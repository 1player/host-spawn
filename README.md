# host-spawn

Run commands on your host machine from inside your flatpak sandbox, [toolbox](https://github.com/containers/toolbox) or [distrobox](https://github.com/89luca89/distrobox) containers.

Originally started as a reimplementation of `flatpak-spawn --host`.

## Recommended setup

**Note:** Distrobox already ships with host-spawn. You might be better served by using their wrapper `distrobox-host-exec` which runs host-spawn under the hood.

* Install host-spawn in a location visible only to the container. I recommend `/usr/local/bin`.
* Make sure it is executable with `chmod +x host-spawn`

## How to use

* `host-spawn` with no argument will open a shell on your host.
* `host-spawn command...` will run the command on your host.

Run `host-spawn -h` for more options.

## Creating shims for host binaries

If there's a process that you always want to execute on the host system, you can
create a symlink to it somewhere in your $PATH and it'll always be executed through `host-spawn`.

Example of creating a shim for the `flatpak` command:

```
# Inside your container:

$ flatpak --version
zsh: command not found: flatpak

# Have host-spawn handle any flatpak command
$ ln -s /usr/local/bin/host-spawn /usr/local/bin/flatpak

# Now flatpak will always be executed on the host
$ flatpak --version
Flatpak 1.12.7
```

**Note:** you will want to store the symlink in a location visible only to the container, to avoid an infinite loop. If you are using toolbox/distrobox, this means anywhere outside your home directory. I recommend `/usr/local/bin`.

## Improvements over flatpak-spawn --host

* Allocates a pty for the spawned process, fixing the following upstream issues: https://github.com/flatpak/flatpak/issues/3697, https://github.com/flatpak/flatpak/issues/3285 and https://github.com/flatpak/flatpak-xdg-utils/issues/57
* Handles SIGWINCH (terminal size changes)
* Passes through `$TERM` environment variable
* Shims host binaries when symlinked, see section above
