# host-spawn

A reimplementation of `flatpak-spawn --host`.

Mostly a proof-of-concept to explore the idea of allocating a new pty for the spawned process, to fix the following upstream issues:

* https://github.com/flatpak/flatpak/issues/3697
* https://github.com/flatpak/flatpak/issues/3285

## References

* https://github.com/owtaylor/PurpleEgg/blob/master/common/host-command.c
* https://github.com/gnunn1/tilix/blob/master/source/gx/tilix/terminal/terminal.d
