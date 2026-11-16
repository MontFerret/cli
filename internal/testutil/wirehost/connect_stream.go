package wirehost

import (
	"context"

	"google.golang.org/grpc"
)

type connectStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *connectStream) Context() context.Context { return s.ctx }
