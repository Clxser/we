package edit

import (
	"fmt"
	"math"
	"testing"

	"github.com/Velvet-MC/s2d/translate"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func TestRotateBlockDoesNotPanicForRegisteredBlocks(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	for _, b := range world.Blocks() {
		t.Run(fmt.Sprintf("%T", b), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("RotateBlock panicked: %v", r)
				}
			}()
			_ = RotateBlock(b, "y", 3)
		})
	}
}

func TestRotateBlockKeepsInertStateBlocksInert(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	got := translate.Lookup("minecraft:smoker[facing=north,lit=false]")
	if !got.Recognized {
		t.Fatal("smoker was not recognized")
	}
	rotated := RotateBlock(got.Block, "y", 1)
	if _, ticking := rotated.(interface {
		Tick(int64, cube.Pos, *world.Tx)
	}); ticking {
		t.Fatalf("rotated schematic block %T must remain inert", rotated)
	}
}

func TestRotateBlockTransformsInertBedrockDirectionProperties(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	tests := []struct {
		name  string
		key   string
		turns int
		prop  string
		want  any
	}{
		{
			name:  "stairs_weirdo_direction",
			key:   "minecraft:sandstone_stairs[facing=north,half=bottom,shape=straight,waterlogged=false]",
			turns: 1,
			prop:  "weirdo_direction",
			want:  int32(0), // north -> east in Bedrock stairs encoding.
		},
		{
			name:  "trapdoor_direction",
			key:   "minecraft:oak_trapdoor[facing=north,half=bottom,open=false,powered=false,waterlogged=false]",
			turns: 1,
			prop:  "direction",
			want:  int32(0), // north -> east in Bedrock trapdoor encoding.
		},
		{
			name:  "portal_axis",
			key:   "minecraft:nether_portal[axis=x]",
			turns: 1,
			prop:  "portal_axis",
			want:  "z",
		},
		{
			name:  "wall_connections",
			key:   "minecraft:cobblestone_wall[east=none,north=tall,south=none,up=true,waterlogged=false,west=low]",
			turns: 1,
			prop:  "wall_connection_type_east",
			want:  "tall", // north connection rotates to east.
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := translate.Lookup(tt.key)
			if !res.Recognized {
				t.Fatalf("%s not recognized", tt.key)
			}
			rotated := RotateBlock(res.Block, "y", tt.turns)
			_, props := rotated.EncodeBlock()
			if got := props[tt.prop]; got != tt.want {
				t.Fatalf("%s property %s = %#v (%T), want %#v (%T)", tt.key, tt.prop, got, got, tt.want, tt.want)
			}
			if _, stateHash := rotated.Hash(); stateHash != math.MaxUint64 {
				t.Fatalf("rotated schematic block should stay inert StateBlock, state hash = %d", stateHash)
			}
		})
	}
}

func TestRotateBlockTransformsInertSignStates(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	tests := []struct {
		name      string
		key       string
		turns     int
		wantProps map[string]any
	}{
		{
			name:  "wall_sign",
			key:   "minecraft:oak_wall_sign[facing=north,waterlogged=false]",
			turns: 1,
			wantProps: map[string]any{
				"facing_direction": int32(5), // north wall sign rotates to east.
			},
		},
		{
			name:  "standing_sign",
			key:   "minecraft:oak_sign[rotation=0,waterlogged=false]",
			turns: 1,
			wantProps: map[string]any{
				"ground_sign_direction": int32(4),
			},
		},
		{
			name:  "wall_hanging_sign",
			key:   "minecraft:oak_wall_hanging_sign[facing=north,waterlogged=false]",
			turns: 1,
			wantProps: map[string]any{
				"facing_direction": int32(5), // north wall hanging sign rotates to east.
			},
		},
		{
			name:  "standing_hanging_sign",
			key:   "minecraft:oak_hanging_sign[attached=false,rotation=0,waterlogged=false]",
			turns: 1,
			wantProps: map[string]any{
				"ground_sign_direction": int32(4),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := translate.Lookup(tt.key)
			if !res.Recognized {
				t.Fatalf("%s not recognized", tt.key)
			}
			rotated := RotateBlock(res.Block, "y", tt.turns)
			_, props := rotated.EncodeBlock()
			for key, want := range tt.wantProps {
				if got := props[key]; got != want {
					t.Fatalf("%s property %s = %#v (%T), want %#v (%T)", tt.key, key, got, got, want, want)
				}
			}
			if _, stateHash := rotated.Hash(); stateHash != math.MaxUint64 {
				t.Fatalf("rotated schematic sign should stay inert StateBlock, state hash = %d", stateHash)
			}
		})
	}
}

