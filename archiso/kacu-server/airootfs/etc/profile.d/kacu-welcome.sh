#!/bin/sh
# Sourced by /etc/profile for interactive login shells, so it must stay POSIX
# sh. The interactive logic itself lives in /etc/kacu/welcome.py.
if [ -n "$PS1" ] && [ -f /etc/kacu/welcome.py ] && command -v python3 >/dev/null 2>&1; then
    python3 /etc/kacu/welcome.py
fi
