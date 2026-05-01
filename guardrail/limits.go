// Package guardrail contains opt-in safety limits for expensive WorldEdit operations.
package guardrail

import "fmt"

// Limits describes optional caps for expensive operations. Zero values are
// unlimited and preserve existing behavior.
type Limits struct {
	MaxSelectionVolume int
	MaxShapeVolume     int
	MaxBrushVolume     int
	MaxStackCopies     int
}

// CheckSelectionVolume rejects a selection volume above MaxSelectionVolume.
func (l Limits) CheckSelectionVolume(volume int64) error {
	return checkVolume("selection", volume, l.MaxSelectionVolume)
}

// CheckShapeVolume rejects a shape bounding volume above MaxShapeVolume.
func (l Limits) CheckShapeVolume(volume int64) error {
	return checkVolume("shape", volume, l.MaxShapeVolume)
}

// CheckBrushVolume rejects a brush bounding volume above MaxBrushVolume.
func (l Limits) CheckBrushVolume(volume int64) error {
	return checkVolume("brush", volume, l.MaxBrushVolume)
}

// CheckStackCopies rejects a stack copy amount above MaxStackCopies.
func (l Limits) CheckStackCopies(copies int) error {
	if l.MaxStackCopies <= 0 || copies <= l.MaxStackCopies {
		return nil
	}
	return fmt.Errorf("stack copies %d exceeds limit %d", copies, l.MaxStackCopies)
}

func checkVolume(kind string, volume int64, max int) error {
	if max <= 0 || volume <= int64(max) {
		return nil
	}
	return fmt.Errorf("%s volume %d exceeds limit %d", kind, volume, max)
}
