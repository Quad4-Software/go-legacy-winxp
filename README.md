# The Go Programming Language

**go-legacy-winxp** is a fork of [go-legacy-win7](https://github.com/thongtech/go-legacy-win7) that extends legacy Windows support down to Windows XP and Server 2003, while keeping Windows 7 through Windows 11 compatibility and the restored deprecated `go get` behaviour from the win7 fork.

![Gopher image](https://golang.org/doc/gopher/fiveyears.jpg)
_Gopher image by [Renee French][rf], licensed under [Creative Commons 4.0 Attribution licence][cc4-by]._

## Differences from Upstream Go

1. **Windows XP through Windows 11 Support**  
   While the official Go project dropped support for Windows 7 and older in Go 1.21, this fork maintains compatibility with Windows XP, Server 2003, Windows 7, 8, 8.1, Server 2008 R2, Server 2012, and Server 2012 R2.

   Windows XP targets use PE major version 5 and avoid linking against Vista+ APIs that are loaded dynamically at runtime when available.

2. **Classic `go get` Behaviour**  
   This fork allows for the deprecated `go get` behaviour when `GO111MODULE` is set to "off" or "auto". This means:

   - In `GOPATH/src`, `go get` and `go install` can operate in `GOPATH` mode.
   - Outside of `GOPATH/src`, these commands can use module-aware mode when appropriate.

3. **Compatibility Notes**  
   Some newer Go features may not be fully compatible with the oldest Windows versions. Use `GOARCH=386` for Windows XP when possible. The race detector build remains intended for Windows 10+ only.

## Changes in Each Release

Current release includes all [go-legacy-win7](https://github.com/thongtech/go-legacy-win7) modifications, plus:

- Set PE minimum target major version to 5 for Windows XP binaries
- Load Vista+ kernel32 APIs dynamically instead of static import (`CreateWaitableTimerExW`, `GetErrorMode`, `GetQueuedCompletionStatusEx`, `RaiseFailFastException`, `WerGetFlags`, `WerSetFlags`, `AddVectoredContinueHandler`)
- Fall back to `SetUnhandledExceptionFilter` when `AddVectoredContinueHandler` is unavailable
- Fall back to `GetQueuedCompletionStatus` in the runtime netpoller when `GetQueuedCompletionStatusEx` is unavailable
- Fall back to `SetErrorMode` when `GetErrorMode` is unavailable
- Fall back to `CancelIo` and no-op proc-thread attribute helpers when those APIs are unavailable
- Fall back to `NtSetInformationFile` when `SetFileInformationByHandle` and `GetFinalPathNameByHandle` are unavailable
- Includes all improvements and bug fixes from the corresponding upstream Go release

Inherited from go-legacy-win7:

- Switched back to `RtlGenRandom` from `ProcessPrng`
- Added back `LoadLibraryA` fallback to load system libraries
- Added back `sysSocket` fallback for socket syscalls
- Added back Windows 7 console handle workaround
- Added back 5ms sleep on Windows 7/8 in (\*Process).Wait
- Restored deprecated `go get` behaviour for use outside modules
- Reverted to the previous `removeall_noat` variant for Windows
- Rolled back `race_windows.syso` to the previous compatible version
- Added `FindFirstFile`/`FindNextFile` fallback for old SMB shares

We provide two build options for Windows amd64:

- **Standard build**  
  Maximum compatibility with legacy Windows, including XP, with the reverted race detector.

- **Race detector build** (version suffix `-race`)  
  Uses Go's latest stable race detector without modifications. Recommended for Windows 10+ when running race tests.

## Download and Install

Binary releases are published from this repository's Releases page after the first workflow run.

### Before you begin
To avoid PATH/GOROOT conflicts and mixed toolchains, uninstall any existing Go installation first.

#### Windows Installation

1. Download the `go-legacy-winxp-<version>.windows_<arch>.zip` file.
2. Extract the ZIP to `C:\` (or any preferred location). This will create a `go-legacy-winxp` folder.
3. Add the following to your system environment variables:
   - Add `C:\go-legacy-winxp\bin` (or your chosen path) to the system `PATH`.
   - Set `GOROOT` to `C:\go-legacy-winxp` (or your chosen path).
4. Add the following to your user environment variables:
   - Add `%USERPROFILE%\go\bin` to the user `PATH`.
   - Set `GOPATH` to `%USERPROFILE%\go`.

#### macOS and Linux Installation

1. Download the appropriate `go-legacy-winxp-<version>.<os>_<arch>.tar.gz` file.

   - For macOS: `go-legacy-winxp-<version>.darwin_<arch>.tar.gz`
   - For Linux: `go-legacy-winxp-<version>.linux_<arch>.tar.gz`

2. Extract the archive to `/usr/local`:

   ```
   sudo tar -C /usr/local -xzf go-legacy-winxp-<version>.<os>_<arch>.tar.gz
   ```

3. Add the following to your shell configuration file:

   - For bash, add to `~/.bash_profile` or `~/.bashrc`
   - For zsh, add to `~/.zshrc`

   ```bash
   export GOROOT=/usr/local/go-legacy-winxp
   export GOPATH=$HOME/go
   export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
   ```

4. Apply the changes:

   - For bash: `source ~/.bash_profile` or `source ~/.bashrc`
   - For zsh: `source ~/.zshrc`

   Note:

   - On macOS Catalina and later, zsh is the default shell.
   - On most Linux distributions, bash is the default shell.

After installation, verify the installation by opening a **new terminal** and running:

```
go version
```

### Install From Source

To install from source, please follow the steps on the [official website](https://go.dev/doc/install/source).

## Contributing

Feedback and issue reports are welcome, and we encourage you to open pull requests to contribute to the project.

Note that the Go project uses the issue tracker for bug reports and
proposals only. See https://go.dev/wiki/Questions for a list of
places to ask questions about the Go language.

[rf]: https://reneefrench.blogspot.com/
[cc4-by]: https://creativecommons.org/licenses/by/4.0/
