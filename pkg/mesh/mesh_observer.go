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
	"context"
	"reflect"

	v1 "github.com/webmeshproj/api/v1"
	"golang.org/x/exp/slog"

	"github.com/webmeshproj/webmesh/pkg/meshdb/peers"
	"github.com/webmeshproj/webmesh/pkg/raft"
)

func (s *meshStore) onObservation(ev raft.Observation) {
	log := s.log.With("event", "observation")
	log.Debug("received observation event", slog.String("type", reflect.TypeOf(ev.Data).String()))
	ctx := context.Background()
	switch data := ev.Data.(type) {
	case raft.PeerObservation:
		log.Debug("PeerObservation", slog.Any("data", data))
		if s.testStore {
			return
		}
		if string(data.Peer.ID) == s.nodeID {
			return
		}
		if err := s.nw.RefreshPeers(ctx); err != nil {
			log.Warn("wireguard refresh peers", slog.String("error", err.Error()))
		}
		if s.plugins.HasWatchers() {
			p := peers.New(s.Storage())
			node, err := p.Get(ctx, string(data.Peer.ID))
			if err != nil {
				log.Warn("failed to lookup peer, can't emit event", slog.String("error", err.Error()))
				return
			}
			err = s.plugins.Emit(ctx, &v1.Event{
				Type: func() v1.WatchEvent {
					if data.Removed {
						return v1.WatchEvent_WATCH_EVENT_NODE_LEAVE
					}
					return v1.WatchEvent_WATCH_EVENT_NODE_JOIN
				}(),
				Event: &v1.Event_Node{
					Node: node.Proto(func() v1.ClusterStatus {
						if data.Removed {
							return v1.ClusterStatus_CLUSTER_STATUS_UNKNOWN
						}
						if data.Peer.Suffrage == raft.Nonvoter {
							return v1.ClusterStatus_CLUSTER_NON_VOTER
						}
						return v1.ClusterStatus_CLUSTER_VOTER
					}()),
				},
			})
			if err != nil {
				log.Warn("error sending node join/leave event", slog.String("error", err.Error()))
			}
		}
	case raft.LeaderObservation:
		log.Debug("LeaderObservation", slog.Any("data", data))
		if s.plugins.HasWatchers() {
			p := peers.New(s.Storage())
			node, err := p.Get(ctx, string(data.LeaderID))
			if err != nil {
				log.Warn("failed to get leader, may be fresh cluster, can't emit event", slog.String("error", err.Error()))
				return
			}
			err = s.plugins.Emit(ctx, &v1.Event{
				Type: v1.WatchEvent_WATCH_EVENT_LEADER_CHANGE,
				Event: &v1.Event_Node{
					Node: node.Proto(v1.ClusterStatus_CLUSTER_LEADER),
				},
			})
			if err != nil {
				log.Warn("error sending leader change event", slog.String("error", err.Error()))
			}
		}
	}
}
