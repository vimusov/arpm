#!/bin/bash

set -e -u -x
set -o pipefail

readonly NEW_ROOT=/new_root
readonly SHARED_DIR=/shared

mkdir -vp -m 0755 $NEW_ROOT/var/{cache/pacman/pkg,lib/pacman,log}
mkdir -vp -m 0755 $NEW_ROOT/{dev,run,etc/pacman.d}
mkdir -vp -m 1777 $NEW_ROOT/tmp
mkdir -vp -m 0555 $NEW_ROOT/{sys,proc}

rm -vf /etc/pacman.conf
install -v -m 0644 $SHARED_DIR/pacman.conf /etc/pacman.conf

ln -svf /proc/mounts /etc/mtab
pacman -Sy  --noconfirm -r $NEW_ROOT base-devel
pacman -Scc --noconfirm -r $NEW_ROOT

rm -vf $NEW_ROOT/etc/{makepkg,pacman}.conf
install -v -m 0644 --target-directory=$NEW_ROOT/etc $SHARED_DIR/{makepkg,pacman}.conf

install -v -m 0755 --target-directory=$NEW_ROOT $SHARED_DIR/makepkg.sh

rm -vf $NEW_ROOT/usr/lib/locale/locale-archive
localedef -c --prefix=$NEW_ROOT -i en_US -f UTF-8 -A $NEW_ROOT/usr/share/locale/locale.alias en_US.UTF-8

useradd --root $NEW_ROOT --user-group --no-log-init --create-home --shell /bin/bash pkgbuild
echo 'pkgbuild ALL=(ALL) NOPASSWD: ALL' >| $NEW_ROOT/etc/sudoers.d/pkgbuild

chmod 0777 $SHARED_DIR
tar cpf $SHARED_DIR/new-rootfs.tar --sort=name --directory=$NEW_ROOT \
    --exclude='./dev/*' --exclude='./proc/*' --exclude='./sys/*' --exclude='./tmp/*' \
    --exclude='./var/log/*.*' .
