# lnx-voice-in-go

**Speak where you type — privately, on your machine.** A lean Linux utility that runs **Whisper** locally (optionally **GPU-accelerated** via CUDA), shows a small **Fyne** overlay, and injects transcribed text into whatever window is focused—no cloud API, just microphone, model, and keyboard injection.

Full technical write-ups: **[English](README.en.md)** · **[Русский](README.ru.md)** (stack, build, licensing).

---

## What it looks like

The app is a small **desktop overlay**, not a full-screen program. The screenshot shows the usual layout while dictating: mic level, recording controls, and the live visualization. Transcribed text is **injected into the focused input** (chat, editor, browser form, etc.) where your cursor is. On **Linux/X11**, you can **reposition** the overlay by dragging anywhere **except** the mic icon; that feature uses **`xdotool`** (may not work on plain Wayland — see [README.ru.md](README.ru.md) / [README.en.md](README.en.md)).

![lnx-voice-in-go overlay: mic level, recording, and rim visualization](assets/images/form-screenshot.png)

---

## Configuration

Runtime options live in **`config.yml`** or **`config.yaml`** at the repo root (or under `~/.voice-input` / `~/.config/lnx-voice-in-go`). Load a specific file with:

```bash
./lnx-voice-in-go --config /path/to/config.yaml
```

The app uses **Viper**: values from the file can be overridden by **`VOICE_`** environment variables (nested YAML keys map to names like `VOICE_MODEL_PATH`, `VOICE_AUDIO_SAMPLE_RATE`, `VOICE_UI_THEME`, …). After that, legacy **`WHISPER_MODEL`**, **`WHISPER_LANG`**, and **`WHISPER_USE_GPU`** still override the **model** block.

| Block | Fields (summary) |
|--------|------------------|
| **model** | `path`, `type` (hint), `lang` (e.g. `auto`, `en`, `ru`), `use_gpu` |
| **audio** | `sample_rate` (16 kHz is what Whisper expects), `max_duration_sec`, `hotkey` (e.g. `F12`; gohook key names) |
| **ui** | `theme` (`dark` / `light`), `main_color` (hex accent), `show_result` (copy transcript to clipboard) |

See **`config.yml`** in the repo for a full example. Details: [README.en.md](README.en.md) · [README.ru.md](README.ru.md).
