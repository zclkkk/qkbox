#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"
VERSION="${QKBOX_VERSION:-$(node -p 'require("./package.json").version' 2>/dev/null || echo "0.1.0")}"
ARCH="${QKBOX_DEB_ARCH:-amd64}"
PKG_ROOT="${ROOT_DIR}/dist/deb/qkbox_${VERSION}_${ARCH}"
OUT_DIR="${ROOT_DIR}/dist/packages"
DEB_PATH="${OUT_DIR}/qkbox_${VERSION}_${ARCH}.deb"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "DEB packaging must run on a Linux host." >&2
  exit 1
fi

if ! command -v dpkg-deb >/dev/null 2>&1; then
  echo "dpkg-deb is required to build the DEB package." >&2
  exit 1
fi

for binary in qkbox qkboxd qkbox-provider; do
  if [[ ! -f "${ROOT_DIR}/bin/${binary}" ]]; then
    echo "Missing ${binary}. Run npm run build on Linux before packaging." >&2
    exit 1
  fi
done

rm -rf "${PKG_ROOT}"
mkdir -p \
  "${PKG_ROOT}/DEBIAN" \
  "${PKG_ROOT}/usr/bin" \
  "${PKG_ROOT}/usr/share/applications" \
  "${PKG_ROOT}/usr/lib/systemd/system" \
  "${OUT_DIR}"

install -m 0755 "${ROOT_DIR}/bin/qkbox" "${PKG_ROOT}/usr/bin/qkbox"
install -m 0755 "${ROOT_DIR}/bin/qkboxd" "${PKG_ROOT}/usr/bin/qkboxd"
install -m 0755 "${ROOT_DIR}/bin/qkbox-provider" "${PKG_ROOT}/usr/bin/qkbox-provider"
install -m 0644 "${ROOT_DIR}/packaging/linux/qkbox.desktop" "${PKG_ROOT}/usr/share/applications/qkbox.desktop"
install -m 0644 "${ROOT_DIR}/packaging/linux/qkbox-provider.service" "${PKG_ROOT}/usr/lib/systemd/system/qkbox-provider.service"

cat > "${PKG_ROOT}/DEBIAN/control" <<CONTROL
Package: qkbox
Version: ${VERSION}
Section: net
Priority: optional
Architecture: ${ARCH}
Maintainer: qkbox contributors
Depends: libc6
Description: qkbox desktop client
 qkbox is a desktop GUI client with a user-scope daemon and explicit runtime owners.
CONTROL

dpkg-deb --build "${PKG_ROOT}" "${DEB_PATH}"

if [[ ! -f "${DEB_PATH}" ]]; then
  echo "dpkg-deb completed but package was not created: ${DEB_PATH}" >&2
  exit 1
fi

echo "Created ${DEB_PATH}"
