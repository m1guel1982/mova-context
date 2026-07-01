Compilación del ejecutable

Si deseas generar el ejecutable desde el código fuente, necesitas tener Go 1.22 o superior instalado.

Compilar para el sistema operativo actual

Desde la carpeta cli/:

cd cli

# Linux / macOS
go build -o mova .

# Windows
go build -o mova.exe .

Compilar todos los ejecutables

Si el proyecto incluye un Makefile:

make build


Esto generará los binarios para las plataformas soportadas en la carpeta de distribución (cli/dist/), por ejemplo:

cli/dist/
├── mova-windows-amd64.exe
├── mova-linux-amd64
├── mova-macos-amd64
└── mova-macos-arm64