#!/bin/bash

set -e -u -x

readonly SUDO=${WITH_SUDO:-sudo}
readonly THIS_FN=$(readlink -e "$0")
readonly THIS_DIR="${THIS_FN%\/*}"
readonly ROOT_DIR="$THIS_DIR"/tmp-root
readonly DST_IMG=bs.tar
readonly SRC_ARCH=archlinux-bootstrap-x86_64.tar.gz

on_exit()
{
    if [ -d "$ROOT_DIR" ]; then
        mountpoint -q "$ROOT_DIR" && $SUDO umount "$ROOT_DIR" || true
        rmdir "$ROOT_DIR"
    else
        true
    fi
    builtin exit 0
}

trap on_exit ERR EXIT

if [ -s "$DST_IMG" ]; then
    echo "ERROR: '$DST_IMG' is already exist." >&2
    exit 1
fi

[ -s "$SRC_ARCH" ] || aria2c \
    "https://mirror.yandex.ru/archlinux/iso/latest/$SRC_ARCH"

mkdir "$ROOT_DIR"
$SUDO mount -t tmpfs -o size=2G none "$ROOT_DIR"
$SUDO tar zxf "$SRC_ARCH" -C "$ROOT_DIR" --strip-components=1
$SUDO tar cf "$THIS_DIR"/"$DST_IMG" -C "$ROOT_DIR" .

readonly ORG_UID=$(id -u)
readonly ORG_GID=$(id -g)
$SUDO chown "$ORG_UID":"$ORG_GID" "$DST_IMG"
