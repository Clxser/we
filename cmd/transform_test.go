package cmd

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestStackDirectionParsesVerticalDirection(t *testing.T) {
	dir, args, ok := stackDirection(cube.Pos{1, 0, 0}, []string{"3", "up", "-a"})
	if !ok {
		t.Fatal("stackDirection rejected up direction")
	}
	if dir != (cube.Pos{0, 1, 0}) {
		t.Fatalf("dir = %v, want up", dir)
	}
	if len(args) != 2 || args[0] != "3" || args[1] != "-a" {
		t.Fatalf("args = %v, want amount and flag only", args)
	}
}

func TestStackDirectionRejectsUnknownDirection(t *testing.T) {
	if _, _, ok := stackDirection(cube.Pos{1, 0, 0}, []string{"3", "sideways"}); ok {
		t.Fatal("stackDirection accepted unknown direction")
	}
}
