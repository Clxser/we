package cmd

import (
	"fmt"
	"time"

	dcf "github.com/df-mc/dragonfly/server/cmd"
)

type operationTimer struct {
	start time.Time
}

func startOperation() operationTimer {
	return operationTimer{start: time.Now()}
}

func (t operationTimer) Printf(o *dcf.Output, format string, args ...any) {
	o.Print(fmt.Sprintf(format, args...) + " (" + formatDuration(time.Since(t.start)) + ")")
}

func (t operationTimer) Print(o *dcf.Output, msg string) {
	o.Print(msg + " (" + formatDuration(time.Since(t.start)) + ")")
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return "<1ms"
	}
	return d.Round(time.Millisecond).String()
}
