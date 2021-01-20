// Example backend application

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	csdspb "github.com/envoyproxy/go-control-plane/envoy/service/status/v3"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	ports = []string{":10001", ":10002", ":10003"}
)

var configDump = `{"xdsConfig":[{"clientStatus":2,"listenerConfig":{"dynamicListeners":[{"activeState":{"lastUpdated":"2021-01-20T19:46:14.720363332Z","listener":{"@type":"type.googleapis.com/envoy.config.listener.v3.Listener","apiListener":{"apiListener":{"@type":"type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager","routeConfig":{"name":"route_config_name","virtualHosts":[{"domains":["*"],"routes":[{"match":{"prefix":""},"route":{"cluster":"cluster_name"}}]}]}}},"name":"server.example.com"},"versionInfo":"1"},"name":"server.example.com"}],"versionInfo":"1"}},{"clientStatus":0,"routeConfig":{"dynamicRouteConfigs":[]}},{"clientStatus":2,"clusterConfig":{"dynamicActiveClusters":[{"cluster":{"@type":"type.googleapis.com/envoy.config.cluster.v3.Cluster","edsClusterConfig":{"edsConfig":{"ads":{}},"serviceName":"eds_service_name"},"name":"cluster_name","type":"EDS"},"lastUpdated":"2021-01-20T19:46:14.731363449Z","versionInfo":"1"}],"versionInfo":"1"}},{"clientStatus":2,"endpointConfig":{"dynamicEndpointConfigs":[{"endpointConfig":{"@type":"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment","clusterName":"eds_service_name","endpoints":[{"lbEndpoints":[{"endpoint":{"address":{"socketAddress":{"address":"127.0.0.1","portValue":19153}}}},{"endpoint":{"address":{"socketAddress":{"address":"127.0.0.1","portValue":15969}}}},{"endpoint":{"address":{"socketAddress":{"address":"127.0.0.1","portValue":31459}}}},{"endpoint":{"address":{"socketAddress":{"address":"127.0.0.1","portValue":24536}}}}],"loadBalancingWeight":3,"locality":{"region":"xds_default_locality_region","subZone":"locality0","zone":"xds_default_locality_zone"}}]},"lastUpdated":"2021-01-20T19:46:14.741363465Z","versionInfo":"1"}],"versionInfo":"1"}}]}`

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

// slow server is used to simulate a server that has a variable delay in its response.
type slowServer struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *slowServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Delay 100ms ~ 200ms before replying
	time.Sleep(time.Duration(100+r.Intn(100)) * time.Millisecond)
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

type mockCsdsServer struct {
	csdspb.UnimplementedClientStatusDiscoveryServiceServer
}

func (*mockCsdsServer) FetchClientStatus(ctx context.Context, req *csdspb.ClientStatusRequest) (*csdspb.ClientStatusResponse, error) {
	var config csdspb.ClientConfig
	if err := protojson.Unmarshal([]byte(configDump), &config); err != nil {
		log.Fatalf("failed to parse config dump: %v", err)
	}
	return &csdspb.ClientStatusResponse{
		Config: []*csdspb.ClientConfig{&config},
	}, nil
}

func main() {
	/***** Set up the server serving channelz service. *****/
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()
	fmt.Println("Serving Admin Services on :50051")
	s := grpc.NewServer()
	service.RegisterChannelzServiceToServer(s)
	csdspb.RegisterClientStatusDiscoveryServiceServer(s, &mockCsdsServer{})
	go s.Serve(lis)
	defer s.Stop()

	/***** Start three GreeterServers(with one of them to be the slowServer). *****/
	for i := 0; i < 3; i++ {
		lis, err := net.Listen("tcp", ports[i])
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		defer lis.Close()
		fmt.Printf("Serving business logic on :%v\n", ports[i])
		s := grpc.NewServer()
		if i == 2 {
			pb.RegisterGreeterServer(s, &slowServer{})
		} else {
			pb.RegisterGreeterServer(s, &server{})
		}
		go s.Serve(lis)
	}

	/***** Wait for user exiting the program *****/
	select {}
}
