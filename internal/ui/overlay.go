package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const waveViewSize = 220

type Visualizer struct {
	window      fyne.Window
	wave        *canvas.Raster
	resultLabel *widget.Label
	recordBtn   *widget.Button
	onRecord    func()

	mu        sync.Mutex
	curVol    float32
	wavePhase float64
	// Эхо: сглаженные копии — «догоняют» основной слой с задержкой
	echoVol   float32
	echoPhase float64
}

func NewOverlay() *Visualizer {
	myApp := app.New()
	w := myApp.NewWindow("Voice Input")

	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(280, 420))

	v := &Visualizer{window: w}

	v.wave = canvas.NewRaster(func(wi, hi int) image.Image {
		return v.micWaveImage(wi, hi)
	})
	// Иначе centerLayout даст объекту размер MinSize() = 1×1 (дефолт у Raster) — ничего не видно.
	v.wave.SetMinSize(fyne.NewSize(waveViewSize, waveViewSize))
	v.wave.Resize(fyne.NewSize(waveViewSize, waveViewSize))

	v.resultLabel = widget.NewLabel("Готов к работе...")
	v.resultLabel.Wrapping = fyne.TextWrapWord
	v.resultLabel.Alignment = fyne.TextAlignCenter

	resultScroll := container.NewScroll(v.resultLabel)
	resultScroll.SetMinSize(fyne.NewSize(260, 150))

	v.recordBtn = widget.NewButton("Начать запись", func() {
		if v.onRecord != nil {
			v.onRecord()
		}
	})

	w.SetContent(container.NewVBox(
		container.NewCenter(v.wave),
		container.NewPadded(resultScroll),
		container.NewPadded(v.recordBtn),
	))
	return v
}

func (v *Visualizer) SetOnRecordToggle(fn func()) {
	v.onRecord = fn
}

func (v *Visualizer) SetRecording(active bool) {
	if active {
		v.recordBtn.SetText("Остановить (F12)")
		v.recordBtn.Importance = widget.DangerImportance
	} else {
		v.recordBtn.SetText("Начать запись (F12)")
		v.recordBtn.Importance = widget.MediumImportance
	}
	v.recordBtn.Refresh()
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

// micWaveImage: спокойный центральный диск и волна по контуру (частота и «высота» от громкости).
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
	// Внешнее «эхо» — чуть дальше от центра, та же форма с отстающими vol/phase
	baseREcho := baseR + minDim*0.048
	amp := minDim * (0.055 + float64(vol)*0.42)
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
	// Ширина мягкого края: основная линия — как эхо, расплывчатая (не резкий dd/hw).
	hwMain := hw*1.35 + 3.2
	hwEcho := hw*1.25 + 1.5

	innerFill := color.RGBA{R: 15, G: 95, B: 190, A: 255}
	innerEdge := color.RGBA{R: 40, G: 140, B: 235, A: 255}
	waveHi := color.RGBA{
		R: uint8(90 + 165*vol),
		G: uint8(180 + 75*vol),
		B: 255,
		A: 255,
	}
	// Непрозрачный фон всю область — иначе на части тем прозрачность «съедает» картинку.
	panelBG := color.RGBA{R: 22, G: 48, B: 88, A: 255}
	ringBG := color.RGBA{R: 28, G: 72, B: 130, A: 255}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Hypot(dx, dy)
			if dist < 1e-6 {
				img.Set(x, y, innerFill)
				continue
			}
			// Atan2 даёт [-π,π] — на ветви влево скачок π→-π рвёт волну; переводим в [0,2π).
			th := math.Atan2(dy, dx)
			if th < 0 {
				th += 2 * math.Pi
			}
			rWave := baseR + amp*math.Sin(cycles*th+phase)
			rEcho := baseREcho + ampEcho*math.Sin(cyclesEcho*th+ephase)

			col := panelBG

			switch {
			case dist < innerR-1:
				col = innerFill
			case dist < innerR+1:
				t := float32(dist - (innerR - 1))
				col = lerpRGBA(innerFill, innerEdge, t)
			default:
				margin := math.Max(hwMain, hwEcho) * 2.6
				lim := math.Max(baseR+amp, baseREcho+ampEcho) + margin
				if dist < lim {
					col = ringBG
				}
				// Эхо: мягкий край через smoothstep
				ddE := math.Abs(dist - rEcho)
				if te := smoothRingMask(ddE, hwEcho) * 0.62; te > 0 {
					echoC := color.RGBA{R: 120, G: 210, B: 255, A: 105}
					col = lerpRGBA(col, echoC, te)
				}
				// Основная волна — такая же расплывчатая линия, чуть ярче и уже по max opacity
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
			}
			img.Set(x, y, col)
		}
	}
	return img
}

// smoothRingMask: 1 на самой линии, 0 на краю полосы halfWidth — без «лесенки» (smoothstep).
func smoothRingMask(dd, halfWidth float64) float32 {
	if halfWidth <= 1e-6 || dd >= halfWidth {
		return 0
	}
	u := dd / halfWidth
	u = u * u * (3 - 2*u)
	return float32(1 - u)
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

// UpdateWave обновляет громкость и фазу бегущей волны по кругу.
func (v *Visualizer) UpdateWave(volume float32) {
	// RMS с микрофона обычно сильно ниже 1 — усиливаем и чуть приподнимаем «тихие» значения для заметной реакции.
	const gain = 4.2
	vol := volume * gain
	if vol > 1 {
		vol = 1
	}
	// Кривая «резче»: средние уровни дают больший сдвиг, пики упираются в 1.
	vol = float32(math.Pow(float64(vol), 0.72))

	v.mu.Lock()
	v.curVol = vol
	v.wavePhase += 0.06 + float64(vol)*0.55

	// Эхо догоняет громкость и фазу — меньше коэффициент = заметнее задержка
	const echoVolFollow = 0.11
	const echoPhaseFollow = 0.09
	v.echoVol += echoVolFollow * (v.curVol - v.echoVol)
	v.echoPhase += echoPhaseFollow * (v.wavePhase - v.echoPhase)

	v.mu.Unlock()
	v.wave.Refresh()
}

// UpdateState обновляет индикатор уровня и заголовок окна.
func (v *Visualizer) UpdateState(volume float32, secondsLeft float64) {
	v.UpdateWave(volume)
	if secondsLeft > 0 {
		v.window.SetTitle(fmt.Sprintf("Voice Input — %.0f с", secondsLeft))
	} else {
		v.window.SetTitle("Voice Input")
	}
}

func (v *Visualizer) SetResultText(s string) {
	v.resultLabel.SetText(s)
	v.resultLabel.Refresh()
}
