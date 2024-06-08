package peer

import (
	"encoding/json"
	"os"

	"github.com/chezzijr/p2p/internal/common/connection"
)

// Used to keep track of the downloading files
// When peer is gracefully shutdown, save the progress of the downloaded files to cache
type CachedFile struct {
    // When peer stop during downloading, some pieces will be missed
    // When peer start again, it will check the cache and download the missing pieces
	Filepath    string `json:"filepath"`
	InfoHash    string `json:"infohash"`

	// Keep track of which pieces have been downloaded
	Bitfield connection.BitField `json:"pieces"`
}

type CachedFiles []*CachedFile
type CachedFilesMap map[string]*CachedFile

func (c CachedFilesMap) SaveCache(path string) error {
	cachedFiles := make(CachedFiles, 0, len(c))
	for _, cachedFile := range c {
		cachedFiles = append(cachedFiles, cachedFile)
	}

	// Open the file at the given path
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(cachedFiles)
}

func LoadCache(path string) (CachedFilesMap, error) {
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

	cachedFilesMap := make(CachedFilesMap)
	for _, cachedFile := range cachedFiles {
		cachedFilesMap[cachedFile.InfoHash] = cachedFile
	}

	return cachedFilesMap, nil
}
