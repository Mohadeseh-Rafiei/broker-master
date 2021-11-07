package broker

import (
	"context"
	"main/pkg/broker"
	"sync"
	"time"
)

//TODO CONSISTANT HASING
//TODO REDIS ALGORITHM

type Module struct {
	chat       map[string][]broker.Message
	clients    map[string][]*chan broker.Message
	expiration map[string][]bool
	close      bool
	mx         sync.Mutex
}

func NewModule() broker.Broker {
	return &Module{
		chat:       map[string][]broker.Message{},
		clients:    map[string][]*chan broker.Message{},
		expiration: map[string][]bool{},
		close:      false,
	}
}

func (m *Module) Close() error {
	m.close = true
	m.mx.Lock()
	defer m.mx.Unlock()
	for _, elements := range m.clients {
		for _, element := range elements {
			close(*element)
		}
	}
	return nil
}

func (m *Module) Publish(ctx context.Context, subject string, msg broker.Message) (int, error) {
	if m.close {
		return 0, broker.ErrUnavailable
	}
	m.mx.Lock()
	m.chat[subject] = append(m.chat[subject], msg)
	m.expiration[subject] = append(m.expiration[subject], false)
	wg := &sync.WaitGroup{}
	for _, n := range m.clients[subject] {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			*n <- msg
			wg.Done()
		}(wg)

	}
	wg.Wait()
	index := len(m.chat[subject]) - 1
	m.mx.Unlock()
	go func() {
		if msg.Expiration == 0 {
			return
		}
		ticker := time.NewTicker(msg.Expiration)
		<-ticker.C
		m.expiration[subject][index] = true
	}()
	return index, nil
}

func (m *Module) Subscribe(ctx context.Context, subject string) (<-chan broker.Message, error) {
	if m.close {
		empty_chan := make(chan broker.Message)
		return empty_chan, broker.ErrUnavailable
	}
	out_chan := make(chan broker.Message, 1000000)
	m.mx.Lock()
	//wg := &sync.WaitGroup{}
	//
	//for _, n := range m.chat[subject]{
	//	wg.Add(1)
	//	go func(wg *sync.WaitGroup) {
	//		out_chan <- n
	//		wg.Done()
	//	}(wg)
	//}
	//wg.Wait()
	m.clients[subject] = append(m.clients[subject], &out_chan)
	m.mx.Unlock()
	return out_chan, nil
}

func (m *Module) Fetch(ctx context.Context, subject string, id int) (broker.Message, error) {
	m.mx.Lock()
	defer m.mx.Unlock()
	msg := broker.Message{}
	if m.close {
		return msg, broker.ErrUnavailable
	}
	if m.expiration[subject][id] {
		return msg, broker.ErrExpiredID
	}
	return m.chat[subject][id], nil
}
