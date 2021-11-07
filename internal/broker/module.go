package broker

import (
	"context"
	"fmt"
	"github.com/go-redis/redis"
	_ "github.com/go-redis/redis"
	"main/pkg/broker"
	"strconv"
	"sync"
	"time"
)

// salam ziba
type Module struct {
	chat       map[string][]broker.Message
	clients    map[string][]*chan broker.Message
	expiration map[string][]bool
	*redis.Client
	close          bool
	mx             sync.Mutex
	mx_for_subject map[string]*sync.Mutex
}

func NewModule() broker.Broker {
	client := redis.NewClient(&redis.Options{Addr: ":6379"})
	return &Module{
		chat:           map[string][]broker.Message{},
		clients:        map[string][]*chan broker.Message{},
		expiration:     map[string][]bool{},
		mx_for_subject: map[string]*sync.Mutex{},
		Client:         client,
		close:          false,
	}
}

func (m *Module) createSubjectMutexIfNotExists(subject string) {
	m.mx.Lock()
	if _, ok := m.mx_for_subject[subject]; !ok {
		m.mx_for_subject[subject] = &sync.Mutex{}
	}
	m.mx.Unlock()
}

func (m *Module) Close() error {
	m.mx.Lock()
	m.close = true
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
	m.createSubjectMutexIfNotExists(subject)
	m.mx_for_subject[subject].Lock()
	m.chat[subject] = append(m.chat[subject], msg)
	wg := &sync.WaitGroup{}
	m.expiration[subject] = append(m.expiration[subject], false)
	index := len(m.chat[subject]) - 1
	for i, _ := range m.clients[subject] {
		wg.Add(1)
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()
			hi, err := m.Client.LPush(subject+":"+strconv.Itoa(i), index).Result()
			fmt.Println("Lpush res: ", hi, err)
		}(i, wg)
	}
	wg.Wait()
	m.mx_for_subject[subject].Unlock()
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
	empty_chan := make(chan broker.Message)
	if m.close {
		return empty_chan, broker.ErrUnavailable
	}
	m.createSubjectMutexIfNotExists(subject)
	out_chan := make(chan broker.Message, 1000000)
	m.mx_for_subject[subject].Lock()
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
	index := len(m.clients[subject]) - 1
	go func() {
		for {
			str, _ := m.Client.BRPop(1000000000, subject+":"+strconv.Itoa(index)).Result()
			fmt.Println("str: ", str)
			if len(str) == 0 {
				break
			}
			id, _ := strconv.Atoi(str[1])
			fmt.Println("id: ", id)
			msg := m.chat[subject][id]
			out_chan <- msg
		}

	}()
	m.mx_for_subject[subject].Unlock()
	return out_chan, nil
}

func (m *Module) Fetch(ctx context.Context, subject string, id int) (broker.Message, error) {
	msg := broker.Message{}
	if _, ok := m.chat[subject]; !ok {
		return msg, broker.ErrUnavailable
	}
	m.mx_for_subject[subject].Lock()
	defer m.mx_for_subject[subject].Unlock()
	if m.close {
		return msg, broker.ErrUnavailable
	}
	if m.expiration[subject][id] {
		return msg, broker.ErrExpiredID
	}
	return m.chat[subject][id], nil
}
