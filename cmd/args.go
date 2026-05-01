package cmd

import (
	"strings"

	dcf "github.com/df-mc/dragonfly/server/cmd"
)

func optionalB(o dcf.Optional[string]) bool {
	v, ok := o.Load()
	return ok && strings.EqualFold(v, "b")
}