func TestRotateBlockPreservesInertBlockNBT(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	res := translate.Lookup("minecraft:red_banner[rotation=0]")
	if !res.Recognized {
		t.Fatal("red banner not recognized")
	}
	rotated := RotateBlock(res.Block, "y", 1)
	nbter, ok := rotated.(world.NBTer)
	if !ok {
		t.Fatalf("rotated banner %T does not implement NBTer", rotated)
	}
	nbt := nbter.EncodeNBT()
	if got := nbt["id"]; got != "Banner" {
		t.Fatalf("id = %#v, want Banner", got)
	}
	if got := nbt["Base"]; got != int32(14) {
		t.Fatalf("Base = %#v (%T), want red banner base 14", got, got)
	}
}

func TestRotateBlockTransformsInertLeverAndButtonStates(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	tests := []struct {
		name      string
		key       string
		turns     int
		wantProps map[string]any
	}{
		{
			name:  "wall_button",
			key:   "minecraft:oak_button[face=wall,facing=north,powered=false]",
			turns: 1,
			wantProps: map[string]any{
				"facing_direction": int32(5),
			},
		},
		{
			name:  "wall_lever",
			key:   "minecraft:lever[face=wall,facing=north,powered=false]",
			turns: 1,
			wantProps: map[string]any{
				"lever_direction": "east",
			},
		},
		{
			name:  "floor_lever_axis",
			key:   "minecraft:lever[face=floor,facing=north,powered=false]",
			turns: 1,
			wantProps: map[string]any{
				"lever_direction": "up_east_west",
			},
		},
		{
			name:  "ceiling_lever_axis",
			key:   "minecraft:lever[face=ceiling,facing=east,powered=false]",
			turns: 1,
			wantProps: map[string]any{
				"lever_direction": "down_north_south",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := translate.Lookup(tt.key)
			if !res.Recognized {
				t.Fatalf("%s not recognized", tt.key)
			}
			rotated := RotateBlock(res.Block, "y", tt.turns)
			_, props := rotated.EncodeBlock()
			for key, want := range tt.wantProps {
				if got := props[key]; got != want {
					t.Fatalf("%s property %s = %#v (%T), want %#v (%T)", tt.key, key, got, got, want, want)
				}
			}
			if _, stateHash := rotated.Hash(); stateHash != math.MaxUint64 {
				t.Fatalf("rotated schematic lever/button should stay inert StateBlock, state hash = %d", stateHash)
			}
		})
	}
}

func TestRotateBlockTransformsInertRailStates(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	tests := []struct {
		name string
		key  string
		want int32
	}{
		{
			name: "normal_rail_corner",
			key:  "minecraft:rail[shape=north_east,waterlogged=false]",
			want: int32(6), // north_east rotates to south_east.
		},
		{
			name: "powered_rail_slope",
			key:  "minecraft:powered_rail[powered=true,shape=ascending_east,waterlogged=false]",
			want: int32(5), // ascending east rotates to ascending south.
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := translate.Lookup(tt.key)
			if !res.Recognized {
				t.Fatalf("%s not recognized", tt.key)
			}
			rotated := RotateBlock(res.Block, "y", 1)
			_, props := rotated.EncodeBlock()
			if got := props["rail_direction"]; got != tt.want {
				t.Fatalf("%s rail_direction = %#v (%T), want %#v", tt.key, got, got, tt.want)
			}
			if _, stateHash := rotated.Hash(); stateHash != math.MaxUint64 {
				t.Fatalf("rotated schematic rail should stay inert StateBlock, state hash = %d", stateHash)
			}
		})
	}
}
