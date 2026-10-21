CC ?= gcc
CFLAGS = -Wall -Wextra -O2 -std=c99
PREFIX ?= /usr/local

.PHONY: all clean install

all: kacu-sysinfo

kacu-sysinfo: kacu-sysinfo.c
	$(CC) $(CFLAGS) -o $@ $<

install: kacu-sysinfo
	install -Dm755 kacu-sysinfo $(DESTDIR)$(PREFIX)/bin/kacu-sysinfo

clean:
	rm -f kacu-sysinfo
