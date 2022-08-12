#!/usr/bin/python

"""
    arpm - ArchLinux repository and packages manager.
    Copyright (C) 2022 Vadim Kuznetsov <vimusov@gmail.com>

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.
"""

import logging
from argparse import ArgumentParser, Namespace
from asyncio import get_running_loop, sleep
from collections import defaultdict
from configparser import RawConfigParser
from contextlib import closing
from os import _exit as os_exit
from pathlib import Path
from socket import SOL_SOCKET, SO_LINGER
from struct import pack
from subprocess import run as run_process
from sys import stderr
from tarfile import TarFile
from tempfile import TemporaryDirectory
from typing import DefaultDict, Dict, Set, Tuple

from aiohttp.web import Application, FileResponse, HTTPNotFound, Response, View, run_app, view
from pyzstd import ZstdFile
from requests import Session
from systemd.daemon import notify as sd_notify

_DB_EXT = 'db.tar.gz'
_PKG_EXT = 'pkg.tar.zst'
_PKG_GLOB = f'*.{_PKG_EXT}'

log = logging.getLogger(__name__)


class _ZstdTarFile(TarFile):
    def __init__(self, path, mode, *args, **kwargs):
        self.__zstd_file = ZstdFile(path, mode='r')
        super().__init__(None, mode, self.__zstd_file, *args, **kwargs)

    def close(self):
        try:
            super().close()
        finally:
            self.__zstd_file.close()


def _get_pkg_name(path: Path) -> str:
    with _ZstdTarFile.open(path) as tar_file:
        for obj_info in tar_file:
            if not obj_info.isreg():
                continue
            if obj_info.name != '.PKGINFO':
                continue
            file_obj = tar_file.extractfile(obj_info)
            with closing(file_obj) as info_file:
                content: str = info_file.read().decode('utf-8')
    for line in content.splitlines():
        if '=' not in line:
            continue
        key, value = line.split('=', maxsplit=1)
        if key.strip() == 'pkgname':
            return value.strip()
    raise ValueError(f"Broken package '{path}:'")


def _load_pkgs(root_dir: Path) -> DefaultDict[str, Set[Path]]:
    result = defaultdict(set)
    for path in root_dir.glob(_PKG_GLOB):
        name = _get_pkg_name(path)
        result[name].add(path)
    return result


class _BranchesHandler(View):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.__root_dir: Path = self.request.app['root_dir']

    async def get(self) -> Response:
        names = []
        for path in self.__root_dir.iterdir():
            count = sum(1 for unused in path.glob(_PKG_GLOB))
            names.append(f'{path.name}: {count} item(s)')
        return Response(text='\n'.join(sorted(names)) if names else 'No entries.')

    async def post(self) -> Response:
        params: Dict[str, str] = await self.request.json()
        path = self.__root_dir / params['name']
        log.info("Creating new branch directory '%s'.", path)
        path.mkdir(exist_ok=True)
        return Response()


