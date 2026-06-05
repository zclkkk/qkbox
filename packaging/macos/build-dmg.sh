#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "macOS DMG packaging must run on a macOS host." >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

if ! command -v hdiutil >/dev/null 2>&1; then
  echo "hdiutil is required to build the DMG." >&2
  exit 1
fi

VERSION="${QKBOX_VERSION:-$(node -p 'require("./package.json").version' 2>/dev/null || echo "0.1.0")}"
STAGE_DIR="${ROOT_DIR}/dist/macos"
OUT_DIR="${ROOT_DIR}/dist/packages"
IMAGE_ROOT="${STAGE_DIR}/image"
APP_DIR="${IMAGE_ROOT}/qkbox.app"
CONTENTS_DIR="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"
DMG_PATH="${OUT_DIR}/qkbox-${VERSION}-macos.dmg"

for binary in qkbox qkboxd qkbox-provider; do
  if [[ ! -f "${ROOT_DIR}/bin/${binary}" ]]; then
    echo "Missing ${binary}. Run npm run build on macOS before packaging." >&2
    exit 1
  fi
done

rm -rf "${IMAGE_ROOT}"
mkdir -p "${MACOS_DIR}" "${RESOURCES_DIR}" "${OUT_DIR}"

install -m 0755 "${ROOT_DIR}/bin/qkbox" "${MACOS_DIR}/qkbox"
install -m 0755 "${ROOT_DIR}/bin/qkboxd" "${MACOS_DIR}/qkboxd"
install -m 0755 "${ROOT_DIR}/bin/qkbox-provider" "${MACOS_DIR}/qkbox-provider"
install -m 0644 "${ROOT_DIR}/packaging/macos/NetworkExtension.README.txt" "${RESOURCES_DIR}/NetworkExtension.README.txt"

cat > "${CONTENTS_DIR}/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>en</string>
  <key>CFBundleExecutable</key>
  <string>qkbox</string>
  <key>CFBundleIdentifier</key>
  <string>dev.qkbox.app</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>qkbox</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>${VERSION}</string>
  <key>CFBundleVersion</key>
  <string>${VERSION}</string>
  <key>LSMinimumSystemVersion</key>
  <string>13.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST

if [[ -n "${QKBOX_CODESIGN_IDENTITY:-}" ]]; then
  codesign --force --deep --options runtime --sign "${QKBOX_CODESIGN_IDENTITY}" "${APP_DIR}"
fi

hdiutil create -volname "qkbox" -srcfolder "${IMAGE_ROOT}" -ov -format UDZO "${DMG_PATH}"

if [[ ! -f "${DMG_PATH}" ]]; then
  echo "hdiutil completed but DMG was not created: ${DMG_PATH}" >&2
  exit 1
fi

echo "Created ${DMG_PATH}"
