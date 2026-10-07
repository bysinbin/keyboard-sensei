#!/usr/bin/env bash
set -e

APP_NAME="Keyboard Sensei"
APP_DIR="${APP_NAME}.app"
STAGING_DIR="/tmp/${APP_NAME}.app"
CONTENTS_DIR="${STAGING_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"

echo "🥋 Building ${APP_NAME}.app bundle..."

# 1. Clean previous build
rm -rf "${STAGING_DIR}"
rm -rf "${APP_DIR}"
mkdir -p "${MACOS_DIR}"
mkdir -p "${RESOURCES_DIR}"

# 2. Compile Go binary
echo "⚙️ Compiling Go binary..."
go build -ldflags="-s -w" -o "${MACOS_DIR}/keyboard-sensei" .
chmod +x "${MACOS_DIR}/keyboard-sensei"

# 3. Copy Icon
if [ -f "AppIcon.icns" ]; then
    cp "AppIcon.icns" "${RESOURCES_DIR}/AppIcon.icns"
fi

# 4. Generate Info.plist
cat <<EOF > "${CONTENTS_DIR}/Info.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>tr</string>
    <key>CFBundleDisplayName</key>
    <string>Keyboard Sensei</string>
    <key>CFBundleExecutable</key>
    <string>keyboard-sensei</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>com.keyboard.sensei</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>Keyboard Sensei</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.2.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>LSMinimumSystemVersion</key>
    <string>12.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

# 5. Sign the App Bundle properly in staging
echo "🔏 Signing app bundle..."
find "${STAGING_DIR}" -name ".DS_Store" -delete
find "${STAGING_DIR}" -name "._*" -delete
dot_clean "${STAGING_DIR}" 2>/dev/null || true
xattr -cr "${STAGING_DIR}" 2>/dev/null || true
codesign --force --deep --sign - "${STAGING_DIR}"

# 6. Copy to local and Applications
ditto "${STAGING_DIR}" "${APP_DIR}"

echo "✅ ${APP_NAME}.app başarıyla oluşturuldu ve imzalandı!"
echo "📍 Konum: $(pwd)/${APP_DIR}"
echo "👉 Uygulamayı Applications klasörüne kopyalamak için: cp -r \"${APP_DIR}\" /Applications/"
