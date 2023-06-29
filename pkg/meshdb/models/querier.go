// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0

package models

import (
	"context"
	"database/sql"
)

type Querier interface {
	DeleteGroup(ctx context.Context, name string) error
	DeleteNetworkACL(ctx context.Context, name string) error
	DeleteNetworkRoute(ctx context.Context, name string) error
	DeleteNode(ctx context.Context, id string) error
	DeleteNodeEdge(ctx context.Context, arg DeleteNodeEdgeParams) error
	DeleteNodeEdges(ctx context.Context, arg DeleteNodeEdgesParams) error
	DeleteRole(ctx context.Context, name string) error
	DeleteRoleBinding(ctx context.Context, name string) error
	EitherNodeExists(ctx context.Context, arg EitherNodeExistsParams) (int64, error)
	GetGroup(ctx context.Context, name string) (Group, error)
	GetIPv4Prefix(ctx context.Context) (string, error)
	GetIPv6Prefix(ctx context.Context) (string, error)
	GetNetworkACL(ctx context.Context, name string) (NetworkAcl, error)
	GetNetworkRoute(ctx context.Context, name string) (NetworkRoute, error)
	GetNode(ctx context.Context, id string) (GetNodeRow, error)
	GetNodeCount(ctx context.Context) (int64, error)
	GetNodeEdge(ctx context.Context, arg GetNodeEdgeParams) (NodeEdge, error)
	GetRole(ctx context.Context, name string) (Role, error)
	GetRoleBinding(ctx context.Context, name string) (RoleBinding, error)
	InsertNode(ctx context.Context, arg InsertNodeParams) (Node, error)
	InsertNodeEdge(ctx context.Context, arg InsertNodeEdgeParams) error
	InsertNodeLease(ctx context.Context, arg InsertNodeLeaseParams) (Lease, error)
	ListAllocatedIPv4(ctx context.Context) ([]string, error)
	ListBoundRolesForNode(ctx context.Context, arg ListBoundRolesForNodeParams) ([]Role, error)
	ListBoundRolesForUser(ctx context.Context, arg ListBoundRolesForUserParams) ([]Role, error)
	ListGroups(ctx context.Context) ([]Group, error)
	ListNetworkACLs(ctx context.Context) ([]NetworkAcl, error)
	ListNetworkRoutes(ctx context.Context) ([]NetworkRoute, error)
	ListNetworkRoutesByDstCidr(ctx context.Context, dstCidrs string) ([]NetworkRoute, error)
	ListNetworkRoutesByNode(ctx context.Context, node string) ([]NetworkRoute, error)
	ListNodeEdges(ctx context.Context) ([]NodeEdge, error)
	ListNodeIDs(ctx context.Context) ([]string, error)
	ListNodes(ctx context.Context) ([]ListNodesRow, error)
	ListNodesByZone(ctx context.Context, zoneAwarenessID sql.NullString) ([]ListNodesByZoneRow, error)
	ListPublicNodes(ctx context.Context) ([]ListPublicNodesRow, error)
	ListRoleBindings(ctx context.Context) ([]RoleBinding, error)
	ListRoles(ctx context.Context) ([]Role, error)
	NodeEdgeExists(ctx context.Context, arg NodeEdgeExistsParams) (int64, error)
	NodeExists(ctx context.Context, id string) (int64, error)
	NodeHasEdges(ctx context.Context, arg NodeHasEdgesParams) (int64, error)
	PutGroup(ctx context.Context, arg PutGroupParams) error
	PutNetworkACL(ctx context.Context, arg PutNetworkACLParams) error
	PutNetworkRoute(ctx context.Context, arg PutNetworkRouteParams) error
	PutRole(ctx context.Context, arg PutRoleParams) error
	PutRoleBinding(ctx context.Context, arg PutRoleBindingParams) error
	ReleaseNodeLease(ctx context.Context, nodeID string) error
	SetIPv4Prefix(ctx context.Context, value string) error
	SetIPv6Prefix(ctx context.Context, value string) error
	UpdateNodeEdge(ctx context.Context, arg UpdateNodeEdgeParams) error
}

var _ Querier = (*Queries)(nil)
