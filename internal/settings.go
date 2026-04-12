package internal

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

const (
	StatusOff     = "off"
	StatusText    = "text"
	StatusVerbose = "verbose"
)

const (
	EffortDefault = "default"
	EffortLow     = "low"
	EffortMedium  = "medium"
	EffortHigh    = "high"
	EffortMax     = "max"
)

type Settings struct {
	StatusLevel string `json:"statusLevel,omitempty"`
	Effort      string `json:"effort,omitempty"`
}

func LoadSettings(dataDir string) Settings {
	data, err := os.ReadFile(filepath.Join(dataDir, "settings.json"))
	if err != nil {
		return Settings{StatusLevel: StatusText}
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		log.Printf("error parsing settings: %v", err)
		return Settings{StatusLevel: StatusText}
	}
	if s.StatusLevel != StatusOff && s.StatusLevel != StatusText && s.StatusLevel != StatusVerbose {
		s.StatusLevel = StatusText
	}
	if s.Effort == "" {
		s.Effort = EffortDefault
	} else if s.Effort != EffortDefault && s.Effort != EffortLow && s.Effort != EffortMedium && s.Effort != EffortHigh && s.Effort != EffortMax {
		s.Effort = EffortDefault
	}
	return s
}

func SaveSettings(dataDir string, s Settings) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Printf("error marshaling settings: %v", err)
		return
	}

	tmp, err := os.CreateTemp(dataDir, "settings-*.json")
	if err != nil {
		log.Printf("error creating temp file for settings: %v", err)
		return
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		log.Printf("error writing settings temp file: %v", err)
		return
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		log.Printf("error closing settings temp file: %v", err)
		return
	}

	if err := os.Rename(tmp.Name(), filepath.Join(dataDir, "settings.json")); err != nil {
		os.Remove(tmp.Name())
		log.Printf("error renaming settings file: %v", err)
	}
}
