package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"lnx-voice-in-go/assets"
	"lnx-voice-in-go/internal/config"
)

const waveViewSize = 300

// micIconSize ratio of visualization area (~inner disk diameter with margin from the rim).
const micIconSizeRatio = 0.285

// micTap: mic PNG with tap handling without full button chrome (no button hover/pressed styling).
type micTap struct {
	widget.BaseWidget
	img   *canvas.Image
	onTap func()
}

func newMicTap(res fyne.Resource, size fyne.Size, onTap func()) *micTap {
	m := &micTap{
		img:   canvas.NewImageFromResource(res),
		onTap: onTap,
	}
	m.img.FillMode = canvas.ImageFillContain
	m.img.SetMinSize(size)
	m.ExtendBaseWidget(m)
	return m
}

func (m *micTap) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.img)
}

func (m *micTap) Tapped(_ *fyne.PointEvent) {
	if m.onTap != nil {
		m.onTap()
	}
}

var _ fyne.Tappable = (*micTap)(nil)

// windowDragLayer: full-cell transparent surface to move the window (below mic stack); mic stays tappable above.
type windowDragLayer struct {
	widget.BaseWidget
	bg  *canvas.Rectangle
	win fyne.Window
}

func newWindowDragLayer(win fyne.Window) *windowDragLayer {
	d := &windowDragLayer{win: win}
	d.bg = canvas.NewRectangle(color.NRGBA{A: 0})
	d.bg.StrokeWidth = 0
	d.bg.SetMinSize(fyne.NewSize(waveViewSize, waveViewSize))
	d.ExtendBaseWidget(d)
	return d
}

func (d *windowDragLayer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(d.bg)
}

func (d *windowDragLayer) Dragged(e *fyne.DragEvent) {
	if d.win == nil {
		return
	}
	moveFyneWindowBy(d.win, e.Dragged.DX, e.Dragged.DY)
}

func (d *windowDragLayer) DragEnd() {}

var _ fyne.Draggable = (*windowDragLayer)(nil)

type Visualizer struct {
	window        fyne.Window
	wave          *canvas.Raster
	micHot        *canvas.Circle    // semi-transparent red “mic on” halo; brightness/size from level
	micPileAnchor *canvas.Rectangle // sets stack MinSize for icon + halo (Circle has no SetMinSize)
	recDot        *canvas.Circle    // fixed corner dot while recording
	onRecord      func()

	innerFill  color.RGBA
	innerEdge  color.RGBA
	waveHiMid  color.RGBA
	ringBGTint color.RGBA

	mu        sync.Mutex
	curVol    float32
	wavePhase float64
	// Echo layer: smoothed copies that lag the main layer
	echoVol   float32
	echoPhase float64
}

func NewOverlay() *Visualizer {
	myApp := app.NewWithID("io.github.lnx-voice-in-go")

	switch strings.ToLower(strings.TrimSpace(config.AppConfig.UI.Theme)) {
	case "light":
		myApp.Settings().SetTheme(theme.LightTheme())
	default:
		myApp.Settings().SetTheme(theme.DarkTheme())
	}

	var w fyne.Window
	if drv, ok := myApp.Driver().(desktop.Driver); ok {
		w = drv.CreateSplashWindow()
	} else {
		w = myApp.NewWindow("Voice Input")
		w.SetPadded(false)
	}
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(waveViewSize, waveViewSize))
	w.SetTitle("Voice Input")

	v := &Visualizer{window: w}

	ac, ok := uiParseHexRGB(config.AppConfig.UI.MainColor)
	if !ok {
		ac = color.RGBA{R: 0, G: 150, B: 255, A: 255}
	}
	v.innerFill, v.innerEdge, v.waveHiMid, v.ringBGTint = uiAccentPalette(ac)

	v.wave = canvas.NewRaster(func(wi, hi int) image.Image {
		return v.micWaveImage(wi, hi)
	})
	// Otherwise centerLayout keeps Raster at default MinSize 1×1 and nothing draws.
	v.wave.SetMinSize(fyne.NewSize(waveViewSize, waveViewSize))
	v.wave.Resize(fyne.NewSize(waveViewSize, waveViewSize))

	micRes := fyne.NewStaticResource("mic_icon.png", assets.MicIconPNG)
	iconSide := float32(waveViewSize) * micIconSizeRatio
	iconSz := fyne.NewSize(iconSide, iconSide)
	micTapW := newMicTap(micRes, iconSz, func() {
		if v.onRecord != nil {
			v.onRecord()
		}
	})

	hotD := iconSide * 1.18
	v.micPileAnchor = canvas.NewRectangle(color.NRGBA{A: 0})
	v.micPileAnchor.StrokeWidth = 0
	v.micPileAnchor.SetMinSize(fyne.NewSize(hotD, hotD))
	v.micPileAnchor.Resize(fyne.NewSize(hotD, hotD))

	v.micHot = canvas.NewCircle(color.RGBA{R: 225, G: 40, B: 58, A: 78})
	v.micHot.StrokeWidth = 0
	v.micHot.Resize(fyne.NewSize(hotD, hotD))

	micPile := container.NewStack(v.micPileAnchor, v.micHot, micTapW)

	v.recDot = canvas.NewCircle(color.RGBA{R: 255, G: 72, B: 88, A: 255})
	v.recDot.StrokeWidth = 0
	v.recDot.Resize(fyne.NewSize(13, 13))
	v.recDot.Hide()

	recTop := container.NewHBox(layout.NewSpacer(), v.recDot)
	micBlock := container.NewBorder(recTop, nil, nil, nil, micPile)
	dragPad := newWindowDragLayer(w)
	waveStack := container.NewStack(v.wave, dragPad, container.NewCenter(micBlock))

	w.SetContent(waveStack)
	return v
}

