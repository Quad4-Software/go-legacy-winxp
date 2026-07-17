---
name: upstream-update
description: >-
  Safely sync go-legacy-winxp from thongtech/go-legacy-win7 while preserving
  Windows XP patches and fork identity. Use when updating Go versions, pulling
  upstream win7 tags, rebasing onto v1.x.y-N, or refreshing patches/0010.
---

# Upstream update (win7 -> winxp)

## Goal

Move this tree to a newer [go-legacy-win7](https://github.com/thongtech/go-legacy-win7) tag (example `v1.26.5-1`) **without losing XP patches**.

## Preconditions

- Working tree clean or changes stashed intentionally
- Remotes present:

```bash
git remote -v
# origin   -> Quad4-Software/go-legacy-winxp
# upstream -> https://github.com/thongtech/go-legacy-win7.git
```

Add upstream if missing:

```bash
git remote add upstream https://github.com/thongtech/go-legacy-win7.git
```

- Pick the win7 tag that matches the desired Go version (`git ls-remote --tags upstream 'v1.*'`).

## Safe sync procedure

Do **not** `git merge` stock golang/go into this repo.

### 1. Preserve XP and fork files

Copy aside before replacing the tree:

- All XP sources listed in `AGENTS.md`
- `README.md`
- `.github/workflows/go-build.yml`
- `.github/workflows/xp-test.yml`
- `patches/0010-Add-Windows-XP-support.patch`
- `scripts/`
- `docker/`
- `testdata/xp-smoke/`
- `AGENTS.md`
- `skills/`

### 2. Check out upstream win7 tag into a temp clone

```bash
TAG=v1.26.5-1   # change as needed
rm -rf /tmp/go-legacy-win7
git clone https://github.com/thongtech/go-legacy-win7.git /tmp/go-legacy-win7
git -C /tmp/go-legacy-win7 checkout "$TAG"
```

Confirm `/tmp/go-legacy-win7/VERSION` is the expected `go1.x.y`.

### 3. Replace tree contents from win7 (keep `.git`)

From the repo root:

```bash
rsync -a --delete \
  --exclude='.git' \
  --exclude='bin' \
  --exclude='pkg' \
  --exclude='AGENTS.md' \
  --exclude='skills' \
  --exclude='scripts' \
  --exclude='docker' \
  --exclude='testdata' \
  /tmp/go-legacy-win7/ ./
```

If you excluded less, restore the preserve list from step 1 afterward.

### 4. Restore XP sources and fork identity

Copy the preserved XP sources and fork files back on top of the win7 tree.

Conflict check: if official or win7 changes touched an XP file (rare on point releases), merge carefully. Prefer keeping dynamic-load XP behavior.

### 5. Regenerate patch 0010

Diff current XP files against the win7 tag checkout and rewrite `patches/0010-Add-Windows-XP-support.patch` so it applies cleanly to that win7 base. Include every XP file from `AGENTS.md`, not only the historical subset.

Also accept updated win7 patches under `patches/0001` through `patches/0009` from upstream.

### 6. Verify

```bash
cat VERSION
cd src && ./make.bash
cd ..
./bin/go version
./scripts/check-xp-pe.sh
```

Optional live XP (needs Docker + `/dev/kvm`):

```bash
./scripts/test-xp-docker.sh
```

### 7. Diff sanity

```bash
# XP files must still differ from win7
diff -u /tmp/go-legacy-win7/src/runtime/os_windows.go src/runtime/os_windows.go | head
# Fork branding intact
grep -n 'go-legacy-winxp\|branch=master' README.md .github/workflows/go-build.yml
```

## Notes

- Point releases often change `os/root_*.go` and similar. Those usually do **not** conflict with XP files. Still check `git diff --stat` after restore.
- Patch `0006` (removeall_noat) is often refreshed by win7 when `os.Root` changes. Take upstream's version.
- Do not commit `bin/`, `pkg/`, or `docker/xp/storage`.
- Commit only when the user asks.

## Failure modes

| Symptom | Fix |
|---------|-----|
| PE target is 6.1 again | `pe.go` lost XP restore |
| Forbidden Vista+ static imports | `os_windows.go` / vendor zsyscall lost dynamic load |
| README says go-legacy-win7 | Fork identity not restored |
| Workflow falls back to `main` | Restored wrong `go-build.yml` |
