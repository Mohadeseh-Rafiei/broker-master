package broker

import (
	"context"
	"github.com/go-redis/redis"
	"main/pkg/broker"
)

//TODO : BOTTLENECK IN CHANS
//TODO : ALL LOCKS

//TODO : SEND MASSAGE FOR EVERY CLIENT OR REDIS MEM CAN HELP

type Module struct {
	*redis.Client
	subs map[string]*redis.PubSub
}

//
//type Module struct {
//	chat map[string] []broker.Message
//	clients map[string][]*chan broker.Message
//	expiration map[string][]bool
//	close bool
//	mx sync.Mutex
//}

func NewModule() broker.Broker {
	client := redis.NewClient(&redis.Options{Addr: ":3679"})
	return &Module{
		Client: client,
		subs:   make(map[string]*redis.PubSub),
	}
}

func (m *Module) Close() error {
	if err := m.Client.Close(); err != nil {
		return err
	}
	return nil
}

func (m *Module) Publish(ctx context.Context, subject string, msg broker.Message) (int, error) {
	//if m.close{
	//	return 0, broker.ErrUnavailable
	//}
	//m.mx.Lock()
	//m.chat[subject] = append(m.chat[subject], msg)
	//m.expiration[subject] = append(m.expiration[subject], false)
	//wg := &sync.WaitGroup{}
	//for _, n := range m.clients[subject]{
	//	wg.Add(1)
	//	go func(wg *sync.WaitGroup, ) {
	//		*n <- msg
	//		wg.Done()
	//	}(wg)
	//
	//}
	//wg.Wait()
	//index := len(m.chat[subject]) - 1
	//m.mx.Unlock()
	//go func() {
	//	if msg.Expiration == 0 {
	//		return
	//	}
	//	ticker := time.NewTicker(msg.Expiration)
	//	<-ticker.C
	//	m.expiration[subject][index] = true
	//}()
	//return index, nil

	_, ok := m.subs[subject]
	if !ok {
		return 0, broker.ErrUnavailable
	}
	if err := m.Client.Publish(subject, msg).Err(); err != nil {
		return 0, err
	}
	return len(m.subs) - 1, nil
}

func (m *Module) Subscribe(ctx context.Context, subject string) (<-chan broker.Message, error) {
	//if m.close{
	//	empty_chan := make(chan broker.Message)
	//	return empty_chan, broker.ErrUnavailable
	//}
	//out_chan := make(chan broker.Message, 1000000)
	//m.mx.Lock()
	////wg := &sync.WaitGroup{}
	////
	////for _, n := range m.chat[subject]{
	////	wg.Add(1)
	////	go func(wg *sync.WaitGroup) {
	////		out_chan <- n
	////		wg.Done()
	////	}(wg)
	////}
	////wg.Wait()
	//m.clients[subject] = append(m.clients[subject], &out_chan)
	//m.mx.Unlock()
	//return  out_chan, nil

	sub := m.Client.Subscribe(subject)
	_, err := sub.Receive()
	if err != nil {
		return nil, broker.ErrUnavailable
	}
	m.subs[subject] = sub
	out := make(chan broker.Message)
	go func() {
		for massage := range sub.Channel() {
			out <- massage
		}
		close(out)
	}()
	return out, nil
}

func (m *Module) Fetch(ctx context.Context, subject string, id int) (broker.Message, error) {
	//m.mx.Lock()
	//defer m.mx.Unlock()
	//msg := broker.Message{}
	//if m.close{
	//	return msg, broker.ErrUnavailable
	//}
	//if m.expiration[subject][id] {
	//	return msg, broker.ErrExpiredID
	//}
	//return m.chat[subject][id], nil
}
