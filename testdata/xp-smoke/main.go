// XP integration smoke test for go-legacy-winxp Docker and CI runs.
package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"time"
)

func main() {
	runtime.GOMAXPROCS(2)
	if len(os.Args) > 1 {
		helperMain(os.Args[1])
		return
	}
	fmt.Println("smoke: starting go-legacy-winxp guest checks")
	fmt.Printf("go-legacy-winxp smoke ok version=%s goos=%s goarch=%s\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)

	passed := 0
	runAllChecks(&passed)

	fmt.Printf("smoke: %d checks passed\n", passed)
	fmt.Println("smoke: all checks passed")
}

// helperMain implements the hidden subcommands used by the os_exec checks.
// The smoke binary re-executes itself so the tests exercise real process
// creation, pipes and signals without depending on cmd.exe.
func helperMain(mode string) {
	switch mode {
	case "__helper_echo":
		// Copy stdin to stdout until EOF.
		_, _ = io.Copy(os.Stdout, os.Stdin)
	case "__helper_sleep":
		secs := 60
		if len(os.Args) > 2 {
			if n, err := strconv.Atoi(os.Args[2]); err == nil {
				secs = n
			}
		}
		time.Sleep(time.Duration(secs) * time.Second)
	case "__helper_env":
		fmt.Println(os.Getenv("XP_SMOKE_ENV"))
	case "__helper_args":
		fmt.Printf("argc=%d arg1=%s\n", len(os.Args), os.Args[1])
	case "__helper_sigint":
		// Wait for os.Interrupt, print markers for the Ctrl+C tests.
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		fmt.Println("READY")
		_ = os.Stdout.Sync()
		<-c
		fmt.Println("GOTINT")
		_ = os.Stdout.Sync()
	}
}
