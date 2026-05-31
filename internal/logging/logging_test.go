package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewHonorsDebugFlag(t *testing.T) {
	var infoOut bytes.Buffer
	New(&infoOut, false).Debug("hidden")
	if strings.Contains(infoOut.String(), "hidden") {
		t.Fatalf("debug message was emitted by info logger: %q", infoOut.String())
	}

	var debugOut bytes.Buffer
	New(&debugOut, true).Debug("visible")
	if !strings.Contains(debugOut.String(), "visible") {
		t.Fatalf("debug message was not emitted by debug logger: %q", debugOut.String())
	}
}
