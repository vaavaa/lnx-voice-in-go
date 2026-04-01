# lnx-voice-in-go

*English: [README.en.md](README.en.md)*

## Что это

Небольшая утилита для **Linux**: захватывает речь с микрофона, расшифровывает её локально через **whisper.cpp** и подставляет текст в активное окно (через `ydotool`). Компактное окно **Fyne** показывает уровень сигнала и управление записью. Лимит длительности записи, горячая клавиша, частота дискретизации, путь к модели, тема UI и др. берутся из **`config.yml`** / **`config.yaml`** (см. раздел **Конфигурация**). **CUDA** задействуется при сборке и при инициализации whisper, если доступны драйвер и библиотеки.

Это не облачный сервис и не полноценный «голосовой ассистент» — задача репозитория: воспроизводимый конвейер **микрофон → PCM → whisper → ввод текста** с минимальным интерфейсом.

## Стек и структура

| Компонент | Назначение |
|-----------|------------|
| **Go 1.26+**, **Cobra** | Точка входа, CLI (`--config`) |
| **Viper** | Загрузка YAML и переопределения `VOICE_*` |
| **Fyne v2** | Оверлей, визуализация, переключение записи |
| **malgo** | Захват аудио: mono **float32**, **16 kHz** |
| **whisper.cpp** (сабмодуль `third_party/whisper`) | Распознавание через CGO (`whisper_full`) |
| **gohook** | Глобальная горячая клавиша из `audio.hotkey` в конфиге (`internal/input`) |
| **ydotool** | Эмуляция ввода в сфокусированный виджет |

Модуль Go: `lnx-voice-in-go`. Исходники: `cmd/`, `internal/audio`, `internal/engine`, `internal/input`, `internal/ui`, `assets/`.

## Требования

- **ОС**: Linux; для GUI — дисплей и зависимости OpenGL/X11 (типичная конфигурация Fyne на X11).
- **Сборка whisper**: CMake, C++; в конфигурации из `Makefile` — **NVIDIA CUDA** и подходящая архитектура GPU (в примере `-DCMAKE_CUDA_ARCHITECTURES=89`; замените под своё железо).
- **Сборка Go**: рабочий **toolchain Go**; включён **CGO**; линковка тянет whisper/ggml и CUDA (см. `#cgo LDFLAGS` в `internal/engine/whisper.go`).
- **Ввод текста**: установленный **`ydotool`** и при необходимости права на uinput (часто на Wayland; см. документацию дистрибутива).

Пакеты разработки Ubuntu/Debian для стека X11/CGO (из репозитория):

```bash
make deps-apt
```

## Модели

Файлы весов GGML кладите в **`models/`** (шаблоны `*.bin` в `.gitignore`). По умолчанию — `models/ggml-small.bin` (поле `model.path` в конфиге), если не заданы `VOICE_MODEL_PATH` или `WHISPER_MODEL`.

Скачивание — скриптами **whisper.cpp** (`third_party/whisper/models/`) или из других источников совместимых `ggml-*.bin`.

## Сборка

Инициализация сабмодуля:

```bash
git submodule update --init --recursive
```

Сборка библиотеки whisper (CUDA как в `Makefile`) и бинарника:

```bash
make all
```

Бинарь: **`lnx-voice-in-go`**. Очистка: `make clean`.

Сборка **без** CUDA или с другими флагами CMake — отдельный запуск `cmake` в `third_party/whisper` и правки `#cgo` в `internal/engine/whisper.go` под ваш набор библиотек (только CPU, другие бэкенды ggml и т.д.).

## Запуск

```bash
./lnx-voice-in-go
```

Явный путь к конфигу:

```bash
./lnx-voice-in-go --config /path/to/config.yaml
```

Проверка CUDA и загрузки модели (системная информация whisper; путь к модели из `model.path` в конфиге):

```bash
./lnx-voice-in-go whisper-selftest
```

## Конфигурация

По умолчанию читается **`config.yaml`** или **`config.yml`** (первый найденный файл в каталоге). Поиск без `--config`: текущий каталог, **`~/.voice-input`**, **`~/.config/lnx-voice-in-go`**.

- **Файл**: см. **`config.yml`** в корне репозитория (`model`, `audio`, `ui`).
- **Префикс `VOICE_`**: переопределяет YAML (вложенные ключи через подчёркивание: `VOICE_MODEL_PATH`, `VOICE_AUDIO_MAX_DURATION_SEC`, `VOICE_UI_SHOW_RESULT`, …).
- **Устаревшие имена**: `WHISPER_MODEL`, `WHISPER_LANG`, `WHISPER_USE_GPU` применяются **после** файла и `VOICE_*` и затрагивают только блок **model**.

Поля (кратко):

| Путь в YAML | Назначение |
|-------------|------------|
| `model.path` / `lang` / `use_gpu` | Веса Whisper, язык, использование GPU |
| `audio.sample_rate` | Частота захвата (16 kHz соответствует Whisper) |
| `audio.max_duration_sec` | Автоостанов записи |
| `audio.hotkey` | Глобальное переключение (имя клавиши gohook, напр. `F12`) |
| `ui.theme` | `dark` или `light` |
| `ui.main_color` | HEX-акцент орба |
| `ui.show_result` | Если `true` — копировать распознанный текст в буфер обмена |

## Переменные окружения

| Переменная | Назначение |
|------------|------------|
| `VOICE_*` | Переопределения для YAML (см. выше) |
| `WHISPER_MODEL` | Путь к модели (поверх `model.path`) |
| `WHISPER_LANG` | Язык whisper (поверх `model.lang`) |
| `WHISPER_USE_GPU` | `0` / `false` — только CPU для контекста whisper |
| `VOICE_ECHO_STUB` | Непустое значение — не вызывать whisper, вернуть тестовую строку (отладка цепочки) |

## Лицензия

Исходники приложения — **Apache License 2.0** (файл `LICENSE`). Зависимости и сабмодуль **whisper.cpp** имеют свои лицензии — см. `third_party/whisper`.
