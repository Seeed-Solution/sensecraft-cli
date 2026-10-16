// Package uploadguard enforces SenseCraft upload size and rate limits on the
// CLI client side (service + HTTP interface layers). Server-side limits still
// apply; this package stops abusive CLI / agent traffic before acquire_file_url.
package uploadguard

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sensecraft-cloud-terminal/internal/config"
)

// Business caps aligned with Console upload.vue / infoForm / model/add.vue.
const (
	MaxAppImageBytes       int64 = 5 * 1024 * 1024
	MaxModelImageBytes     int64 = 2 * 1024 * 1024
	MaxNonImageUploadBytes int64 = 100 * 1024 * 1024
)

// Rate limits: stop burst OSS upload abuse while allowing a full
// app images[] batch (max 16) with retries.
const (
	BurstWindow   = 10 * time.Second
	BurstMax      = 8
	SustainWindow = time.Minute
	SustainMax    = 30
)

var (
	ErrEmptyUpload = errors.New("empty file uploads are not allowed (0 bytes)")
	ErrRateLimited = errors.New("upload rate limit exceeded; wait and retry (max 8 per 10s, 30 per minute per machine)")
)

var (
	rateMu     sync.Mutex
	ratePath   string
	rateLoaded bool
)

// SetStatePathForTest overrides the on-disk rate ledger (tests only).
func SetStatePathForTest(path string) {
	rateMu.Lock()
	defer rateMu.Unlock()
	ratePath = path
	rateLoaded = true
}

func resolveStatePath() (string, error) {
	if rateLoaded && ratePath != "" {
		return ratePath, nil
	}
	cfg, err := config.ResolveConfigPath()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(cfg)
	return filepath.Join(dir, "sensecraft-upload-rate.json"), nil
}

type rateLedger struct {
	Events []int64 `json:"events"` // unix milliseconds
}

// AllowUpload records one upload attempt. identity is optional (key id / profile);
// empty uses a machine-wide bucket. Sandbox callers should skip this.
func AllowUpload(identity string) error {
	rateMu.Lock()
	defer rateMu.Unlock()

	path, err := resolveStatePath()
	if err != nil {
		// Fail closed on path errors only when we cannot locate config dir;
		// fall back to in-memory-only by using temp — prefer fail open with
		// memory bucket keyed by process if config missing.
		path = filepath.Join(os.TempDir(), "sensecraft-upload-rate.json")
	}
	_ = identity // reserved for per-key buckets; machine-wide for now

	now := time.Now()
	led, _ := loadLedger(path)
	cutoff := now.Add(-SustainWindow).UnixMilli()
	kept := led.Events[:0]
	for _, ts := range led.Events {
		if ts >= cutoff {
			kept = append(kept, ts)
		}
	}
	led.Events = kept

	burstCutoff := now.Add(-BurstWindow).UnixMilli()
	burst := 0
	for _, ts := range led.Events {
		if ts >= burstCutoff {
			burst++
		}
	}
	if burst >= BurstMax {
		return fmt.Errorf("%w (burst %d/%d in %s)", ErrRateLimited, burst, BurstMax, BurstWindow)
	}
	if len(led.Events) >= SustainMax {
		return fmt.Errorf("%w (sustained %d/%d in %s)", ErrRateLimited, len(led.Events), SustainMax, SustainWindow)
	}

	led.Events = append(led.Events, now.UnixMilli())
	if err := saveLedger(path, led); err != nil {
		// Still allow if we cannot persist — memory already counted this turn
		// via led; next process may under-count. Prefer continuing with best effort.
		_ = err
	}
	return nil
}

func loadLedger(path string) (rateLedger, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return rateLedger{}, err
	}
	var led rateLedger
	if err := json.Unmarshal(b, &led); err != nil {
		return rateLedger{}, err
	}
	return led, nil
}

func saveLedger(path string, led rateLedger) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(led)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ValidateOSSUploadSize mirrors Console upload.vue size gates.
// fileType: 1=image, 2–6=non-image. forAppImage selects 5MB vs 2MB for type 1.
func ValidateOSSUploadSize(fileType int, size int64, forAppImage bool) error {
	if size <= 0 {
		return ErrEmptyUpload
	}
	max := MaxNonImageUploadBytes
	if fileType == 1 {
		if forAppImage {
			max = MaxAppImageBytes
		} else {
			max = MaxModelImageBytes
		}
	}
	if size > max {
		if fileType == 1 {
			if forAppImage {
				return fmt.Errorf("app image exceeds Console limit of 5 MB (%d bytes); got %d bytes", max, size)
			}
			return fmt.Errorf("image exceeds Console limit of 2 MB (%d bytes); got %d bytes (model pic_url); use --for-app for app images up to 5 MB", max, size)
		}
		return fmt.Errorf("file exceeds Console limit of 100 MB (%d bytes); got %d bytes (file_type=%d)", max, size, fileType)
	}
	return nil
}
