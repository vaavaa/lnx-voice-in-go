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
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"lnx-voice-in-go/assets"
)

const waveViewSize = 300

// micIconSize — доля от области визуализации (~диаметр внутреннего диска с отступами от края круга).
const micIconSizeRatio = 0.285

// micTap — PNG микрофона с обработкой нажатия без оформления кнопки (нет hover/pressed у кнопки).
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

type Visualizer struct {
	window        fyne.Window
	wave          *canvas.Raster
	micHot        *canvas.Circle    // полупрозрачный красный «микрофон включён», пульс от уровня
	micPileAnchor *canvas.Rectangle // задаёт MinSize стека иконки + подсветки (у Circle нет SetMinSize)
	recDot        *canvas.Circle    // фиксированная точка в углу: идёт запись
	onRecord      func()

	mu        sync.Mutex
	curVol    float32
	wavePhase float64
	// Эхо: сглаженные копии — «догоняют» основной слой с задержкой
	echoVol   float32
	echoPhase float64
}

func NewOverlay() *Visualizer {
	myApp := app.NewWithID("io.github.lnx-voice-in-go")

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

	v.wave = canvas.NewRaster(func(wi, hi int) image.Image {
		return v.micWaveImage(wi, hi)
	})
	// Иначе centerLayout даст объекту размер MinSize() = 1×1 (дефолт у Raster) — ничего не видно.
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
	waveStack := container.NewStack(v.wave, container.NewCenter(micBlock))

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
	// Широкий мягкий край диска — как у линий волны (smoothRingMask), без ступенек по пикселям.
	innerSoft := math.Max(5.5, minDim*0.052)

	// Внешнее «эхо» — чуть дальше от центра, та же форма с отстающими vol/phase
	baseREcho := baseR + minDim*0.048
	// Основная волна: положе, чем раньше (меньше «крутизна» пиков), плюс жёсткий потолок —
	// иначе при sin=-1 кольцо уходит под мягкий край диска и выглядит обрезанным.
	ampMainRaw := minDim * (0.048 + float64(vol)*0.30) * 0.88
	minRWave := innerR + innerSoft + minDim*0.024 // зазор от внешнего края мягкого диска
	ampCap := baseR - minRWave
	if ampCap < 0 {
		ampCap = 0
	}
	ampMain := ampMainRaw
	if ampMain > ampCap {
		ampMain = ampCap
	}
	// Эхо без изменения характера
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

	// Не даём волне вылезти за край квадрата растра (обрезание по бокам).
	edgePad := minDim * 0.035
	maxR := minDim*0.5 - edgePad
	if baseR+ampMain+hwMain > maxR {
		ampMain = math.Max(0, maxR-baseR-hwMain)
	}
	if baseREcho+ampEcho+hwEcho > maxR {
		ampEcho = math.Max(0, maxR-baseREcho-hwEcho)
	}

	innerFill := color.RGBA{R: 15, G: 95, B: 190, A: 255}
	innerEdge := color.RGBA{R: 40, G: 140, B: 235, A: 255}
	waveHi := color.RGBA{
		R: uint8(90 + 165*vol),
		G: uint8(180 + 75*vol),
		B: 255,
		A: 255,
	}
	// Вне «орба» — прозрачно (композитор покажет рабочий стол, если окно с альфой поддерживается).
	panelBG := color.RGBA{A: 0}
	ringBG := color.RGBA{R: 28, G: 72, B: 130, A: 210}

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
			rWave := baseR + ampMain*math.Sin(cycles*th+phase)
			rEcho := baseREcho + ampEcho*math.Sin(cyclesEcho*th+ephase)

			// Центр диска — сплошная заливка; волны под него не рисуем
			if dist < innerR-innerSoft {
				img.Set(x, y, innerFill)
				continue
			}

			col := panelBG
			if dist < innerR {
				// из центра к номинальному радиусу: fill → ободок
				t := float32(smootherstep01((dist - (innerR - innerSoft)) / innerSoft))
				col = lerpRGBA(innerFill, innerEdge, t)
			} else if dist < innerR+innerSoft {
				// наружу от края: ободок → фон кольца
				t := float32(smootherstep01((dist - innerR) / innerSoft))
				col = lerpRGBA(innerEdge, ringBG, t)
			}

			margin := math.Max(hwMain, hwEcho) * 2.6
			lim := math.Max(baseR+ampMain, baseREcho+ampEcho) + margin
			if dist >= innerR+innerSoft && dist < lim {
				col = ringBG
			}

			// Эхо
			ddE := math.Abs(dist - rEcho)
			if te := smoothRingMask(ddE, hwEcho) * 0.62; te > 0 {
				echoC := color.RGBA{R: 120, G: 210, B: 255, A: 105}
				col = lerpRGBA(col, echoC, te)
			}
			// Основная волна
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

			// Внешний край орба — плавно в прозрачность, без резкого кольца на lim.
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

// smootherstep01 — более плавный S-изгиб (Perlin), для мягких заливок как у контуров волны.
func smootherstep01(u float64) float64 {
	if u <= 0 {
		return 0
	}
	if u >= 1 {
		return 1
	}
	return u * u * u * (u*(u*6-15) + 10)
}

// smoothRingMask: 1 на самой линии, 0 на краю полосы halfWidth — без «лесенки».
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
	v.updateMicHotPulse(vol)
}

// updateMicHotPulse — красная подсветка под иконкой: яркость и размер от громкости (0 = тихий «мик вкл»).
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

// UpdateState обновляет индикатор уровня и заголовок окна.
func (v *Visualizer) UpdateState(volume float32, secondsLeft float64) {
	v.UpdateWave(volume)
	if secondsLeft > 0 {
		v.window.SetTitle(fmt.Sprintf("Voice Input — %.0f s", secondsLeft))
	} else {
		v.window.SetTitle("Voice Input")
	}
}

// SetClipboardRecognized кладёт в системный буфер только успешный текст распознавания (вызывать из любого потока).
func (v *Visualizer) SetClipboardRecognized(text string) {
	if text == "" {
		return
	}
	fyne.Do(func() {
		v.window.Clipboard().SetContent(text)
	})
}
