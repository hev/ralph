package ralph

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hev/ralph/internal/config"
)

// HTTP client with timeout
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

const (
	soundsCacheFile = "ralph_sounds.txt"
)

// SoundPlayer handles fetching and playing Ralph Wiggum audio clips
type SoundPlayer struct {
	config config.SoundConfig
	sounds []string
}

// NewSoundPlayer creates a new SoundPlayer with the given config
func NewSoundPlayer(cfg config.SoundConfig) *SoundPlayer {
	return &SoundPlayer{
		config: cfg,
	}
}

// Play fetches a random Ralph quote and plays it asynchronously
func (p *SoundPlayer) Play() error {
	if !p.config.Enabled || p.config.Mute {
		return nil
	}

	// Load sounds if not already loaded (do this synchronously on first call)
	if len(p.sounds) == 0 {
		if err := p.loadSounds(); err != nil {
			return fmt.Errorf("failed to load sounds: %w", err)
		}
	}

	if len(p.sounds) == 0 {
		return fmt.Errorf("no sounds available")
	}

	// Pick a random sound
	rand.Seed(time.Now().UnixNano())
	soundURL := p.sounds[rand.Intn(len(p.sounds))]

	// Play asynchronously so we don't block the loop
	go func() {
		tmpFile, err := p.downloadSound(soundURL)
		if err != nil {
			return
		}
		defer os.Remove(tmpFile)
		p.playSound(tmpFile)
	}()

	return nil
}

// loadSounds loads the sound URLs from cache or fetches them
func (p *SoundPlayer) loadSounds() error {
	// Ensure cache directory exists
	if err := os.MkdirAll(p.config.CacheDir, 0755); err != nil {
		return err
	}

	cacheFile := filepath.Join(p.config.CacheDir, soundsCacheFile)

	// Try to load from cache first
	if data, err := os.ReadFile(cacheFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				p.sounds = append(p.sounds, line)
			}
		}
		if len(p.sounds) > 0 {
			return nil
		}
	}

	// Fetch from the sound page
	if err := p.fetchSounds(); err != nil {
		return err
	}

	// Cache the sounds
	if len(p.sounds) > 0 {
		data := strings.Join(p.sounds, "\n")
		os.WriteFile(cacheFile, []byte(data), 0644)
	}

	return nil
}

// fetchSounds fetches the sound page and extracts MP3 URLs
func (p *SoundPlayer) fetchSounds() error {
	resp, err := httpClient.Get(p.config.PageURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch sound page: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse base URL for resolving relative paths
	baseURL, err := url.Parse(p.config.PageURL)
	if err != nil {
		return err
	}

	// Extract MP3 links using regex
	// Look for href="...mp3" or src="...mp3" patterns
	mp3Pattern := regexp.MustCompile(`(?:href|src)=["']([^"']*\.mp3)["']`)
	matches := mp3Pattern.FindAllStringSubmatch(string(body), -1)

	for _, match := range matches {
		if len(match) > 1 {
			mp3URL := match[1]
			// Resolve relative URLs
			if !strings.HasPrefix(mp3URL, "http://") && !strings.HasPrefix(mp3URL, "https://") {
				ref, err := url.Parse(mp3URL)
				if err != nil {
					continue
				}
				mp3URL = baseURL.ResolveReference(ref).String()
			}
			p.sounds = append(p.sounds, mp3URL)
		}
	}

	return nil
}

// downloadSound downloads a sound file to a temporary location
func (p *SoundPlayer) downloadSound(soundURL string) (string, error) {
	resp, err := httpClient.Get(soundURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download sound: %s", resp.Status)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "ralph_*.mp3")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

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

// ClearCache removes the cached sound URLs
func (p *SoundPlayer) ClearCache() error {
	cacheFile := filepath.Join(p.config.CacheDir, soundsCacheFile)
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	p.sounds = nil
	return nil
}
