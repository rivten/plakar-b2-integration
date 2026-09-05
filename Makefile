PLAKAR = /home/hugo/go/bin/plakar
VERSION = v0.0.1

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
PTAR := backblaze_$(VERSION)_$(GOOS)_$(GOARCH).ptar

build:
	go build -v -o b2Storage ./plugin/storage

package: build
	rm -f $(PTAR)
	$(PLAKAR) pkg create ./manifest.yaml $(VERSION)

uninstall:
	-$(PLAKAR) pkg rm backblaze

install: package
	$(PLAKAR) pkg add -allow-unsigned ./$(PTAR)

reinstall: uninstall install
