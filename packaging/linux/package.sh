#!/bin/bash
# 打包 Linux 版本：.deb 安装包和便携 .tar.gz
# 用法: packaging/linux/package.sh <可执行文件> <版本号> <输出目录>
set -euo pipefail
BIN=$1
VERSION=${2#v}
OUT=$3
HERE=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$HERE/../.." && pwd)
mkdir -p "$OUT"
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# .deb
PKG=$WORK/deb
install -Dm755 "$BIN" "$PKG/usr/bin/m4s-converter"
install -Dm644 "$HERE/m4s-converter.desktop" "$PKG/usr/share/applications/m4s-converter.desktop"
install -Dm644 "$ROOT/app/build/appicon.png" "$PKG/usr/share/icons/hicolor/512x512/apps/m4s-converter.png"
mkdir -p "$PKG/DEBIAN"
cat > "$PKG/DEBIAN/control" <<CONTROL
Package: m4s-converter
Version: $VERSION
Section: video
Priority: optional
Architecture: amd64
Maintainer: crucibleu <105141329+crucibleu@users.noreply.github.com>
Depends: libgtk-3-0t64 | libgtk-3-0, libwebkit2gtk-4.1-0, gstreamer1.0-plugins-base, gstreamer1.0-plugins-good
Recommends: gstreamer1.0-libav, gstreamer1.0-plugins-bad, xdg-utils
Homepage: https://github.com/crucibleu/FromM4StoMp4_A-Converter-With-User-friendly-GUI
Description: Convert Bilibili m4s cache to mp4, with a library and player
 Browse videos cached by the Bilibili client, convert the ones you choose
 to mp4 without re-encoding, preview them and play them with danmaku.
CONTROL
dpkg-deb --root-owner-group --build "$PKG" "$OUT/m4s-converter_${VERSION}_amd64.deb"

# 便携版
TAR=$WORK/m4s-converter-${VERSION}-linux-amd64
install -Dm755 "$BIN" "$TAR/m4s-converter"
install -Dm644 "$HERE/m4s-converter.desktop" "$TAR/m4s-converter.desktop"
install -Dm644 "$ROOT/app/build/appicon.png" "$TAR/m4s-converter.png"
cat > "$TAR/README.txt" <<README
M4S Converter $VERSION (Linux x86_64)

运行 ./m4s-converter 即可。需要系统已安装 WebKitGTK 4.1 和 GStreamer：
Run ./m4s-converter. It needs WebKitGTK 4.1 and GStreamer:

  Debian/Ubuntu: sudo apt install libwebkit2gtk-4.1-0 gstreamer1.0-plugins-good gstreamer1.0-libav
  Fedora:        sudo dnf install webkit2gtk4.1 gstreamer1-plugins-good gstreamer1-libav
README
tar -C "$WORK" -czf "$OUT/m4s-converter-${VERSION}-linux-amd64.tar.gz" "$(basename "$TAR")"
ls -la "$OUT"
