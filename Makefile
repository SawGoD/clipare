.PHONY: mac windows windows-console test

mac:
	mkdir -p dist/Clipare.app/Contents/MacOS
	cp assets/Info.plist dist/Clipare.app/Contents/Info.plist
	CGO_ENABLED=1 go build -trimpath -o dist/Clipare.app/Contents/MacOS/Clipare ./cmd/clipare

windows:
	mkdir -p dist
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H=windowsgui" -o dist/Clipare.exe ./cmd/clipare

windows-console:
	mkdir -p dist
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o dist/clipare-console.exe ./cmd/clipare

test:
	go test -race ./...
	go vet ./...
