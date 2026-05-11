package edit

import (
	"math"
	"reflect"
	"strings"

	"github.com/Velvet-MC/s2d/translate"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/parse"
)

var (
	directionType = reflect.TypeOf(cube.North)
	faceType      = reflect.TypeOf(cube.FaceDown)
	axisType      = reflect.TypeOf(cube.X)
)

// RotateBlock rotates b's simple directional state around axis by turns
// quarter-turns. It covers blocks whose orientation is represented by a single
// exported cube.Direction, cube.Face, or cube.Axis field, plus common Bedrock
// direction properties. Multi-block or multi-property shapes such as rails,
// beds, doors, vines, chest pairs, and redstone wire require specialised
// transforms and are intentionally left unchanged here.
func RotateBlock(b world.Block, axis string, turns int) world.Block {
	return transformBlock(b, blockTransform{axis: strings.ToLower(axis), turns: ((turns % 4) + 4) % 4})
}

// FlipBlock mirrors b's simple directional state across axis. See RotateBlock
// for the supported-state scope.
func FlipBlock(b world.Block, axis string) world.Block {
	return transformBlock(b, blockTransform{axis: strings.ToLower(axis), flip: true})
}

type blockTransform struct {
	axis  string
	turns int
	flip  bool
}

type blockTransformCache map[parse.BlockKey]world.Block

func (c blockTransformCache) transform(b world.Block, t blockTransform) world.Block {
	if c == nil {
		return transformBlock(b, t)
	}
	key := parse.BlockKeyOf(b)
	if out, ok := c[key]; ok {
		return out
	}
	out := transformBlock(b, t)
	c[key] = out
	return out
}

func transformBlock(b world.Block, t blockTransform) world.Block {
	if b == nil {
		return nil
	}
	if t.turns == 0 && !t.flip {
		return b
	}
	if out, ok := transformBlockFields(b, t); ok {
		return out
	}
	name, props := b.EncodeBlock()
	if out, ok := transformBlockProperties(b, name, props, t); ok {
		return out
	}
	return b
}

func transformBlockFields(b world.Block, t blockTransform) (world.Block, bool) {
	v := reflect.ValueOf(b)
	ptr := false
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return b, false
		}
		v = v.Elem()
		ptr = true
	}
	if v.Kind() != reflect.Struct {
		return b, false
	}
	cp := reflect.New(v.Type()).Elem()
	cp.Set(v)
	changed := false
	for i := 0; i < cp.NumField(); i++ {
		field := cp.Field(i)
		if !field.CanSet() {
			continue
		}
		switch field.Type() {
		case directionType:
			old := cube.Direction(field.Int())
			next := transformDirection(old, t)
			if next != old {
				field.SetInt(int64(next))
				changed = true
			}
		case faceType:
			old := cube.Face(field.Int())
			next := transformFace(old, t)
			if next != old {
				field.SetInt(int64(next))
				changed = true
			}
		case axisType:
			old := cube.Axis(field.Int())
			next := transformAxis(old, t)
			if next != old {
				field.SetInt(int64(next))
				changed = true
			}
		}
	}
	if !changed {
		return b, false
	}
	if ptr {
		out, ok := cp.Addr().Interface().(world.Block)
		return out, ok
	}
	out, ok := cp.Interface().(world.Block)
	return out, ok
}

