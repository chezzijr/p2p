package database

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/chezzijr/p2p/internal/common/torrent"
)

func (s *service) GetRecentTorrents(limit, offset int) ([]*torrent.TorrentFile, error) {
    stmt, err := s.db.Prepare("SELECT id, file, created_at FROM torrents ORDER BY created_at DESC LIMIT $1 OFFSET $2")
    if err != nil {
        return nil, err
    }

    rows, err := stmt.Query(limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var torrents []*torrent.TorrentFile
    for rows.Next() {
        var id int
        var buf bytes.Buffer
        var createdAt time.Time

        err := rows.Scan(&id, &buf, &createdAt)
        if err != nil {
            return nil, err
        }
        var torrent torrent.TorrentFile
        gob.NewDecoder(&buf).Decode(&torrent)
        torrents = append(torrents, &torrent)
    }

    return torrents, nil
}

func (s *service) GetTorrentByID(id int) (*torrent.TorrentFile, error) {
    stmt, err := s.db.Prepare("SELECT file FROM torrents WHERE id = $1")
    if err != nil {
        return nil, err
    }

    var buf bytes.Buffer
    err = stmt.QueryRow(id).Scan(&buf)
    if err != nil {
        return nil, err
    }

    var torrent torrent.TorrentFile
    gob.NewDecoder(&buf).Decode(&torrent)

    return &torrent, nil
}

func (s *service) AddTorrent(torrent *torrent.TorrentFile) error {
    stmt, err := s.db.Prepare("INSERT INTO torrents (file) VALUES ($1)")
    if err != nil {
        return err
    }

    var buf bytes.Buffer
    gob.NewEncoder(&buf).Encode(torrent)

    _, err = stmt.Exec(&buf)
    if err != nil {
        return err
    }

    return nil
}

func (s *service) AddBulkTorrents(torrents []*torrent.TorrentFile) error {
    stmt, err := s.db.Prepare("INSERT INTO torrents (file) VALUES ($1)")
    if err != nil {
        return err
    }

    for _, torrent := range torrents {
        var buf bytes.Buffer
        gob.NewEncoder(&buf).Encode(torrent)

        _, err = stmt.Exec(&buf)
        if err != nil {
            return err
        }
    }

    return nil
}
