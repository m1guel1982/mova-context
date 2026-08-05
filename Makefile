# Mova Context — Build

ifeq ($(OS),Windows_NT)
    MKDIR_DIST = if not exist dist mkdir dist
    RM_RF = rmdir /s /q dist
    GO_BUILD = go build -ldflags="-s -w"
    
    # Detectar arquitectura en Windows (AMD64 o ARM64)
    ARCH = amd64
    ifeq ($(PROCESSOR_ARCHITECTURE),ARM64)
        ARCH = arm64
    endif
    
    BINARY_NAME = mova-windows-$(ARCH).exe
    TARGET_NAME = mova.exe

build-all:
	$(MKDIR_DIST)
	cmd /c "set GOOS=windows&& set GOARCH=amd64&& $(GO_BUILD) -o dist/mova-windows-amd64.exe ./src/cli"
#	cmd /c "set GOOS=linux&& set GOARCH=amd64&& $(GO_BUILD) -o dist/mova-linux-amd64 ./src/cli"
#	cmd /c "set GOOS=darwin&& set GOARCH=amd64&& $(GO_BUILD) -o dist/mova-macos-amd64 ./src/cli"
#	cmd /c "set GOOS=darwin&& set GOARCH=arm64&& $(GO_BUILD) -o dist/mova-macos-arm64 ./src/cli"

install:
	@for /f "delims=" %%g in ('go env GOPATH') do ( \
		if not exist "%%g\bin" mkdir "%%g\bin" && \
		copy /Y "dist\$(BINARY_NAME)" "%%g\bin\$(TARGET_NAME)" && \
		powershell -NoProfile -ExecutionPolicy Bypass -Command \
			"$$gopathBin = '%%g\bin'; \
			 $$oldPath = [Environment]::GetEnvironmentVariable('Path', 'User'); \
			 if (-not $$oldPath.Split(';').Contains($$gopathBin)) { \
				 [Environment]::SetEnvironmentVariable('Path', $$oldPath + ';' + $$gopathBin, 'User'); \
				 Write-Host 'Agregado %%g\bin al PATH de usuario en Windows.'; \
			 }" \
	)
	@echo Installed successfully as $(TARGET_NAME).

else
    MKDIR_DIST = mkdir -p dist
    RM_RF = rm -rf dist
    GO_BUILD = go build -ldflags="-s -w"
    
    GOPATH_DIR := $(shell go env GOPATH)
    
    UNAME_S := $(shell uname -s | tr '[:upper:]' '[:lower:]')
    ifeq ($(UNAME_S),darwin)
        OS_NAME = macos
        SHELL_PROFILE = $(HOME)/.zshrc
    else
        OS_NAME = linux
        SHELL_PROFILE = $(HOME)/.bashrc
    endif

    UNAME_M := $(shell uname -m)
    ifeq ($(UNAME_M),x86_64)
        ARCH_NAME = amd64
    else ifeq ($(UNAME_M),arm64)
        ARCH_NAME = arm64
    else ifeq ($(UNAME_M),aarch64)
        ARCH_NAME = arm64
    endif

    BINARY_NAME = mova-$(OS_NAME)-$(ARCH_NAME)
    TARGET_NAME = mova

build-all:
	$(MKDIR_DIST)
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o dist/mova-windows-amd64.exe ./src/cli
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -o dist/mova-linux-amd64 ./src/cli
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o dist/mova-macos-amd64 ./src/cli
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -o dist/mova-macos-arm64 ./src/cli

install:
	@mkdir -p "$(GOPATH_DIR)/bin"
	@cp "dist/$(BINARY_NAME)" "$(GOPATH_DIR)/bin/$(TARGET_NAME)"
	@grep -qF '$(GOPATH_DIR)/bin' $(SHELL_PROFILE) 2>/dev/null || \
		(echo 'export PATH="$(GOPATH_DIR)/bin:$$PATH"' >> $(SHELL_PROFILE) && \
		 echo "Agregado $(GOPATH_DIR)/bin a $(SHELL_PROFILE)"); \
	echo "Installed successfully to $(GOPATH_DIR)/bin/$(TARGET_NAME)."

endif

build:
	$(MKDIR_DIST)
	$(GO_BUILD) -o dist/$(TARGET_NAME) ./src/cli

clean:
	$(RM_RF)

test:
	cd src && go test ./...