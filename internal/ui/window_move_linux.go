//go:build linux

package ui

import (
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
)

var (
	xdoOnce   sync.Once
	xdoAvail  bool
	winIDMu   sync.Mutex
	winIDHint string
)

func xdotoolOnPATH() bool {
	xdoOnce.Do(func() {
		_, err := exec.LookPath("xdotool")
		xdoAvail = err == nil
	})
	return xdoAvail
}

func resolveXDotoolWindowID() string {
	winIDMu.Lock()
	defer winIDMu.Unlock()
	if winIDHint != "" {
		return winIDHint
	}
	out, err := exec.Command("xdotool", "search", "--pid", strconv.Itoa(os.Getpid())).Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		id := strings.TrimSpace(lines[i])
		if id != "" {
			winIDHint = id
			break
		}
	}
	return winIDHint
}

// moveFyneWindowBy shifts the native window by (dx, dy) in Fyne coordinates (scaled to pixels).
func moveFyneWindowBy(w fyne.Window, dx, dy float32) {
	if w == nil || !xdotoolOnPATH() {
		return
	}
	wid := resolveXDotoolWindowID()
	if wid == "" {
		return
	}
	scale := float32(1)
	if c := w.Canvas(); c != nil {
		scale = c.Scale()
	}
	pdx := int(math.Round(float64(dx * scale)))
	pdy := int(math.Round(float64(dy * scale)))
	if pdx == 0 && pdy == 0 {
		return
	}
	_ = exec.Command("xdotool", "windowmove", "--relative", wid, strconv.Itoa(pdx), strconv.Itoa(pdy)).Run()
}
