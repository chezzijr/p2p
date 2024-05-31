package api

import (
	"net/url"
	"strconv"
	"time"
)

type AnnounceRequest struct {
	InfoHash   string `query:"info_hash"`
	PeerID     string `query:"peer_id"`
	Port       uint16 `query:"port"`
	Uploaded   int    `query:"uploaded"`
	Downloaded int    `query:"downloaded"`
	Left       int    `query:"left"`
    Event      string `query:"event"`
}

type AnnounceResponse struct {
	Interval time.Duration `bencode:"interval"`
	Peers    string        `bencode:"peers"`
}

func (req *AnnounceRequest) ToUrlValues() url.Values {
    v := url.Values{}
    v.Add("info_hash", req.InfoHash)
    v.Add("peer_id", req.PeerID)
    v.Add("port", strconv.Itoa(int(req.Port)))
    v.Add("uploaded", strconv.Itoa(req.Uploaded))
    v.Add("downloaded", strconv.Itoa(req.Downloaded))
    v.Add("left", strconv.Itoa(req.Left))
    v.Add("event", req.Event)
    return v
}
