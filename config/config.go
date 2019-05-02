package config

import (
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

var (
	TiDiffPath        string
	TiDiffHistoryPath string
	TiDiffConfigPath  string
)

func init() {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	TiDiffPath = filepath.Join(home, ".config/tidiff")
	TiDiffConfigPath = filepath.Join(TiDiffPath, "config")
	TiDiffHistoryPath = filepath.Join(TiDiffPath, "history")
}
