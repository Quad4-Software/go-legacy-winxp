# AGENTS.md

Guidance for automated agents and humans maintaining this repository.

## What this repo is

**go-legacy-winxp** is a fork of [go-legacy-win7](https://github.com/thongtech/go-legacy-win7) that keeps Go working on Windows XP and Server 2003 while also supporting Windows 7 through Windows 11.

It inherits win7 work such as `RtlGenRandom`, `LoadLibraryA` fallbacks, `sysSocket` fallbacks, and restored GOPATH-mode `go get`. On top of that it adds XP-specific PE targeting and dynamic loading of Vista+ APIs.

Upstream Go itself dropped Windows 7 and older in Go 1.21. This tree is not upstream golang/go.

## Remotes

| Remote | URL | Role |
|--------|-----|------|
| `origin` | `git@github.com:Quad4-Software/go-legacy-winxp.git` | This fork |
| `upstream` | `https://github.com/thongtech/go-legacy-win7.git` | win7 parent to sync from |

Official Go tags live at `https://go.googlesource.com/go` (for reference only). Day-to-day version bumps come from **upstream** win7 tags such as `v1.26.5-1`, not by re-applying every win7 patch onto stock Go.

Default branch is `master`.

## Layout that matters

```
VERSION                      Go version string for this tree
patches/                     Ordered patch series (0001-0009 win7 lineage, 0010 XP)
patches/0010-*.patch         Full XP delta versus current win7 base
src/                         Go source (already patched in-tree)
scripts/xp-files.list        XP sources preserved across sync
scripts/fork-files.list      Fork identity paths preserved across sync
scripts/scaffold-version.sh  Sync onto a win7 tag
scripts/regenerate-xp-patch.sh
scripts/check-xp-pe.sh       Static PE 5.1 and forbidden-import checks
scripts/test-xp-docker.sh
docker/xp/                   dockurr/windows VERSION=xp smoke harness
testdata/xp-smoke/           Guest smoke program
.github/workflows/           Release, XP test, and upstream watch
skills/                      Task skills for maintenance agents
```

Build artifacts (`bin/`, `pkg/`, generated `z*.go`) are local. Do not commit them.

## Hard rules

1. **Never drop XP patches when updating.** Preserve every change covered by `patches/0010-Add-Windows-XP-support.patch` plus fork identity files listed below.
2. **Do not replace this tree with stock Go.** Sync from `upstream` win7 tags first.
3. **Keep fork branding.** `README.md` and `.github/workflows/go-build.yml` stay winxp-named (`master` fallback, not win7 `main` / `RELEASE_TOKEN`).
4. **Regenerate `patches/0010-*.patch` after every successful sync** so the series matches the tree.
5. **Do not invent new markdown docs** unless asked. Update `AGENTS.md` and `skills/` when workflows change.
6. **No emojis** in repo text. Prefer plain ASCII in scripts and docs.

## Fork identity files (always keep)

Authoritative list: `scripts/fork-files.list` (dirs and files preserved by scaffold).

Includes README, AGENTS.md, workflows (`go-build`, `xp-test`, `watch-upstream`), `patches/0010`, `scripts/`, `docker/`, `testdata/`, and `skills/`.

## XP source files (must survive upstream sync)

Authoritative list: `scripts/xp-files.list`. Summary:

- `src/cmd/link/internal/ld/pe.go` (PE major version 5)
- `src/cmd/vendor/golang.org/x/sys/windows/zsyscall_windows.go`
- `src/internal/runtime/syscall/windows/defs_windows.go`
- `src/internal/syscall/windows/at_windows.go`
- `src/internal/syscall/windows/syscall_windows.go`
- `src/internal/syscall/windows/symlink_windows.go`
- `src/internal/syscall/windows/types_windows.go`
- `src/internal/syscall/windows/zsyscall_windows.go`
- `src/net/fd_windows.go` (ignore unsupported UDP WSAIoctl on XP)
- `src/os/dir_windows.go`
- `src/os/file_windows.go`
- `src/os/types_windows.go`
- `src/runtime/netpoll_windows.go`
- `src/runtime/os_windows.go`
- `src/runtime/signal_windows.go`
- `src/syscall/zsyscall_windows.go`

Vista+ APIs must stay **dynamically loaded** (not `cgo_import_dynamic` static imports) when missing on XP.

## Skills

Read and follow these before the matching task:

| Skill | When |
|-------|------|
| [skills/upstream-update/SKILL.md](skills/upstream-update/SKILL.md) | Bump Go / sync from win7 / apply a new `vX.Y.Z-N` tag |
| [skills/xp-testing/SKILL.md](skills/xp-testing/SKILL.md) | Validate XP PE target, imports, or run Docker XP smoke tests |

## Common commands

Sync onto a win7 tag (preserves XP + fork identity, regenerates `0010`):

```bash
./scripts/scaffold-version.sh v1.27.0-2
```

Regenerate XP patch only (needs matching win7 checkout):

```bash
./scripts/regenerate-xp-patch.sh /tmp/go-legacy-win7
```

Bootstrap toolchain (needs a host Go for bootstrap):

```bash
cd src
./make.bash
```

Use the built toolchain:

```bash
./bin/go version
export PATH="$PWD/bin:$PATH"
```

Static XP checks (builds `windows/386` and `windows/amd64` smoke binaries):

```bash
./scripts/check-xp-pe.sh
```

Live XP smoke test via Docker + KVM (`dockurr/windows`, `VERSION=xp`):

```bash
./scripts/test-xp-docker.sh
```

Web UI while the guest runs: `http://127.0.0.1:8006/`

First XP boot downloads and installs the guest (slow). Disk cache lives under `/tmp/go-legacy-winxp-xp-storage` when the project filesystem is btrfs. Later boots reuse that disk.

Cross-compile example:

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=386 ./bin/go build -o hello.exe .
```

Expect PE OS and subsystem **5.1** for XP-compatible binaries.

## Releases

`.github/workflows/go-build.yml` is `workflow_dispatch` with a version input. It builds matrix targets and publishes draft GitHub releases. Prefer release branch `release-branch.goX.Y` when present, else `master`.

`.github/workflows/watch-upstream.yml` runs daily and on `workflow_dispatch`. It opens an `upstream-sync` issue when a newer `thongtech/go-legacy-win7` tag (`vX.Y.Z-N`) appears.

## Do not

- Force-push `master` unless explicitly requested
- Commit secrets, `bin/`, `pkg/`, or `docker/xp/storage`
- Treat Wine as proof of XP compatibility
- Assume GitHub-hosted runners have `/dev/kvm` (live XP Docker job may skip there)
