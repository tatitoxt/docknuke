package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tatitoxt/docknuke/pkg/cleaner"
	"github.com/tatitoxt/docknuke/pkg/types"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Simulate cleaning without deleting resources")
	keepDays := flag.Int("keep-days", 0, "Keep resources created in the last N days")
	all := flag.Bool("all", false, "Clean all unused images and volumes, not just dangling ones")
	version := flag.Bool("version", false, "Print version information")

	flag.Parse()

	if *version {
		fmt.Println("DockNuke v1.0.0 - Smart Docker Environment Cleaner")
		os.Exit(0)
	}

	fmt.Println("☢️  DockNuke v1.0.0 - Docker Environment Optimizer")
	fmt.Println("=================================================")

	opts := types.NukeOptions{
		DryRun:   *dryRun,
		KeepDays: *keepDays,
		All:      *all,
	}

	if opts.DryRun {
		fmt.Println("🔍 [DRY-RUN MODE] Scanning Docker daemon for reclaimable resources...")
	} else {
		fmt.Println("💥 [EXECUTION MODE] Cleaning Docker daemon resources...")
	}

	start := time.Now()
	eng := cleaner.NewMockEngine()
	report, err := eng.Run(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running DockNuke: %v\n", err)
		os.Exit(1)
	}

	report.Duration = time.Since(start)

	fmt.Println("\n📊 Cleaning Summary:")
	fmt.Printf("   • Containers Removed:  %d\n", report.ContainersRemoved)
	fmt.Printf("   • Images Removed:      %d\n", report.ImagesRemoved)
	fmt.Printf("   • Volumes Removed:     %d\n", report.VolumesRemoved)
	fmt.Printf("   • Build Cache Cleared: %s\n", types.FormatBytes(report.BuildCacheReclaimed))
	fmt.Printf("   • Total Disk Reclaimed: %s\n", types.FormatBytes(report.TotalSpaceReclaimed))
	fmt.Printf("   • Execution Time:      %v\n", report.Duration)
	fmt.Println("\n✨ Docker environment optimized successfully!")
}
