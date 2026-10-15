# Apptainer on gokrazy

This directory packages Apptainer for [gokrazy](https://gokrazy.org/), a
Go-only Linux appliance platform with a read-only root file system.

gokrazy builds every package as a pure Go program, but Apptainer's runtime
`starter` is written partly in C. Therefore this package consists of:

- `apptainer/`: a small pure Go wrapper, installed by gokrazy as
  `/user/apptainer`, which prepares writable directories and then runs the
  real `apptainer` binary.
- `apptainer/_gokrazy/extrafiles_<arch>.tar`: a statically linked, non-SUID
  Apptainer installation under `/usr/local/apptainer`, including static
  `mksquashfs`, `unsquashfs`, `newuidmap` and `newgidmap`, plus minimal
  `/etc/passwd`, `/etc/group`, `/etc/subuid` and `/etc/subgid` files and a
  `/root` mount point. These tarballs are built by `build.sh` and are checked
  in, so the package can be added to a gokrazy instance without building
  them first. Rebuild and commit them whenever the Apptainer sources change.

## Building

Docker with BuildKit (`docker buildx`) is required. From the top of the
Apptainer source tree run:

```sh
dist/gokrazy/build.sh amd64 arm64
```

The build runs natively for the target platform inside a
`golang:<version>-alpine` container. To build `arm64` tarballs on an `amd64`
host, register QEMU user-mode emulation first, for example with:

```sh
docker run --privileged --rm tonistiigi/binfmt --install arm64
```

## Adding to a gokrazy instance

The prebuilt tarballs are committed, so the package can be added directly
without building anything:

```sh
gok -i <instance> add github.com/ascension-association/gk-apptainer/dist/gokrazy/apptainer@bare-metal
gok -i <instance> update
```

Alternatively, add the wrapper package from a local checkout (for example
after running `build.sh`):

```sh
gok -i <instance> add ./dist/gokrazy/apptainer
```

## Usage

From a [breakglass](https://github.com/gokrazy/breakglass) shell:

```sh
cd /perm
apptainer pull alpine.sif docker://alpine
apptainer exec alpine.sif cat /etc/os-release
```

The wrapper sets the following, unless already set:

- `HOME` to `/perm/apptainer/home` when it is unset or `/`.
- `APPTAINER_CACHEDIR` to `/perm/apptainer/cache`.
- `APPTAINER_TMPDIR` to `/perm/apptainer/tmp`.
- `APPTAINER_CONFIG_FILE` to `/perm/apptainer/apptainer.conf`, if that file
  exists. Copy `/usr/local/apptainer/etc/apptainer/apptainer.conf` there to
  customize the configuration.

If `/perm` is not available, `/tmp/apptainer` is used instead. The wrapper
also creates `/var/tmp` (`/perm/var/tmp`), which Apptainer binds into
containers.

## Limitations

- Apptainer runs as root, as all gokrazy programs do by default.
- `--userns` and `--fakeroot` require `CONFIG_USER_NS`. It is enabled in
  the gokrazy arm64 kernel (`github.com/gokrazy/kernel.arm64`) but not in
  the amd64 kernel (`github.com/gokrazy/kernel.amd64`). `--fakeroot` maps
  container IDs 1-65536 to the host's subordinate IDs 100000-165535.
- Built without seccomp, CNI network plugins (`--net`), subid support, or
  `squashfuse`/`fuse2fs`/`gocryptfs`. Without `squashfuse`, `--userns` and
  `--fakeroot` extract SIF images to a temporary sandbox on every run.
- The package ships `/etc/passwd`, `/etc/group`, `/etc/subuid`,
  `/etc/subgid` and `/root`. These collide with any other gokrazy package
  providing the same paths.
