#!/bin/bash

set -e -u
set -o pipefail

readonly SUDO=${WITH_SUDO:-sudo}

THIS_FN=$(readlink -e "$0")
LOCAL_REPO=''
RESULT_DIR=''
TARBALL_DIR=''

readonly THIS_DIR="${THIS_FN%\/*}"
readonly SHARED_DIR="$THIS_DIR"/shared
readonly ARCH_CONT_NAME=docker.io/archlinux/archlinux
readonly HALF_CONT_NAME=half-backed-image
readonly FULL_CONT_NAME=arch-makepkg

on_exit()
{
    $SUDO podman images | fgrep -q  $HALF_CONT_NAME && $SUDO podman rmi --force $HALF_CONT_NAME
    $SUDO podman images | fgrep -q  $ARCH_CONT_NAME && $SUDO podman rmi --force $ARCH_CONT_NAME
    rm -rfv "$SHARED_DIR"
    [ -z "$TARBALL_DIR" ] || rm -rfv "$TARBALL_DIR"
    builtin exit 0
}

cont_ready()
{
    $SUDO podman images | fgrep -q  $FULL_CONT_NAME
}

do_cleanup()
{
    cont_ready || return 0
    $SUDO podman rmi --force $FULL_CONT_NAME
}

do_update()
{
    install -D -m 0644 --target-directory="$SHARED_DIR" \
        "$THIS_DIR"/makepkg.sh /etc/{makepkg,pacman}.conf

    pushd "$THIS_DIR" > /dev/null
    $SUDO podman build \
        --rm --force-rm --no-cache \
        --from $ARCH_CONT_NAME \
        --tag $HALF_CONT_NAME \
        --volume "$SHARED_DIR":/shared .
    popd > /dev/null

    $SUDO podman import \
        --change CMD='' \
        --change ENTRYPOINT=/makepkg.sh \
        "$SHARED_DIR"/new-rootfs.tar \
        $FULL_CONT_NAME
}

usage()
{
    local cur_dir=$(readlink -e "$PWD")
    cat >&2 << EOF
Usage: ${0##*\/} [-h] [-r] [-u] [-l <DIR>] [-o <DIR>] [SRC]

Options:
    -r: Remove container and quit;
    -u: Update container and quit;
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

trap on_exit ERR EXIT

while getopts "hrul:o:" opt; do
    case $opt in
        h)
            usage
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
        curl -s "$SRC_DIR" | bsdtar -x -v -C "$TARBALL_DIR" -f -
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
    --volume "$SRC_DIR":/sources:ro \
    --volume "$RESULT_DIR":/result \
    ${LOCAL_REPO:+--volume "$LOCAL_REPO":/local_repo:ro} \
    $FULL_CONT_NAME