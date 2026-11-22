package cleaner

import (
	"context"
	"fmt"
	"strings"

	"github.com/tatitoxt/docknuke/pkg/docker"
	"github.com/tatitoxt/docknuke/pkg/types"
)

// CliEngine runs a clean against the local Docker daemon via the docker CLI.
type CliEngine struct {
	Client *docker.Client
}

func NewCliEngine() *CliEngine {
	return &CliEngine{Client: docker.New(nil)}
}

// Run collects candidates, then either reports them (dry-run) or deletes them.
func (e *CliEngine) Run(ctx context.Context, opts types.NukeOptions) (*types.NukeReport, error) {
	report := &types.NukeReport{Detail: &types.NukeDetail{}}

	containers, err := e.Client.StoppedContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	images, err := e.Client.Images(ctx, opts.All)
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}
	var volumes []docker.Volume
	if opts.IncludeVols || opts.All {
		volumes, err = e.Client.DanglingVolumes(ctx)
		if err != nil {
			return nil, fmt.Errorf("list volumes: %w", err)
		}
	}

	var containerIDs, imageIDs, volumeNames []string
	for _, c := range containers {
		if ShouldKeep(c.Created, opts.KeepDays) || excluded(c.Name, opts.Excludes) {
			continue
		}
		containerIDs = append(containerIDs, c.ID)
		report.Detail.Containers = append(report.Detail.Containers, fmt.Sprintf("%s (%s)", c.Name, shortID(c.ID)))
	}
	for _, img := range images {
		if ShouldKeep(img.Created, opts.KeepDays) || excluded(img.RepoTag, opts.Excludes) {
			continue
		}
		imageIDs = append(imageIDs, img.ID)
		report.Detail.Images = append(report.Detail.Images, fmt.Sprintf("%s (%s)", img.RepoTag, img.ID))
		report.ImagesSize += img.Size
	}
	for _, v := range volumes {
		if excluded(v.Name, opts.Excludes) {
			continue
		}
		volumeNames = append(volumeNames, v.Name)
		report.Detail.Volumes = append(report.Detail.Volumes, v.Name)
	}

	report.ContainersRemoved = len(containerIDs)
	report.ImagesRemoved = len(imageIDs)
	report.VolumesRemoved = len(volumeNames)

	if opts.DryRun {
		return report, nil
	}

	before, _ := e.Client.Reclaimable(ctx)

	if err := e.Client.RemoveContainers(ctx, containerIDs); err != nil {
		return nil, fmt.Errorf("remove containers: %w", err)
	}
	if err := e.Client.RemoveImages(ctx, imageIDs); err != nil {
		return nil, fmt.Errorf("remove images: %w", err)
	}
	if err := e.Client.RemoveVolumes(ctx, volumeNames); err != nil {
		return nil, fmt.Errorf("remove volumes: %w", err)
	}
	if opts.IncludeBuild || opts.All {
		if err := e.Client.PruneBuildCache(ctx); err != nil {
			return nil, fmt.Errorf("prune build cache: %w", err)
		}
		report.BuildCachePruned = true
	}

	after, err := e.Client.Reclaimable(ctx)
	if err == nil && before != nil {
		var reclaimed int64
		for t, b := range before {
			if d := b - after[t]; d > 0 {
				reclaimed += d
			}
		}
		report.TotalSpaceReclaimed = reclaimed
		report.BuildCacheReclaimed = before["build cache"] - after["build cache"]
		if report.BuildCacheReclaimed < 0 {
			report.BuildCacheReclaimed = 0
		}
	}
	return report, nil
}

// shortID truncates a docker id for display (full ids are 64 chars).
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// excluded reports whether name matches any --exclude substring.
func excluded(name string, excludes []string) bool {
	for _, ex := range excludes {
		if ex != "" && strings.Contains(name, ex) {
			return true
		}
	}
	return false
}
