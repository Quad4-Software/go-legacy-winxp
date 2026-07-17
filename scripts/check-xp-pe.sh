#!/usr/bin/env bash
# Static checks for Windows XP PE target and forbidden Vista+ imports.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GO="${GO:-$ROOT/bin/go}"
OUTDIR="${1:-$ROOT/docker/xp/oem}"

if [[ ! -x "$GO" ]]; then
  echo "missing go binary at $GO (build with cd src && ./make.bash first)" >&2
  exit 1
fi

mkdir -p "$OUTDIR"
export CGO_ENABLED=0

echo "building windows/386 and windows/amd64 smoke binaries"
GOOS=windows GOARCH=386 "$GO" build -C "$ROOT/testdata/xp-smoke" -o "$OUTDIR/xp-smoke-386.exe" .
GOOS=windows GOARCH=amd64 "$GO" build -C "$ROOT/testdata/xp-smoke" -o "$OUTDIR/xp-smoke-amd64.exe" .

# Also stage copies for shared-folder retests.
SHARED="$ROOT/docker/xp/shared"
mkdir -p "$SHARED"
cp -f "$OUTDIR/xp-smoke-386.exe" "$SHARED/xp-smoke-386.exe"
cp -f "$OUTDIR/xp-smoke-amd64.exe" "$SHARED/xp-smoke-amd64.exe"

python3 - "$OUTDIR/xp-smoke-386.exe" "$OUTDIR/xp-smoke-amd64.exe" <<'PY'
import struct
import sys

FORBIDDEN = {
    b"CreateWaitableTimerExW",
    b"GetQueuedCompletionStatusEx",
    b"AddVectoredContinueHandler",
    b"RaiseFailFastException",
    b"WerGetFlags",
    b"WerSetFlags",
    b"GetErrorMode",
    b"SetFileInformationByHandle",
    b"GetFinalPathNameByHandle",
}

def read_u16(data, off):
    return struct.unpack_from("<H", data, off)[0]

def read_u32(data, off):
    return struct.unpack_from("<I", data, off)[0]

def pe_info(path):
    with open(path, "rb") as f:
        data = f.read()
    if data[:2] != b"MZ":
        raise SystemExit(f"{path}: not MZ")
    pe_off = read_u32(data, 0x3C)
    if data[pe_off:pe_off + 4] != b"PE\0\0":
        raise SystemExit(f"{path}: missing PE signature")
    opt = pe_off + 24
    magic = read_u16(data, opt)
    major_os = read_u16(data, opt + 40)
    minor_os = read_u16(data, opt + 42)
    major_sub = read_u16(data, opt + 48)
    minor_sub = read_u16(data, opt + 50)
    if magic == 0x10B:
        dd = opt + 96
    elif magic == 0x20B:
        dd = opt + 112
    else:
        raise SystemExit(f"{path}: unknown optional magic 0x{magic:x}")
    import_rva = read_u32(data, dd + 8)
    import_size = read_u32(data, dd + 12)
    num_sections = read_u16(data, pe_off + 6)
    size_opt = read_u16(data, pe_off + 20)
    sec = pe_off + 24 + size_opt

    def rva_to_off(rva):
        for i in range(num_sections):
            s = sec + i * 40
            va = read_u32(data, s + 12)
            raw_size = read_u32(data, s + 16)
            raw_ptr = read_u32(data, s + 20)
            virt_size = read_u32(data, s + 8)
            size = max(raw_size, virt_size)
            if va <= rva < va + size and raw_ptr:
                return raw_ptr + (rva - va)
        return None

    names = set()
    if import_rva and import_size:
        off = rva_to_off(import_rva)
        if off is None:
            raise SystemExit(f"{path}: cannot map import directory")
        while True:
            name_rva = read_u32(data, off + 12)
            first_thunk = read_u32(data, off + 16)
            if name_rva == 0 and first_thunk == 0:
                break
            name_off = rva_to_off(name_rva)
            dll = data[name_off:data.find(b"\0", name_off)].decode("ascii", "replace")
            ilt_rva = read_u32(data, off + 0) or first_thunk
            thunk = rva_to_off(ilt_rva)
            entry_size = 8 if magic == 0x20B else 4
            while True:
                if entry_size == 8:
                    raw = struct.unpack_from("<Q", data, thunk)[0]
                    ordinal_flag = 1 << 63
                else:
                    raw = read_u32(data, thunk)
                    ordinal_flag = 1 << 31
                if raw == 0:
                    break
                if raw & ordinal_flag == 0:
                    hint_rva = raw & 0x7FFFFFFF
                    hint_off = rva_to_off(hint_rva)
                    if hint_off is not None:
                        fn = data[hint_off + 2:data.find(b"\0", hint_off + 2)]
                        names.add(fn)
                thunk += entry_size
            off += 20
            _ = dll
    return major_os, minor_os, major_sub, minor_sub, names

failed = False
for path in sys.argv[1:]:
    major_os, minor_os, major_sub, minor_sub, names = pe_info(path)
    print(f"{path}: OS={major_os}.{minor_os} Subsystem={major_sub}.{minor_sub}")
    if major_os != 5 or minor_os != 1 or major_sub != 5 or minor_sub != 1:
        print(f"  ERROR: expected PE 5.1 target, got OS {major_os}.{minor_os} subsystem {major_sub}.{minor_sub}")
        failed = True
    bad = sorted(n.decode("ascii", "replace") for n in (names & FORBIDDEN))
    if bad:
        print(f"  ERROR: forbidden static imports: {', '.join(bad)}")
        failed = True
    else:
        print("  OK: no forbidden Vista+ static imports")

if failed:
    raise SystemExit(1)
print("PE checks passed")
PY
