/*
Copyright © 2026 VaaVaa

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"github.com/spf13/cobra"

	"lnx-voice-in-go/internal/audio"
	"lnx-voice-in-go/internal/engine"
	"lnx-voice-in-go/internal/input"
	"lnx-voice-in-go/internal/ui"
)

var whisperSelftestCmd = &cobra.Command{
	Use:   "whisper-selftest",
	Short: "Whisper self-test: print system info and load the model",
	Run: func(cmd *cobra.Command, args []string) {
		engine.CheckCUDA()
	},
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lnx-voice-in-go",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		rec, err := audio.NewRecorder()
		if err != nil {
			fmt.Fprintf(os.Stderr, "microphone unavailable: %v\n", err)
			os.Exit(1)
		}

		vis := ui.NewOverlay()
		const maxRecord = 60 * time.Second

		var sess struct {
			mu        sync.Mutex
			recording bool
			start     time.Time
		}

		var toggleRecording func()
		toggleRecording = func() {
			sess.mu.Lock()
			active := sess.recording
			sess.mu.Unlock()

			if !active {
				sess.mu.Lock()
				sess.recording = true
				sess.start = time.Now()
				sess.mu.Unlock()

				rec.BeginSession()
				vis.SetRecording(true)
				fmt.Println("Recording started...")
			} else {
				sess.mu.Lock()
				sess.recording = false
				sess.mu.Unlock()

				vis.SetRecording(false)
				fmt.Println("Processing...")

				go func() {
					samples := rec.EndSession()
					text := engine.Process(samples)

					if len(samples) == 0 {
						fmt.Fprintln(os.Stderr, "voice: buffer has 0 samples — recording too short or microphone not capturing.")
						return
					}
					if strings.HasPrefix(text, "Whisper error:") {
						fmt.Fprintln(os.Stderr, "voice:", text)
						return
					}
					if text == "" {
						fmt.Fprintf(os.Stderr, "voice: whisper returned empty text (~%.2f s PCM). Language: see WHISPER_LANG.\n", float64(len(samples))/16000)
						return
					}

					fmt.Fprintf(os.Stderr, "voice: transcribed %q\n", text)
					vis.SetClipboardRecognized(text)
					input.Type(text)
				}()
			}
		}

		vis.SetOnRecordToggle(toggleRecording)

		uiTicker := time.NewTicker(30 * time.Millisecond)
		go func() {
			defer uiTicker.Stop()
			for range uiTicker.C {
				sess.mu.Lock()
				active := sess.recording
				start := sess.start
				sess.mu.Unlock()

				if active {
					elapsed := time.Since(start)
					if elapsed >= maxRecord {
						fyne.Do(toggleRecording)
						continue
					}
					secLeft := maxRecord.Seconds() - elapsed.Seconds()
					lv := rec.GetRMS()
					sl := secLeft
					fyne.Do(func() {
						vis.UpdateState(lv, sl)
					})
				} else {
					fyne.Do(func() {
						vis.UpdateState(0, 0)
					})
				}
			}
		}()

		vis.RunApp()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(whisperSelftestCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.lnx-voice-in-go.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
