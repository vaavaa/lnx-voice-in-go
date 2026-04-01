package audio

import (
	"encoding/binary"
	"math"
	"sync"

	"github.com/gen2brain/malgo"
)

type Recorder struct {
	ctx         *malgo.AllocatedContext
	device      *malgo.Device
	Samples     []float32
	IsRecording bool
	mu          sync.Mutex // Защищает Samples и IsRecording
}

func NewRecorder() (*Recorder, error) {
	r := &Recorder{}

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}
	r.ctx = ctx

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = 16000

	onRec := func(pOutput, pInput []byte, frameCount uint32) {
		r.mu.Lock()
		defer r.mu.Unlock()

		if !r.IsRecording {
			return
		}

		newSamples := bytesToFloat32(pInput)
		r.Samples = append(r.Samples, newSamples...)
	}

	cbs := malgo.DeviceCallbacks{Data: onRec}
	device, err := malgo.InitDevice(ctx.Context, deviceConfig, cbs)
	if err != nil {
		_ = ctx.Uninit()
		ctx.Free()
		return nil, err
	}
	r.device = device
	if err := device.Start(); err != nil {
		device.Uninit()
		_ = ctx.Uninit()
		ctx.Free()
		return nil, err
	}

	return r, nil
}

func bytesToFloat32(p []byte) []float32 {
	n := len(p) / 4
	if n == 0 {
		return nil
	}
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(p[i*4 : i*4+4]))
	}
	return out
}

// BeginSession очищает буфер и включает запись в колбэке.
func (r *Recorder) BeginSession() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Samples = make([]float32, 0, 16000*10)
	r.IsRecording = true
}

// EndSession выключает запись и возвращает копию накопленных сэмплов.
func (r *Recorder) EndSession() []float32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.IsRecording = false
	out := make([]float32, len(r.Samples))
	copy(out, r.Samples)
	return out
}

// Clear чистит буфер перед новой записью.
func (r *Recorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Samples = make([]float32, 0, 16000*10)
}

// GetRMS возвращает текущий уровень громкости (Root Mean Square).
func (r *Recorder) GetRMS() float32 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.IsRecording {
		return 0
	}

	window := 1600 // ~100 ms при 16 kHz
	n := len(r.Samples)
	if n < window {
		return 0
	}

	var sum float32
	for i := n - window; i < n; i++ {
		sum += r.Samples[i] * r.Samples[i]
	}
	return float32(math.Sqrt(float64(sum / float32(window))))
}

// GetData возвращает копию накопленных данных для передачи в Whisper.
func (r *Recorder) GetData() []float32 {
	r.mu.Lock()
	defer r.mu.Unlock()

	dest := make([]float32, len(r.Samples))
	copy(dest, r.Samples)
	return dest
}
