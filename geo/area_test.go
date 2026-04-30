package geo

import (
	"reflect"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestNewAreaNormalisesCorners(t *testing.T) {
	a := NewArea(3, -1, 7, -2, 4, 1)

	if want := (cube.Pos{-2, -1, 1}); a.Min != want {
		t.Fatalf("Min = %v, want %v", a.Min, want)
	}
	if want := (cube.Pos{3, 4, 7}); a.Max != want {
		t.Fatalf("Max = %v, want %v", a.Max, want)
	}
	if got, want := [3]int{a.Dx(), a.Dy(), a.Dz()}, [3]int{6, 6, 7}; got != want {
		t.Fatalf("dimensions = %v, want %v", got, want)
	}
}

func TestAreaRangeVisitsInclusiveNormalisedArea(t *testing.T) {
	a := NewArea(1, 2, 4, 0, 2, 3)

	var got []cube.Pos
	a.Range(func(x, y, z int) {
		got = append(got, cube.Pos{x, y, z})
	})

	want := []cube.Pos{
		{0, 2, 3},
		{0, 2, 4},
		{1, 2, 3},
		{1, 2, 4},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Range visited %v, want %v", got, want)
	}
}
