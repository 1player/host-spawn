## 1.6.0 (28 Apr 2024)

### Added

* Build and distribute binary for loongarch64. Thanks @shenmo7192.
* Add `-cwd` flag to change the working directory of the spawned process. Thanks @someone13574.

## 1.5.1 (21 Dec 2023)

### Minor

* Update golang.org/x/sys dependency to v0.15.0. Thanks @klugier.
* Refactor build scripts to target single architecture

## 1.5.0 (2 Sep 2023)

### Added

* Forward signals to the host process (https://github.com/1player/host-spawn/issues/18)

## 1.4.2 (3 Jun 2023)

### Fixed

* Do not allocate a pty for `podman` (https://github.com/1player/host-spawn/issues/21). Thanks @lbssousa.

## 1.4.1 (18 Feb 2023)

### Fixed

* Do not propagate environment variables that are not set (https://github.com/1player/host-spawn/issues/17)

## 1.4.0 (14 Jan 2023)

### Added

* Do not allocate a pty if the command is known to misbehave when attached to one. Thanks @89luca89

## 1.3.0 (12 Oct 2022)

### Added

* `-env` command line argument to specify which environment variables to pass to the host process. If unspecified, defaults to "TERM". Thanks @travier

## 1.2.1 (12 Aug 2022)

### Fixed

* Don't fail if stdin is redirected (https://github.com/1player/host-spawn/issues/11)

## 1.2.0 (27 Jul 2022)

### Added

* Spawn a shell on the host if no command is passed.

## 1.1.0 (24 Jul 2022)

### Added

* Shim host binaries when symlinked. See example at https://github.com/1player/host-spawn#creating-shims-for-host-binaries

## 1.0.2 (15 Jul 2022)

### Changed

* Added `--no-pty` flag to work around misbehaving processes that terminate too early. See https://github.com/1player/host-spawn/issues/7. Thanks @89luca89

## 1.0.1 (11 Jul 2022)

### Changed

* Terminal no longer gets scrambled on error. Thanks @89luca89

## 1.0 (9 Jul 2022)

Compiled, statically linked version
