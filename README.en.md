# lnx-voice-in-go

*Русский: [README.ru.md](README.ru.md)*

## What this is

A small **Linux** utility: it captures speech from the microphone, transcribes it locally with **whisper.cpp**, and injects the text into the focused window (via `ydotool`). A compact **Fyne** overlay shows a level meter and a record control. Recording duration is capped (currently up to 60 s). **CUDA** is used at build time and when initializing whisper if drivers and libraries are available.

This is not a cloud service or a full‑featured “voice assistant.” The point of the repo is a reproducible **microphone → PCM → whisper → text input** pipeline with minimal UI.

## Stack and layout

| Piece | Role |
|--------|------|
| **Go 1.26+**, **Cobra** | Entry point, CLI |
| **Fyne v2** | Overlay window, visualization, record toggle |
| **malgo** | Audio capture: mono **float32**, **16 kHz** |
| **whisper.cpp** (submodule `third_party/whisper`) | Transcription via CGO (`whisper_full`) |
| **gohook** | Keyboard hooks (`internal/input`; hotkey flows can be extended separately) |
| **ydotool** | Injects keystrokes into the focused widget |

Go module: `lnx-voice-in-go`. Sources: `cmd/`, `internal/audio`, `internal/engine`, `internal/input`, `internal/ui`, `assets/`.

## Requirements

- **OS**: Linux; GUI needs a display and OpenGL/X11 dependencies (typical Fyne setup on X11).
- **Whisper build**: CMake, C++ toolchain; for the Makefile’s target config — **NVIDIA CUDA** and a matching GPU architecture (the sample uses `-DCMAKE_CUDA_ARCHITECTURES=89`; change it for your hardware).
- **Go build**: a working **Go toolchain**; **CGO** on; linking pulls whisper/ggml and CUDA (see `#cgo LDFLAGS` in `internal/engine/whisper.go`).
- **Typing**: **`ydotool`** installed, plus uinput permissions where required (often on Wayland; see your distro docs).

Ubuntu/Debian dev packages for the X11/CGO stack (from this repo):

```bash
make deps-apt
```

## Models

Place GGML weight files under **`models/`** (`*.bin` patterns are gitignored). Default path is `models/ggml-small.bin` unless overridden by environment.

Download models with upstream **whisper.cpp** scripts (`third_party/whisper/models/`) or any source of compatible `ggml-*.bin` files.

## Build

Initialize the submodule:

```bash
git submodule update --init --recursive
```

Build the whisper library (CUDA as in `Makefile`) and the binary:

```bash
make all
```

Output binary: **`lnx-voice-in-go`**. Remove CMake artifacts and the binary: `make clean`.

**Non‑CUDA** builds or different CMake options require your own `cmake` setup under `third_party/whisper` and edits to `#cgo` in `internal/engine/whisper.go` to match your library set (CPU-only, other ggml backends, etc.).

## Run

```bash
./lnx-voice-in-go
```

Check CUDA and model load (whisper system info):

```bash
./lnx-voice-in-go whisper-selftest
```

## Environment variables

| Variable | Purpose |
|----------|---------|
| `WHISPER_MODEL` | Path to the model file (default `models/ggml-small.bin`) |
| `WHISPER_LANG` | Whisper language, e.g. `ru`; empty uses code default (`auto`) |
| `WHISPER_USE_GPU` | Set to `0` to disable GPU when creating the whisper context |
| `VOICE_ECHO_STUB` | Any non-empty value skips whisper and returns a test string (pipeline debug) |

## License

Application source is under **Apache License 2.0** (see `LICENSE`). Dependencies and the **whisper.cpp** submodule have their own licenses — see `third_party/whisper`.
