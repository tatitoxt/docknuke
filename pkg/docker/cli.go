// Package docker wraps the docker CLI with typed, machine-readable calls.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Runner executes a docker command and returns stdout. Injected for tests.
type Runner func(ctx context.Context, args ...string) (string, error)

// CLIRunner is the production runner.
func CLIRunner(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "docker", args...).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("docker %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

type Client struct{ run Runner }

func New(run Runner) *Client {
	if run == nil {
		run = CLIRunner
	}
	return &Client{run: run}
}

// Container is a stopped container candidate.
type Container struct {
	ID      string
	Name    string
	Created time.Time
}

// Image is an image candidate (dangling or unused).
type Image struct {
	ID      string
	RepoTag string // "<none>:<none>" for dangling
	Size    int64
	Created time.Time
	InUse   bool
}

// Volume is a dangling volume candidate.
type Volume struct{ Name string }

func ids(out string) []string {
	var ids []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			ids = append(ids, l)
		}
	}
	return ids
}

// StoppedContainers returns containers in exited/created/dead state.
func (c *Client) StoppedContainers(ctx context.Context) ([]Container, error) {
	out, err := c.run(ctx, "ps", "-aq", "-f", "status=exited", "-f", "status=created", "-f", "status=dead")
	if err != nil {
		return nil, err
	}
	list := ids(out)
	if len(list) == 0 {
		return nil, nil
	}
	args := append([]string{"inspect", "-f", `{{.Id}} {{.Name}} {{.Created}}`}, list...)
	out, err = c.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var res []Container
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.Fields(l)
		if len(f) < 2 {
			continue
		}
		created, _ := time.Parse(time.RFC3339Nano, strings.Join(f[2:], " "))
		res = append(res, Container{ID: f[0], Name: strings.TrimPrefix(f[1], "/"), Created: created})
	}
	return res, nil
}

// Images returns dangling images, or every image (unused included) when all is
// true. excludeContainers simulates a post-removal world for dry-run previews:
// an image referenced only by those container ids counts as unused.
func (c *Client) Images(ctx context.Context, all bool, excludeContainers map[string]bool) ([]Image, error) {
	out, err := c.run(ctx, "images", "-a", "--format", `{{.ID}}|{{.Repository}}:{{.Tag}}|{{.Size}}|{{.CreatedAt}}`)
	if err != nil {
		return nil, err
	}
	users, err := c.imageUsers(ctx)
	if err != nil {
		return nil, err
	}
	var res []Image
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if l == "" {
			continue
		}
		f := strings.SplitN(l, "|", 4)
		if len(f) < 4 {
			continue
		}
		img := Image{
			ID:      f[0],
			RepoTag: f[1],
			Size:    parseHumanSize(f[2]),
			InUse:   inUseBy(users[f[0]], excludeContainers),
		}
		img.Created = parseDockerTime(f[3])
		dangling := f[1] == "<none>:<none>"
		if (img.InUse || !dangling) && !(all && !img.InUse) {
			continue
		}
		res = append(res, img)
	}
	return res, nil
}

func inUseBy(containers []string, excluded map[string]bool) bool {
	for _, c := range containers {
		if !excluded[c] {
			return true
		}
	}
	return false
}

// imageUsers maps each image's short id to the ids of containers using it.
func (c *Client) imageUsers(ctx context.Context) (map[string][]string, error) {
	out, err := c.run(ctx, "ps", "-aq")
	if err != nil {
		return nil, err
	}
	list := ids(out)
	users := map[string][]string{}
	if len(list) == 0 {
		return users, nil
	}
	args := append([]string{"inspect", "-f", `{{.Id}}|{{.Image}}`}, list...)
	out, err = c.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.SplitN(l, "|", 2)
		if len(f) != 2 {
			continue
		}
		ctr, img := f[0], strings.TrimPrefix(f[1], "sha256:")
		key := img
		if len(key) > 12 {
			key = key[:12]
		}
		users[key] = append(users[key], ctr)
	}
	return users, nil
}

// volumeUsers maps each volume name to the ids of containers mounting it.
func (c *Client) volumeUsers(ctx context.Context) (map[string][]string, error) {
	out, err := c.run(ctx, "ps", "-aq")
	if err != nil {
		return nil, err
	}
	list := ids(out)
	users := map[string][]string{}
	if len(list) == 0 {
		return users, nil
	}
	args := append([]string{"inspect", "-f", `{{.Id}}|{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}};{{end}}{{end}}`}, list...)
	out, err = c.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.SplitN(l, "|", 2)
		if len(f) != 2 {
			continue
		}
		for _, name := range strings.Split(strings.TrimSuffix(f[1], ";"), ";") {
			if name != "" {
				users[name] = append(users[name], f[0])
			}
		}
	}
	return users, nil
}

