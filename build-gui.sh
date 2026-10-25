#!/bin/bash
# 编译图形界面版本（Fyne，需要 CGO）
#
# Linux 依赖（Debian/Ubuntu）:
#   sudo apt install gcc pkg-config libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev
# 在 Linux 上交叉编译 Windows 版本需要以下任意一种 C 编译器:
#   - zig  (https://ziglang.org/download/)
#   - mingw-w64: sudo apt install gcc-mingw-w64-x86-64
set -e
cd "$(dirname "$0")"

echo build gui for linux-amd64...
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s" -o m4s-converter-gui-linux_amd64 ./gui

echo build gui for windows-amd64...
if command -v x86_64-w64-mingw32-gcc >/dev/null; then
	export CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++
elif command -v zig >/dev/null; then
	export CC="zig cc -target x86_64-windows-gnu" CXX="zig c++ -target x86_64-windows-gnu"
else
	echo "skip windows: install zig or mingw-w64 to cross-compile" >&2
	exit 0
fi
# -H=windowsgui 隐藏控制台窗口；zig 的链接器需要显式指定 --subsystem
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build \
	-ldflags "-w -s -H=windowsgui -extldflags=-Wl,--subsystem,windows" \
	-o m4s-converter-gui-amd64.exe ./gui
