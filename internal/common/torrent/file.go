package torrent

import (
	"bufio"
	"crypto/sha1"
	"io"
	"log/slog"
	"os"

	"github.com/jackpal/bencode-go"
)

func Open(filePath string) (*TorrentFile, error) {
	// open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// decode bencode
	var btf torrentBencode
	err = bencode.Unmarshal(file, &btf)
	if err != nil {
		return nil, err
	}

	// convert to TorrentFile
	return btf.toTorrentFile()
}

func (t *TorrentFile) Save(filePath string) error {
	// open file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// encode bencode
	err = bencode.Marshal(file, t.toTorrentBencode())
	if err != nil {
		return err
	}

	return nil
}

func ReadPiecesFromFile(filePath string, pieceLength int) ([][]byte, error) {
    // open file
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    fileStat, err := file.Stat()
    if err != nil {
        return nil, err
    }

    r := bufio.NewReader(file)

    numHashes := int(fileStat.Size()) / pieceLength + 1
    pieceHashes := make([][]byte, numHashes)

    for i := 0; i < numHashes; i++ {
        begin := i * pieceLength
        end := min((i+1)*pieceLength, int(fileStat.Size()))

        buf := make([]byte, end-begin)
        // read piece length byte
        _, err := r.Read(buf[:])
        if err != nil {
            return nil, err
        }
        pieceHashes[i] = buf
    }

    return pieceHashes, nil

}

func GenerateTorrentFromSingleFile(filePath, trackerUrl string, pieceLength int) (*TorrentFile, error) {
	// open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileStat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	r := bufio.NewReader(file)

    numHashes := int(fileStat.Size()) / pieceLength + 1
	pieceHashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		begin := i * pieceLength
		end := min((i+1)*pieceLength, int(fileStat.Size()))

        buf := make([]byte, end - begin)
		// read piece length byte
		_, err := r.Read(buf[:])
		if err != nil {
			return nil, err
		}
        slog.Info("Read piece", "bytes" , buf[:], "index", i)

		pieceHash := sha1.Sum(buf)
		pieceHashes[i] = pieceHash
        slog.Info("Generated piece hash", "index", i, "hash", pieceHash)
	}

	pieceHashesString := make([]byte, 0, len(pieceHashes)*sha1.Size)
	for _, hash := range pieceHashes {
		pieceHashesString = append(pieceHashesString, hash[:]...)
	}

	bt := torrentBencode{
		Announce: trackerUrl,
		Info: torrentBencodeInfo{
			Pieces:      string(pieceHashesString),
			PieceLength: pieceLength,
			Length:      int(fileStat.Size()),
			Name:        fileStat.Name(),
		},
	}
	return bt.toTorrentFile()
}

func OpenFromReader(r io.Reader) (*TorrentFile, error) {
	// decode bencode
	var btf torrentBencode
	err := bencode.Unmarshal(r, &btf)
	if err != nil {
		return nil, err
	}

	return btf.toTorrentFile()
}

func (t *TorrentFile) Write(w io.Writer) error {
    return bencode.Marshal(w, t.toTorrentBencode())
}
