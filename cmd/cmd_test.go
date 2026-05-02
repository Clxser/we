package cmd_test

import (
	"testing"

	dcf "github.com/df-mc/dragonfly/server/cmd"
	_ "github.com/df-mc/we/cmd"
)

func TestDoubleSlashCommandAliasesRegistered(t *testing.T) {
	for _, name := range []string{"/wand", "/set", "/copy", "/paste", "/clearclipboard", "/undo", "/brush", "/layer", "/removeabove", "/removebelow", "/removenear", "/naturalize", "/searchitem", "/search", "/l"} {
		if _, ok := dcf.ByAlias(name); !ok {
			t.Fatalf("command alias %q is not registered", name)
		}
	}
}
