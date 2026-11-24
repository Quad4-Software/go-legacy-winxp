// xp-agent is a guest-side controller for headless Windows XP testing.
//
// It polls a directory (typically the dockurr/windows Samba share) for
// cmd-*.txt files, executes the command they contain, and writes the
// result to out-<name>.txt next to the command file.
//
// Command file format: the whole file is one command line. The first
// whitespace-separated token is the verb; the rest of the line is the
// argument string. Verbs that take a single blob (runsh, consoles) use
// the remainder of the line verbatim, so quoting is not needed.
// An optional second line may contain "timeout <milliseconds>".
//
// Verbs:
//
//	run <exe> <args...>      run hidden, capture combined output, report exit code
//	runsh <cmdline>          run "cmd /c <cmdline>" hidden, capture output
//	console <exe> <args...>  start exe in a new console + process group, report pid
//	consoles <cmdline>       start "cmd /c <cmdline>" in a new console + group
//	sconsole <exe> <args...>  start exe in the agent's own console, new group
//	sconsoles <cmdline>       start "cmd /c <cmdline>" in the agent's console
//	ctrl <event> <pgid>      GenerateConsoleCtrlEvent; event is c|break|0|1|2
//	kill <pid>               TerminateProcess
//	alive <pid>              report whether pid is running and its exit code
//	ps                       list processes as "pid<TAB>name" lines
//	mkdir <path>             MkdirAll
//	sleep <ms>               sleep
//	ping                     health check, writes "pong"
//	quit                     agent exits
package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	createNewConsole         = 0x00000010
	createNewProcessGroup    = 0x00000200
	createNoWindow           = 0x08000000
	createUnicodeEnvironment = 0x00000400
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procGenerateCtrl = kernel32.NewProc("GenerateConsoleCtrlEvent")
	procOpenProcess  = kernel32.NewProc("OpenProcess")
	procCloseHandle  = kernel32.NewProc("CloseHandle")
	procTermProcess  = kernel32.NewProc("TerminateProcess")
	procGetExitCode  = kernel32.NewProc("GetExitCodeProcess")
	procSnapProcess  = kernel32.NewProc("CreateToolhelp32Snapshot")
)

const (
	processTerminate        = 0x0001
	processQueryInformation = 0x0400
	synchronize             = 0x00100000
	stillActive             = 259
)

func main() {
	// The agent shares its console with test processes when delivering
	// CTRL_C_EVENT to the whole group. signal.Ignore is not enough on
	// Windows: it clears the wanted bit so sigsend declines and the
	// default handler still terminates the process. Notify keeps the
	// handler returning TRUE; drain the channel so it survives.
	c := make(chan os.Signal, 8)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
		}
	}()

	share := `\\host.lan\Data`
	if len(os.Args) > 1 {
		share = os.Args[1]
	}
	dir := filepath.Join(share, "agent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "xp-agent: mkdir %s: %v\n", dir, err)
		os.Exit(1)
	}
	logf, err := os.OpenFile(filepath.Join(dir, "agent.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {
		defer logf.Close()
		fmt.Fprintf(logf, "xp-agent started share=%s pid=%d\n", share, os.Getpid())
	}

	for {
		entries, err := os.ReadDir(dir)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasPrefix(name, "cmd-") || !strings.HasSuffix(name, ".txt") {
				continue
			}
			cmdPath := filepath.Join(dir, name)
			id := strings.TrimSuffix(strings.TrimPrefix(name, "cmd-"), ".txt")
			data, err := os.ReadFile(cmdPath)
			if err != nil {
				continue
			}
			out := runCommand(string(data))
			outPath := filepath.Join(dir, "out-"+id+".txt")
			_ = os.WriteFile(outPath, []byte(out), 0o644)
			_ = os.Remove(cmdPath)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func runCommand(spec string) string {
	lines := strings.Split(spec, "\n")
	cmdline := strings.TrimSpace(lines[0])
	if cmdline == "" {
		return "error=empty command\n"
	}
	timeout := 120 * time.Second
	for _, l := range lines[1:] {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "timeout ") {
			if ms, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(l, "timeout "))); err == nil {
				timeout = time.Duration(ms) * time.Millisecond
			}
		}
	}

	verb, rest, _ := strings.Cut(cmdline, " ")
	rest = strings.TrimSpace(rest)

	switch verb {
	case "ping":
		return "pong\n"
	case "quit":
		go func() { time.Sleep(200 * time.Millisecond); os.Exit(0) }()
		return "bye\n"
	case "sleep":
		ms, _ := strconv.Atoi(rest)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return "slept\n"
	case "mkdir":
		if err := os.MkdirAll(rest, 0o755); err != nil {
			return "error=" + err.Error() + "\n"
		}
		return "ok\n"
	case "run", "runsh":
		return runHidden(verb, rest, timeout)
	case "console", "consoles", "sconsole", "sconsoles", "aconsole", "aconsoles":
		return startConsole(verb, rest)
	case "ctrl":
		return sendCtrl(rest)
	case "kill":
		return killPID(rest)
	case "alive":
		return alive(rest)
	case "ps":
		return listProcesses()
	default:
		return "error=unknown verb " + verb + "\n"
	}
}

