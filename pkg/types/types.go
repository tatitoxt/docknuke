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

type NukeReport struct {
	ContainersRemoved   int
	ImagesRemoved       int
	VolumesRemoved      int
	BuildCacheReclaimed int64
	TotalSpaceReclaimed int64
	Duration            time.Duration
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
