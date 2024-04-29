package database

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chezzijr/p2p/internal/common/torrent"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

type Postgres interface {
	Health() map[string]string
    GetRecentTorrents(limit, offset int) ([]*torrent.TorrentFile, error)
    GetTorrentByID(id int) (*torrent.TorrentFile, error)
    AddTorrent(torrent *torrent.TorrentFile) error
    AddBulkTorrents(torrents []*torrent.TorrentFile) error
}

type postgres struct {
	db *sql.DB
}

var (
	pgDatabase   = os.Getenv("PG_DATABASE")
	pgPassword   = os.Getenv("PG_PASSWORD")
	pgUsername   = os.Getenv("PG_USERNAME")
	pgPort       = os.Getenv("PG_PORT")
	pgHost       = os.Getenv("PG_HOST")
	pgInstance *postgres
)

func NewPostgres() (Postgres, error) {
	// Reuse Connection
	if pgInstance != nil {
		return pgInstance, nil
	}
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", pgUsername, pgPassword, pgHost, pgPort, pgDatabase)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
        return nil, err
	}
	pgInstance = &postgres{
		db: db,
	}
	return pgInstance, nil
}

func (s *postgres) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := s.db.PingContext(ctx)
	if err != nil {
		log.Fatalf(fmt.Sprintf("db down: %v", err))
	}

	return map[string]string{
		"message": "It's healthy",
	}
}

func (s *postgres) GetRecentTorrents(limit, offset int) ([]*torrent.TorrentFile, error) {
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

func (s *postgres) GetTorrentByID(id int) (*torrent.TorrentFile, error) {
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

func (s *postgres) AddTorrent(torrent *torrent.TorrentFile) error {
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

func (s *postgres) AddBulkTorrents(torrents []*torrent.TorrentFile) error {
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
