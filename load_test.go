package main

import (
	pb "broker-massage/api/proto"
	"context"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"testing"
	"time"
)

func TestDataRace(t *testing.T) {
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(":4000", grpc.WithInsecure())
	if err != nil {
		log.Info("did not connect")
	}
	defer conn.Close()
	ctx := context.Background()
	c := pb.NewBrokerClient(conn)

	for i := 0; i < 100000; i++ {
		pub_request := pb.PublishRequest{
			Body:    []byte("hello"),
			Subject: "alaki",
		}
		go func(pub_request *pb.PublishRequest) {
			c.Publish(ctx, pub_request)
		}(&pub_request)
	}
	for i := 0; i < 100000; i++ {
		sub_request := pb.SubscribeRequest{
			Subject: "ali",
		}
		go func(sub_request *pb.SubscribeRequest) {
			c.Subscribe(ctx, sub_request)
		}(&sub_request)
	}
	time.Sleep(20 * time.Second)
}
