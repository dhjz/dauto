@echo off
REM chcp 65001
setlocal enabledelayedexpansion

go env -w GOOS=linux
go build -ldflags "-s -w" -o ./dist/
echo "build linux success..."

@REM go env -w GOOS=linux GOARCH=arm  GOARM=7 
@REM go build -ldflags "-s -w" -o ./dist/
@REM echo "build linux-armv7 success..."

@REM go env -w GOOS=linux GOARCH=arm64  GOARM= 
@REM go build -ldflags "-s -w" -o ./dist/
@REM echo "build linux-arm64 success..."

go env -w GOOS=windows GOARCH=amd64 GOARM=
go build -ldflags "-s -w" -o ./dist/dauto_debug.exe
go build -ldflags "-s -w -H=windowsgui" -o ./dist/
echo "build windows exe success..."
pause
endlocal