package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
)

const Version = "1.0.0"

const (
	ServerAddr  = "87.228.5.28:7777"
	PackBaseURL = "https://raw.githubusercontent.com/konstms/blagrp-launcher/pack/"
	SampURL     = "https://sa-mp.mp/downloads/"
	AppTitle    = "Благовещенск RP"
)

// Размеры gta_sa.exe версии 1.0 US (оригинал и распространённая «компактная» сборка).
var goodExeSizes = map[int64]bool{14383616: true, 14405632: true}

type Config struct {
	Nick string `json:"nick"`
	Game string `json:"game"`
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "BlagRP", "launcher.json")
}

func loadConfig() *Config {
	c := &Config{}
	if b, err := os.ReadFile(configPath()); err == nil {
		_ = json.Unmarshal(b, c)
	}
	return c
}

func saveConfig(c *Config) {
	p := configPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	b, _ := json.MarshalIndent(c, "", " ")
	_ = os.WriteFile(p, b, 0o644)
}

var nickRe = regexp.MustCompile(`^[A-Za-z0-9_\[\]$@=().]{3,20}$`)
var rpNickRe = regexp.MustCompile(`^[A-Z][a-z]+_[A-Z][a-z]+$`)

type GameCheck int

const (
	GameOK GameCheck = iota
	GameMissing
	GameBadVersion
	GameNoSamp
)

func checkGame(dir string) GameCheck {
	if dir == "" {
		return GameMissing
	}
	fi, err := os.Stat(filepath.Join(dir, "gta_sa.exe"))
	if err != nil {
		return GameMissing
	}
	if !goodExeSizes[fi.Size()] {
		return GameBadVersion
	}
	if _, err := os.Stat(filepath.Join(dir, "samp.dll")); err != nil {
		return GameNoSamp
	}
	if _, err := os.Stat(filepath.Join(dir, "samp.exe")); err != nil {
		return GameNoSamp
	}
	return GameOK
}

func statePath(game string) string { return filepath.Join(game, "blagrp_state.json") }
