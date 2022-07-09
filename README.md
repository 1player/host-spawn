# host-spawn

A reimplementation of `flatpak-spawn --host`.

Improvements over the original:

* Allocates a pty for the spawned process, fixing the following upstream issues: https://github.com/flatpak/flatpak/issues/3697 and https://github.com/flatpak/flatpak/issues/3285
* Handles SIGWINCH (window size changes)

## References

* https://github.com/owtaylor/PurpleEgg/blob/master/common/host-command.c
* https://github.com/gnunn1/tilix/blob/master/source/gx/tilix/terminal/terminal.d
