# Mova Context — Build

ifeq ($(OS),Windows_NT)
	MKDIR_DIST = if not exist dist mkdir dist
	RM_RF = rmdir /s /q dist
	GO_BUILD = go build

build-all:
	$(MKDIR_DIST)
#	set GOOS=linux&& set GOARCH=amd64&& $(GO_BUILD) -ldflags="-s -w" -o dist/mova-linux-amd64 ./src/cli
#	set GOOS=darwin&& set GOARCH=amd64&& $(GO_BUILD) -ldflags="-s -w" -o dist/mova-macos-amd64 ./src/cli
#	set GOOS=darwin&& set GOARCH=arm64&& $(GO_BUILD) -ldflags="-s -w" -o dist/mova-macos-arm64 ./src/cli
	set GOOS=windows&& set GOARCH=amd64&& $(GO_BUILD) -ldflags="-s -w" -o dist/mova-windows-amd64.exe ./src/cli
  
else
	MKDIR_DIST = mkdir -p dist
	RM_RF = rm -rf dist
	GO_BUILD = go build

build-all:
	$(MKDIR_DIST)
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -ldflags="-s -w" -o dist/mova-linux-amd64 ./src/cli
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -ldflags="-s -w" -o dist/mova-macos-amd64 ./src/cli
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -ldflags="-s -w" -o dist/mova-macos-arm64 ./src/cli
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -ldflags="-s -w" -o dist/mova-windows-amd64.exe ./src/cli

endif

build:
	$(MKDIR_DIST)
	go build -ldflags="-s -w" -o dist/mova ./src/cli

clean:
	$(RM_RF)

test:
	go test ./...