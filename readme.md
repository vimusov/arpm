# What?

A bunch of scripts to build packages for Arch Linux. Plus a server and CLI tool for sharing and deploying packages.

# Why?

When you are running more than one server under Arch Linux you may need a shared repository with some custom packages.
And this is it!

# Features

- Repeatable build in a clean container environment;
- Build from a directory, from a tarball, fetching an URL (from AUR for instance);
- The remote server keeps the packages database steady and solid;
- A CLI tool which allows to run these actions on a remote server:
  - List all branches;
  - Create new branch;
  - List packages in a branch;
  - Download a package;
  - Upload packages with replacing the old ones;
  - Remove packages (branch removal is not implemented for the safety reasons);
  - Update the server (for debug and development purposes);

**WARNING!** The server does not support any authorization!

# Requirements

For build packages:

- `/etc/makepkg.conf` (copies into container);
- `/etc/pacman.conf` (copies into container);
- `podman`
- `sudo`
- `aria2c` or `wget` or `curl`

# Server setup

1. Create an unprivileged user, `arpm` for instance.

1. Create a systemd unit:

   ```
   [Unit]
   Description=ArchLinux repository and packages server

   [Service]
   Type=notify
   User=arpm
   Group=arpm
   ExecStart=/usr/bin/arpm server /srv/archlinux/x86_64
   Restart=always

   [Install]
   WantedBy=multi-user.target
   ```

   Where `/srv/archlinux/x86_64` - is the packages root directory.

1. Run the server:

   `systemctl enable --now arpm`

# CLI tool usage

1. Create a config `~/.config/arpm.conf`:

   ```
   server = 'http://example.com:31847'
   ```

   Where `example.com` is the address of the server.

1. Build a package:

   `./build.sh 'https://aur.archlinux.org/cgit/aur.git/snapshot/google-chrome.tar.gz'`

1. Create a new branch on server, `custom` for instance:

   `./arpm.py branch mk custom`

1. Upload package to the server:

   `./arpm.py pkg put custom out/*.pkg.tar.zstd`

1. Append the URL with a new branch to the `/etc/pacman.conf`:

   ```
   [custom]
   Server = http://example.com/archlinux/$arch/$repo
   ```

# License

GPL.
