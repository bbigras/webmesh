// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0

package raftdb

import (
	"context"
)

type Querier interface {
	CreateNode(ctx context.Context, arg CreateNodeParams) (Node, error)
	DeleteNode(ctx context.Context, id string) error
	GetIPv4Prefix(ctx context.Context) (string, error)
	GetNode(ctx context.Context, id string) (GetNodeRow, error)
	GetNodePeer(ctx context.Context, id string) (GetNodePeerRow, error)
	GetNodePrivateRPCAddress(ctx context.Context, nodeID string) (interface{}, error)
	GetNodePrivateRPCAddresses(ctx context.Context, nodeID string) ([]interface{}, error)
	GetNodePublicRPCAddress(ctx context.Context, nodeID string) (interface{}, error)
	GetNodePublicRPCAddresses(ctx context.Context, nodeID string) ([]interface{}, error)
	GetULAPrefix(ctx context.Context) (string, error)
	InsertNodeLease(ctx context.Context, arg InsertNodeLeaseParams) (Lease, error)
	ListAllocatedIPv4(ctx context.Context) ([]string, error)
	ListNodePeers(ctx context.Context, id string) ([]ListNodePeersRow, error)
	ListNodes(ctx context.Context) ([]ListNodesRow, error)
	ListPublicRPCAddresses(ctx context.Context) ([]ListPublicRPCAddressesRow, error)
	ReleaseNodeLease(ctx context.Context, nodeID string) error
	SetIPv4Prefix(ctx context.Context, value string) error
	SetULAPrefix(ctx context.Context, value string) error
	UpdateNode(ctx context.Context, arg UpdateNodeParams) (Node, error)
}

var _ Querier = (*Queries)(nil)