package userconfig

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/asdfgasfhsn/sunbeams/internal/config"
)

// LoadWithOverride returns the embedded defaults, with fields from the
// user TOML file (if present) overwriting defaults. Slices (devices, modes,
// standard_timings) are replaced wholesale when present in the override.
// Missing override file is not an error.
func LoadWithOverride(path string) (*config.Config, error) {
	cfg, err := config.LoadDefaults()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var override config.Config
	if _, err := toml.Decode(string(data), &override); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	mergeInPlace(cfg, &override)
	return cfg, nil
}

// mergeInPlace merges src over dst. For EDID numeric fields, a zero value in src
// is treated as "not set" and preserves the default. To override an EDID numeric
// field to 0, edit defaults.toml and rebuild — user overrides cannot set EDID
// fields to zero via this merge.
func mergeInPlace(dst, src *config.Config) {
	// EDID struct: field-by-field, non-zero wins
	dv := reflect.ValueOf(&dst.EDID).Elem()
	sv := reflect.ValueOf(&src.EDID).Elem()
	for i := 0; i < dv.NumField(); i++ {
		sf := sv.Field(i)
		if !sf.IsZero() {
			dv.Field(i).Set(sf)
		}
	}
	// CTA: only replace if non-empty
	if len(src.CTA.VICCodes) > 0 {
		dst.CTA.VICCodes = src.CTA.VICCodes
	}
	// Slices: replace wholesale if present
	if len(src.StandardTimings) > 0 {
		dst.StandardTimings = src.StandardTimings
	}
	if len(src.Devices) > 0 {
		dst.Devices = src.Devices
	}
	if len(src.Modes) > 0 {
		dst.Modes = src.Modes
	}
}
