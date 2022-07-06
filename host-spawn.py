#!/usr/bin/env python

import sys
import os
import dbus
from dbus.types import *

bus = dbus.SessionBus()
proxy = bus.get_object("org.freedesktop.Flatpak", "/org/freedesktop/Flatpak/Development")

cwd_path = ByteArray(os.getcwd().encode('utf-8') + b'\0')
argv = map(lambda arg: ByteArray(arg.encode('utf-8') + b'\0'), sys.argv[1:])
envs = {
    'TERM': 'xterm-256color'
}
fds = {
    0: UnixFd(0),
    1: UnixFd(1),
    2: UnixFd(2)
}
flags = 0

pids = proxy.HostCommand(cwd_path, argv, fds, envs, flags, dbus_interface='org.freedesktop.Flatpak.Development')
print(pids)
