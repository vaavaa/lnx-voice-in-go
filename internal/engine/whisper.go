package engine

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/whisper/include -I${SRCDIR}/../../third_party/whisper/ggml/include
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/whisper/build/src -L${SRCDIR}/../../third_party/whisper/build/ggml/src -L${SRCDIR}/../../third_party/whisper/build/ggml/src/ggml-cuda -lwhisper -lggml -lggml-cpu -lggml-cuda -lggml-base -lm -lgomp -lpthread -ldl -lstdc++ -lcudart -lcublas -lcublasLt -lcuda

#include "whisper.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"lnx-voice-in-go/internal/config"
)

var (
	whisperMu      sync.Mutex
	whisperCtx     *C.struct_whisper_context
	whisperInitErr error
)

// Process transcribes mono float32 PCM via whisper_full (16 kHz expected; see audio.sample_rate in config).
// Model path, language, and GPU come from config.yaml plus VOICE_* env; legacy WHISPER_* still supported. Debug without model: VOICE_ECHO_STUB=1.
func Process(samples []float32) string {
	if len(samples) == 0 {
		return ""
	}
	sr := float64(config.AppConfig.Audio.SampleRate)
	if sr <= 0 {
		sr = 16000
	}
	if os.Getenv("VOICE_ECHO_STUB") != "" {
		sec := float64(len(samples)) / sr
		return fmt.Sprintf("echo-test %.1fs ", sec)
	}

	whisperMu.Lock()
	defer whisperMu.Unlock()

	ctx, err := ensureCtxLocked()
	if err != nil {
		return fmt.Sprintf("Whisper error: %v", err)
	}

	params := C.whisper_full_default_params(C.WHISPER_SAMPLING_GREEDY)
	params.n_threads = C.int(runtime.NumCPU())
	params.print_progress = C.bool(false)
	params.print_realtime = C.bool(false)
	params.print_timestamps = C.bool(false)

	lang := config.AppConfig.Model.Lang
	if lang == "" {
		lang = "auto"
	}
	langC := C.CString(lang)
	defer C.free(unsafe.Pointer(langC))
	params.language = langC

	pcm := (*C.float)(unsafe.Pointer(&samples[0]))
	n := C.int(len(samples))
	rc := C.whisper_full(ctx, params, pcm, n)
	if rc != 0 {
		return fmt.Sprintf("Whisper error: whisper_full returned code %d", int(rc))
	}

	nSeg := int(C.whisper_full_n_segments(ctx))
	var b strings.Builder
	for i := range nSeg {
		t := C.whisper_full_get_segment_text(ctx, C.int(i))
		if t == nil {
			continue
		}
		b.WriteString(C.GoString(t))
	}
	return strings.TrimSpace(b.String())
}

func ensureCtxLocked() (*C.struct_whisper_context, error) {
	if whisperInitErr != nil {
		return nil, whisperInitErr
	}
	if whisperCtx != nil {
		return whisperCtx, nil
	}

	path := config.AppConfig.Model.Path
	if path == "" {
		path = "models/ggml-small.bin"
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	cparams := C.whisper_context_default_params()
	if !config.AppConfig.Model.UseGPU {
		cparams.use_gpu = C.bool(false)
	}

	ctx := C.whisper_init_from_file_with_params(cpath, cparams)
	if ctx == nil {
		whisperInitErr = fmt.Errorf("failed to load model %q (file, permissions, CUDA/libs)", path)
		return nil, whisperInitErr
	}
	whisperCtx = ctx
	fmt.Fprintf(os.Stderr, "whisper: model loaded: %s (use_gpu=%v)\n", path, bool(cparams.use_gpu))
	return whisperCtx, nil
}

func CheckCUDA() {
	info := C.whisper_print_system_info()
	fmt.Printf("Whisper System Info: %s\n", C.GoString(info))

	params := C.whisper_context_default_params()
	params.use_gpu = C.bool(true)

	mp := config.AppConfig.Model.Path
	if mp == "" {
		mp = "models/ggml-small.bin"
	}
	modelPath := C.CString(mp)
	defer C.free(unsafe.Pointer(modelPath))

	ctx := C.whisper_init_from_file_with_params(modelPath, params)
	if ctx == nil {
		fmt.Println("❌ Error: failed to initialize model. Check model path and CUDA drivers.")
	} else {
		fmt.Println("✅ OK: model loaded, GPU (CUDA) is available.")
		C.whisper_free(ctx)
	}
}
