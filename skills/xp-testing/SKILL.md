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
   - Requires Docker. Uses KVM when it is safe, otherwise TCG (`XP_KVM=N`)
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
# Nested hosts often need TCG: XP_KVM=N ./scripts/test-xp-docker.sh
```

## Commands

```bash
./scripts/check-xp-pe.sh
./scripts/fetch-xp-iso.sh
./scripts/test-xp-docker.sh
```

Environment:

- `XP_TEST_TIMEOUT` seconds to wait for guest `PASS`/`FAIL` (KVM default `7200` first install / `1800` cached disk, TCG default `14400` / `3600`)
- `XP_KVM` `Y` or `N` (unset: auto, nested hypervisor uses TCG)
- `XP_KEEP=1` leaves the guest running after the script exits
- `GO` path to go binary (default `./bin/go`)

Web viewer during Docker runs: `http://127.0.0.1:8006/`

## Storage / filesystem caveat

Windows Setup is unreliable on **btrfs**-backed disks. `test-xp-docker.sh` places the disk image under `/var/tmp/go-legacy-winxp-xp-storage` when the project is on btrfs. Do not delete that cache unless you want a full XP reinstall.

The XP SP3 ISO is fetched from Internet Archive by `scripts/fetch-xp-iso.sh` and mounted as `custom.iso` so dockurr does not rely on dead bobpony/files.dog mirrors.

## Result protocol

- Ignore intermediate markers such as `starting` in `result.txt`
- Success only when the file content is `PASS`, `smoke.out` contains `go-legacy-winxp smoke ok`, `smoke: all checks passed`, `goarch=386`, `init_rfc3339=` (issue 10 `time.Zone` path), and `check time_json: ok`
- `smoke.exit` must be `0` on PASS
- Guest also writes `smoke.log` (status + exit + output). Host writes `smoke-report.txt` after the run
- Failure when content is `FAIL`, output validation fails, or on timeout
- Prefer writing final status only as `PASS` or `FAIL` from guest scripts

## CI

`.github/workflows/ci.yml` builds the toolchain and runs `check-xp-pe.sh` on pull requests and `master`.

`.github/workflows/xp-test.yml` is `workflow_dispatch` only. It probes `/dev/kvm` before cloning the tree. Pull requests use `ci.yml` instead of waiting on QEMU.

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
- `RegLoadMUIStringW`
- `GetFileInformationByHandleEx`
- `GetTempPath2W`

## After changing XP runtime / link code

1. `./scripts/check-xp-pe.sh`
2. `./scripts/test-xp-docker.sh`
3. Regenerate patch 0010 against the matching win7 checkout:
   `./scripts/regenerate-xp-patch.sh /tmp/go-legacy-win7`
   (see `skills/upstream-update/SKILL.md`)