func (v *Visualizer) SetOnRecordToggle(fn func()) {
	v.onRecord = fn
}

func (v *Visualizer) SetRecording(active bool) {
	if v.recDot == nil {
		return
	}
	if active {
		v.recDot.Show()
	} else {
		v.recDot.Hide()
	}
	v.recDot.Refresh()
}

func (v *Visualizer) Show() {
	v.window.Show()
}

func (v *Visualizer) Hide() {
	v.window.Hide()
}

func (v *Visualizer) RunApp() {
	v.window.ShowAndRun()
}

// micWaveImage: calm central disk and rim wave (frequency and excursion from volume).
func (v *Visualizer) micWaveImage(w, h int) image.Image {
	v.mu.Lock()
	vol := v.curVol
	phase := v.wavePhase
	evol := v.echoVol
	ephase := v.echoPhase
	v.mu.Unlock()

	if vol > 1 {
		vol = 1
	}
	if evol > 1 {
		evol = 1
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx := float64(w-1) / 2
	cy := float64(h-1) / 2
	minDim := float64(w)
	if h < w {
		minDim = float64(h)
	}

	innerR := minDim * 0.30
	baseR := minDim * 0.38
	// Wide soft disk edge like wave strokes (smoothRingMask), no pixel stair-steps.
	innerSoft := math.Max(5.5, minDim*0.052)

	// Outer echo ring: slightly farther out, same shape with lagging vol/phase
	baseREcho := baseR + minDim*0.048
	// Main wave: shallower peaks plus hard cap so at sin=-1 the ring does not clip under the soft disk edge.
	ampMainRaw := minDim * (0.048 + float64(vol)*0.30) * 0.88
	minRWave := innerR + innerSoft + minDim*0.024 // gap past outer soft disk edge
	ampCap := baseR - minRWave
	if ampCap < 0 {
		ampCap = 0
	}
	ampMain := ampMainRaw
	if ampMain > ampCap {
		ampMain = ampCap
	}
	// Echo keeps the same character as main wave
	ampEcho := minDim * (0.055 + float64(evol)*0.42)
	cycles := math.Round(2.5 + float64(vol)*12.0)
	if cycles < 2 {
		cycles = 2
	}
	cyclesEcho := math.Round(2.5 + float64(evol)*12.0)
	if cyclesEcho < 2 {
		cyclesEcho = 2
	}
	hw := 2.5 + float64(vol)*7.00
	// Soft stroke width: main line softer than raw dd/hw (blurrier edge).
	hwMain := hw*1.35 + 3.2
	hwEcho := hw*1.25 + 1.5

	// Keep wave inside raster bounds (no side clipping artifact).
	edgePad := minDim * 0.035
	maxR := minDim*0.5 - edgePad
	if baseR+ampMain+hwMain > maxR {
		ampMain = math.Max(0, maxR-baseR-hwMain)
	}
	if baseREcho+ampEcho+hwEcho > maxR {
		ampEcho = math.Max(0, maxR-baseREcho-hwEcho)
	}

	innerFill := v.innerFill
	innerEdge := v.innerEdge
	waveHi := color.RGBA{
		R: uint8(math.Min(255, float64(v.waveHiMid.R)*(0.55+0.45*float64(vol)))),
		G: uint8(math.Min(255, float64(v.waveHiMid.G)*(0.55+0.45*float64(vol)))),
		B: uint8(math.Min(255, float64(v.waveHiMid.B)*(0.62+0.38*float64(vol)))),
		A: 255,
	}
	// Outside orb: transparent (compositor shows desktop if window alpha is supported).
	panelBG := color.RGBA{A: 0}
	ringBG := v.ringBGTint

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Hypot(dx, dy)
			if dist < 1e-6 {
				img.Set(x, y, innerFill)
				continue
			}
			// Atan2 is [-π,π]; branch cut would break the wave; use [0,2π).
			th := math.Atan2(dy, dx)
			if th < 0 {
				th += 2 * math.Pi
			}
			rWave := baseR + ampMain*math.Sin(cycles*th+phase)
			rEcho := baseREcho + ampEcho*math.Sin(cyclesEcho*th+ephase)

			// Solid disk center; skip wave math underneath
			if dist < innerR-innerSoft {
				img.Set(x, y, innerFill)
				continue
			}

			col := panelBG
			if dist < innerR {
				// Inward from center: fill → rim
				t := float32(smootherstep01((dist - (innerR - innerSoft)) / innerSoft))
				col = lerpRGBA(innerFill, innerEdge, t)
			} else if dist < innerR+innerSoft {
				// Outward from rim: rim → ring background
				t := float32(smootherstep01((dist - innerR) / innerSoft))
				col = lerpRGBA(innerEdge, ringBG, t)
			}

			margin := math.Max(hwMain, hwEcho) * 2.6
			lim := math.Max(baseR+ampMain, baseREcho+ampEcho) + margin
			if dist >= innerR+innerSoft && dist < lim {
				col = ringBG
			}

			// Echo stroke
			ddE := math.Abs(dist - rEcho)
			if te := smoothRingMask(ddE, hwEcho) * 0.62; te > 0 {
				echoC := color.RGBA{R: 120, G: 210, B: 255, A: 105}
				col = lerpRGBA(col, echoC, te)
			}
			// Main wave stroke
			dd := math.Abs(dist - rWave)
			if tm := smoothRingMask(dd, hwMain) * 0.92; tm > 0 {
				var stroke color.RGBA
				if dist < rWave {
					stroke = lerpRGBA(innerEdge, waveHi, 0.82)
				} else {
					stroke = waveHi
					if stroke.A > 245 {
						stroke.A = 245
					}
				}
				col = lerpRGBA(col, stroke, tm)
			}

			// Feather outer orb edge into transparency (no hard ring at lim).
			outerW := math.Max(6.0, minDim*0.046)
			if dist > lim-outerW {
				u := (dist - (lim - outerW)) / (outerW * 1.38)
				if u > 1 {
					u = 1
				}
				col = lerpRGBA(panelBG, col, float32(1-smoothstep01(u)))
			}

			img.Set(x, y, col)
		}
	}
	return img
}

