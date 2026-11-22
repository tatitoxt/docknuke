package cleaner

import (
	"context"
	"testing"
	"time"

	"github.com/tatitoxt/docknuke/pkg/docker"
	"github.com/tatitoxt/docknuke/pkg/types"
)

// runnerStub answers the docker calls an engine run makes, in order.
func runnerStub() docker.Runner {
	created := time.Now().Add(-48 * time.Hour).Format("2006-01-02 15:04:05 -0700 MST")
	return func(_ context.Context, args ...string) (string, error) {
		switch {
		case args[0] == "ps" && args[1] == "-aq" && len(args) > 2:
			return "idA\nidB\n", nil // stopped containers
		case args[0] == "inspect":
			return "idA /keep-me " + created + "\nidB /drop-me " + created + "\n", nil
		case args[0] == "images":
			return "img1|<none>:<none>|10MB|2026-11-01 10:00:00 +0000 UTC\n", nil
		case args[0] == "volume":
			return "vol1\nvol2-keep\n", nil
		case args[0] == "system":
			return `{"Type":"Images","Reclaimable":"10MB (1%)"}` + "\n", nil
		default:
			return "", nil
		}
	}
}

func TestEngineDryRunFiltersAndDeletesNothing(t *testing.T) {
	eng := &CliEngine{Client: docker.New(runnerStub())}
	report, err := eng.Run(context.Background(), types.NukeOptions{
		DryRun:      true,
		IncludeVols: true,
		Excludes:    []string{"keep"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.ContainersRemoved != 1 || report.Detail.Containers[0][:7] != "drop-me" {
		t.Fatalf("expected only drop-me container, got %+v", report.Detail.Containers)
	}
	if report.VolumesRemoved != 1 || report.Detail.Volumes[0] != "vol1" {
		t.Fatalf("expected only vol1, got %+v", report.Detail.Volumes)
	}
	if report.ImagesRemoved != 1 {
		t.Fatalf("expected the dangling image, got %d", report.ImagesRemoved)
	}
	if report.TotalSpaceReclaimed != 0 {
		t.Fatal("dry-run must report zero reclaimed")
	}
}
