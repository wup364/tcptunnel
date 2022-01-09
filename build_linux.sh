export B_HOME=$(pwd)
export B_BOOT=$B_HOME/boot
export B_BIN=$B_HOME/bin

echo build for linux

cd $B_BOOT/client
go build -o $B_BIN/tunnel-client
cd $B_BOOT/server
go build -o $B_BIN/tunnel-server

echo build succeed
