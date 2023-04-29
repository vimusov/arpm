#!/bin/bash

set -e -u -x

opt=''
CONFIRM='--noconfirm'

on_error()
{
    echo "Unexpected error occurred. Dropping to an emergency shell.\n"
    /bin/bash
    builtin exit 1
}

while getopts "ie" opt; do
    case $opt in
        i)
            CONFIRM=''
            ;;
        e)
            trap on_error ERR
            ;;
    esac
done

readonly PKG_EXT='*.pkg.tar.zst'
readonly WORK_DIR=/tmp/workdir
readonly RESULT_DIR=/result
readonly SRC_DIR_PATH=/sources
readonly SRC_REPO_DIR=/local_repo
readonly DST_REPO_DIR=/tmp/local_repo

[ -d $SRC_REPO_DIR ] && {
    install -vD -m 0644 --target-directory=$DST_REPO_DIR $SRC_REPO_DIR/$PKG_EXT
    pushd $DST_REPO_DIR > /dev/null
    repo-add pkgbuild.db.tar.gz $PKG_EXT
    popd > /dev/null
    echo -e "\n[pkgbuild]\nServer = file://$DST_REPO_DIR\n" >> /etc/pacman.conf
} || true
pacman -Syu $CONFIRM

mkdir $WORK_DIR
for path in $SRC_DIR_PATH/* ; do
    if [ -d "$path" ]; then
        pkg_found=0
        for pkg in "$path"/"$PKG_EXT"; do
            [ -s "$pkg" ] && pkg_found=1 || continue
        done
        [ $pkg_found -eq 1 ] || cp -rv "$path" $WORK_DIR
    elif [ -f "$path" ]; then
        cp -v "$path" $WORK_DIR
    else
        echo "Skipping unknown file '$path'."
    fi
done
chown -Rv pkgbuild:pkgbuild $WORK_DIR
pushd $WORK_DIR > /dev/null
sudo -u pkgbuild -- makepkg --skippgpcheck --syncdeps $CONFIRM
popd > /dev/null

install -v -m 0644 --target-directory=$RESULT_DIR $WORK_DIR/$PKG_EXT
