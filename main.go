package main

import (
	"context"
	_ "flag"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	_ "google.golang.org/protobuf/proto"
	"gopkg.in/Graylog2/go-gelf.v1/gelf"
	"io"
	pb "main/api/proto"
	internal "main/internal/broker"
	pkg2 "main/pkg"
	pkg "main/pkg/broker"
	"net"
	_ "net"
	"os"
	"time"
)

type Server struct {
	pb.UnimplementedBrokerServer
	ctx context.Context
}

var (
	broker pkg.Broker
)

func GetBroker() pkg.Broker {
	if broker == nil {
		broker = internal.NewModule()
	}
	return broker
}

func (s *Server) Publish(ctx context.Context, in *pb.PublishRequest) (*pb.PublishResponse, error) {
	log.WithField("Received:", in.String())
	pkg2.GetMethodCountMetric().With(prometheus.Labels{"method_name": "publish", "subject": in.Subject}).Inc()
	start := time.Now()
	brk := GetBroker()
	msgId, err := brk.Publish(ctx, in.Subject, pkg.Message{
		Body:       string(in.Body),
		Expiration: time.Duration(in.ExpirationSeconds),
	})
	if err != nil {
		log.WithFields(log.Fields{
			"request": in,
			"error":   err.Error(),
		}).Error("can't publish on topc: ", in.Subject)
		pkg2.GetErrorRateMetric().With(prometheus.Labels{"method_name": "publish", "subject": in.Subject, "error": err.Error()}).Observe(1)
		defer pkg2.GetMethodDurationMetric().With(
			prometheus.Labels{"method_name": "publish", "subject": in.Subject, "successful": "false"},
		).Observe(float64(time.Until(start).Milliseconds()))
		return &pb.PublishResponse{}, err
	}
	log.Info("Publish was successfully!")
	defer pkg2.GetMethodDurationMetric().With(
		prometheus.Labels{"method_name": "publish", "subject": in.Subject, "successful": "true"},
	).Observe(float64(time.Until(start).Milliseconds()))
	return &pb.PublishResponse{Id: int32(msgId)}, err
}
func (s *Server) Subscribe(in *pb.SubscribeRequest, stream pb.Broker_SubscribeServer) error {
	log.WithField("Received: ", in.String())
	pkg2.GetMethodCountMetric().With(prometheus.Labels{"method_name": "subscribe", "subject": in.Subject}).Inc()
	pkg2.GetActiveSubscribesMetric().With(prometheus.Labels{"subject": in.Subject}).Inc()
	start := time.Now()
	brk := GetBroker()
	chans, err := brk.Subscribe(s.ctx, in.Subject)

	if err != nil {
		log.WithFields(log.Fields{
			"request": in,
			"error":   err.Error(),
		}).Error("client can't Subscribe on Subject: ", in.Subject)
		pkg2.GetErrorRateMetric().With(prometheus.Labels{"method_name": "subscribe", "subject": in.Subject, "error": err.Error()}).Observe(1)
		defer pkg2.GetMethodDurationMetric().With(
			prometheus.Labels{"method_name": "subscribe", "subject": in.Subject, "successful": "false"},
		).Observe(float64(time.Until(start).Milliseconds()))

		return err
	}
	for {
		select {
		case str := <-chans:
			{
				response := pb.MessageResponse{
					Body: []byte(str.Body),
				}
				err := stream.Send(&response)
				if err != nil {
					log.WithFields(log.Fields{
						"request": in,
						"error":   err.Error(),
					}).Error("client can't Subscribe on Subject: ", in.Subject)
					pkg2.GetErrorRateMetric().With(prometheus.Labels{"method_name": "subscribe", "subject": in.Subject, "error": err.Error()}).Observe(1)
					defer pkg2.GetMethodDurationMetric().With(
						prometheus.Labels{"method_name": "subscribe", "subject": in.Subject, "successful": "false"},
					).Observe(float64(time.Until(start).Milliseconds()))

					return err
				}
				defer pkg2.GetMethodDurationMetric().With(
					prometheus.Labels{"method_name": "subscribe", "subject": in.Subject, "successful": "true"},
				).Observe(float64(time.Until(start).Milliseconds()))
				log.Infof("Received Message: %v from Server was Successfuly!", response.Body)
			}
		}
	}
}
func (s *Server) Fetch(ctx context.Context, fch *pb.FetchRequest) (*pb.MessageResponse, error) {
	log.Info("Received: ", fch.String())
	pkg2.GetMethodCountMetric().With(prometheus.Labels{"method_name": "fetch", "subject": fch.Subject}).Inc()
	brk := GetBroker()
	start := time.Now()
	msg, err := brk.Fetch(ctx, fch.Subject, int(fch.Id))
	if err != nil {
		log.WithFields(log.Fields{
			"request": fch,
			"error":   err.Error(),
		}).Error("client can't Fetch Message on Subject: ", fch.Subject)
		pkg2.GetErrorRateMetric().With(prometheus.Labels{"method_name": "fetch", "subject": fch.Subject, "error": err.Error()}).Observe(1)
		defer pkg2.GetMethodDurationMetric().With(
			prometheus.Labels{"method_name": "subscribe", "fetch": fch.Subject, "successful": "false"},
		).Observe(float64(time.Until(start).Milliseconds()))

		return &pb.MessageResponse{}, err
	}
	data := []byte(msg.Body)
	log.Infof("Fetch %v From server was successfuly!", msg.Body)

	defer pkg2.GetMethodDurationMetric().With(
		prometheus.Labels{"method_name": "publish", "fetch": fch.Subject, "successful": "true"},
	).Observe(float64(time.Until(start).Milliseconds()))

	return &pb.MessageResponse{Body: data}, err
}

// Main requirements:
// 1. All tests should be passed
// 2. Your logs should be accessible in Graylog
// 3. Basic prometheus metrics ( latency, throughput, etc. ) should be implemented
// 	  for every base functionality ( publish, subscribe etc. )

const (
	port = ":50051"
)

func init() {
	graylogAddr := ":12201"
	gelfWriter, err := gelf.NewWriter(graylogAddr)
	if err != nil {
		log.Fatalf("gelf.NewWriter: %s", err)
	}
	// log to both stderr and graylog2
	log.SetOutput(io.MultiWriter(os.Stderr, gelfWriter))
	log.Printf("logging to stderr & graylog2@'%s'", graylogAddr)

	log.Printf("Hello gray World")

}
func main() {

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	pkg2.StartPrometheusServer()
	s := grpc.NewServer()
	pb.RegisterBrokerServer(s, &Server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