// runHidden runs a command without a console window and captures output.
func runHidden(verb, rest string, timeout time.Duration) string {
	var cmd *exec.Cmd
	if verb == "runsh" {
		cmd = exec.Command("cmd.exe", "/c", rest)
	} else {
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return "error=run needs an exe\n"
		}
		cmd = exec.Command(fields[0], fields[1:]...)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow | createUnicodeEnvironment}
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		return fmt.Sprintf("error=%v\n%s", err, out)
	}
	return fmt.Sprintf("exit=%d\n--- output ---\n%s", code, out)
}

// startConsole launches a process in its own console and process group so
// that console control events can be targeted at it later. The s* verbs
// keep the agent's console and only start a new process group, which is
// what GenerateConsoleCtrlEvent can reach (CTRL_C_EVENT only works with
// process group 0, meaning every process on the caller's console).
//
// The a* verbs share the agent's console AND process group: no
// CREATE_NEW_PROCESS_GROUP, so the children receive CTRL_C_EVENT exactly
// like a normal `go run` under cmd.exe. This is the faithful repro for
// issue #12.
func startConsole(verb, rest string) string {
	shared := strings.HasPrefix(verb, "s") || strings.HasPrefix(verb, "a")
	inner := strings.TrimLeft(verb, "sa")
	var cmd *exec.Cmd
	if inner == "consoles" {
		cmd = exec.Command("cmd.exe", "/c", rest)
	} else {
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return "error=console needs an exe\n"
		}
		cmd = exec.Command(fields[0], fields[1:]...)
	}
	flags := uint32(createUnicodeEnvironment)
	if !shared {
		flags |= createNewConsole
	}
	if !strings.HasPrefix(verb, "a") {
		// s* verbs start a new process group (CTRL_C-immune, CTRL_BREAK-able).
		// a* verbs stay in the agent's group so CTRL_C_EVENT reaches them.
		flags |= createNewProcessGroup
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: flags,
	}
	// Detach stdio so the child does not share the agent's (nonexistent) pipes.
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return "error=" + err.Error() + "\n"
	}
	pid := cmd.Process.Pid
	// Do not wait; release the handle reference. The process group id equals
	// the pid of the process created with CREATE_NEW_PROCESS_GROUP.
	go cmd.Wait()
	return fmt.Sprintf("pid=%d pgid=%d\n", pid, pid)
}

