package transport

import (
	"context"
	"log"
	"time"

	csdspb "github.com/envoyproxy/go-control-plane/envoy/service/status/v3"
	"google.golang.org/grpc"
	zpb "google.golang.org/grpc/channelz/grpc_channelz_v1"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

var conn *grpc.ClientConn
var channelzClient zpb.ChannelzClient
var csdsClient csdspb.ClientStatusDiscoveryServiceClient
var healthClient healthpb.HealthClient

// Connect connects to the service at address and creates stubs
func Connect(address, certFile, serverNameOverride string) {
	var err error
	var credOption grpc.DialOption
	if certFile != "" {
		cred, err := credentials.NewClientTLSFromFile(certFile, serverNameOverride)
		if err != nil {
			log.Fatalf("failed to create credential: %v", err)
		}
		credOption = grpc.WithTransportCredentials(cred)
	} else {
		credOption = grpc.WithInsecure()
	}
	// Dial
	conn, err = grpc.Dial(address, credOption)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	channelzClient = zpb.NewChannelzClient(conn)
	csdsClient = csdspb.NewClientStatusDiscoveryServiceClient(conn)
	healthClient = healthpb.NewHealthClient(conn)
	// Wait for ready
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	var state connectivity.State = conn.GetState()
	for state != connectivity.Ready {
		conn.WaitForStateChange(ctx, state)
		if ctx.Err() != nil {
			log.Fatalf("failed to establish connection to address: %v", address)
		}
		state = conn.GetState()
	}
}

// IsConnected checks if the connection is ready to use
func IsConnected() bool {
	if conn == nil {
		return false
	}
	return conn.GetState() == connectivity.Ready
}

// Channels returns all available channels
func Channels() []*zpb.Channel {
	channels, err := channelzClient.GetTopChannels(context.Background(), &zpb.GetTopChannelsRequest{})
	if err != nil {
		log.Fatalf("failed to fetch top channels: %v", err)
	}
	return channels.Channel
}

// Subchannel returns the queried subchannel
func Subchannel(subchannelID int64) *zpb.Subchannel {
	subchannel, err := channelzClient.GetSubchannel(context.Background(), &zpb.GetSubchannelRequest{SubchannelId: subchannelID})
	if err != nil {
		log.Fatalf("failed to fetch subchannel (id=%v): %v", subchannelID, err)
	}
	return subchannel.Subchannel
}

// Subchannels traverses all channels and fetches all subchannels
func Subchannels() []*zpb.Subchannel {
	var s []*zpb.Subchannel
	for _, channel := range Channels() {
		for _, subchannelRef := range channel.SubchannelRef {
			s = append(s, Subchannel(subchannelRef.SubchannelId))
		}
	}
	return s
}

// Servers returns all available servers
func Servers() []*zpb.Server {
	servers, err := channelzClient.GetServers(context.Background(), &zpb.GetServersRequest{})
	if err != nil {
		log.Fatalf("failed to fetch servers: %v", err)
	}
	return servers.Server
}

// Socket returns a socket
func Socket(socketID int64) *zpb.Socket {
	socket, err := channelzClient.GetSocket(context.Background(), &zpb.GetSocketRequest{SocketId: socketID})
	if err != nil {
		log.Fatalf("failed to fetch socket (id=%v): %v", socketID, err)
	}
	return socket.Socket
}

// ServerSocket returns all sockets of this server
func ServerSocket(serverId int64) []*zpb.Socket {
	var s []*zpb.Socket
	serverSocketResp, err := channelzClient.GetServerSockets(
		context.Background(),
		&zpb.GetServerSocketsRequest{ServerId: serverId},
	)
	if err != nil {
		log.Fatalf("failed to fetch server sockets (id=%v): %v", serverId, err)
	}
	for _, socketRef := range serverSocketResp.SocketRef {
		s = append(s, Socket(socketRef.SocketId))
	}
	return s
}

// ServerSockets returns all sockets for servers
func Sockets() []*zpb.Socket {
	var s []*zpb.Socket
	// Gather client sockets
	for _, subchannel := range Subchannels() {
		for _, socketRef := range subchannel.SocketRef {
			s = append(s, Socket(socketRef.SocketId))
		}
	}
	// Gather server sockets
	for _, server := range Servers() {
		s = append(s, ServerSocket(server.Ref.ServerId)...)
	}
	return s
}

// FetchClientStatus fetches the xDS resources status
func FetchClientStatus() *csdspb.ClientStatusResponse {
	resp, err := csdsClient.FetchClientStatus(context.Background(), &csdspb.ClientStatusRequest{})
	if err != nil {
		log.Fatalf("failed to fetch xds config: %v", err)
	}
	return resp
}

func GetHealthStatus(service string) string {
	resp, err := healthClient.Check(context.Background(), &healthpb.HealthCheckRequest{Service: service})
	if err != nil {
		log.Fatalf("failed to fetch health status for \"%s\": %v", service, err)
	}
	return healthpb.HealthCheckResponse_ServingStatus_name[int32(resp.Status)]
}
