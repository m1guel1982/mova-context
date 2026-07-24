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

install: build
	@for /f "delims=" %%g in ('go env GOPATH') do copy /Y dist\mova.exe "%%g\bin\mova.exe"
	@echo Installed to %GOPATH%\bin\mova.exe — make sure that folder is in your PATH.

else
	MKDIR_DIST = mkdir -p dist
	RM_RF = rm -rf dist
	GO_BUILD = go build

build-all:
	$(MKDIR_DIST)
#	GOOS=linux GOARCH=amd64 $(GO_BUILD) -ldflags="-s -w" -o dist/mova-linux-amd64 ./src/cli
#	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -ldflags="-s -w" -o dist/mova-macos-amd64 ./src/cli
#	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -ldflags="-s -w" -o dist/mova-macos-arm64 ./src/cli
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -ldflags="-s -w" -o dist/mova-windows-amd64.exe ./src/cli

# install builds and copies the binary into `go env GOPATH`/bin — the same
# folder `go install` itself always uses, on Linux and macOS alike. We
# don't use `go install ./src/cli` directly because Go would name the
# binary after the last path element ("cli"), not "mova".
install: build
	@mkdir -p "$$(go env GOPATH)/bin"
	@cp dist/mova "$$(go env GOPATH)/bin/mova"
	@echo "Installed to $$(go env GOPATH)/bin/mova — make sure that folder is in your PATH."

endif

build:
	$(MKDIR_DIST)
	go build -ldflags="-s -w" -o dist/mova ./src/cli

clean:
	$(RM_RF)

test:
	cd src && go test ./...