func transformBlockProperties(original world.Block, name string, props map[string]any, t blockTransform) (world.Block, bool) {
	if len(props) == 0 {
		return nil, false
	}
	if _, stateHash := original.Hash(); stateHash == math.MaxUint64 {
		state, changed := translate.TransformBedrockState(
			translate.BedrockState{Name: name, Properties: props},
			t.axis,
			t.turns,
			t.flip,
		)
		if changed {
			return translate.NewStateBlock(state), true
		}
		return nil, false
	}
	cp := make(map[string]any, len(props))
	for k, v := range props {
		cp[k] = v
	}
	changed := false
	for _, key := range []string{"minecraft:cardinal_direction", "cardinal_direction"} {
		if v, ok := cp[key]; ok {
			if s, ok := v.(string); ok {
				if d, ok := directionFromString(s); ok {
					next := transformDirection(d, t)
					if next != d {
						cp[key] = next.String()
						changed = true
					}
				}
			}
		}
	}
	if v, ok := cp["facing_direction"]; ok {
		if d, ok := intWallFacingDirection(v); ok && isWallMountedState(name) {
			next := transformDirection(d, t)
			if next != d {
				cp["facing_direction"] = wallFacingDirectionInt(next)
				changed = true
			}
		} else if face, ok := intFace(v); ok {
			next := transformFace(face, t)
			if next != face {
				cp["facing_direction"] = int32(next)
				changed = true
			}
		}
	}
	if v, ok := cp["weirdo_direction"]; ok {
		if d, ok := intStairsDirection(v); ok {
			next := transformDirection(d, t)
			if next != d {
				cp["weirdo_direction"] = stairsDirectionInt(next)
				changed = true
			}
		}
	}
	if v, ok := cp["direction"]; ok && isTrapdoorState(name) {
		if d, ok := intTrapdoorDirection(v); ok {
			next := transformDirection(d, t)
			if next != d {
				cp["direction"] = trapdoorDirectionInt(next)
				changed = true
			}
		}
	}
	if v, ok := cp["ground_sign_direction"]; ok && t.axis == "y" {
		if o, ok := intValue(v); ok {
			next := transformOrientation(o, t)
			if next != o {
				cp["ground_sign_direction"] = int32(next)
				changed = true
			}
		}
	}
	if v, ok := cp["pillar_axis"]; ok {
		if s, ok := v.(string); ok {
			if a, ok := axisFromString(s); ok {
				next := transformAxis(a, t)
				if next != a {
					cp["pillar_axis"] = next.String()
					changed = true
				}
			}
		}
	}
	if v, ok := cp["portal_axis"]; ok {
		if s, ok := v.(string); ok {
			if a, ok := axisFromString(s); ok {
				next := transformAxis(a, t)
				if next != a {
					cp["portal_axis"] = next.String()
					changed = true
				}
			}
		}
	}
	if v, ok := cp["lever_direction"]; ok {
		if s, ok := v.(string); ok {
			next := transformLeverDirection(s, t)
			if next != s {
				cp["lever_direction"] = next
				changed = true
			}
		}
	}
	if v, ok := cp["torch_facing_direction"]; ok {
		if s, ok := v.(string); ok {
			if face, ok := torchFaceFromString(s); ok {
				next := transformFace(face, t)
				if next != face {
					cp["torch_facing_direction"] = torchFaceString(next)
					changed = true
				}
			}
		}
	}
	if transformWallConnections(cp, t) {
		changed = true
	}
	if !changed {
		return nil, false
	}
	out, ok := world.BlockByName(name, cp)
	return out, ok
}

func transformDirection(d cube.Direction, t blockTransform) cube.Direction {
	if !validDirection(d) {
		return d
	}
	if t.flip {
		switch t.axis {
		case "x":
			if d == cube.East || d == cube.West {
				return d.Opposite()
			}
		case "z":
			if d == cube.North || d == cube.South {
				return d.Opposite()
			}
		}
		return d
	}
	if t.axis != "y" {
		return d
	}
	for i := 0; i < t.turns; i++ {
		d = d.RotateRight()
	}
	return d
}

func validDirection(d cube.Direction) bool {
	switch d {
	case cube.North, cube.South, cube.West, cube.East:
		return true
	default:
		return false
	}
}

func transformFace(f cube.Face, t blockTransform) cube.Face {
	vec, ok := faceVector(f)
	if !ok {
		return f
	}
	return vectorFace(transformVector(vec, t))
}

