package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Model struct {
		Path   string `mapstructure:"path"`
		Type   string `mapstructure:"type"`
		Lang   string `mapstructure:"lang"`
		UseGPU bool   `mapstructure:"use_gpu"`
	} `mapstructure:"model"`

	Audio struct {
		SampleRate     int    `mapstructure:"sample_rate"`
		MaxDurationSec int    `mapstructure:"max_duration_sec"`
		Hotkey         string `mapstructure:"hotkey"`
	} `mapstructure:"audio"`

	UI struct {
		Theme      string `mapstructure:"theme"`
		MainColor  string `mapstructure:"main_color"`
		ShowResult bool   `mapstructure:"show_result"`
	} `mapstructure:"ui"`
}

var AppConfig Config

// LoadConfig reads config YAML (config.yaml / config.yml in search dirs, or path from --config) and VOICE_* env.
// Nested keys in YAML map to VOICE_MODEL_PATH, VOICE_AUDIO_SAMPLE_RATE, etc.
// Legacy: WHISPER_MODEL, WHISPER_LANG, WHISPER_USE_GPU still override model settings after load.
func LoadConfig(explicitFile string) error {
	viper.Reset()

	if explicitFile != "" {
		viper.SetConfigFile(explicitFile)
	} else {
		dirs := []string{"."}
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs, filepath.Join(home, ".voice-input"), filepath.Join(home, ".config", "lnx-voice-in-go"))
		}
		var found string
	outer:
		for _, dir := range dirs {
			for _, base := range []string{"config.yaml", "config.yml"} {
				full := filepath.Join(dir, base)
				if st, err := os.Stat(full); err == nil && !st.IsDir() {
					found = full
					break outer
				}
			}
		}
		if found != "" {
			viper.SetConfigFile(found)
		} else {
			viper.SetConfigName("config")
			viper.SetConfigType("yaml")
			for _, dir := range dirs {
				viper.AddConfigPath(dir)
			}
		}
	}

	viper.SetEnvPrefix("VOICE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("model.path", "models/ggml-small.bin")
	viper.SetDefault("model.type", "small")
	viper.SetDefault("model.lang", "auto")
	viper.SetDefault("model.use_gpu", true)
	viper.SetDefault("audio.sample_rate", 16000)
	viper.SetDefault("audio.max_duration_sec", 60)
	viper.SetDefault("audio.hotkey", "F12")
	viper.SetDefault("ui.theme", "dark")
	viper.SetDefault("ui.main_color", "#0096FF")
	viper.SetDefault("ui.show_result", true)

	if err := viper.ReadInConfig(); err != nil {
		if explicitFile != "" {
			return fmt.Errorf("read config: %w", err)
		}
		var nf viper.ConfigFileNotFoundError
		if !errors.As(err, &nf) {
			return fmt.Errorf("read config: %w", err)
		}
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	// Legacy WHISPER_* env vars override file and VOICE_* after load.
	if v := os.Getenv("WHISPER_MODEL"); v != "" {
		AppConfig.Model.Path = v
	}
	if v := os.Getenv("WHISPER_LANG"); v != "" {
		AppConfig.Model.Lang = v
	}
	switch strings.TrimSpace(os.Getenv("WHISPER_USE_GPU")) {
	case "0", "false", "FALSE":
		AppConfig.Model.UseGPU = false
	case "1", "true", "TRUE":
		AppConfig.Model.UseGPU = true
	}

	if AppConfig.Audio.SampleRate <= 0 {
		AppConfig.Audio.SampleRate = 16000
	}
	if AppConfig.Audio.MaxDurationSec <= 0 {
		AppConfig.Audio.MaxDurationSec = 60
	}
	if AppConfig.Audio.Hotkey == "" {
		AppConfig.Audio.Hotkey = "F12"
	}
	if AppConfig.Model.Lang == "" {
		AppConfig.Model.Lang = "auto"
	}
	if AppConfig.Model.Path == "" {
		AppConfig.Model.Path = "models/ggml-small.bin"
	}

	return nil
}
