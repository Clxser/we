// Package brush contains the legacy item-bound brush registry.
//
// New code should prefer editbrush, which stores brush configuration directly
// on item metadata and records changes through the shared history package. This
// package remains for older palette/action flows.
package brush
