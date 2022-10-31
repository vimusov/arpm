#!/bin/bash

set -e -u -x
set -o pipefail

readonly LANG=en_US.UTF-8
readonly NEW_ROOT=/new_root
readonly SHARED_DIR=/shared

update_configs()
{
    local name=''
    local names=(
        makepkg.conf
        pacman.conf
        pacman.d/mirrorlist
    )
    local root_dir="$1"

    for name in "${names[@]}"; do
        rm -vf "$root_dir"/etc/"$name"
        install -v -m 0644 $SHARED_DIR/"${name##*/}" "$root_dir"/etc/"$name"
    done
}

mkdir -vp -m 0755 $NEW_ROOT/var/{cache/pacman/pkg,lib/pacman,log}
mkdir -vp -m 0755 $NEW_ROOT/{dev,run,etc/pacman.d}
mkdir -vp -m 1777 $NEW_ROOT/tmp
mkdir -vp -m 0555 $NEW_ROOT/{sys,proc}

update_configs ''

ln -svf /proc/mounts /etc/mtab
pacman -Sy  --noconfirm -r $NEW_ROOT base-devel
pacman -Scc --noconfirm -r $NEW_ROOT

update_configs $NEW_ROOT

install -v -m 0755 --target-directory=$NEW_ROOT $SHARED_DIR/makepkg.sh

rm -vf $NEW_ROOT/usr/lib/locale/locale-archive
localedef -c --prefix=$NEW_ROOT -i "${LANG%%.*}" -f "${LANG##*.}" -A $NEW_ROOT/usr/share/locale/locale.alias "$LANG"
echo "LANG=$LANG" > $NEW_ROOT/etc/locale.conf

useradd --root $NEW_ROOT --user-group --no-log-init --create-home --shell /bin/bash pkgbuild
echo 'pkgbuild ALL=(ALL) NOPASSWD: ALL' >| $NEW_ROOT/etc/sudoers.d/pkgbuild

chmod 0777 $SHARED_DIR
tar cpf $SHARED_DIR/new-rootfs.tar --sort=name --directory=$NEW_ROOT \
    --exclude='./dev/*' --exclude='./proc/*' --exclude='./sys/*' --exclude='./tmp/*' \
    --exclude='./var/log/*.*' .
