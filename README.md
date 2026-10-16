# DockNuke

DockNuke es una pequeña CLI en Go para inspeccionar y limpiar recursos de Docker desde la terminal. Incluye un modo de vista previa para revisar las acciones antes de borrar recursos.

```bash
git clone https://github.com/tatitoxt/docknuke.git && cd docknuke
go build -o docknuke ./cmd/docknuke
./docknuke --dry-run
```

## Requisitos

- Go 1.22+
- Docker Engine disponible localmente

## Compilar y ejecutar

```bash
go test ./...
```

Para previsualizar una limpieza:

```bash
./docknuke --dry-run
```

El proyecto todavía es pequeño; revisá la salida y las opciones antes de ejecutar una limpieza real. Las pruebas actuales cubren los paquetes `cleaner` y `types`.

## Licencia

MIT. Ver [LICENSE](LICENSE).
