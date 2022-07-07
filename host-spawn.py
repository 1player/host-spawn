#!/usr/bin/env python

import sys
import os
import dbus
from gi.repository import GLib
from dbus.mainloop.glib import DBusGMainLoop
from dbus.types import *

DBusGMainLoop(set_as_default=True)
loop = GLib.MainLoop()

bus = dbus.SessionBus()
proxy = bus.get_object(
    "org.freedesktop.Flatpak", "/org/freedesktop/Flatpak/Development"
)


def host_command_exited(pid, status):
    loop.quit()


proxy.connect_to_signal(
    "HostCommandExited",
    host_command_exited,
    dbus_interface="org.freedesktop.Flatpak.Development",
)

cwd_path = ByteArray(os.getcwd().encode("utf-8") + b"\0")
argv = map(lambda arg: ByteArray(arg.encode("utf-8") + b"\0"), sys.argv[1:])
envs = {"TERM": "xterm-256color"}
fds = {0: UnixFd(0), 1: UnixFd(1), 2: UnixFd(2)}
flags = 0

pids = proxy.HostCommand(
    cwd_path,
    argv,
    fds,
    envs,
    flags,
    dbus_interface="org.freedesktop.Flatpak.Development",
)
print(pids)

loop.run()
