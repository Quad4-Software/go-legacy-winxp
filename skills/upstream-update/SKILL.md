---
name: upstream-update
description: >-
  Safely sync go-legacy-winxp from thongtech/go-legacy-win7 while preserving
  Windows XP patches and fork identity. Use when updating Go versions, pulling
  upstream win7 tags, rebasing onto v1.x.y-N, or refreshing patches/0010.
---

# Upstream update (win7 -> winxp)

## Goal

Move this tree to a newer [go-legacy-win7](https://github.com/thongtech/go-legacy-win7) tag (example `v1.27.0-2`) **without losing XP patches**.

## Preconditions

- Working tree clean (or pass `--force` to scaffold)
- Remotes present:

```bash
git remote -v
# origin   -> Quad4-Software/go-legacy-winxp
# upstream -> https://github.com/thongtech/go-legacy-win7.git
```

`scripts/scaffold-version.sh` adds `upstream` if missing.

- Pick the win7 tag (`git ls-remote --tags upstream 'v1.*'`).
  Scheduled workflow `.github/workflows/watch-upstream.yml` opens an issue when a newer tag appears.

## Sync (preferred)

Do **not** `git merge` stock golang/go into this repo.

```bash
./scripts/scaffold-version.sh v1.27.0-2
```

Flags:

- `--work-dir DIR`  win7 checkout path (default `/tmp/go-legacy-win7`)
- `--skip-build`    skip `src/make.bash`
- `--skip-pe`       skip `scripts/check-xp-pe.sh`
- `--force`         allow dirty worktree

What it does:

1. Preserves XP sources from `scripts/xp-files.list` and fork paths from `scripts/fork-files.list`
2. Checks out the win7 tag into the work dir
3. Rsyncs win7 into this tree (keeps `.git`, excludes fork dirs)
4. Restores preserved XP + fork identity
5. Regenerates `patches/0010-Add-Windows-XP-support.patch` via `scripts/regenerate-xp-patch.sh`
6. Builds and runs PE checks unless skipped

Does not commit or push.

## Regenerate patch only

If the tree is already on the right win7 base and only `0010` is stale:

```bash
./scripts/regenerate-xp-patch.sh /tmp/go-legacy-win7
```

The argument must be a checkout of the matching win7 tag.

## Verify

```bash
cat VERSION
./bin/go version
./scripts/check-xp-pe.sh
```

Optional live XP (needs Docker + `/dev/kvm`):

```bash
./scripts/test-xp-docker.sh
```

Sanity:

```bash
diff -u /tmp/go-legacy-win7/src/runtime/os_windows.go src/runtime/os_windows.go | head
grep -n 'go-legacy-winxp\|branch=master' README.md .github/workflows/go-build.yml
```

## Notes

- Point releases often change `os/root_*.go` and similar. Those usually do **not** conflict with XP files. Still check `git diff --stat` after scaffold.
- Patch `0006` (removeall_noat) is often refreshed by win7 when `os.Root` changes. Take upstream's version (scaffold keeps win7 `0001`-`0009`).
- Do not commit `bin/`, `pkg/`, or `docker/xp/storage`.
- Commit only when the user asks.
- File lists are authoritative: `scripts/xp-files.list` and `scripts/fork-files.list`.

## Failure modes

| Symptom | Fix |
|---------|-----|
| PE target is 6.1 again | XP restore missed `pe.go` check `xp-files.list` |
| Forbidden Vista+ static imports | XP restore missed runtime/zsyscall dynamic load |
| README says go-legacy-win7 | Fork identity not restored check `fork-files.list` |
| Workflow falls back to `main` | Restored wrong `go-build.yml` |
| Dirty tree refused | Stash/commit or pass `--force` |
