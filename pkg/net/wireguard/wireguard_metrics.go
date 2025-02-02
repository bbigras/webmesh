/*
Copyright 2023 Avi Zimmerman <avi.zimmerman@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package wireguard

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	v1 "github.com/webmeshproj/api/v1"
	"golang.org/x/exp/slog"
)

// Peer Metrics
var (
	// BytesSentTotal tracks bytes sent over a wireguard interface
	BytesSentTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "webmesh",
		Name:      "wireguard_bytes_sent_total",
		Help:      "Total bytes sent over the wireguard interface.",
	}, []string{"node_id"})

	// BytesRecvdTotal tracks bytes received over a wireguard interface.
	BytesRecvdTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "webmesh",
		Name:      "wireguard_bytes_rcvd_total",
		Help:      "Total bytes received over the wireguard interface.",
	}, []string{"node_id"})

	// ConnectedPeers tracks the remote peers on a wireguard interface.
	ConnectedPeers = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "webmesh",
		Name:      "wireguard_connected_peers",
		Help:      "The current number of wireguard peers.",
	}, []string{"node_id", "peer"})

	// PeerBytesSentTotal tracks bytes sent over a wireguard interface
	// to a specific peer.
	PeerBytesSentTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "webmesh",
		Name:      "wireguard_peer_bytes_sent_total",
		Help:      "Total bytes sent over the wireguard interface by peer.",
	}, []string{"node_id", "peer"})

	// PeerBytesRecvdTotal tracks bytes received over a wireguard interface
	// from a specific peer.
	PeerBytesRecvdTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "webmesh",
		Name:      "wireguard_peer_bytes_rcvd_total",
		Help:      "Total bytes received over the wireguard interface by peer.",
	}, []string{"node_id", "peer"})
)

// Metrics returns the metrics for the wireguard interface.
func (w *wginterface) Metrics() (*v1.InterfaceMetrics, error) {
	device, err := w.cli.Device(w.Name())
	if err != nil {
		return nil, err
	}
	metrics := &v1.InterfaceMetrics{
		DeviceName:         device.Name,
		PublicKey:          device.PublicKey.String(),
		AddressV4:          w.Interface.AddressV4().String(),
		AddressV6:          w.Interface.AddressV6().String(),
		Type:               device.Type.String(),
		ListenPort:         int32(device.ListenPort),
		TotalReceiveBytes:  0,
		TotalTransmitBytes: 0,
		NumPeers:           int32(len(device.Peers)),
		Peers:              make([]*v1.PeerMetrics, len(device.Peers)),
	}
	for i, peer := range device.Peers {
		metrics.TotalReceiveBytes += uint64(peer.ReceiveBytes)
		metrics.TotalTransmitBytes += uint64(peer.TransmitBytes)
		metrics.Peers[i] = &v1.PeerMetrics{
			PublicKey:           peer.PublicKey.String(),
			Endpoint:            peer.Endpoint.String(),
			PersistentKeepAlive: peer.PersistentKeepaliveInterval.String(),
			LastHandshakeTime:   peer.LastHandshakeTime.UTC().Format(time.RFC3339),
			AllowedIps: func() []string {
				var ips []string
				for _, ip := range peer.AllowedIPs {
					ips = append(ips, ip.String())
				}
				return ips
			}(),
			ProtocolVersion: int64(peer.ProtocolVersion),
			ReceiveBytes:    uint64(peer.ReceiveBytes),
			TransmitBytes:   uint64(peer.TransmitBytes),
		}
	}
	return metrics, nil
}

// MetricsRecorder records metrics for a wireguard interface.
type MetricsRecorder struct {
	wg        *wginterface
	totalSent uint64
	totalRcvd uint64
	connected map[string]struct{}
	peerSent  map[string]uint64
	peerRcvd  map[string]uint64
	mux       sync.Mutex
	log       *slog.Logger
}

// NewMetricsRecorder returns a new MetricsRecorder.
func NewMetricsRecorder(wg Interface) *MetricsRecorder {
	return &MetricsRecorder{
		wg:        wg.(*wginterface),
		connected: make(map[string]struct{}),
		peerSent:  make(map[string]uint64),
		peerRcvd:  make(map[string]uint64),
		log:       slog.Default().With("component", "wireguard-metrics"),
	}
}

// Run starts the metrics recorder.
func (m *MetricsRecorder) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.log.Debug("updating interface metrics")
			if err := m.updateMetrics(); err != nil {
				m.log.Error("update metrics", slog.String("error", err.Error()))
			}
		}
	}
}

// updateMetrics updates the prometheus metrics for the wireguard interface.
func (m *MetricsRecorder) updateMetrics() error {
	m.mux.Lock()
	defer m.mux.Unlock()

	nodeID := m.wg.opts.NodeID
	metrics, err := m.wg.Metrics()
	if err != nil {
		return fmt.Errorf("get metrics: %w", err)
	}

	// Update global metrics.
	var sentDiff, rcvdDiff uint64
	if m.totalSent > 0 {
		sentDiff = metrics.TotalTransmitBytes - m.totalSent
	} else {
		sentDiff = metrics.TotalTransmitBytes
	}
	if m.totalRcvd > 0 {
		rcvdDiff = metrics.TotalReceiveBytes - m.totalRcvd
	} else {
		rcvdDiff = metrics.TotalReceiveBytes
	}
	BytesSentTotal.WithLabelValues(nodeID).Add(float64(sentDiff))
	BytesRecvdTotal.WithLabelValues(nodeID).Add(float64(rcvdDiff))

	// Update peer metrics.
	seen := make(map[string]struct{})
	for _, peer := range metrics.GetPeers() {
		peerID, ok := m.wg.peerByPublicKey(peer.PublicKey)
		if !ok {
			continue
		}
		seen[peerID] = struct{}{}
		m.connected[peerID] = struct{}{}
		// Set the peer as connected.
		ConnectedPeers.WithLabelValues(nodeID, peerID).Set(1)
		// Update the peer metrics.
		var sentDiff, rcvdDiff uint64
		if m.peerSent[peerID] > 0 {
			sentDiff = peer.TransmitBytes - m.peerSent[peerID]
		} else {
			sentDiff = peer.TransmitBytes
		}
		if m.peerRcvd[peerID] > 0 {
			rcvdDiff = peer.ReceiveBytes - m.peerRcvd[peerID]
		} else {
			rcvdDiff = peer.ReceiveBytes
		}
		m.peerSent[peerID] = peer.TransmitBytes
		m.peerRcvd[peerID] = peer.ReceiveBytes
		PeerBytesSentTotal.WithLabelValues(nodeID, peerID).Add(float64(sentDiff))
		PeerBytesRecvdTotal.WithLabelValues(nodeID, peerID).Add(float64(rcvdDiff))
	}

	// Decrement the connected peers that are no longer connected.
	for peerID := range m.connected {
		if _, ok := seen[peerID]; !ok {
			ConnectedPeers.WithLabelValues(nodeID, peerID).Set(0)
			delete(m.connected, peerID)
		}
	}
	return nil
}