func transformAxis(a cube.Axis, t blockTransform) cube.Axis {
	vec := axisVector(a)
	next := transformVector(vec, t)
	if next[0] != 0 {
		return cube.X
	}
	if next[1] != 0 {
		return cube.Y
	}
	return cube.Z
}

func transformOrientation(o int, t blockTransform) int {
	o = ((o % 16) + 16) % 16
	if t.flip {
		switch t.axis {
		case "x":
			return (16 - o) % 16
		case "z":
			return (8 - o + 16) % 16
		default:
			return o
		}
	}
	return (o + t.turns*4) % 16
}

func transformVector(v [3]int, t blockTransform) [3]int {
	if t.flip {
		switch t.axis {
		case "x":
			v[0] = -v[0]
		case "y":
			v[1] = -v[1]
		case "z":
			v[2] = -v[2]
		}
		return v
	}
	for i := 0; i < t.turns; i++ {
		switch t.axis {
		case "x":
			v = [3]int{v[0], -v[2], v[1]}
		case "z":
			v = [3]int{-v[1], v[0], v[2]}
		default:
			v = [3]int{-v[2], v[1], v[0]}
		}
	}
	return v
}

func faceVector(f cube.Face) ([3]int, bool) {
	switch f {
	case cube.FaceDown:
		return [3]int{0, -1, 0}, true
	case cube.FaceUp:
		return [3]int{0, 1, 0}, true
	case cube.FaceNorth:
		return [3]int{0, 0, -1}, true
	case cube.FaceSouth:
		return [3]int{0, 0, 1}, true
	case cube.FaceWest:
		return [3]int{-1, 0, 0}, true
	case cube.FaceEast:
		return [3]int{1, 0, 0}, true
	default:
		return [3]int{}, false
	}
}

func vectorFace(v [3]int) cube.Face {
	switch v {
	case [3]int{0, -1, 0}:
		return cube.FaceDown
	case [3]int{0, 1, 0}:
		return cube.FaceUp
	case [3]int{0, 0, -1}:
		return cube.FaceNorth
	case [3]int{0, 0, 1}:
		return cube.FaceSouth
	case [3]int{-1, 0, 0}:
		return cube.FaceWest
	case [3]int{1, 0, 0}:
		return cube.FaceEast
	default:
		return cube.FaceUp
	}
}

func axisVector(a cube.Axis) [3]int {
	switch a {
	case cube.X:
		return [3]int{1, 0, 0}
	case cube.Y:
		return [3]int{0, 1, 0}
	default:
		return [3]int{0, 0, 1}
	}
}

func directionFromString(s string) (cube.Direction, bool) {
	switch strings.ToLower(s) {
	case "north":
		return cube.North, true
	case "south":
		return cube.South, true
	case "west":
		return cube.West, true
	case "east":
		return cube.East, true
	default:
		return cube.North, false
	}
}

func axisFromString(s string) (cube.Axis, bool) {
	switch strings.ToLower(s) {
	case "x":
		return cube.X, true
	case "y":
		return cube.Y, true
	case "z":
		return cube.Z, true
	default:
		return cube.Y, false
	}
}

func intFace(v any) (cube.Face, bool) {
	n, ok := intValue(v)
	if !ok || n < int(cube.FaceDown) || n > int(cube.FaceEast) {
		return cube.FaceDown, false
	}
	return cube.Face(n), true
}

func intValue(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case uint8:
		return int(n), true
	case float64:
		return int(n), n == float64(int(n))
	default:
		return 0, false
	}
}

func isTrapdoorState(name string) bool {
	name = strings.TrimPrefix(name, "minecraft:")
	return name == "trapdoor" || strings.HasSuffix(name, "_trapdoor")
}

func isWallMountedState(name string) bool {
	name = strings.TrimPrefix(name, "minecraft:")
	return name == "wall_sign" || name == "wall_banner" ||
		strings.HasSuffix(name, "_wall_sign") || strings.HasSuffix(name, "_wall_banner") ||
		strings.HasSuffix(name, "_wall_skull") || strings.HasSuffix(name, "_wall_head") ||
		strings.HasSuffix(name, "_hanging_sign")
}