func smoothstep01(u float64) float64 {
	if u <= 0 {
		return 0
	}
	if u >= 1 {
		return 1
	}
	return u * u * (3 - 2*u)
}

// smootherstep01: smoother S-curve (Perlin) for soft fills such as wave contours.
func smootherstep01(u float64) float64 {
	if u <= 0 {
		return 0
	}
	if u >= 1 {
		return 1
	}
	return u * u * u * (u*(u*6-15) + 10)
}

// smoothRingMask: 1 on the stroke center, 0 at halfWidth — avoids pixel ladder.
func smoothRingMask(dd, halfWidth float64) float32 {
	if halfWidth <= 1e-6 || dd >= halfWidth {
		return 0
	}
	return float32(1 - smoothstep01(dd/halfWidth))
}

func lerpRGBA(a, b color.RGBA, t float32) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	inv := 1 - t
	return color.RGBA{
		R: uint8(inv*float32(a.R) + t*float32(b.R)),
		G: uint8(inv*float32(a.G) + t*float32(b.G)),
		B: uint8(inv*float32(a.B) + t*float32(b.B)),
		A: uint8(inv*float32(a.A) + t*float32(b.A)),
	}
}

// UpdateWave updates volume-driven phase of the traveling rim wave.
func (v *Visualizer) UpdateWave(volume float32) {
	// Mic RMS is usually far below 1 — gain up and lift quiet inputs for visible motion.
	const gain = 4.2
	vol := volume * gain
	if vol > 1 {
		vol = 1
	}
	// Sharper curve: mid levels move more, peaks clamp at 1.
	vol = float32(math.Pow(float64(vol), 0.72))

	v.mu.Lock()
	v.curVol = vol
	v.wavePhase += 0.06 + float64(vol)*0.55

	// Echo follows volume/phase; smaller coeff = more visible lag
	const echoVolFollow = 0.11
	const echoPhaseFollow = 0.09
	v.echoVol += echoVolFollow * (v.curVol - v.echoVol)
	v.echoPhase += echoPhaseFollow * (v.wavePhase - v.echoPhase)

	v.mu.Unlock()
	v.wave.Refresh()
	v.updateMicHotPulse(vol)
}

