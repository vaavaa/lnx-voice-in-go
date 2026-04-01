# lnx-voice-in-go

*Russian: [README.ru.md](README.ru.md)*

## What this is

A small **Linux** utility: it captures speech from the microphone, transcribes it locally with **whisper.cpp**, and injects the text into the focused window (via `ydotool`). A compact **Fyne** overlay shows a level meter and a record control. Recording length cap, hotkey, sample rate, model path, UI theme, and more are read from **`config.yaml`** (see **Configuration** below). **CUDA** is used at build time and when initializing whisper if drivers and libraries are available.

This is not a cloud service or a full‑featured “voice assistant.” The point of the repo is a reproducible **microphone → PCM → whisper → text input** pipeline with minimal UI.

## Stack and layout

| Piece | Role |
|--------|------|
| **Go 1.26+**, **Cobra** | Entry point, CLI (`--config`) |
| **Viper** | Load `config.yaml` + `VOICE_*` env overrides |
| **Fyne v2** | Overlay window, visualization, record toggle |
| **malgo** | Audio capture: mono **float32**, **16 kHz** |
| **whisper.cpp** (submodule `third_party/whisper`) | Transcription via CGO (`whisper_full`) |
| **gohook** | Global hotkey from `audio.hotkey` in config (`internal/input`) |
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

Place GGML weight files under **`models/`** (`*.bin` patterns are gitignored). Default path is `models/ggml-small.bin` (see `model.path` in `config.yaml`) unless overridden by `VOICE_MODEL_PATH` or `WHISPER_MODEL`.

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

Optional explicit config path:

```bash
./lnx-voice-in-go --config /path/to/config.yaml
```

Check CUDA and model load (whisper system info; uses `model.path` from config):

```bash
./lnx-voice-in-go whisper-selftest
```

## Configuration

Settings are loaded from **`config.yaml`** or **`config.yml`** by default (first match wins per directory). Without `--config`, Viper searches the current directory, then **`~/.voice-input`**, then **`~/.config/lnx-voice-in-go`**.

- **File**: see root **`config.yml`** for field names (`model`, `audio`, `ui`).
- **`VOICE_` prefix**: overrides YAML after load (nested keys use underscores, e.g. `VOICE_MODEL_PATH`, `VOICE_AUDIO_MAX_DURATION_SEC`, `VOICE_UI_SHOW_RESULT`).
- **Legacy**: `WHISPER_MODEL`, `WHISPER_LANG`, `WHISPER_USE_GPU` are still applied **after** the file and `VOICE_*`, for the model block only.

Notable fields:

| YAML path | Role |
|-----------|------|
| `model.path` / `lang` / `use_gpu` | Whisper weights, language, GPU toggle |
| `audio.sample_rate` | Capture rate (16 kHz matches Whisper) |
| `audio.max_duration_sec` | Auto-stop recording |
| `audio.hotkey` | Global toggle key (gohook name, e.g. `F12`) |
| `ui.theme` | `dark` or `light` |
| `ui.main_color` | Hex accent for the orb |
| `ui.show_result` | If true, copy successful transcript to clipboard |

## Environment variables

| Variable | Purpose |
|----------|---------|
| `VOICE_*` | Overrides for `config.yaml` (see above) |
| `WHISPER_MODEL` | Path to model (overrides `model.path` after load) |
| `WHISPER_LANG` | Whisper language (overrides `model.lang`) |
| `WHISPER_USE_GPU` | `0` / `false` forces CPU for whisper context |
| `VOICE_ECHO_STUB` | Non-empty skips whisper and returns a test string (pipeline debug) |

## License

Application source is under **Apache License 2.0** (see `LICENSE`). Dependencies and the **whisper.cpp** submodule have their own licenses — see `third_party/whisper`.
