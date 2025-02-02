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

// Package admin provides the admin gRPC server.
package admin

import (
	v1 "github.com/webmeshproj/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/webmeshproj/webmesh/pkg/context"
	rbacdb "github.com/webmeshproj/webmesh/pkg/meshdb/rbac"
	"github.com/webmeshproj/webmesh/pkg/services/rbac"
)

var deleteRoleAction = rbac.Actions{
	{
		Resource: v1.RuleResource_RESOURCE_ROLES,
		Verb:     v1.RuleVerb_VERB_DELETE,
	},
}

func (s *Server) DeleteRole(ctx context.Context, role *v1.Role) (*emptypb.Empty, error) {
	if !s.store.Raft().IsLeader() {
		return nil, status.Error(codes.FailedPrecondition, "not the leader")
	}
	if role.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if ok, err := s.rbacEval.Evaluate(ctx, deleteRoleAction.For(role.GetName())); !ok {
		if err != nil {
			context.LoggerFrom(ctx).Error("failed to evaluate delete role action", "error", err)
		}
		return nil, status.Error(codes.PermissionDenied, "caller does not have permission to delete roles")
	}
	if rbacdb.IsSystemRole(role.GetName()) {
		return nil, status.Error(codes.InvalidArgument, "cannot delete system roles")
	}
	err := s.rbac.DeleteRole(ctx, role.GetName())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}
