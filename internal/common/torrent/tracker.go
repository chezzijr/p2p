package torrent

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"p2p/internal/common/api"
	"p2p/internal/common/peers"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
)


func (t *TorrentFile) buildTrackerUrl(peerID [20]byte, port uint16) (string, error) {
    // build tracker url
    base, err := url.Parse(t.Announce)
    if err != nil {
        return "", err
    }
    params := url.Values{
        "info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Length)},
    }
    base.RawQuery = params.Encode()
    return base.String(), nil 
}

func (t *TorrentFile) RequestPeers(peerID [20]byte, port uint16) ([]peers.Peer, error) {
    return []peers.Peer{
        {
            Ip: net.IPv4(127, 0, 0, 1),
            Port: 1357,
        },
    }, nil

    trackerUrl, err := t.buildTrackerUrl(peerID, port)
    if err != nil {
        return nil, err
    }
    client := &http.Client{
        Timeout: 15 * time.Second,
    }

    resp, err := client.Get(trackerUrl)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // read response to string
    respbytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    slog.Debug("Tracker response: %s", string(respbytes))


    var r api.AnnounceResponse
    err = bencode.Unmarshal(resp.Body, &r)
    if err != nil {
        return nil, err
    }

    return peers.Unmarshal([]byte(r.Peers))
}
