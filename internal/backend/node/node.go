package node

import "context"

type Node interface {
	Provision(context.Context) error
	Teardown(context.Context) error
}
