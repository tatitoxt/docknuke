package types

import (
	"fmt"
	"time"
)

type NukeOptions struct {
	DryRun       bool
	Force        bool
	All          bool
	KeepDays     int
	Excludes     []string
	IncludeVols  bool
	IncludeBuild bool
}

// NukeDetail lists exactly what was (or would be) removed.
type NukeDetail struct {
	Containers []string
	Images     []string
	Volumes    []string
}

type NukeReport struct {
	ContainersRemoved   int
	ImagesRemoved       int
	VolumesRemoved      int
	ImagesSize          int64 // sum of removed image sizes, best effort
	BuildCacheReclaimed int64
	BuildCachePruned    bool
	TotalSpaceReclaimed int64
	Duration            time.Duration
	Detail              *NukeDetail
}

func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
