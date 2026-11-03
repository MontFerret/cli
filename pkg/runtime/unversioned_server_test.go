package runtime

import (
	"google.golang.org/grpc"

	wirev1 "github.com/MontFerret/wire/gen/ferret/wire/v1"
)

// unversionedServer simulates a pre-alpha.2 handshake with valid protocol metadata.
type unversionedServer struct {
	wirev1.UnimplementedRuntimeServiceServer
}

func (s *unversionedServer) Connect(_ *wirev1.ConnectRequest, stream grpc.ServerStreamingServer[wirev1.ConnectResponse]) error {
	if err := stream.Send(&wirev1.ConnectResponse{
		ConnectionId: &wirev1.ConnectionId{Value: "legacy-connection"},
		Protocol:     &wirev1.ProtocolInfo{Name: "ferret-wire", Version: "1"},
	}); err != nil {
		return err
	}

	<-stream.Context().Done()

	return stream.Context().Err()
}
