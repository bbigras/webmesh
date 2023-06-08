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

// Package admin provides the admin gRPC server.
package admin

import (
	"context"

	v1 "github.com/webmeshproj/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	rbacdb "github.com/webmeshproj/node/pkg/meshdb/rbac"
	"github.com/webmeshproj/node/pkg/services/rbac"
)

var getRoleBindingAction = &rbac.Action{
	Resource: v1.RuleResource_RESOURCE_ROLE_BINDINGS,
	Verb:     v1.RuleVerbs_VERB_GET,
}

func (s *Server) GetRoleBinding(ctx context.Context, rb *v1.RoleBinding) (*v1.RoleBinding, error) {
	if rb.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if ok, err := s.rbacEval.Evaluate(ctx, getRoleBindingAction.For(rb.GetName())); !ok {
		return nil, status.Error(codes.PermissionDenied, "caller does not have permission to get rolebindings")
	} else if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	rb, err := s.rbac.GetRoleBinding(ctx, rb.GetName())
	if err != nil {
		if err == rbacdb.ErrRoleBindingNotFound {
			return nil, status.Errorf(codes.NotFound, "rolebinding %q not found", rb.GetName())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return rb, nil
}