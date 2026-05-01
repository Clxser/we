package brush

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
)

func TestFindIgnoresInvalidBrushItemData(t *testing.T) {
	stack := item.NewStack(item.Axe{Tier: item.ToolTierWood}, 1)
	for _, value := range []any{"not-a-uuid", 42} {
		if _, ok := find(stack.WithValue("brush", value)); ok {
			t.Fatalf("find returned brush for invalid value %v", value)
		}
	}
}
