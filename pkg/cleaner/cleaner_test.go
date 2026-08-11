package cleaner

import (
	"testing"
	"time"

	"github.com/tatitoxt/docknuke/pkg/types"
)

func TestFilterRetention(t *testing.T) {
	now := time.Now()
	created3DaysAgo := now.AddDate(0, 0, -3)
	created10DaysAgo := now.AddDate(0, 0, -10)

	opts := types.NukeOptions{
		KeepDays: 7,
	}

	if ShouldKeep(created3DaysAgo, opts.KeepDays) == false {
		t.Errorf("Expected resource created 3 days ago to be kept when KeepDays=7")
	}

	if ShouldKeep(created10DaysAgo, opts.KeepDays) == true {
		t.Errorf("Expected resource created 10 days ago to NOT be kept when KeepDays=7")
	}
}
