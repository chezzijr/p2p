package peer

import (
	"os"
	"path"

	"github.com/spf13/viper"
)

const APPNAME = "chezzijr-p2p"

var (
    configPath string
    cachePath  string
    logPath    string
)

func init() {
    var err error
    configPath, err = os.UserConfigDir()
    if err != nil {
        // use temp dir if user config dir not found
        configPath = os.TempDir()
    }
    configPath = path.Join(configPath, APPNAME)

    cachePath, err = os.UserCacheDir()
    if err != nil {
        // use temp dir if user cache dir not found
        cachePath = os.TempDir()
    }
    cachePath = path.Join(cachePath, APPNAME)

    logPath = path.Join(os.TempDir(), APPNAME)
}

type Config struct {
	CachePath            string
	LogPath              string
	DefaultBlockSize     int    
	SeedOnFileDownloaded bool   
	// This is hard to implement
	SeedOnPieceDownloaded bool
}

func LoadConfig() (*Config, error) {
	viper.AddConfigPath(configPath)
    viper.AddConfigPath(path.Join(os.TempDir(), APPNAME))

	viper.SetConfigName("config")
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            if err := createDefaultConfig(); err != nil {
                return nil, err
            }
        } else {
            return nil, err
        }
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// This will call if config file not found
// So the error of ReadInConfig will not be ConfigFileNotFoundError
func createDefaultConfig() error {
    configFilePath := path.Join(configPath, "config.json")
    cacheFilePath := path.Join(cachePath, "cache.json")
    logFilePath := path.Join(logPath, "log.txt")

    defaultCfg := Config{
        CachePath:            cacheFilePath,
        LogPath:              logFilePath,
        DefaultBlockSize:     1024,
        SeedOnFileDownloaded: true,
        SeedOnPieceDownloaded: false,
    }

    viper.SetDefault("CachePath", defaultCfg.CachePath)
    viper.SetDefault("LogPath", defaultCfg.LogPath)
    viper.SetDefault("DefaultBlockSize", defaultCfg.DefaultBlockSize)
    viper.SetDefault("SeedOnFileDownloaded", defaultCfg.SeedOnFileDownloaded)
    viper.SetDefault("SeedOnPieceDownloaded", defaultCfg.SeedOnPieceDownloaded)

    createFileIfNotExist(configFilePath)

    // write default config
    err := viper.SafeWriteConfig()
    if err != nil {
        return err
    }

    err = viper.ReadInConfig()
    return err
}

func createFileIfNotExist(filePath string) error {
    _, err := os.Stat(filePath)
    if os.IsNotExist(err) {
        err = os.MkdirAll(path.Dir(filePath), os.ModePerm)
        return err
    }
    return err
}
