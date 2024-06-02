package peer

import (
	"encoding/json"
	"log/slog"
	"os"
	"path"
)

const AppName = "chezzijr-p2p"

type Config struct {
	CachePath        string `json:"cache_path"`
	LogPath          string `json:"log_path"`
	DefaultBlockSize int    `json:"default_block_size"`
}

func LoadConfig() (*Config, error) {
    configDir, err := os.UserConfigDir()
    if err != nil {
        return nil, err
    }
    configPath := path.Join(configDir, AppName, "config.json")

    f, err := os.Open(configPath)
    if err != nil {
        if os.IsNotExist(err) {
            slog.Warn("Config file not found, creating default config")
            return CreateDefaultConfig()
        }
        return nil, err
    }
    defer f.Close()

    config := &Config{}
    err = json.NewDecoder(f).Decode(config)
    if err != nil {
        return nil, err
    }

    return config, nil
}

func CreateDefaultConfig() (*Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
    configPath := path.Join(configDir, AppName, "config.json")

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
    cachePath := path.Join(cacheDir, AppName, "cache.json")

    logPath := path.Join(configDir, "log", "log.txt")

    defaultConfig := &Config{
        CachePath:        cachePath,
        LogPath:          logPath,
        DefaultBlockSize: 1024,
    }

    err = defaultConfig.Save(configPath)
    if err != nil {
        return nil, err
    }

    return defaultConfig, nil
}
func (c *Config) Save(path string) error {
    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()

    return json.NewEncoder(f).Encode(c)
}
