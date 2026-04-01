# Настройки путей
WHISPER_PATH = $(shell pwd)/third_party/whisper
BUILD_PATH = $(WHISPER_PATH)/build
BINARY_NAME = lnx-voice-in-go

.PHONY: all build-whisper build-go clean deps-apt

all: build-whisper build-go

# Заголовки для CGO: gohook (xkbcommon, Xlib-xcb), glfw (Xrandr, …). Выполните: make deps-apt
deps-apt:
	sudo apt-get install -y libxkbcommon-dev libxkbcommon-x11-dev libx11-xcb-dev libxrandr-dev libxcursor-dev libx11-dev libgl1-mesa-dev libxi-dev libxinerama-dev libxxf86vm-dev pkg-config xorg-dev

# 1. Сборка ядра Whisper с поддержкой CUDA
build-whisper:
	@echo "--- Сборка Whisper.cpp с CUDA ---"
	cmake -S $(WHISPER_PATH) -B $(BUILD_PATH) \
		-DGGML_CUDA=ON \
		-DBUILD_SHARED_LIBS=OFF \
		-DCMAKE_CUDA_ARCHITECTURES=89
	cmake --build $(BUILD_PATH) --config Release --target whisper

# 2. Сборка вашего Go-приложения
build-go:
	@echo "--- Сборка Go-сервиса ---"
	# Добавляем пути к CUDA библиотекам Ubuntu в LDFLAGS
	export CGO_LDFLAGS="-L/usr/local/cuda/lib64 -L$(BUILD_PATH)/src -L$(BUILD_PATH)/ggml/src -L$(BUILD_PATH)/ggml/src/ggml-cuda" && \
	go build -o $(BINARY_NAME) main.go

clean:
	rm -rf $(BUILD_PATH)
	rm -f $(BINARY_NAME)