package peer

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/chezzijr/p2p/internal/common/connection"
	"github.com/chezzijr/p2p/internal/common/utils"
)

// Used to keep track of the downloading files
// When peer is gracefully shutdown, save the progress of the downloaded files to cache
type CachedFile struct {
	// When peer stop during downloading, some pieces will be missed
	// When peer start again, it will check the cache and download the missing pieces
	Filepath string `json:"filepath"`
	InfoHash string `json:"infohash"`

	// Keep track of which pieces have been downloaded
	Bitfield connection.BitField `json:"pieces"`
}

// TODO: also cached the seeding files
// so that next time the peer start, it can continue seeding the files
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

// the path is provided by the config
// so this function creates a file at the given path if it doesn't exist
// instead of creating a file when creating config
// This ensures that the file exists
func LoadCache(path string) (CachedFilesMap, error) {
    // Failed to create, exit
    err := utils.CreateFileIfNotExist(path)
    if err != nil {
        return nil, err
    }

	file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
	defer file.Close()

	var cachedFiles CachedFiles

    // If the file was newly created
    // json decoder will return EOF error
    // when gracefully shutdown, the cache will be saved
    // overwriting the file so we just return an empty map
    err = json.NewDecoder(file).Decode(&cachedFiles)
    if err != nil {
        if errors.Is(err, io.EOF) {
            return make(CachedFilesMap), nil
        } else {
            return nil, err
        }
    }

	cachedFilesMap := make(CachedFilesMap)
	for _, cachedFile := range cachedFiles {
		cachedFilesMap[cachedFile.InfoHash] = cachedFile
	}

	return cachedFilesMap, nil
}
