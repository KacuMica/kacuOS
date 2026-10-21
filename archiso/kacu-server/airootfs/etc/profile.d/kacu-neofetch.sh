#!/bin/sh
if [ -n "$PS1" ] && command -v fastfetch >/dev/null 2>&1; then
    fastfetch --config /etc/kacu/fastfetch.jsonc
fi
