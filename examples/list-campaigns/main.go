// Command list-campaigns prints every campaign in the workspace.
//
//	AYINBOX_API_KEY=aybox_... go run ./examples/list-campaigns
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ayinbox/ayinbox-go"
)

func main() {
	client, err := ayinbox.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pager := client.Campaigns.ListAutoPaging(&ayinbox.CampaignListParams{Limit: 100})
	for pager.Next(ctx) {
		c := pager.Current()
		fmt.Printf("%s  %-12s  %s\n", c.ID, c.Status, c.Name)
	}
	if err := pager.Err(); err != nil {
		log.Fatal(err)
	}
}
