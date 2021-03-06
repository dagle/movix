# build, test, install
#
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
MANDIR := $(PREFIX)/share/man
SCRIPT := $(PREFIX)/share/movix
SRC = cmd/movix/main.go

movix:
	go build -ldflags "-X main.LuaPath=${SCRIPT}" -o $@ ./cmd/$@

install: movix
	install -Dm755 movix -t '$(DESTDIR)$(BINDIR)'
	install -Dm755 movix.lua -t '$(DESTDIR)$(SCRIPT)'
