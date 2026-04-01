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
	sampleRate  int
	Samples     []float32
	IsRecording bool
	mu          sync.Mutex // Protects Samples and IsRecording.
}

func NewRecorder(sampleRate int) (*Recorder, error) {
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	r := &Recorder{sampleRate: sampleRate}

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}
	r.ctx = ctx

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = uint32(sampleRate)

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

// BeginSession clears the buffer and enables capture in the device callback.
func (r *Recorder) BeginSession() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Samples = make([]float32, 0, r.sampleRate*10)
	r.IsRecording = true
}

// EndSession stops capture and returns a copy of accumulated samples.
func (r *Recorder) EndSession() []float32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.IsRecording = false
	out := make([]float32, len(r.Samples))
	copy(out, r.Samples)
	return out
}

// Clear resets the sample buffer before a new session.
func (r *Recorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Samples = make([]float32, 0, r.sampleRate*10)
}

// GetRMS returns the current loudness (root mean square) over a short tail window.
func (r *Recorder) GetRMS() float32 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.IsRecording {
		return 0
	}

	window := r.sampleRate / 10 // ~100 ms
	if window < 1 {
		window = 1600
	}
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

// GetData returns a copy of accumulated samples (for passing to the engine).
func (r *Recorder) GetData() []float32 {
	r.mu.Lock()
	defer r.mu.Unlock()

	dest := make([]float32, len(r.Samples))
	copy(dest, r.Samples)
	return dest
}
