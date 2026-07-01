# Building the Executable

To build the executable from the source code, you need **Go 1.22 or later** installed.

## Build for the Current Operating System

From the `cli/` directory:

```bash
cd cli

# Linux / macOS
go build -o mova .

# Windows
go build -o mova.exe .
```

## Build All Executables

If the project includes a `Makefile`:

```bash
make build
```

This will generate the binaries for all supported platforms in the distribution directory (`cli/dist/`), for example:

```text
cli/dist/
├── mova-windows-amd64.exe
├── mova-linux-amd64
├── mova-macos-amd64
└── mova-macos-arm64
```
