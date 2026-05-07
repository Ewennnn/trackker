package api

import "context"

type ServerBuilder func() Server

type Server interface {
	Start(ctx context.Context) error
	Shutdown() error
}
