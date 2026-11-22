# DockNuke

A small, dependency-free Go CLI that inspects and cleans up Docker resources —
stopped containers, dangling or unused images, dangling volumes and builder
cache — with a safe `--dry-run` preview that shows exactly what would be
deleted and how much space it frees.

It talks to the local Docker daemon through the `docker` CLI (no SDK
dependency): candidates are collected from `docker ps`/`images`/`volume ls`,
space reclaimed is measured from `docker system df` before and after a run.

## Requirements

- Go 1.22+
- Docker Engine running locally

## Usage

```bash
git clone https://github.com/tatitoxt/docknuke.git && cd docknuke
go build -o docknuke ./cmd/docknuke

./docknuke --dry-run          # preview: lists every resource it would remove
./docknuke                    # remove stopped containers + dangling images
./docknuke --volumes          # also remove dangling volumes
./docknuke --all              # all unused images + volumes + build cache
./docknuke --build-cache      # also prune builder cache
./docknuke --keep-days 7      # keep anything created in the last 7 days
./docknuke --exclude postgres --exclude my-data  # skip matching names (repeatable)
```

By default DockNuke touches **only** stopped (`exited`/`created`/`dead`)
containers and *dangling* images — running containers and tagged images in use
are never removed. Reclaimed space is reported from the daemon's own
`docker system df` numbers, not estimates.

## Testing

```bash
go test ./...          # unit tests (runner stubbed; no daemon needed)
bash tests/e2e.sh      # E2E against the local daemon — creates real resources
                       # and deletes them. NOTE: also prunes other stale
                       # stopped containers/dangling resources on the daemon.
```

## Measured

A dry-run scan of a daemon holding **~180 resources** (29 images, 27
containers, 14 volumes, 114 build-cache entries) completes in **~0.24 s**
end-to-end, cold. See `docs/benchmarks.md` for the measurement. The E2E suite
(`tests/e2e.sh`) verifies on a live daemon that dry-run deletes nothing, that a
real run removes exactly the created fixtures (and that running containers and
in-use tagged images survive), and that a second run is a no-op.

## Limitations

- Operates on the **local** daemon only; no remote Docker contexts or SSH.
- Space reclaim is measured as the daemon-wide `docker system df` delta, so a
  concurrent build/pull during the run shifts the reported number.
- `--exclude` matches substrings of container names, image repo:tags and volume
  names — no glob or regex.
- `--keep-days` applies to containers and images (both expose `Created`);
  volumes carry no creation timestamp, so `--volumes` removes all dangling
  ones regardless of age.
- Windows/macOS: works wherever the `docker` CLI is on `PATH`, but only tested
  on Linux.

## License

MIT — see [LICENSE](LICENSE).
