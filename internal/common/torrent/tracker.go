package torrent

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/chezzijr/p2p/internal/common/api"
	"github.com/chezzijr/p2p/internal/common/peers"

	"github.com/jackpal/bencode-go"
)


func (t *TorrentFile) buildTrackerUrl(peerID [20]byte, port uint16) (string, error) {
    // build tracker url
    base, err := url.Parse(t.Announce)
    if err != nil {
        return "", err
    }
  //   params := url.Values{
  //       "info_hash":  []string{string(t.InfoHash[:])},
		// "peer_id":    []string{string(peerID[:])},
		// "port":       []string{strconv.Itoa(int(port))},
		// "uploaded":   []string{"0"},
		// "downloaded": []string{"0"},
		// "compact":    []string{"1"},
		// "left":       []string{strconv.Itoa(t.Length)},
  //   }
    params := api.AnnounceRequest{
        InfoHash:  t.InfoHash.String(),
        PeerID:    string(peerID[:]),
        Port:      port,
        Uploaded:  0,
        Downloaded: 0,
        Left:      t.Length,
    }
    base.RawQuery = params.ToUrlValues().Encode()
    return base.String(), nil 
}

func (t *TorrentFile) RequestPeers(peerID [20]byte, port uint16) ([]peers.Peer, error) {
    trackerUrl, err := t.buildTrackerUrl(peerID, port)
    if err != nil {
        return nil, err
    }
    slog.Info("Build tracker url", "url", trackerUrl)
    client := &http.Client{
        Timeout: 15 * time.Second,
    }

    slog.Info("Requesting peers from tracker", "torrent", t.Name)
    resp, err := client.Get(trackerUrl)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Failed to request peers: %s", resp.Status)
    }

    slog.Info("Received response from tracker", "status", resp.Status)
    var r api.AnnounceResponse
    err = bencode.Unmarshal(resp.Body, &r)
    if err != nil {
        return nil, err
    }

    slog.Info("Unmarshalled response", "response", r)

    return peers.Unmarshal([]byte(r.Peers))
}
