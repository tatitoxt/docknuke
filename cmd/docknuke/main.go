package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tatitoxt/docknuke/pkg/cleaner"
	"github.com/tatitoxt/docknuke/pkg/types"
)

var version = "1.1.0"

type excludes []string

func (e *excludes) String() string { return strings.Join(*e, ",") }
func (e *excludes) Set(v string) error {
	*e = append(*e, v)
	return nil
}

func main() {
	dryRun := flag.Bool("dry-run", false, "List what would be removed without deleting anything")
	keepDays := flag.Int("keep-days", 0, "Keep resources created in the last N days")
	all := flag.Bool("all", false, "Clean everything unused: all unused images, dangling volumes and build cache (not just dangling images)")
	volumes := flag.Bool("volumes", false, "Also remove dangling volumes")
	buildCache := flag.Bool("build-cache", false, "Also prune all builder cache")
	var ex excludes
	flag.Var(&ex, "exclude", "Skip containers/images/volumes whose name contains this substring (repeatable)")
	ver := flag.Bool("version", false, "Print version information")

	flag.Parse()

	if *ver {
		fmt.Printf("DockNuke v%s - Docker environment cleaner\n", version)
		os.Exit(0)
	}

	opts := types.NukeOptions{
		DryRun:       *dryRun,
		KeepDays:     *keepDays,
		All:          *all,
		Excludes:     ex,
		IncludeVols:  *volumes,
		IncludeBuild: *buildCache,
	}

	if opts.DryRun {
		fmt.Println("[dry-run] scanning docker daemon for reclaimable resources...")
	} else {
		fmt.Println("cleaning docker daemon resources...")
	}

	start := time.Now()
	report, err := cleaner.NewCliEngine().Run(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "docknuke: %v\n", err)
		os.Exit(1)
	}
	report.Duration = time.Since(start)

	verb := "removed"
	if opts.DryRun {
		verb = "would remove"
	}
	if d := report.Detail; d != nil && (len(d.Containers)+len(d.Images)+len(d.Volumes) > 0) {
		fmt.Println()
		printSection("containers", d.Containers, verb)
		printSection("images", d.Images, verb)
		printSection("volumes", d.Volumes, verb)
	}
	fmt.Println()
	fmt.Printf("containers %s: %d\n", verb, report.ContainersRemoved)
	fmt.Printf("images %s:     %d (~%s)\n", verb, report.ImagesRemoved, types.FormatBytes(report.ImagesSize))
	fmt.Printf("volumes %s:    %d\n", verb, report.VolumesRemoved)
	if report.BuildCachePruned {
		fmt.Printf("build cache:   pruned (%s reclaimed)\n", types.FormatBytes(report.BuildCacheReclaimed))
	}
	if !opts.DryRun {
		fmt.Printf("total reclaimed: %s in %v\n", types.FormatBytes(report.TotalSpaceReclaimed), report.Duration.Round(time.Millisecond))
	}
	if opts.DryRun {
		fmt.Println("nothing was deleted — rerun without --dry-run to apply")
	}
}

func printSection(label string, items []string, verb string) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("  %s (%d):\n", label, len(items))
	for _, it := range items {
		fmt.Printf("    - %s\n", it)
	}
}
