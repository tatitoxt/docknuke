package docker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestParseHumanSize(t *testing.T) {
	cases := map[string]int64{
		"0B":       0,
		"512B":     512,
		"1.389GB":  1389000000,
		"875.2MB":  875200000,
		"36.86kB":  36860,
		"4.554GB":  4554000000,
		"1.00 GiB": 1 << 30,
	}
	for in, want := range cases {
		if got := parseHumanSize(in); got != want {
			t.Errorf("parseHumanSize(%q) = %d, want %d", in, got, want)
		}
	}
}

func fakeRunner(responses map[string]string) Runner {
	return func(_ context.Context, args ...string) (string, error) {
		key := args[0] + " " + args[1]
		if out, ok := responses[key]; ok {
			return out, nil
		}
		if args[0] == "inspect" {
			return responses["inspect"], nil
		}
		return "", errors.New("unexpected call: " + key)
	}
}

func TestImagesFiltersDanglingUnlessAll(t *testing.T) {
	client := New(fakeRunner(map[string]string{
		"images -a": "aaa000000000|<none>:<none>|100MB|2026-11-01 10:00:00 +0000 UTC\n" +
			"bbb000000000|busybox:1.36|4MB|2026-11-01 10:00:00 +0000 UTC\n" +
			"ccc000000000|old:unused|50MB|2026-11-01 10:00:00 +0000 UTC\n",
		"ps -aq":  "ctr1\n",
		"inspect": "sha256:bbb000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n", // container uses bbb
	}))

	dangling, err := client.Images(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(dangling) != 1 || dangling[0].ID != "aaa000000000" {
		t.Fatalf("expected only dangling aaa, got %+v", dangling)
	}

	unused, err := client.Images(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(unused) != 2 {
		t.Fatalf("expected dangling + unused (in-use excluded), got %+v", unused)
	}
}

func TestReclaimableParsesDf(t *testing.T) {
	client := New(fakeRunner(map[string]string{
		"system df": `{"Type":"Images","Reclaimable":"1.389GB (12%)"}` + "\n" +
			`{"Type":"Build Cache","Reclaimable":"4.554GB (49%)"}` + "\n",
	}))
	got, err := client.Reclaimable(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got["images"] != 1389000000 || got["build cache"] != 4554000000 {
		t.Fatalf("bad parse: %+v", got)
	}
}

func TestParseDockerTime(t *testing.T) {
	if parseDockerTime("2026-11-01 10:00:00 +0000 UTC").IsZero() {
		t.Error("docker images timestamp not parsed")
	}
	if !time.Now().After(parseDockerTime("2026-11-01 10:00:00 +0000 UTC")) {
		t.Error("bad time direction")
	}
}
