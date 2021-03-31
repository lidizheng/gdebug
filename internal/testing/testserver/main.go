// Testserver mocking the responses of Channelz/CSDS/Health

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/reflection"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/fault/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	csdspb "github.com/envoyproxy/go-control-plane/envoy/service/status/v3"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	servingPortFlag = flag.Int("serving", 10001, "the serving port")
	adminPortFlag   = flag.Int("admin", 50051, "the admin port")
	healthFlag      = flag.Bool("health", true, "the health checking status")
)

// Implements the Greeter service
type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

// Implements the CSDS service
type mockCsdsServer struct {
	csdspb.UnimplementedClientStatusDiscoveryServiceServer
}

func (*mockCsdsServer) FetchClientStatus(ctx context.Context, req *csdspb.ClientStatusRequest) (*csdspb.ClientStatusResponse, error) {
	file, err := os.Open("csds_config_dump.json")
	if err != nil {
		panic(err)
	}
	configDump, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	var response csdspb.ClientStatusResponse
	if err := protojson.Unmarshal([]byte(configDump), &response); err != nil {
		panic(err)
	}
	return &response, nil
}

func main() {
	// Creates the primary server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *servingPortFlag))
	if err != nil {
		panic(err)
	}
	defer lis.Close()
	fmt.Printf("Serving business logic on :%d\n", *servingPortFlag)
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	go s.Serve(lis)
	// Creates the admin server
	lis, err = net.Listen("tcp", fmt.Sprintf(":%d", *adminPortFlag))
	if err != nil {
		panic(err)
	}
	defer lis.Close()
	s = grpc.NewServer()
	reflection.Register(s)
	service.RegisterChannelzServiceToServer(s)
	csdspb.RegisterClientStatusDiscoveryServiceServer(s, &mockCsdsServer{})
	healthcheck := health.NewServer()
	if *healthFlag {
		healthcheck.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	} else {
		healthcheck.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	}
	healthpb.RegisterHealthServer(s, healthcheck)
	fmt.Printf("Serving Admin Services on :%d\n", *adminPortFlag)
	go s.Serve(lis)
	/***** Wait for user exiting the program *****/
	select {}
}
