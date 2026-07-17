---
name: xp-testing
description: >-
  Validate go-legacy-winxp Windows XP compatibility with PE checks and
  dockurr/windows Docker smoke tests. Use when testing XP, checking PE 5.1,
  running scripts/check-xp-pe.sh or scripts/test-xp-docker.sh, or debugging
  XP CI.
---

# XP compatibility testing

## Layers

1. **Static (always)** `./scripts/check-xp-pe.sh`
   - Builds `windows/386` and `windows/amd64` smoke EXEs with this tree's `bin/go`
   - Asserts PE OS and subsystem **5.1**
   - Fails if Vista+ APIs appear as static imports

2. **Live guest (optional)** `./scripts/test-xp-docker.sh`
   - Requires Docker and `/dev/kvm`
   - Runs `dockurr/windows` with `VERSION=xp`
   - Guest runs `docker/xp/oem/install.bat` (first install) or `docker/xp/shared/run.bat` (retest)
   - Host waits for `docker/xp/shared/result.txt` containing only `PASS` or `FAIL`

Wine is **not** an XP oracle.

## Prerequisites

```bash
# Toolchain
cd src && ./make.bash && cd ..
./bin/go version

# Live test only
docker info
ls -l /dev/kvm
```

## Commands

```bash
./scripts/check-xp-pe.sh
./scripts/test-xp-docker.sh
```

Environment:

- `XP_TEST_TIMEOUT` seconds to wait for guest `PASS`/`FAIL` (default `3600`)
- `GO` path to go binary (default `./bin/go`)

Web viewer during Docker runs: `http://127.0.0.1:8006/`

## Storage / filesystem caveat

Windows Setup is unreliable on **btrfs**-backed disks. `test-xp-docker.sh` relocates the disk image to `/tmp/go-legacy-winxp-xp-storage` when the project filesystem is btrfs. Do not delete that cache unless you want a full XP reinstall.

## Result protocol

- Ignore intermediate markers such as `starting` in `result.txt`
- Success only when the file content is `PASS`
- Failure when content is `FAIL` or on timeout
- Prefer writing final status only as `PASS` or `FAIL` from guest scripts

## CI

`.github/workflows/xp-test.yml`:

- `pe-check` job always builds the toolchain and runs `check-xp-pe.sh`
- `xp-docker` job runs only when `/dev/kvm` exists (typically self-hosted). On GitHub-hosted runners it skips with guidance to run locally.

## Forbidden static imports

These must not appear in the PE import table of XP smoke binaries (dynamic `GetProcAddress` / `NewProc` is fine):

- `CreateWaitableTimerExW`
- `GetQueuedCompletionStatusEx`
- `AddVectoredContinueHandler`
- `RaiseFailFastException`
- `WerGetFlags`
- `WerSetFlags`
- `GetErrorMode`
- `SetFileInformationByHandle`
- `GetFinalPathNameByHandle`

## After changing XP runtime / link code

1. `./scripts/check-xp-pe.sh`
2. If KVM available, `./scripts/test-xp-docker.sh`
3. Regenerate `patches/0010-Add-Windows-XP-support.patch` if sources changed (see `skills/upstream-update/SKILL.md`)