// updateMicHotPulse: red halo under icon scales with volume (quiet = dim “mic on”).
func (v *Visualizer) updateMicHotPulse(vol float32) {
	if v.micHot == nil {
		return
	}
	iconSide := float32(waveViewSize) * micIconSizeRatio
	baseD := iconSide * 1.18
	pulse := vol * 26
	d := baseD + pulse
	alpha := uint8(72 + vol*175)
	if alpha > 235 {
		alpha = 235
	}
	v.micHot.FillColor = color.RGBA{R: 225, G: 38, B: 58, A: alpha}
	sz := fyne.NewSize(d, d)
	if v.micPileAnchor != nil {
		v.micPileAnchor.SetMinSize(sz)
		v.micPileAnchor.Resize(sz)
	}
	v.micHot.Resize(sz)
	v.micHot.Refresh()
}

// UpdateState refreshes level meter and window title countdown.
func (v *Visualizer) UpdateState(volume float32, secondsLeft float64) {
	v.UpdateWave(volume)
	if secondsLeft > 0 {
		v.window.SetTitle(fmt.Sprintf("Voice Input — %.0f s", secondsLeft))
	} else {
		v.window.SetTitle("Voice Input")
	}
}

// SetClipboardRecognized copies successful transcription to the system clipboard (safe from any goroutine).
func (v *Visualizer) SetClipboardRecognized(text string) {
	if text == "" {
		return
	}
	fyne.Do(func() {
		v.window.Clipboard().SetContent(text)
	})
}

func uiParseHexRGB(s string) (color.RGBA, bool) {
	s = strings.TrimSpace(s)
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	s = strings.ToLower(s)
	if len(s) != 6 {
		return color.RGBA{}, false
	}
	var r, g, b uint8
	n, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
	if err != nil || n != 3 {
		return color.RGBA{}, false
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}, true
}

func uiClampU8(x float32) uint8 {
	if x < 0 {
		return 0
	}
	if x > 255 {
		return 255
	}
	return uint8(x)
}

func uiAccentPalette(ac color.RGBA) (innerFill, innerEdge, waveHiMid, ringBG color.RGBA) {
	innerFill = color.RGBA{
		R: uiClampU8(float32(ac.R) * 0.06),
		G: uiClampU8(float32(ac.G) * 0.12),
		B: uiClampU8(float32(ac.B) * 0.16),
		A: 255,
	}
	innerEdge = color.RGBA{
		R: uiClampU8(float32(ac.R)*0.18 + 35),
		G: uiClampU8(float32(ac.G)*0.25 + 50),
		B: uiClampU8(float32(ac.B)*0.35 + 55),
		A: 255,
	}
	waveHiMid = color.RGBA{
		R: uiClampU8(float32(ac.R)*0.5 + 70),
		G: uiClampU8(float32(ac.G)*0.45 + 90),
		B: 255,
		A: 255,
	}
	ringBG = color.RGBA{
		R: uiClampU8(float32(ac.R)*0.12 + 22),
		G: uiClampU8(float32(ac.G)*0.17 + 38),
		B: uiClampU8(float32(ac.B)*0.22 + 48),
		A: 210,
	}
	return innerFill, innerEdge, waveHiMid, ringBG
}
