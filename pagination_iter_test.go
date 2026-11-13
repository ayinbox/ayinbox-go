//go:build go1.23

package ayinbox

import (
	"context"
	"testing"
)

func TestPagerAll(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	n := 0
	for camp, err := range c.Campaigns.ListAutoPaging(nil).All(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		if camp.ID == "" {
			t.Fatal("empty row")
		}
		n++
	}
	if n != 3 {
		t.Fatalf("rows %d", n)
	}
	for range c.Campaigns.ListAutoPaging(nil).All(context.Background()) {
		break
	}
}
