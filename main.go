package main

import (
	"context"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	_ "google.golang.org/protobuf/proto"
	"log"
	pb "main/api/proto"
	pkg "main/pkg/broker"
	"net"
	_ "net"
	"time"
)

type Server struct {
	pb.UnimplementedBrokerServer
	mybroker pkg.Broker
}

func (s *Server) Publish(ctx context.Context, in *pb.PublishRequest) (*pb.PublishResponse, error) {
	log.Printf("Received: %v", in.String())
	msgId, err := s.mybroker.Publish(ctx, in.Subject, pkg.Message{
		Body:       string(in.Body),
		Expiration: time.Duration(in.ExpirationSeconds),
	})
	log.Printf("massage_id: %v", msgId)
	return nil, err
}
func (s *Server) Subscribe(in *pb.SubscribeRequest, br pb.Broker_SubscribeServer) error {
	return status.Errorf(codes.Unimplemented, "method Subscribe not implemented")
}
func (s *Server) Fetch(context.Context, *pb.FetchRequest) (*pb.MessageResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Fetch not implemented")
}

//
//func (s *Server) Publish(ctx context.Context, in *pb.PublishRequest) (*pb.PublishResponse, error) {
//	log.Printf("Received: %v", in.String())
//	msgId , err := s.mybroker.Publish(ctx, in.Subject, pkg.Message{
//		Body: string(in.GetBody()),
//		Expiration: time.Duration(in.ExpirationSeconds),
//	})
//	return &pb.PublishResponse{
//		Id: int32(msgId),
//	}, err
//}

// Main requirements:
// 1. All tests should be passed
// 2. Your logs should be accessible in Graylog
// 3. Basic prometheus metrics ( latency, throughput, etc. ) should be implemented
// 	  for every base functionality ( publish, subscribe etc. )

const (
	port = ":50051"
)

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterBrokerServer(s, &Server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
