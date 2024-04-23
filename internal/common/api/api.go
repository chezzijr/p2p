package api

import (
	"time"
)

type AnnounceRequest struct {
	InfoHash   string `query:"info_hash"`
	PeerID     string `query:"peer_id"`
	Port       uint16 `query:"port"`
	Uploaded   int    `query:"uploaded"`
	Downloaded int    `query:"downloaded"`
	Left       int    `query:"left"`
	Compact    int    `query:"compact"`
}

type AnnounceResponse struct {
	Interval time.Duration `bencode:"interval"`
	Peers    []byte        `bencode:"peers"`
}
