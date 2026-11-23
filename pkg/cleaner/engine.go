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
// A real run collects images and volumes a second time after container removal:
// the last container of an image or the last user of a volume is itself a
// candidate, so resources it released must be picked up in the same pass.
func (e *CliEngine) Run(ctx context.Context, opts types.NukeOptions) (*types.NukeReport, error) {
	report := &types.NukeReport{Detail: &types.NukeDetail{}}

	containers, err := e.Client.StoppedContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	var containerIDs []string
	for _, c := range containers {
		if ShouldKeep(c.Created, opts.KeepDays) || excluded(c.Name, opts.Excludes) {
			continue
		}
		containerIDs = append(containerIDs, c.ID)
		report.Detail.Containers = append(report.Detail.Containers, fmt.Sprintf("%s (%s)", c.Name, shortID(c.ID)))
	}
	report.ContainersRemoved = len(containerIDs)

	// Ids already queued across collection passes — separate per resource kind
	// so a volume name colliding with an image id can't hide a candidate.
	seenImages := map[string]bool{}
	seenVols := map[string]bool{}
	collect := func(excludeContainers map[string]bool) ([]string, []string, error) {
		images, err := e.Client.Images(ctx, opts.All, excludeContainers)
		if err != nil {
			return nil, nil, fmt.Errorf("list images: %w", err)
		}
		var volumes []docker.Volume
		if opts.IncludeVols || opts.All {
			volumes, err = e.Client.DanglingVolumes(ctx, excludeContainers)
			if err != nil {
				return nil, nil, fmt.Errorf("list volumes: %w", err)
			}
		}
		var imageIDs, volumeNames []string
		for _, img := range images {
			if ShouldKeep(img.Created, opts.KeepDays) || excluded(img.RepoTag, opts.Excludes) {
				continue
			}
			if seenImages[img.ID] {
				continue
			}
			seenImages[img.ID] = true
			imageIDs = append(imageIDs, img.ID)
			report.Detail.Images = append(report.Detail.Images, fmt.Sprintf("%s (%s)", img.RepoTag, img.ID))
			report.ImagesSize += img.Size
			report.ImagesRemoved++
		}
		for _, v := range volumes {
			if excluded(v.Name, opts.Excludes) || seenVols[v.Name] {
				continue
			}
			seenVols[v.Name] = true
			volumeNames = append(volumeNames, v.Name)
			report.Detail.Volumes = append(report.Detail.Volumes, v.Name)
			report.VolumesRemoved++
		}
		return imageIDs, volumeNames, nil
	}

	imageIDs, volumeNames, err := collect(nil)
	if err != nil {
		return nil, err
	}

	removing := map[string]bool{}
	for _, id := range containerIDs {
		removing[id] = true
	}
	if opts.DryRun {
		// Preview the post-removal world too: images and volumes held only by
		// the containers above would be released — a real run deletes them.
		if _, _, err := collect(removing); err != nil {
			return nil, err
		}
		return report, nil
	}

	before, _ := e.Client.Reclaimable(ctx)

	if err := e.Client.RemoveContainers(ctx, containerIDs); err != nil {
		return nil, fmt.Errorf("remove containers: %w", err)
	}
	// Second pass: removing the containers above may have released their
	// images and volumes — pick them up before deleting.
	extraImages, extraVolumes, err := collect(nil)
	if err != nil {
		return nil, err
	}
	imageIDs = append(imageIDs, extraImages...)
	volumeNames = append(volumeNames, extraVolumes...)

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