class _PackagesHandler(View):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.__branch: str = self.request.match_info['branch']
        self.__root_dir: Path = self.request.app['root_dir'] / self.__branch

    def __rebuild_db(self):
        branch = self.__branch
        root_dir = self.__root_dir

        log.info("Going to rebuilt index in '%s'.", root_dir)
        for path in root_dir.glob(f'{branch}.*'):
            path.unlink(missing_ok=True)
            log.warning("File '%s' removed successfully.", path)

        files = [str(path) for path in root_dir.glob(_PKG_GLOB)]
        if not files:
            log.warning("No files in DB '%s', doing nothing.", root_dir)
            return

        args = ['repo-add', f'{branch}.{_DB_EXT}'] + files
        log.debug('Executing %r.', args)

        log.info("Rebuilding DB in '%s'.", root_dir)
        result = run_process(args, capture_output=True, cwd=str(root_dir))

        if result.returncode:
            log.error("Failed to build DB in '%s', code %d, stdout=%r, stdout=%r.", root_dir, result.returncode, result.stdout, result.stderr)
            raise RuntimeError('Unable to rebuild DB:')

        log.info("DB in '%s' has been rebuilt successfully.", root_dir)

    async def get(self) -> Response | FileResponse:
        root_dir = self.__root_dir
        if not root_dir.is_dir():
            return HTTPNotFound()
        if (name := self.request.query.get('name')) is not None:
            path = root_dir / name
            if path.is_file():
                return FileResponse(path)
            return HTTPNotFound()
        names = sorted(path.name for path in root_dir.glob(_PKG_GLOB))
        return Response(text='\n'.join(names) if names else 'No entries.')

    async def post(self) -> Response:
        root_dir = self.__root_dir
        file_name = self.request.query['name']
        log.info("Got request with file name '%s'.", file_name)

        with TemporaryDirectory(dir=root_dir.parent) as tmp_dir:
            new_path = Path(tmp_dir) / file_name

            log.info("Storing new package '%s'.", new_path)
            with new_path.open(mode='wb') as pkg_file:
                async for chunk in self.request.content.iter_any():
                    pkg_file.write(chunk)
                pkg_file.flush()

            pkgs = _load_pkgs(root_dir)
            new_name = _get_pkg_name(new_path)

            for path in pkgs.get(new_name, {}):
                log.warning("Removing old package '%s'.", path)
                path.unlink(missing_ok=True)

            dst_path = root_dir / file_name
            new_path.replace(dst_path)
            log.info("Moved: '%s' => '%s'.", new_path, dst_path)

        self.__rebuild_db()
        return Response()

    async def delete(self) -> Response:
        root_dir = self.__root_dir
        pkgs = _load_pkgs(root_dir)

        for name in self.request.query['names'].split(','):
            if name.endswith(_PKG_EXT):
                log.info("Looking for file with name '%s'.", name)
                path = root_dir / name
                if path.is_file():
                    log.warning("Removing package '%s'.", path)
                    path.unlink(missing_ok=True)
                else:
                    log.error("No such file '%s'.", name)
                continue

            log.info("Looking for package with name '%s'.", name)
            paths = pkgs.get(name, {})
            if not paths:
                log.error("No package with name '%s'.", name)
                continue

            for path in paths:
                log.warning("Removing package '%s'.", path)
                path.unlink(missing_ok=True)

        self.__rebuild_db()
        return Response()


class _UpgradeHandler(View):
    @staticmethod
    async def __reload():
        await sleep(2)
        log.warning('Going to forced restart.')
        os_exit(42)

    async def __flush(self) -> Response:
        response = Response()
        response.headers['connection'] = 'close'
        await response.prepare(self.request)
        await response.write_eof()
        sock = self.request.get_extra_info('socket')
        sock.setsockopt(SOL_SOCKET, SO_LINGER, pack('ii', 1, 1))
        sock.close()
        return response

    async def post(self) -> Response:
        log.warning('Going to upgrade myself.')
        new_code = await self.request.read()
        if not new_code:
            raise ValueError('Empty upgrade:')
        with Path(__file__).open('wb') as self_file:
            self_file.write(new_code)
            self_file.flush()
        get_running_loop().create_task(self.__reload())
        return await self.__flush()


def _get_server(args: Namespace) -> Tuple[str, int]:
    config_path = Path(args.config).expanduser()
    try:
        log.debug("Loading config from '%s'.", config_path)
        with config_path.open(mode='rt') as config_file:
            config = RawConfigParser()
            config.read_file(config_file)
            host = config.get('server', 'host')
            port = config.getint('server', 'port')
            return host, port
    except FileNotFoundError:
        print(f"ERROR: Config file '{config_path!s}' is not found.", file=stderr)
        os_exit(1)


def _get_base_url(args: Namespace):
    host, port = _get_server(args)
    return f'http://{host}:{port}'


def _notify_start():
    log.debug('Notifying systemd about successful start.')
    sd_notify('READY=1')


def _run_server(args: Namespace):
    logging.basicConfig(level=logging.DEBUG if args.debug else logging.INFO)
    logging.getLogger('aiohttp').setLevel(logging.DEBUG if args.debug else logging.ERROR)
    logging.getLogger('asyncio').setLevel(logging.ERROR)

    app = Application()
    app.router.add_routes([
        view('/branches', _BranchesHandler),
        view('/packages/{branch:[^/]+}', _PackagesHandler),
        view('/upgrade', _UpgradeHandler),
    ])
    app['root_dir'] = args.root
    host, port = _get_server(args)

    async def send_notify(unused_arg):
        log.info('Start listening on %s:%d.', host, port)
        if args.notify:
            _notify_start()
        yield

    app.cleanup_ctx.append(send_notify)

    run_app(app, host=host, port=port, print=None)
    log.info('Shutdown.')


def _on_create_branch(args: Namespace):
    with Session() as session:
        response = session.post(f'{_get_base_url(args)}/branches', json=dict(name=args.name))
        response.raise_for_status()


def _on_list_branches(args: Namespace):
    with Session() as session:
        response = session.get(f'{_get_base_url(args)}/branches')
        response.raise_for_status()
        print(response.text)