func intWallFacingDirection(v any) (cube.Direction, bool) {
	n, ok := intValue(v)
	if !ok {
		return cube.North, false
	}
	switch n {
	case 2:
		return cube.North, true
	case 3:
		return cube.South, true
	case 4:
		return cube.West, true
	case 5:
		return cube.East, true
	default:
		return cube.North, false
	}
}

func wallFacingDirectionInt(d cube.Direction) int32 {
	switch d {
	case cube.North:
		return 2
	case cube.South:
		return 3
	case cube.West:
		return 4
	case cube.East:
		return 5
	default:
		return 2
	}
}

func intStairsDirection(v any) (cube.Direction, bool) {
	n, ok := intValue(v)
	if !ok {
		return cube.North, false
	}
	switch n {
	case 0:
		return cube.East, true
	case 1:
		return cube.West, true
	case 2:
		return cube.South, true
	case 3:
		return cube.North, true
	default:
		return cube.North, false
	}
}

func stairsDirectionInt(d cube.Direction) int32 {
	switch d {
	case cube.East:
		return 0
	case cube.West:
		return 1
	case cube.South:
		return 2
	case cube.North:
		return 3
	default:
		return 3
	}
}

func intTrapdoorDirection(v any) (cube.Direction, bool) {
	return intStairsDirection(v)
}

func trapdoorDirectionInt(d cube.Direction) int32 {
	return stairsDirectionInt(d)
}

func transformWallConnections(props map[string]any, t blockTransform) bool {
	keys := map[string]cube.Direction{
		"wall_connection_type_north": cube.North,
		"wall_connection_type_east":  cube.East,
		"wall_connection_type_south": cube.South,
		"wall_connection_type_west":  cube.West,
	}
	values := make(map[cube.Direction]any, len(keys))
	seen := false
	for key, dir := range keys {
		if v, ok := props[key]; ok {
			values[dir] = v
			seen = true
		}
	}
	if !seen {
		return false
	}
	for key := range keys {
		props[key] = "none"
	}
	changed := false
	for from, value := range values {
		to := transformDirection(from, t)
		key := wallConnectionKey(to)
		if key == "" {
			continue
		}
		if props[key] != value {
			changed = true
		}
		props[key] = value
	}
	return changed
}

func wallConnectionKey(d cube.Direction) string {
	switch d {
	case cube.North:
		return "wall_connection_type_north"
	case cube.East:
		return "wall_connection_type_east"
	case cube.South:
		return "wall_connection_type_south"
	case cube.West:
		return "wall_connection_type_west"
	default:
		return ""
	}
}

func transformLeverDirection(s string, t blockTransform) string {
	if d, ok := directionFromString(s); ok {
		return transformDirection(d, t).String()
	}
	switch s {
	case "up_north_south", "up_east_west", "down_north_south", "down_east_west":
		prefix := "up_"
		axisPart := strings.TrimPrefix(s, "up_")
		if strings.HasPrefix(s, "down_") {
			prefix = "down_"
			axisPart = strings.TrimPrefix(s, "down_")
		}
		axis := cube.Z
		if axisPart == "east_west" {
			axis = cube.X
		}
		next := transformAxis(axis, t)
		if next == cube.X {
			return prefix + "east_west"
		}
		return prefix + "north_south"
	default:
		return s
	}
}

func torchFaceFromString(s string) (cube.Face, bool) {
	switch strings.ToLower(s) {
	case "top":
		return cube.FaceDown, true
	case "north":
		return cube.FaceNorth, true
	case "south":
		return cube.FaceSouth, true
	case "west":
		return cube.FaceWest, true
	case "east":
		return cube.FaceEast, true
	default:
		return cube.FaceDown, false
	}
}

func torchFaceString(f cube.Face) string {
	if f == cube.FaceDown {
		return "top"
	}
	return f.String()
}
