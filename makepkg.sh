#!/bin/bash

set -e -u -x

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
pacman -Syu --noconfirm

mkdir $WORK_DIR
find $SRC_DIR_PATH -mindepth 1 -not \( -path '*/.*' -or -iname $PKG_EXT \) -exec cp -rv '{}' $WORK_DIR \;
chown -Rv pkgbuild:pkgbuild $WORK_DIR
pushd $WORK_DIR > /dev/null
sudo -u pkgbuild -- makepkg --skippgpcheck --noconfirm --syncdeps
popd > /dev/null

chmod 0777 $RESULT_DIR
install -v -m 0644 --target-directory=$RESULT_DIR $WORK_DIR/$PKG_EXT
