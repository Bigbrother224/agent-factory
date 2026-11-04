# TeleBlock — construction et livraison
#
# Aucune toolchain croisée n'est nécessaire : Go produit un binaire
# linux/arm64 statique (CGO_ENABLED=0) depuis n'importe quelle machine.

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
ARM64_BINARY := dist/teleblock-linux-arm64

.PHONY: build arm64 pi-image clean

build:
	go build -o dist/teleblock .

# Binaire statique pour le Pi : CGO désactivé, symboles retirés, version
# estampillée. C'est exactement ce que build-image.sh pose dans l'image.
arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath \
		-ldflags "-s -w -X main.version=$(VERSION)" -o $(ARM64_BINARY) .

# Charge utile de l'image Raspberry Pi : rootfs/ + manifeste + archive.
# Puis : sudo tar -C <partition root de la SD> -xzf dist/teleblock-pi-rootfs.tar.gz
pi-image:
	VERSION=$(VERSION) bash deploy/pi/build-image.sh

clean:
	rm -rf dist
