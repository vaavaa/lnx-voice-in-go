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
)

var (
	whisperMu      sync.Mutex
	whisperCtx     *C.struct_whisper_context
	whisperInitErr error
)

// Process распознаёт PCM mono float32 @ 16 kHz через whisper_full.
// Путь к модели: WHISPER_MODEL (по умолчанию models/ggml-small.bin).
// Язык: WHISPER_LANG (по умолчанию auto). GPU: по умолчанию из whisper; WHISPER_USE_GPU=0 — только CPU.
// Отладка цепочки без модели: VOICE_ECHO_STUB=1.
func Process(samples []float32) string {
	if len(samples) == 0 {
		return ""
	}
	if os.Getenv("VOICE_ECHO_STUB") != "" {
		sec := float64(len(samples)) / 16000
		return fmt.Sprintf("echo-test %.1fs ", sec)
	}

	whisperMu.Lock()
	defer whisperMu.Unlock()

	ctx, err := ensureCtxLocked()
	if err != nil {
		return fmt.Sprintf("Ошибка Whisper: %v", err)
	}

	params := C.whisper_full_default_params(C.WHISPER_SAMPLING_GREEDY)
	params.n_threads = C.int(runtime.NumCPU())
	params.print_progress = C.bool(false)
	params.print_realtime = C.bool(false)
	params.print_timestamps = C.bool(false)

	lang := os.Getenv("WHISPER_LANG")
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
		return fmt.Sprintf("Ошибка whisper_full (код %d)", int(rc))
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

	path := os.Getenv("WHISPER_MODEL")
	if path == "" {
		path = "models/ggml-small.bin"
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	cparams := C.whisper_context_default_params()
	if os.Getenv("WHISPER_USE_GPU") == "0" {
		cparams.use_gpu = C.bool(false)
	}

	ctx := C.whisper_init_from_file_with_params(cpath, cparams)
	if ctx == nil {
		whisperInitErr = fmt.Errorf("не удалось загрузить модель %q (файл, права, CUDA/библиотеки)", path)
		return nil, whisperInitErr
	}
	whisperCtx = ctx
	fmt.Fprintf(os.Stderr, "whisper: модель загружена: %s (use_gpu=%v)\n", path, bool(cparams.use_gpu))
	return whisperCtx, nil
}

func CheckCUDA() {
	info := C.whisper_print_system_info()
	fmt.Printf("Whisper System Info: %s\n", C.GoString(info))

	params := C.whisper_context_default_params()
	params.use_gpu = C.bool(true)

	modelPath := C.CString("models/ggml-small.bin")
	defer C.free(unsafe.Pointer(modelPath))

	ctx := C.whisper_init_from_file_with_params(modelPath, params)
	if ctx == nil {
		fmt.Println("❌ Ошибка: Не удалось инициализировать модель. Проверьте путь и CUDA драйверы.")
	} else {
		fmt.Println("✅ Успех: Модель загружена, GPU (CUDA) доступен!")
		C.whisper_free(ctx)
	}
}
