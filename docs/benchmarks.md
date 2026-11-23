# Benchmarks

Measured 2026-11-22 on the dev machine (Docker Engine 29.9.2, Go 1.26, Linux).

## Scan latency (dry-run)

Daemon contents at measurement time: 29 images, 27 containers, 14 volumes,
114 build-cache entries (~180 resources total).

```bash
go build -o docknuke ./cmd/docknuke
/usr/bin/time ./docknuke --dry-run --volumes
```

| Run | Wall time |
| --- | --------- |
| 1 | 0.25 s |
| 2 | 0.24 s |
| 3 | 0.23 s |

**Result: ~0.24 s** for a full candidate scan (containers + images + volumes)
across ~180 daemon resources. The scan is dominated by `docker` CLI process
startup; resource count scales linearly in the inspect batch calls.

## Reclaim verification

`tests/e2e.sh` asserts on a live daemon that a real run deletes the created
fixtures and that `total reclaimed` (from `docker system df` deltas) reports
the bytes docker itself counts as freed — in the recorded run, 7.58 MB from
the fixture resources alone.
