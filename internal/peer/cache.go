package peer

import (
	"encoding/json"
	"os"

	"github.com/chezzijr/p2p/internal/common/connection"
)

// This is used for caching files and their corresponding torrent files
// It is also used to keep track of the downloaded files
// When peer is gracefully shutdown, save the progress of the downloaded files to cache
type CachedFile struct {
	Filepath    string `json:"filepath"`
	Torrentpath string `json:"torrentpath"`
	Downloaded  bool   `json:"downloaded"`

	// Keep track of which pieces have been downloaded
	Pieces connection.BitField `json:"pieces"`
}

type CachedFiles []*CachedFile

func (c CachedFiles) SaveCache(path string) error {
	// Open the file at the given path
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(c)
}

func LoadCache(path string) (CachedFiles, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cachedFiles CachedFiles
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cachedFiles); err != nil {
		return nil, err
	}

	return cachedFiles, nil
}
