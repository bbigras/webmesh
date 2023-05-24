/*
Copyright 2023.

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
	"net"
	"net/netip"
	"time"

	"golang.org/x/exp/slog"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// PutPeer updates a peer in the wireguard configuration.
func (w *wginterface) PutPeer(ctx context.Context, peer *Peer) error {
	w.peersMux.Lock()
	defer w.peersMux.Unlock()
	w.log.Debug("put peer", slog.Any("peer", peer))
	key, err := wgtypes.ParseKey(peer.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}
	var keepAlive *time.Duration
	var endpoint *net.UDPAddr
	var allowedIPs []net.IPNet
	if w.opts.PersistentKeepAlive != 0 {
		keepAlive = &w.opts.PersistentKeepAlive
	}
	if peer.PrivateIPv6.IsValid() {
		allowedIPs = append(allowedIPs, net.IPNet{
			IP:   peer.PrivateIPv6.Addr().AsSlice(),
			Mask: net.CIDRMask(peer.PrivateIPv6.Bits(), 128),
		})
	}
	if peer.PrivateIPv4.IsValid() {
		// TODO: We force this to 32 for now, but we should make this configurable
		allowedIPs = append(allowedIPs, net.IPNet{
			IP:   peer.PrivateIPv4.Addr().AsSlice(),
			Mask: net.CIDRMask(32, 32),
		})
	}
	if w.peerConfigs != nil {
		for _, ip := range w.peerConfigs.AllowedIPs(peer.ID) {
			var ipnet net.IPNet
			if ip.Addr().Is4() {
				ipnet = net.IPNet{
					IP:   ip.Addr().AsSlice(),
					Mask: net.CIDRMask(ip.Bits(), 32),
				}
			} else {
				ipnet = net.IPNet{
					IP:   ip.Addr().AsSlice(),
					Mask: net.CIDRMask(ip.Bits(), 128),
				}
			}
			allowedIPs = append(allowedIPs, ipnet)
		}
	}
	if peer.IsPubliclyRoutable() {
		// The peer is publicly accessible
		udpAddr, err := net.ResolveUDPAddr("udp", peer.Endpoint)
		if err != nil {
			return fmt.Errorf("failed to resolve peer endpoint: %w", err)
		}
		endpoint = udpAddr
		if !w.IsPublic() {
			// We are behind a NAT and the peer isn't.
			// Allow all network traffic to the peer.
			if w.opts.NetworkV6.IsValid() {
				allowedIPs = append(allowedIPs, net.IPNet{
					IP:   w.opts.NetworkV6.Addr().AsSlice(),
					Mask: net.CIDRMask(w.opts.NetworkV6.Bits(), 128),
				})
			}
			if w.opts.NetworkV4.IsValid() {
				allowedIPs = append(allowedIPs, net.IPNet{
					IP:   w.opts.NetworkV4.Addr().AsSlice(),
					Mask: net.CIDRMask(w.opts.NetworkV4.Bits(), 32),
				})
			}
			// Set the keepalive interval to 25 seconds (maybe make this configurable)
			if keepAlive != nil {
				*keepAlive = 25 * time.Second
			}
		}
	} else if !w.IsPublic() {
		// We are behind a NAT and the peer is too.
		// No reason to track them
		return nil
	}
	w.log.Debug("computed allowed IPs for peer",
		slog.String("peer-id", peer.ID),
		slog.Any("allowed-ips", allowedIPs))
	peerCfg := wgtypes.PeerConfig{
		PublicKey:                   key,
		UpdateOnly:                  false,
		ReplaceAllowedIPs:           true,
		Endpoint:                    endpoint,
		AllowedIPs:                  allowedIPs,
		PersistentKeepaliveInterval: keepAlive,
	}
	w.log.Debug("configuring peer", slog.Any("peer", peerCfg))
	err = w.cli.ConfigureDevice(w.Name(), wgtypes.Config{
		Peers: []wgtypes.PeerConfig{peerCfg},
	})
	if err != nil {
		return fmt.Errorf("failed to configure wireguard interface: %w", err)
	}
	// Add the peer to our map
	w.peers[peer.ID] = key
	// Add routes to the allowed IPs
	for _, ip := range allowedIPs {
		addr, _ := netip.AddrFromSlice(ip.IP)
		_, bits := ip.Mask.Size()
		prefix := netip.PrefixFrom(addr, bits)
		if prefix.Addr().Is6() && w.opts.NetworkV6.IsValid() {
			if w.opts.NetworkV6.Contains(addr) {
				// Don't readd routes to our own network
				continue
			}
			w.log.Debug("adding ipv6 route", slog.Any("prefix", prefix))
			err = w.AddRoute(ctx, prefix)
			if err != nil && !IsRouteExists(err) {
				return fmt.Errorf("failed to add route: %w", err)
			}
		}
		if prefix.Addr().Is4() && w.opts.NetworkV4.IsValid() {
			if w.opts.NetworkV4.Contains(addr) {
				// Don't readd routes to our own network
				continue
			}
			w.log.Debug("adding ipv4 route", slog.Any("prefix", prefix))
			err = w.AddRoute(ctx, prefix)
			if err != nil && !IsRouteExists(err) {
				return fmt.Errorf("failed to add route: %w", err)
			}
		}
	}
	return nil
}

// DeletePeer removes a peer from the wireguard configuration.
func (w *wginterface) DeletePeer(ctx context.Context, peer *Peer) error {
	w.peersMux.Lock()
	defer w.peersMux.Unlock()
	if key, ok := w.peers[peer.ID]; ok {
		delete(w.peers, peer.ID)
		return w.cli.ConfigureDevice(w.Name(), wgtypes.Config{
			Peers: []wgtypes.PeerConfig{
				{
					PublicKey: key,
					Remove:    true,
				},
			},
		})
	}
	return nil
}