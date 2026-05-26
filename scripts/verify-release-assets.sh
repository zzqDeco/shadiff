#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <dist-dir> <tag>" >&2
  exit 2
fi

dist_dir="$1"
tag="$2"

if [[ ! -d "${dist_dir}" ]]; then
  echo "release asset directory not found: ${dist_dir}" >&2
  exit 1
fi

required_assets=(
  "shadiff_${tag}_linux_amd64.tar.gz"
  "shadiff_${tag}_linux_arm64.tar.gz"
  "shadiff_${tag}_darwin_amd64.tar.gz"
  "shadiff_${tag}_darwin_arm64.tar.gz"
  "shadiff_${tag}_windows_amd64.zip"
  "shadiff_${tag}_windows_arm64.zip"
  "checksums.txt"
)

for asset in "${required_assets[@]}"; do
  if [[ ! -f "${dist_dir}/${asset}" ]]; then
    echo "missing release asset: ${asset}" >&2
    exit 1
  fi
done

if command -v sha256sum >/dev/null 2>&1; then
  (cd "${dist_dir}" && sha256sum -c checksums.txt)
elif command -v shasum >/dev/null 2>&1; then
  (cd "${dist_dir}" && shasum -a 256 -c checksums.txt)
else
  echo "sha256sum or shasum is required to verify release checksums" >&2
  exit 1
fi

for archive in "${dist_dir}"/shadiff_"${tag}"_{linux,darwin}_*.tar.gz; do
  if ! tar -tzf "${archive}" | grep -qx "shadiff"; then
    echo "archive does not contain shadiff binary: ${archive}" >&2
    exit 1
  fi
done

for archive in "${dist_dir}"/shadiff_"${tag}"_windows_*.zip; do
  if ! unzip -Z1 "${archive}" | grep -qx "shadiff.exe"; then
    echo "archive does not contain shadiff.exe binary: ${archive}" >&2
    exit 1
  fi
done

case "$(uname -s)-$(uname -m)" in
  Linux-x86_64|Linux-amd64)
    tmp_dir="$(mktemp -d)"
    trap 'rm -rf "${tmp_dir}"' EXIT
    tar -xzf "${dist_dir}/shadiff_${tag}_linux_amd64.tar.gz" -C "${tmp_dir}"
    if ! "${tmp_dir}/shadiff" version | grep -q "${tag}"; then
      echo "linux amd64 binary version output does not include ${tag}" >&2
      exit 1
    fi
    ;;
  *)
    echo "skipping linux amd64 binary execution on $(uname -s)/$(uname -m)"
    ;;
esac
