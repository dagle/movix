# Maintainer: Per Odlund <per.odlund@gmail.com>

pkgname='movix'
_pkgname='movix'
pkgver=0.0.1
pkgrel=1
pkgdesc='An offline media server'
arch=('x86_64')
url='https://github.com/dagle/movix'
license=('MIT')
# options=('!strip' '!emptydirs')
provides=('movix')
conflicts=('movix')
source=("git+https://github.com/dagle/movix")
sha256sums=('SKIP')
makedepends=(
    'go'
)

pkgver() {
	cd "$pkgname"
	git describe --long --abbrev=7 | sed 's/\([^-]*-g\)/r\1/;s/-/./g'
}

build() {
    go build -ldflags "-X main.LuaPath=${SCRIPT}" 
}

package() {
	make movix "${pkgdir}"
}

package() {
	make package "${pkgdir}"
}
