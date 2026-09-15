# Mova Context — Build
#
# CGO_ENABLED=0 on every target below is intentional, not incidental:
# mova has NO C dependencies (see docs/PROJECT.md § "AST filtering" —
# even the tree-sitter-compatible AST engine used by focus/
# context-trace is github.com/odvcencio/gotreesitter, a pure-Go
# runtime). That is what makes single-command cross-compilation to
# windows/linux/darwin, amd64/arm64, from ONE machine below actually
# work: a CGO build would need a matching C cross-compiler per target,
# which this Makefile never sets up. If a future dependency ever pulls
# in "import "C"", `go build` with CGO_ENABLED=0 fails loudly right
# here instead of silently producing a binary that needs a C
# toolchain the person building it doesn't have — see the "Refactor
# to remove tree-sitter's CGo binding" note in CHANGELOG-worthy history
# for exactly the situation this guards against.

# GO_TAGS: the astfilter package (core/focus/astfilter) supports a
# grammar_subset build mode that embeds ONLY the languages Mova
# Context actually uses (see astfilter/languages.go and
# docs/PROJECT.md § "AST filtering") instead of all ~200+ grammars the
# gotreesitter registry ships — this is what keeps the pure-Go AST
# engine from bloating the installer: a handful of MB for 11
# languages instead of ~20MB for the full registry. Add a new
# grammar_subset_<lang> tag here (comma-separated, no spaces — see
# grammars/z_subset_blob_embed_*.go for the exact per-language tag
# names) whenever astfilter/languages.go gains a new extension entry.
GO_TAGS = grammar_subset,grammar_subset_go,grammar_subset_python,grammar_subset_javascript,grammar_subset_typescript,grammar_subset_java,grammar_subset_c_sharp,grammar_subset_c,grammar_subset_cpp,grammar_subset_php,grammar_subset_ruby,grammar_subset_rust

ifeq ($(OS),Windows_NT)
    MKDIR_DIST = if not exist dist mkdir dist
    RM_RF = rmdir /s /q dist
    GO_BUILD = go build -tags "$(GO_TAGS)" -ldflags="-s -w"

    # Detectar arquitectura en Windows (AMD64 o ARM64)
    ARCH = amd64
    ifeq ($(PROCESSOR_ARCHITECTURE),ARM64)
        ARCH = arm64
    endif
    
    BINARY_NAME = mova-windows-$(ARCH).exe
    TARGET_NAME = mova.exe

build-all:
	$(MKDIR_DIST)
	cmd /c "set GOOS=windows&& set GOARCH=amd64&& set CGO_ENABLED=0&& $(GO_BUILD) -o dist/mova-windows-amd64.exe ./src/cli"
#	cmd /c "set GOOS=linux&& set GOARCH=amd64&& set CGO_ENABLED=0&& $(GO_BUILD) -o dist/mova-linux-amd64 ./src/cli"
#	cmd /c "set GOOS=darwin&& set GOARCH=amd64&& set CGO_ENABLED=0&& $(GO_BUILD) -o dist/mova-macos-amd64 ./src/cli"
#	cmd /c "set GOOS=darwin&& set GOARCH=arm64&& set CGO_ENABLED=0&& $(GO_BUILD) -o dist/mova-macos-arm64 ./src/cli"

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
    GO_BUILD = CGO_ENABLED=0 go build -tags "$(GO_TAGS)" -ldflags="-s -w"
    
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
	cd src && CGO_ENABLED=0 go test ./...