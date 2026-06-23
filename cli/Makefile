build:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o dist/mova-linux-amd64   .
	GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o dist/mova-macos-amd64   .
	GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o dist/mova-macos-arm64   .
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/mova-windows-amd64.exe .

clean:
	rm -rf dist/

test:
	go test ./...
