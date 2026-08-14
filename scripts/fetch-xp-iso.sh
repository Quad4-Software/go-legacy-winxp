#!/usr/bin/env bash
# Fetch the Windows XP SP3 x86 ISO used by the Docker smoke test.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ISO_DIR="$ROOT/docker/xp/iso"
ISO_PATH="$ISO_DIR/xp-sp3.iso"
ISO_URL="https://archive.org/download/en_windows_xp_professional_with_service_pack_3_x86_cd_x14-80428/en_windows_xp_professional_with_service_pack_3_x86_cd_x14-80428.iso"
ISO_SHA256="62b6c91563bad6cd12a352aa018627c314cfc5162d8e9f8af0756a642e602a46"
ISO_SIZE=617756672

mkdir -p "$ISO_DIR"

iso_ok() {
  [[ -f "$ISO_PATH" ]] || return 1
  [[ "$(stat -c%s "$ISO_PATH")" -eq "$ISO_SIZE" ]] || return 1
  sha256sum -c - <<< "${ISO_SHA256}  ${ISO_PATH}" >/dev/null 2>&1
}

if iso_ok; then
  echo "XP ISO already present at $ISO_PATH"
  exit 0
fi

echo "fetching Windows XP SP3 ISO to $ISO_PATH"
tmp="${ISO_PATH}.partial"
rm -f "$tmp"

curl -fL --retry 5 --retry-delay 5 \
  -o "$tmp" \
  "$ISO_URL"

actual_size="$(stat -c%s "$tmp")"
if [[ "$actual_size" -ne "$ISO_SIZE" ]]; then
  echo "unexpected ISO size: got $actual_size want $ISO_SIZE" >&2
  rm -f "$tmp"
  exit 1
fi

if ! sha256sum -c - <<< "${ISO_SHA256}  ${tmp}"; then
  echo "ISO checksum mismatch" >&2
  rm -f "$tmp"
  exit 1
fi

mv -f "$tmp" "$ISO_PATH"
echo "XP ISO ready at $ISO_PATH"
