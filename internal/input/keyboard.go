package input

import (
	"os/exec"

	hook "github.com/robotn/gohook"
)

func Type(text string) {
	if text == "" {
		return
	}
	_ = exec.Command("ydotool", "type", text).Run()
}

func ListenF12(onPress func()) {
	evChan := hook.Start()
	defer hook.End()

	for ev := range evChan {
		if ev.Kind == hook.KeyDown && ev.Keycode == hook.Keycode["f12"] {
			onPress()
		}
	}
}