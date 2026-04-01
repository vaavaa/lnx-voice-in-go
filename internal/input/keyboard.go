package input

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	hook "github.com/robotn/gohook"
)

func Type(text string) {
	if text == "" {
		return
	}
	_ = exec.Command("ydotool", "type", text).Run()
}

// ListenHotkey blocks the current goroutine: global keyboard hook via gohook.
// keyName is normalized to lower case and must match hook.Keycode keys, e.g. "f12", "pause".
func ListenHotkey(keyName string, onPress func()) {
	keyName = strings.ToLower(strings.TrimSpace(keyName))
	code, ok := hook.Keycode[keyName]
	if !ok {
		fmt.Fprintf(os.Stderr, "voice: unknown hotkey %q, using f12\n", keyName)
		code = hook.Keycode["f12"]
	}

	evChan := hook.Start()
	defer hook.End()

	for ev := range evChan {
		if ev.Kind == hook.KeyDown && ev.Keycode == code {
			onPress()
		}
	}
}