def _on_list_pkgs(args: Namespace):
    with Session() as session:
        response = session.get(f'{_get_base_url(args)}/packages/{args.branch}')
        response.raise_for_status()
        print(response.text)


def _on_get_pkg(args: Namespace):
    name = args.package
    with Session() as session, Path().cwd().joinpath(name).open(mode='wb') as pkg_file:
        response = session.get(f'{_get_base_url(args)}/packages/{args.branch}', params=dict(name=name), stream=True)
        response.raise_for_status()
        for chunk in response.iter_content():
            pkg_file.write(chunk)
        pkg_file.flush()


def _on_put_pkgs(args: Namespace):
    with Session() as session:
        for name in args.files:
            path = Path(name).resolve()
            with path.open(mode='rb') as pkg_file:
                response = session.post(f'{_get_base_url(args)}/packages/{args.branch}', params=dict(name=path.name), data=pkg_file)
                response.raise_for_status()


def _on_remove_pkgs(args: Namespace):
    with Session() as session:
        names = ','.join(args.names)
        response = session.delete(f'{_get_base_url(args)}/packages/{args.branch}', params=dict(names=names))
        response.raise_for_status()


def _on_upgrade(args: Namespace):
    server = _get_base_url(args)
    print(f"Going to upgrade server on address '{server}'.\nIs everything correct? (y/n)")
    if input().strip().lower() in ['y', 'yes']:
        with Session() as session:
            response = session.post(f'{server}/upgrade', data=Path(__file__).read_bytes())
            response.raise_for_status()


def main():
    parser = ArgumentParser(prog='arpm', description='ArchLinux repository and packages manager.')
    parser.add_argument('-c', '--config', default='~/.config/arpm.conf', help='Config path.')
    subparsers = parser.add_subparsers()

    def add_server_parsers():
        server_parser = subparsers.add_parser('server', help='Run repository and packages server.')
        server_parser.add_argument('-d', '--debug', action='store_true', help='Enable debug mode.')
        server_parser.add_argument('-n', '--notify', action='store_true', help='Notify systemd about successful start.')
        server_parser.add_argument('root', type=Path, help='Repositories root directory.')
        server_parser.set_defaults(func=_run_server)

    def add_branches_parsers():
        branches_parser = subparsers.add_parser('branch', help='Manage branches on server.')
        branches_subparsers = branches_parser.add_subparsers(required=True)

        create_branch_parser = branches_subparsers.add_parser('mk', help='Create new branch.')
        create_branch_parser.add_argument('name', help='branch name.')
        create_branch_parser.set_defaults(func=_on_create_branch)

        list_branches_parser = branches_subparsers.add_parser('ls', help='List branches.')
        list_branches_parser.set_defaults(func=_on_list_branches)

    def add_pkgs_parsers():
        pkgs_parser = subparsers.add_parser('pkg', help='Manage packages in branch.')
        pkgs_parser.add_argument('branch', help='Use branch name.')
        pkgs_subparser = pkgs_parser.add_subparsers(required=True)

        def add_pkg_commands():
            list_pkgs_parser = pkgs_subparser.add_parser('ls', help='List all packages in branch.')
            list_pkgs_parser.set_defaults(func=_on_list_pkgs)

            down_pkg_parser = pkgs_subparser.add_parser('get', help='Download packages.')
            down_pkg_parser.add_argument('package', help='Package name to download.')
            down_pkg_parser.set_defaults(func=_on_get_pkg)

            publish_pkg_parser = pkgs_subparser.add_parser('put', help='Publish packages.')
            publish_pkg_parser.add_argument('files', nargs='+', help='Packages to publish.')
            publish_pkg_parser.set_defaults(func=_on_put_pkgs)

            remove_pkgs_parser = pkgs_subparser.add_parser('rm', help='Remove packages.')
            remove_pkgs_parser.add_argument('names', nargs='+', help='Packages names to remove.')
            remove_pkgs_parser.set_defaults(func=_on_remove_pkgs)

        add_pkg_commands()

    def add_upgrade_parsers():
        upgrade_parser = subparsers.add_parser('upgrade', help='Upgrade the remote server (for development purposes only).')
        upgrade_parser.set_defaults(func=_on_upgrade)

    add_server_parsers()
    add_branches_parsers()
    add_pkgs_parsers()
    add_upgrade_parsers()

    def on_unknown_arg(argv: Namespace):
        argv.parser.print_help(stderr)
        os_exit(1)

    parser.set_defaults(func=on_unknown_arg, parser=parser)
    args = parser.parse_args()
    args.func(args)


if __name__ == '__main__':
    main()
