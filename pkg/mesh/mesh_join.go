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

package mesh

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	v1 "github.com/webmeshproj/api/v1"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/webmeshproj/webmesh/pkg/context"
	meshnet "github.com/webmeshproj/webmesh/pkg/net"
)

var (
	errFatalJoin = fmt.Errorf("fatal join error")
)

func (s *meshStore) joinWithPeerDiscovery(ctx context.Context, features []v1.Feature) error {
	log := s.log.With(slog.String("peer-discovery-addrs", strings.Join(s.opts.Mesh.PeerDiscoveryAddresses, ",")))
	ctx = context.WithLogger(ctx, log)
	log.Info("discovering joinable peers")
	var err error
	for _, addr := range s.opts.Mesh.PeerDiscoveryAddresses {
		var c *grpc.ClientConn
		c, err = s.newGRPCConn(ctx, addr)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Error("failed to dial peer discovery address", slog.String("error", err.Error()))
			continue
		}
		defer c.Close()
		cli := v1.NewPeerDiscoveryClient(c)
		var resp *v1.ListRaftPeersResponse
		resp, err = cli.ListPeers(ctx, &emptypb.Empty{})
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Error("failed to list peers", slog.String("error", err.Error()))
			continue
		}
		log.Info("discovered joinable peers", slog.Any("peers", resp.Peers))
	Peers:
		for _, peer := range resp.Peers {
			err = s.join(ctx, features, peer.Address, s.opts.Mesh.MaxJoinRetries)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Error("failed to join peer", slog.String("error", err.Error()))
				continue Peers
			}
		}
		// If we got this far, we aren't going to try another discovery server.
		// They'll all have the same peers.
		break
	}
	if err != nil {
		return fmt.Errorf("join with peer discovery: %w", err)
	}
	return nil
}

func (s *meshStore) join(ctx context.Context, features []v1.Feature, joinAddr string, maxRetries int) error {
	log := s.log.With(slog.String("join-addr", joinAddr))
	ctx = context.WithLogger(ctx, log)
	log.Info("joining mesh")
	var tries int
	var err error
	for tries <= maxRetries {
		if tries > 0 {
			log.Info("retrying join request", slog.Int("tries", tries))
		}
		var conn *grpc.ClientConn
		conn, err = s.newGRPCConn(ctx, joinAddr)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			err = fmt.Errorf("dial join node: %w", err)
			log.Error("gRPC dial failed", slog.String("error", err.Error()))
			if tries >= maxRetries {
				return err
			}
			tries++
			time.Sleep(time.Second)
			continue
		}
		err = s.joinWithConn(ctx, conn, features)
		if err != nil {
			if errors.Is(err, errFatalJoin) {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			err = fmt.Errorf("join node: %w", err)
			log.Error("join failed", slog.String("error", err.Error()))
			if tries >= maxRetries {
				return err
			}
			tries++
			time.Sleep(time.Second)
			continue
		}
		break
	}
	return err
}

func (s *meshStore) joinWithConn(ctx context.Context, c *grpc.ClientConn, features []v1.Feature) error {
	log := context.LoggerFrom(ctx)
	client := v1.NewNodeClient(c)
	defer c.Close()
	if s.opts.Mesh.GRPCAdvertisePort == 0 {
		// Assume the default port.
		s.opts.Mesh.GRPCAdvertisePort = 8443
	}
	key, err := s.loadWireGuardKey(ctx)
	if err != nil {
		return fmt.Errorf("load wireguard key: %w", err)
	}
	req := &v1.JoinRequest{
		Id:                 s.ID(),
		PublicKey:          key.PublicKey().String(),
		RaftPort:           int32(s.raft.ListenPort()),
		GrpcPort:           int32(s.opts.Mesh.GRPCAdvertisePort),
		MeshdnsPort:        int32(s.opts.Mesh.MeshDNSAdvertisePort),
		PrimaryEndpoint:    s.opts.Mesh.PrimaryEndpoint,
		WireguardEndpoints: s.opts.WireGuard.Endpoints,
		ZoneAwarenessId:    s.opts.Mesh.ZoneAwarenessID,
		AssignIpv4:         !s.opts.Mesh.NoIPv4,
		PreferRaftIpv6:     s.opts.Raft.PreferIPv6,
		AsVoter:            s.opts.Mesh.JoinAsVoter,
		Routes:             s.opts.Mesh.Routes,
		DirectPeers:        s.opts.Mesh.DirectPeers,
		Features:           features,
	}
	log.Debug("sending join request to node", slog.Any("req", req))
	resp, err := client.Join(ctx, req)
	if err != nil {
		return fmt.Errorf("join request: %w", err)
	}
	log.Debug("received join response", slog.Any("resp", resp))
	s.meshDomain = resp.GetMeshDomain()
	if !strings.HasSuffix(s.meshDomain, ".") {
		s.meshDomain += "."
	}
	var addressv4, addressv6, networkv4, networkv6 netip.Prefix
	// We always parse addresses and let the net manager decide what to use
	if resp.AddressIpv4 != "" {
		addressv4, err = netip.ParsePrefix(resp.AddressIpv4)
		if err != nil {
			return fmt.Errorf("parse ipv4 address: %w", err)
		}
	}
	networkv4, err = netip.ParsePrefix(resp.NetworkIpv4)
	if err != nil {
		return fmt.Errorf("parse ipv4 network: %w", err)
	}
	addressv6, err = netip.ParsePrefix(resp.AddressIpv6)
	if err != nil {
		return fmt.Errorf("parse ipv6 address: %w", err)
	}
	networkv6, err = netip.ParsePrefix(resp.NetworkIpv6)
	if err != nil {
		return fmt.Errorf("parse ipv6 network: %w", err)
	}
	opts := &meshnet.StartOptions{
		Key:       key,
		AddressV4: addressv4,
		AddressV6: addressv6,
		NetworkV4: networkv4,
		NetworkV6: networkv6,
	}
	log.Debug("starting network manager", slog.Any("opts", opts))
	err = s.nw.Start(ctx, opts)
	if err != nil {
		return fmt.Errorf("%w starting network manager: %w", errFatalJoin, err)
	}
	for _, peer := range resp.GetPeers() {
		log.Info("adding peer", slog.Any("peer", peer))
		err = s.nw.AddPeer(ctx, peer, resp.GetIceServers())
		if err != nil {
			return fmt.Errorf("%w adding peer: %w", errFatalJoin, err)
		}
	}
	if s.opts.Mesh.UseMeshDNS {
		var servers []netip.AddrPort
		if s.opts.Mesh.MeshDNSAdvertisePort != 0 {
			// Use our local port.
			addr := "127.0.0.1"
			if s.opts.Mesh.NoIPv4 {
				addr = "::1"
			}
			servers = append(servers, netip.AddrPortFrom(netip.MustParseAddr(addr), uint16(s.opts.Mesh.MeshDNSAdvertisePort)))
		} else {
			for _, server := range resp.GetDnsServers() {
				addr, err := netip.ParseAddrPort(server)
				if err != nil {
					return fmt.Errorf("%w parsing dns server: %w", errFatalJoin, err)
				}
				servers = append(servers, addr)
			}
		}
		err = s.nw.AddDNSServers(ctx, servers)
		if err != nil {
			return fmt.Errorf("%w adding dns servers: %w", errFatalJoin, err)
		}
	}
	return nil
}
