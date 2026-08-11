package cleaner

import (
	"context"
	"time"

	"github.com/tatitoxt/docknuke/pkg/types"
)

type Engine interface {
	Run(ctx context.Context, opts types.NukeOptions) (*types.NukeReport, error)
}

type MockEngine struct{}

func NewMockEngine() Engine {
	return &MockEngine{}
}

func (m *MockEngine) Run(ctx context.Context, opts types.NukeOptions) (*types.NukeReport, error) {
	return &types.NukeReport{
		ContainersRemoved:   3,
		ImagesRemoved:       5,
		VolumesRemoved:      2,
		BuildCacheReclaimed: 1073741824, // 1 GB
		TotalSpaceReclaimed: 2147483648, // 2 GB
		Duration:            250 * time.Millisecond,
	}, nil
}

func ShouldKeep(createdAt time.Time, keepDays int) bool {
	if keepDays <= 0 {
		return false
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	return createdAt.After(cutoff)
}
