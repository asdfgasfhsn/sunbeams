package config

import (
	_ "embed"
	"fmt"

	"github.com/BurntSushi/toml"
)

//go:embed defaults.toml
var defaultsTOML []byte

type EDIDConfig struct {
	ManufacturerID   string `toml:"manufacturer_id"`
	ProductCode      uint16 `toml:"product_code"`
	Serial           uint32 `toml:"serial"`
	Week             byte   `toml:"week"`
	Year             int    `toml:"year"`
	MonitorName      string `toml:"monitor_name"`
	MaxPixelClockMHz int    `toml:"max_pixel_clock_mhz"`
	MinVRate         int    `toml:"min_vrate"`
	MaxVRate         int    `toml:"max_vrate"`
	MinHRate         int    `toml:"min_hrate"`
	MaxHRate         int    `toml:"max_hrate"`
	MaxTMDSMHz       int    `toml:"max_tmds_mhz"`
	MaxFRLRate       int    `toml:"max_frl_rate"`
}

type CTAConfig struct {
	VICCodes []int `toml:"vic_codes"`
	Y420VICs []int `toml:"y420_vics"`
}

type StandardTiming struct {
	Width   int `toml:"width"`
	Height  int `toml:"height"`
	Refresh int `toml:"refresh"`
}

type Device struct {
	Slug       string `toml:"slug"`
	Label      string `toml:"label"`
	Width      int    `toml:"width"`
	Height     int    `toml:"height"`
	MaxRefresh int    `toml:"max_refresh"`
	HDR        bool   `toml:"hdr"`
}

type Mode struct {
	Width       int    `toml:"width"`
	Height      int    `toml:"height"`
	Refresh     int    `toml:"refresh"`
	Description string `toml:"description"`
}

type GamingConfig struct {
	HelperPath     string `toml:"helper_path"`
	ModesCfg       string `toml:"modes_cfg"`
	SafeRevertMode string `toml:"safe_revert_mode"`
}

type Config struct {
	EDID            EDIDConfig       `toml:"edid"`
	CTA             CTAConfig        `toml:"cta"`
	StandardTimings []StandardTiming `toml:"standard_timings"`
	Devices         []Device         `toml:"devices"`
	Modes           []Mode           `toml:"modes"`
	Gaming          GamingConfig     `toml:"gaming"`
}

// LoadDefaults parses the embedded default config.
func LoadDefaults() (*Config, error) {
	var cfg Config
	if err := toml.Unmarshal(defaultsTOML, &cfg); err != nil {
		return nil, fmt.Errorf("parse embedded defaults: %w", err)
	}
	return &cfg, nil
}

// DefaultsTOML returns the embedded default TOML bytes.
func DefaultsTOML() []byte {
	out := make([]byte, len(defaultsTOML))
	copy(out, defaultsTOML)
	return out
}
