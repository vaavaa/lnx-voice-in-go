//go:build !linux

package ui

import "fyne.io/fyne/v2"

func moveFyneWindowBy(_ fyne.Window, _, _ float32) {}
