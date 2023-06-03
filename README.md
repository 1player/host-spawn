# host-spawn

A reimplementation of `flatpak-spawn --host`.

Run commands on your host machine from inside your flatpak sandbox, [toolbox](https://github.com/containers/toolbox) or [distrobox](https://github.com/89luca89/distrobox) containers.

## Improvements over the original

* Allocates a pty for the spawned process, fixing the following upstream issues: https://github.com/flatpak/flatpak/issues/3697, https://github.com/flatpak/flatpak/issues/3285 and https://github.com/flatpak/flatpak-xdg-utils/issues/57
* Handles SIGWINCH (terminal size changes)
* Passes through `$TERM` environment variable
* Shims host binaries when symlinked, see section below

## Creating shims for host binaries

If there's a process that only makes sense to be executed on the host system, you can
create a symlink to it somewhere in your $PATH and it'll always be executed through `host-spawn`.

*Note:* you will want to store the symlink in a location visible only to the container, to avoid an infinite loop. If you are using toolbox/distrobox, this means anywhere outside your home directory. I recommend `/usr/local/bin`. See https://github.com/1player/host-spawn/issues/19 for details.

Example of creating a shim for the `flatpak` command:

```
# Inside your container

$ flatpak --version
zsh: command not found: flatpak
$ ln -s /usr/local/bin/host-spawn /usr/local/bin/flatpak
# Now the flatpak command will always be executed on the host
$ flatpak --version
Flatpak 1.12.7
```

## References

* https://github.com/owtaylor/PurpleEgg/blob/master/common/host-command.c
* https://github.com/gnunn1/tilix/blob/master/source/gx/tilix/terminal/terminal.d
