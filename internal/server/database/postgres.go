package database

import (
	"bytes"
	"context"
	"database/sql"
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
	pgDatabase   = os.Getenv("POSTGRES_DB")
	pgPassword   = os.Getenv("POSTGRES_PASSWORD")
	pgUsername   = os.Getenv("POSTGRES_USER")
	pgPort       = os.Getenv("POSTGRES_PORT")
	pgHost       = os.Getenv("POSTGRES_HOST")
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
    stmt, err := s.db.Prepare("SELECT * FROM torrents")
    if err != nil {
        return nil, err
    }

    rows, err := stmt.Query()
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    torrents := make([]*torrent.TorrentFile, 0, limit)
    for rows.Next() {
        var id int
        var buf []byte
        var createdAt time.Time

        err := rows.Scan(&id, &buf, &createdAt)
        if err != nil {
            return nil, err
        }

        t, err := torrent.OpenFromReader(bytes.NewBuffer(buf))
        if err != nil {
            return nil, err
        }

        torrents = append(torrents, t)
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

    // gob.NewDecoder(&buf).Decode(&torrent)
    t, err := torrent.OpenFromReader(&buf)
    if err != nil {
        return nil, err
    }

    return t, nil
}

func (s *postgres) AddTorrent(t *torrent.TorrentFile) error {
    stmt, err := s.db.Prepare("INSERT INTO torrents (file) VALUES ($1)")
    if err != nil {
        return err
    }

    var buf bytes.Buffer
    // gob.NewEncoder(&buf).Encode(torrent)
    err = t.Write(&buf)
    if err != nil {
        return err
    }

    _, err = stmt.Exec(buf.Bytes())
    if err != nil {
        return err
    }

    return nil
}

func (s *postgres) AddBulkTorrents(ts []*torrent.TorrentFile) error {
    for _, torrent := range ts {
        err := s.AddTorrent(torrent)
        if err != nil {
            return err
        }
    }

    return nil
}
