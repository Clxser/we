package service

import (
	"sort"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
)

const defaultSearchLimit = 20

// SearchItems returns registered item and block names containing query.
func SearchItems(query string, limit int) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	seen := map[string]struct{}{}
	for _, it := range world.Items() {
		name, _ := it.EncodeItem()
		addSearchMatch(seen, name, query)
	}
	for _, b := range world.Blocks() {
		name, _ := b.EncodeBlock()
		addSearchMatch(seen, name, query)
	}
	matches := make([]string, 0, len(seen))
	for name := range seen {
		matches = append(matches, name)
	}
	sort.Strings(matches)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}

func addSearchMatch(matches map[string]struct{}, name, query string) {
	short := strings.TrimPrefix(name, "minecraft:")
	if strings.Contains(strings.ToLower(name), query) || strings.Contains(strings.ToLower(short), query) {
		matches[name] = struct{}{}
	}
}
