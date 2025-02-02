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

package meshdns

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"github.com/webmeshproj/webmesh/pkg/context"
	"github.com/webmeshproj/webmesh/pkg/meshdb"
	"github.com/webmeshproj/webmesh/pkg/meshdb/peers"
)

func (s *Server) appendPeerToMessage(ctx context.Context, st meshdb.Store, r, m *dns.Msg, peerID string, ipv6Only bool) error {
	peer, err := peers.New(st.Storage()).Get(ctx, peerID)
	if err != nil {
		return err
	}
	fqdn := newFQDN(st, peer.ID)
	for i, q := range r.Question {
		switch q.Qtype {
		case dns.TypeTXT:
			s.log.Debug("handling peer TXT question")
			m.Answer = append(m.Answer, newPeerTXTRecord(fqdn, &peer))
			if !ipv6Only && peer.PrivateIPv4.IsValid() {
				m.Extra = append(m.Extra, &dns.A{
					Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 1},
					A:   peer.PrivateIPv4.Addr().AsSlice(),
				})
			}
			if peer.PrivateIPv6.IsValid() {
				m.Extra = append(m.Extra, &dns.AAAA{
					Hdr:  dns.RR_Header{Name: fqdn, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 1},
					AAAA: peer.PrivateIPv6.Addr().AsSlice(),
				})
			}
		case dns.TypeA:
			if ipv6Only {
				if i != len(r.Question)-1 {
					// Maybe they asked for a AAAA also
					continue
				}
				return errNoIPv4{}
			}
			s.log.Debug("handling peer A question")
			if !peer.PrivateIPv4.IsValid() {
				s.log.Debug("no private IPv4 address for peer")
				return errNoIPv4{}
			}
			m.Answer = append(m.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 1},
				A:   peer.PrivateIPv4.Addr().AsSlice(),
			})
			m.Extra = append(m.Extra, newPeerTXTRecord(fqdn, &peer))
		case dns.TypeAAAA:
			s.log.Debug("handling peer AAAA question")
			if !peer.PrivateIPv6.IsValid() {
				s.log.Debug("no private IPv6 address for peer")
				return errNoIPv6{}
			}
			m.Answer = append(m.Answer, &dns.AAAA{
				Hdr:  dns.RR_Header{Name: fqdn, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 1},
				AAAA: peer.PrivateIPv6.Addr().AsSlice(),
			})
			m.Extra = append(m.Extra, newPeerTXTRecord(fqdn, &peer))
		}
	}
	return nil
}

func newPeerTXTRecord(name string, peer *peers.Node) *dns.TXT {
	txtData := []string{
		fmt.Sprintf("id=%s", peer.ID),
		fmt.Sprintf("raft_port=%d", peer.RaftPort),
		fmt.Sprintf("grpc_port=%d", peer.GRPCPort),
		fmt.Sprintf("wireguard_endpoints=%s", func() string {
			if len(peer.WireGuardEndpoints) > 0 {
				return strings.Join(peer.WireGuardEndpoints, ",")
			}
			return "<none>"
		}()),
		fmt.Sprintf("primary_endpoint=%s", func() string {
			if peer.PrimaryEndpoint != "" {
				return peer.PrimaryEndpoint
			}
			return "<none>"
		}()),
	}
	return &dns.TXT{
		Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 1},
		Txt: txtData,
	}
}
