# build, test, install
#
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
MANDIR := $(PREFIX)/share/man
SCRIPT := $(PREFIX)/share/movix
SRC = main.go

movix:
	go install -ldflags "-X main.LuaPath=${SCRIPT}"
	
man:
	scdoc < movix.1.scd > movix.1

install: movix
	install -Dm755 movix -t '$(DESTDIR)$(BINDIR)'
	install -Dm755 movix.lua -t '$(DESTDIR)$(SCRIPT)'
