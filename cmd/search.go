package cmd

import (
	"strings"

	dcf "github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/service"
)

// SearchItemCommand implements //searchitem <query> — lists matching item/block IDs.
type SearchItemCommand struct {
	Query dcf.Varargs `cmd:"query"`
}

func (SearchItemCommand) Allow(dcf.Source) bool { return true }

func (c SearchItemCommand) Run(_ dcf.Source, o *dcf.Output, _ *world.Tx) {
	query := strings.TrimSpace(string(c.Query))
	if query == "" {
		o.Error("usage: //searchitem <query>")
		return
	}
	matches := service.SearchItems(query, 20)
	if len(matches) == 0 {
		o.Printf("No items found for %q.", query)
		return
	}
	o.Print("Matches: " + strings.Join(matches, ", "))
}
