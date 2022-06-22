echo off
set B_HOME=%cd%
set B_BOOT=%B_HOME%/boot
set B_BIN=%B_HOME%/bin

echo build for windows

cd %B_BOOT%/client
go build -o %B_BIN%/tunnel-client.exe
cd %B_BOOT%/server
go build -o %B_BIN%/tunnel-server.exe

echo build succeed