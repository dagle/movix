# build, test, install
#
# install -Dm755 movix -t '$(DESTDIR)$(BINDIR)'

PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
MANDIR := $(PREFIX)/share/man
SCRIPT := $(PREFIX)/share/movix
SRC = main.go

movix-install:
	go install -ldflags "-X main.LuaPath=${SCRIPT}"

movix:
	go build -ldflags "-X main.LuaPath=${SCRIPT}"
	
movix.1:
	scdoc < movix.1.scd > movix.1

package: movix movix.1
	scdoc < movix.1.scd > movix.1
	install -Dm755 movix -t '$(BINDIR)'
	install -Dm644 movix.1 -t '$(MANDIR)'
	install -Dm755 movix.lua -t '$(SCRIPT)'
