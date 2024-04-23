package torrent

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"github.com/jackpal/bencode-go"
)

var (
	ErrMalformedPieces = errors.New("Received malformed pieces")
)

type TorrentFile struct {
	Announce    string
	InfoHash    [sha1.Size]byte
	PieceHashes [][sha1.Size]byte
	PieceLength int
	Length      int
	Name        string
}

type torrentBencode struct {
	Announce string             `bencode:"announce"`
	Info     torrentBencodeInfo `bencode:"info"`
}

type torrentBencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

func (info *torrentBencodeInfo) hash() ([sha1.Size]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *info)
	if err != nil {
		return [sha1.Size]byte{}, err
	}
	return sha1.Sum(buf.Bytes()), nil
}

func (info *torrentBencodeInfo) splitPieceHashes() ([][sha1.Size]byte, error) {
	hashLen := sha1.Size // default length of sha1 hash in bytes
	buf := []byte(info.Pieces)
	if len(buf)%hashLen != 0 {
		return nil, ErrMalformedPieces
	}
	numHashes := len(buf) / hashLen
	hashes := make([][sha1.Size]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

func (btf *torrentBencode) toTorrentFile() (*TorrentFile, error) {
	infoHash, err := btf.Info.hash()
	if err != nil {
		return nil, err
	}
	pieceHashes, err := btf.Info.splitPieceHashes()
	if err != nil {
		return nil, err
	}
	return &TorrentFile{
		Announce:    btf.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: btf.Info.PieceLength,
		Length:      btf.Info.Length,
		Name:        btf.Info.Name,
	}, nil
}

func (t *TorrentFile) toTorrentBencode() torrentBencode {
	hashBytes := make([]byte, len(t.PieceHashes)*20)
	for i, pieceHash := range t.PieceHashes {
		copy(hashBytes[i*20:], pieceHash[:])
	}

	return torrentBencode{
		Announce: t.Announce,
		Info: torrentBencodeInfo{
			Pieces:      string(hashBytes),
			PieceLength: t.PieceLength,
			Length:      t.Length,
			Name:        t.Name,
		},
	}
}
