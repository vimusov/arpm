#!/bin/bash

set -ueo pipefail

readonly SUDO=${WITH_SUDO:-sudo}

readonly THIS_FN=$(readlink -e "$0")
LOCAL_REPO=''
RESULT_DIR=''
TARBALL_DIR=''
MIRRORS_CONF='/etc/pacman.d/mirrorlist'

readonly ORG_UID=$(id -u)
readonly ORG_GID=$(id -g)
readonly THIS_DIR="${THIS_FN%\/*}"
readonly SHARED_DIR="$THIS_DIR"/shared
readonly HALF_CONT_NAME=half-backed-image
readonly FULL_CONT_NAME=arch-makepkg
readonly SRC_ARCH=archlinux-bootstrap-x86_64.tar.zst
readonly ROOT_DIR="$THIS_DIR"/tmp-root
readonly DST_IMG=bs.tar
readonly IMG_URL="https://mirror.yandex.ru/archlinux/iso/latest/$SRC_ARCH"

on_exit()
{
    if [ -d "$ROOT_DIR" ]; then
        mountpoint -q "$ROOT_DIR" && $SUDO umount "$ROOT_DIR" || true
        rmdir "$ROOT_DIR"
    else
        true
    fi
    $SUDO podman images | grep -qF  $HALF_CONT_NAME && $SUDO podman rmi --force $HALF_CONT_NAME
    rm -rfv "$SHARED_DIR"
    [ -z "$TARBALL_DIR" ] || rm -rfv "$TARBALL_DIR"
    builtin exit 0
}

trap on_exit ERR EXIT

cont_ready()
{
    $SUDO podman images | grep -qF  $FULL_CONT_NAME
}

do_cleanup()
{
    cont_ready || return 0
    $SUDO podman rmi --force $FULL_CONT_NAME
}

do_bootstrap()
{
    [ -s "$DST_IMG" ] && return 0

	if ! [ -s "$SRC_ARCH" ]; then
    	local loader=''
    	for loader in aria2c wget curl; do
        	type $loader > /dev/null 2>&1 && break
    	done
    	if [ -z "$loader" ]; then
	        echo "ERROR: Loader is not found, aria2c or wget or curl needed." >&2
        	exit 1
    	fi
	    $loader "$IMG_URL"
	fi

    mkdir "$ROOT_DIR"
    $SUDO mount -t tmpfs -o size=2G none "$ROOT_DIR"
    $SUDO bsdtar xf "$SRC_ARCH" -C "$ROOT_DIR" --strip-components=1
    $SUDO tar cf "$THIS_DIR"/"$DST_IMG" -C "$ROOT_DIR" .
    $SUDO chown "$ORG_UID":"$ORG_GID" "$DST_IMG"
}

do_update()
{
    install -D -m 0644 --target-directory="$SHARED_DIR" \
        "$THIS_DIR"/makepkg.sh /etc/{makepkg,pacman}.conf \
        "$MIRRORS_CONF"

    do_bootstrap

    pushd "$THIS_DIR" > /dev/null
    $SUDO podman build \
        --network=host \
        --rm --force-rm --no-cache \
        --tag $HALF_CONT_NAME \
        --volume "$SHARED_DIR":/shared .
    popd > /dev/null

    $SUDO podman import \
        "$SHARED_DIR"/new-rootfs.tar \
        $FULL_CONT_NAME
}

usage()
{
    local cur_dir=$(readlink -e "$PWD")
    cat >&2 << EOF
Usage: ${0##*\/} [-h] [-e] [-i] [-r] [-u] [-m <FILE>] [-l <DIR>] [-o <DIR>] [SRC]

Options:
    -e: Run shell if build failed;
    -i: Run pacman interactively;
    -r: Remove container and quit;
    -u: Update container and quit;
    -m <FILE>: Use mirrorlist from the specified file;
    -l <DIR>: Use the directory as a local repository;
    -o <DIR>: The directory for the built packages ('$cur_dir/out' by default);

Arguments:
    SRC: Source. Can be:

        1. Undefined - the current directory is a directory with sources;

        2. Path to directory - this directory is a directory with sources;

        3. Path to tarball file - the file will be extracted and its content
           used as a sources;

        4. URL with tarball - the file will be downloaded, extracted and its
           content used as a sources;

EOF
    exit 1
}

CONT_ARGS=()

while getopts "heirum:l:o:" opt; do
    case $opt in
        h)
            usage
            ;;
        e)
            CONT_ARGS+=(-e)
            ;;
        i)
            CONT_ARGS+=(-i)
            ;;
        r)
            do_cleanup
            exit $?
            ;;
        u)
            do_cleanup
            do_update
            exit $?
            ;;
        m)
            MIRRORS_CONF=$(readlink -e "$OPTARG")
            ;;
        l)
            LOCAL_REPO=$(readlink -e "$OPTARG")
            ;;
        o)
            RESULT_DIR=$(readlink -m "$OPTARG")
            ;;
        \?)
            usage
            ;;
    esac
done

shift $(($OPTIND-1))
[ $# -eq 1 ] && SRC_DIR="$1" || SRC_DIR="$PWD"

if [ -d "$SRC_DIR" ]; then
    SRC_DIR=$(readlink -e "$SRC_DIR")
else
    TARBALL_DIR=$(mktemp -d)
    if [[ "${SRC_DIR%%:*}" =~ ^(http|https|ftp)$ ]]; then
        curl "$SRC_DIR" | bsdtar -x -v -C "$TARBALL_DIR" -f -
    else
        bsdtar -x -v -C "$TARBALL_DIR" -f $(readlink -e "$SRC_DIR")
    fi
    SRC_DIR=$(echo "$TARBALL_DIR"/*)
fi

if ! [ -s "$SRC_DIR"/PKGBUILD ]; then
    echo "ERROR: Unable to find PKGBUILD in '$SRC_DIR'." >&2
    exit 1
fi

[ -n "$RESULT_DIR" ] || RESULT_DIR=$(readlink -m "$PWD"/out)
mkdir -p "$RESULT_DIR"

cont_ready || do_update

$SUDO podman run -it --rm \
    --network=host \
    --volume "$SRC_DIR":/sources:ro \
    --volume "$RESULT_DIR":/result \
    ${LOCAL_REPO:+--volume "$LOCAL_REPO":/local_repo:ro} \
    $FULL_CONT_NAME /makepkg.sh "${CONT_ARGS[@]}"

$SUDO chown -R "$ORG_UID":"$ORG_GID" "$RESULT_DIR"
