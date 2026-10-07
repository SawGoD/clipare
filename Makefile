.PHONY: mac windows windows-console release-mac release-windows test test-macos-ui

VERSION := $(strip $(shell sed -n '1p' VERSION))
COMMIT := $(shell git rev-parse HEAD)
LDFLAGS := -X clipare.Commit=$(COMMIT)

mac:
	mkdir -p dist/Clipare.app/Contents/MacOS
	sed -e 's/@VERSION@/$(VERSION)/g' -e 's/@BUILD@/$(VERSION)/g' assets/Info.plist > dist/Clipare.app/Contents/Info.plist
	CGO_ENABLED=1 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/Clipare.app/Contents/MacOS/Clipare ./cmd/clipare

windows:
	mkdir -p dist
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS) -H=windowsgui" -o dist/Clipare.exe ./cmd/clipare

windows-console:
	mkdir -p dist
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/clipare-console.exe ./cmd/clipare

release-mac:
	go run ./scripts/build -os darwin -arch arm64
	go run ./scripts/build -os darwin -arch amd64

release-windows:
	go run ./scripts/build -os windows -arch amd64
	go run ./scripts/build -os windows -arch arm64

test:
	go test -race ./...
	go vet ./...

# Runs in an interactive macOS session; briefly opens a test window.
test-macos-ui:
	mkdir -p dist
	clang -fblocks -framework AppKit -o dist/edit-menu-test tests/macos/edit_menu_test.m
	./dist/edit-menu-test
