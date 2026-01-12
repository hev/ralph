package ralph

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hev/ralph/internal/config"
)

//go:embed sounds/*
var soundFiles embed.FS

// SoundType represents the type of sound event
type SoundType string

const (
	SoundSessionStart    SoundType = "session-start.wav"
	SoundIterationFinish SoundType = "iteration-finish.mp3"
	SoundTodoComplete    SoundType = "todo-complete.mp3"
)

// SoundPlayer handles playing Ralph Wiggum audio clips
type SoundPlayer struct {
	config   config.SoundConfig
	cacheDir string
}

// NewSoundPlayer creates a new SoundPlayer with the given config
func NewSoundPlayer(cfg config.SoundConfig) *SoundPlayer {
	return &SoundPlayer{
		config:   cfg,
		cacheDir: cfg.CacheDir,
	}
}

// PlaySessionStart plays the session start sound (synchronous - blocks until done)
func (p *SoundPlayer) PlaySessionStart() error {
	return p.playHookSoundSync(SoundSessionStart)
}

// PlayIterationFinish plays the iteration finish sound (asynchronous)
func (p *SoundPlayer) PlayIterationFinish() error {
	return p.playHookSoundAsync(SoundIterationFinish)
}

// PlayTodoComplete plays the todo list complete sound (synchronous - blocks until done)
func (p *SoundPlayer) PlayTodoComplete() error {
	return p.playHookSoundSync(SoundTodoComplete)
}

// Play plays the iteration finish sound (for backwards compatibility)
func (p *SoundPlayer) Play() error {
	return p.PlayIterationFinish()
}

// playHookSoundSync plays a specific embedded sound file synchronously (blocks until done)
func (p *SoundPlayer) playHookSoundSync(soundType SoundType) error {
	if !p.config.Enabled || p.config.Mute {
		return nil
	}

	tmpFile, err := p.extractSound(soundType)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	return p.playSound(tmpFile)
}

// playHookSoundAsync plays a specific embedded sound file asynchronously
func (p *SoundPlayer) playHookSoundAsync(soundType SoundType) error {
	if !p.config.Enabled || p.config.Mute {
		return nil
	}

	tmpFile, err := p.extractSound(soundType)
	if err != nil {
		return err
	}

	go func() {
		defer os.Remove(tmpFile)
		p.playSound(tmpFile)
	}()

	return nil
}

// extractSound extracts an embedded sound to a temp file and returns the path
func (p *SoundPlayer) extractSound(soundType SoundType) (string, error) {
	soundPath := "sounds/" + string(soundType)
	data, err := soundFiles.ReadFile(soundPath)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded sound %s: %w", soundType, err)
	}

	ext := filepath.Ext(string(soundType))
	tmpFile, err := os.CreateTemp("", "ralph_*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write sound data: %w", err)
	}
	tmpFile.Close()

	return tmpFile.Name(), nil
}

// playSound plays the sound file using an available player
func (p *SoundPlayer) playSound(filePath string) error {
	// List of players to try in order of preference
	// afplay is macOS native, ffplay is cross-platform, mpg123/mpg321 are common on Linux
	players := []struct {
		name string
		args []string
	}{
		{"afplay", []string{filePath}},
		{"ffplay", []string{"-nodisp", "-autoexit", "-loglevel", "quiet", filePath}},
		{"mpg123", []string{"-q", filePath}},
		{"mpg321", []string{"-q", filePath}},
	}

	// If a preferred player is set, try it first
	if p.config.Player != "" {
		for i, player := range players {
			if player.name == p.config.Player {
				// Move preferred player to front
				players = append([]struct {
					name string
					args []string
				}{player}, append(players[:i], players[i+1:]...)...)
				break
			}
		}
	}

	// Try each player until one works
	var lastErr error
	for _, player := range players {
		if path, err := exec.LookPath(player.name); err == nil {
			cmd := exec.Command(path, player.args...)
			cmd.Stdout = nil
			cmd.Stderr = nil
			if err := cmd.Run(); err != nil {
				lastErr = err
				continue
			}
			return nil
		}
	}

	if lastErr != nil {
		return fmt.Errorf("all players failed, last error: %w", lastErr)
	}
	return fmt.Errorf("no audio player found (tried: afplay, ffplay, mpg123, mpg321)")
}

// ClearCache is kept for backwards compatibility but is now a no-op
func (p *SoundPlayer) ClearCache() error {
	return nil
}
