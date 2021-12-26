package deckard

import (
	"io/ioutil"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	CodeFolder string                   `toml:"code_folder"`
	Projects   map[string]ConfigProject `toml:"project"`
}

type ConfigProject struct {
	Icon string `toml:"icon"`
	Repo string `toml:"repo"`
}

// TODO Create config with project data and render them on the screen in the top bar
func LoadConfig() (*Config, error) {
	bytes, err := ioutil.ReadFile("config.toml")
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = toml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
