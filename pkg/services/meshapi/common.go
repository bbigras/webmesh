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

package meshapi

import (
	"github.com/hashicorp/raft"
	v1 "github.com/webmeshproj/api/v1"

	"github.com/webmeshproj/webmesh/pkg/meshdb/peers"
)

func dbNodeToAPINode(node *peers.Node, leader string, servers []raft.Server) *v1.MeshNode {
	return node.Proto(func() v1.ClusterStatus {
		for _, srv := range servers {
			if string(srv.ID) == node.ID {
				if string(srv.ID) == leader {
					return v1.ClusterStatus_CLUSTER_LEADER
				}
				switch srv.Suffrage {
				case raft.Voter:
					return v1.ClusterStatus_CLUSTER_VOTER
				case raft.Nonvoter:
					return v1.ClusterStatus_CLUSTER_NON_VOTER
				}
			}
		}
		return v1.ClusterStatus_CLUSTER_STATUS_UNKNOWN
	}())
}
