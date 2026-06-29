.PHONY: build frontend appimage clean

VERSION ?= $(shell git describe --tags --always --dirty)

frontend:
	cd web && npm ci && npm run build

build: frontend
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o dist/abacus ./cmd/abacus

appimage: build
	rm -rf Abacus.AppDir
	mkdir -p Abacus.AppDir/usr/bin
	cp dist/abacus Abacus.AppDir/usr/bin/abacus
	cp packaging/appimage/Abacus.desktop Abacus.AppDir/
	cp packaging/appimage/abacus.png Abacus.AppDir/
	ln -sf abacus.png Abacus.AppDir/.DirIcon
	cp packaging/appimage/AppRun Abacus.AppDir/AppRun
	chmod +x Abacus.AppDir/AppRun
	ARCH=x86_64 appimagetool Abacus.AppDir dist/Abacus-$(VERSION)-x86_64.AppImage

clean:
	rm -rf dist/ Abacus.AppDir
