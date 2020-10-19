#!/bin/sh
# run ddns on remote host via ssh
ARCH=`ssh $1 uname -m`
KERNEL=`ssh $1 uname -s | tr '[:upper:]' '[:lower:]'`

case "$ARCH" in
    "x86_64") ARCH="amd64"
    ;;
    "aarch64") ARCH="arm64"
    ;;
esac

BINARY="ddns-$KERNEL-$ARCH"
rsync -avz --progress $BINARY $1:~/
rsync -avz --progress ~/.ddns $1:~/
ssh $1 "~/$BINARY"