// DanglingVolumes returns volumes not referenced by any container — plus, when
// excludeContainers is set, volumes referenced only by those container ids
// (the post-removal world a dry-run previews).
func (c *Client) DanglingVolumes(ctx context.Context, excludeContainers map[string]bool) ([]Volume, error) {
	out, err := c.run(ctx, "volume", "ls", "-q", "-f", "dangling=true")
	if err != nil {
		return nil, err
	}
	var res []Volume
	seen := map[string]bool{}
	for _, name := range ids(out) {
		seen[name] = true
		res = append(res, Volume{Name: name})
	}
	if len(excludeContainers) == 0 {
		return res, nil
	}
	users, err := c.volumeUsers(ctx)
	if err != nil {
		return nil, err
	}
	for name, containers := range users {
		if seen[name] {
			continue
		}
		if !inUseBy(containers, excludeContainers) {
			seen[name] = true
			res = append(res, Volume{Name: name})
		}
	}
	return res, nil
}

// RemoveContainers / RemoveImages / RemoveVolumes delete the given ids.
func (c *Client) RemoveContainers(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := c.run(ctx, append([]string{"rm"}, ids...)...)
	return err
}

// RemoveImages deletes images by id, one at a time. Deleting by id pins the
// exact image — a mutable tag cannot redirect the removal to a replacement
// build, and digest-only rows (repository:<none>) still resolve. Removal falls
// back to `rmi -f` so a dangling parent whose layers are retained by a child
// image untags instead of aborting the whole clean.
func (c *Client) RemoveImages(ctx context.Context, ids []string) error {
	var errs []string
	for _, id := range ids {
		if _, err := c.run(ctx, "rmi", id); err != nil {
			if _, ferr := c.run(ctx, "rmi", "-f", id); ferr != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", id, ferr))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("rmi failures: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (c *Client) RemoveVolumes(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}
	_, err := c.run(ctx, append([]string{"volume", "rm"}, names...)...)
	return err
}

// PruneBuildCache clears builder cache (all, not just dangling entries).
func (c *Client) PruneBuildCache(ctx context.Context) error {
	_, err := c.run(ctx, "builder", "prune", "-af")
	return err
}

// dfEntry is one row of `docker system df --format json`.
type dfEntry struct {
	Type        string `json:"Type"`
	Reclaimable string `json:"Reclaimable"`
}

// Reclaimable returns bytes docker reports as reclaimable per resource type.
func (c *Client) Reclaimable(ctx context.Context) (map[string]int64, error) {
	out, err := c.run(ctx, "system", "df", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	res := map[string]int64{}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if l == "" {
			continue
		}
		var e dfEntry
		if json.Unmarshal([]byte(l), &e) != nil {
			continue
		}
		// "1.389GB (12%)" → strip the percentage
		size := strings.TrimSpace(strings.SplitN(e.Reclaimable, "(", 2)[0])
		res[strings.ToLower(e.Type)] = parseHumanSize(size)
	}
	return res, nil
}

// parseHumanSize parses docker sizes like "1.389GB", "875.2MB", "0B".
func parseHumanSize(s string) int64 {
	s = strings.TrimSpace(s)
	units := map[string]float64{"KB": 1e3, "MB": 1e6, "GB": 1e9, "TB": 1e12, "KIB": 1 << 10, "MIB": 1 << 20, "GIB": 1 << 30}
	upper := strings.ToUpper(s)
	for suffix, mult := range units {
		if strings.HasSuffix(upper, suffix) {
			v, _ := strconv.ParseFloat(strings.TrimSpace(s[:len(s)-len(suffix)]), 64)
			return int64(v * mult)
		}
	}
	if strings.HasSuffix(upper, "B") {
		v, _ := strconv.ParseFloat(strings.TrimSpace(s[:len(s)-1]), 64)
		return int64(v)
	}
	v, _ := strconv.ParseFloat(s, 64)
	return int64(v)
}

func parseDockerTime(s string) time.Time {
	// "2026-11-22 06:38:00 -0300 -03" or RFC3339 variants
	for _, layout := range []string{"2006-01-02 15:04:05 -0700 MST", time.RFC3339Nano, "2006-01-02 15:04:05 -0700"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