// sendCtrl calls GenerateConsoleCtrlEvent. event: c|break|0|1|2
func sendCtrl(arg string) string {
	fields := strings.Fields(arg)
	if len(fields) != 2 {
		return "error=ctrl needs <event> <pgid>\n"
	}
	var event uint32
	switch strings.ToLower(fields[0]) {
	case "c", "ctrl-c", "ctrl_c":
		event = 0 // CTRL_C_EVENT
	case "break", "ctrl-break", "ctrl_break":
		event = 1 // CTRL_BREAK_EVENT
	case "close":
		event = 2 // CTRL_CLOSE_EVENT
	case "0":
		event = 0
	case "1":
		event = 1
	case "2":
		event = 2
	default:
		return "error=unknown event " + fields[0] + "\n"
	}
	pgid, err := strconv.ParseUint(fields[1], 10, 32)
	if err != nil {
		return "error=bad pgid\n"
	}
	r, _, e := procGenerateCtrl.Call(uintptr(event), uintptr(pgid))
	if r == 0 {
		return fmt.Sprintf("error=GenerateConsoleCtrlEvent failed errno=%d\n", e)
	}
	return fmt.Sprintf("sent event=%d pgid=%d\n", event, pgid)
}

func openProcess(pid uint32, access uint32) uintptr {
	h, _, _ := procOpenProcess.Call(uintptr(access), 0, uintptr(pid))
	return h
}

func killPID(arg string) string {
	pid, err := strconv.ParseUint(strings.TrimSpace(arg), 10, 32)
	if err != nil {
		return "error=bad pid\n"
	}
	h := openProcess(uint32(pid), processTerminate)
	if h == 0 {
		return fmt.Sprintf("error=OpenProcess failed for pid=%d\n", pid)
	}
	defer procCloseHandle.Call(h)
	r, _, e := procTermProcess.Call(h, 1)
	if r == 0 {
		return fmt.Sprintf("error=TerminateProcess failed errno=%d\n", e)
	}
	return fmt.Sprintf("killed pid=%d\n", pid)
}

func alive(arg string) string {
	pid, err := strconv.ParseUint(strings.TrimSpace(arg), 10, 32)
	if err != nil {
		return "error=bad pid\n"
	}
	h := openProcess(uint32(pid), processQueryInformation|synchronize)
	if h == 0 {
		return fmt.Sprintf("pid=%d state=dead or inaccessible\n", pid)
	}
	defer procCloseHandle.Call(h)
	var code uint32
	procGetExitCode.Call(h, uintptr(unsafe.Pointer(&code)))
	if code == stillActive {
		return fmt.Sprintf("pid=%d state=running\n", pid)
	}
	return fmt.Sprintf("pid=%d state=exited code=%d\n", pid, code)
}

func listProcesses() string {
	const th32csSnapProcess = 0x2
	const maxPath = 260
	snap, _, _ := procSnapProcess.Call(th32csSnapProcess, 0)
	if snap == ^uintptr(0) {
		return "error=CreateToolhelp32Snapshot failed\n"
	}
	defer procCloseHandle.Call(snap)

	type processEntry32 struct {
		Size            uint32
		Usage           uint32
		ProcessID       uint32
		DefaultHeapID   uintptr
		ModuleID        uint32
		Threads         uint32
		ParentProcessID uint32
		PriClassBase    int32
		Flags           uint32
		ExeFile         [maxPath]uint16
	}
	var pe processEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))

	procProcess32First := kernel32.NewProc("Process32FirstW")
	procProcess32Next := kernel32.NewProc("Process32NextW")

	var b strings.Builder
	ok, _, _ := procProcess32First.Call(snap, uintptr(unsafe.Pointer(&pe)))
	for ok != 0 {
		name := syscall.UTF16ToString(pe.ExeFile[:])
		fmt.Fprintf(&b, "%d\t%s\n", pe.ProcessID, name)
		pe.Size = uint32(unsafe.Sizeof(pe))
		ok, _, _ = procProcess32Next.Call(snap, uintptr(unsafe.Pointer(&pe)))
	}
	return b.String()
}
