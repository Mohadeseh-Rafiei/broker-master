package broker

import (
	"context"
	"fmt"
	"github.com/go-redis/redis"
	_ "github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
	"main/pkg/broker"
	"strconv"
	"sync"
	"time"
)

type SubjectInfo struct {
	chats           []broker.Message
	expirations     []bool
	mx              *sync.Mutex
	numberOfClients int
}

type Module struct {
	redisClient *redis.Client
	close       bool
	quit        chan struct{}
	mx          sync.Mutex
	subjectInfo sync.Map
}

func NewModule() broker.Broker {
	client := redis.NewClient(&redis.Options{Addr: ":6379"})
	return &Module{
		redisClient: client,
		close:       false,
		quit:        make(chan struct{}),
	}
}

func (m *Module) createSubjectInfoIfNotExistsAndLock(subject string) *SubjectInfo {
	value, _ := m.subjectInfo.LoadOrStore(subject, &SubjectInfo{mx: &sync.Mutex{}, numberOfClients: 0})
	mtx := value.(*SubjectInfo)
	mtx.mx.Lock()
	return mtx
}

func (m *Module) Close() error {
	m.close = true
	close(m.quit)
	return nil
}

func (m *Module) Publish(ctx context.Context, subject string, msg broker.Message) (int, error) {

	log.Infof("Subject and msg: %v %v", subject, msg)

	if m.close {
		log.WithFields(log.Fields{
			"request": subject,
			"error":   broker.ErrUnavailable,
		}).Error("can't Publish from  topic: ", subject)
		return 0, broker.ErrUnavailable
	}
	subjectInfo := m.createSubjectInfoIfNotExistsAndLock(subject)
	subjectInfo.chats = append(subjectInfo.chats, msg)
	subjectInfo.expirations = append(subjectInfo.expirations, false)

	wg := &sync.WaitGroup{}
	chatId := len(subjectInfo.chats) - 1
	for i := 0; i < subjectInfo.numberOfClients; i++ {
		wg.Add(1)
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()
			data, err := m.redisClient.LPush("musub:"+subject+":"+strconv.Itoa(i), chatId).Result()
			if err != nil {
				log.WithFields(log.Fields{
					"request": subject,
					"error":   err.Error(),
				}).Error("can't publish on topic: ", subject)
				return
			}
			log.Info("published: ", data)
		}(i, wg)
	}
	wg.Wait()
	subjectInfo.mx.Unlock()
	go func() {
		if msg.Expiration == 0 {
			return
		}
		ticker := time.NewTicker(msg.Expiration)
		<-ticker.C
		subjectInfo.expirations[chatId] = true
	}()
	return chatId, nil
}

func (m *Module) Subscribe(ctx context.Context, subject string) (<-chan broker.Message, error) {
	empty_chan := make(chan broker.Message)
	if m.close {
		log.WithFields(log.Fields{
			"request": subject,
			"error":   broker.ErrUnavailable,
		}).Error("can't Subscribe on topic: ", subject)
		return empty_chan, broker.ErrUnavailable
	}
	out_chan := make(chan broker.Message)
	subjectInfo := m.createSubjectInfoIfNotExistsAndLock(subject)
	defer subjectInfo.mx.Unlock()
	queueId := subjectInfo.numberOfClients
	subjectInfo.numberOfClients++

	go func(subjectInfo *SubjectInfo, CLOSE chan struct{}) {
		for {
			select {
			case <-CLOSE:
				close(out_chan)
				fmt.Println("closed chan")
				return
			default:
				str, _ := m.redisClient.BRPop(1*time.Second, "musub:"+subject+":"+strconv.Itoa(queueId)).Result()
				fmt.Println("Rpop result: ", str)
				if len(str) == 0 {
					break
				}
				id, _ := strconv.Atoi(str[1])
				fmt.Println("id: ", id)
				msg := subjectInfo.chats[id]
				out_chan <- msg
			}
		}

	}(subjectInfo, m.quit)
	log.Info("Subscribed...")
	return out_chan, nil
}

func (m *Module) Fetch(ctx context.Context, subject string, id int) (broker.Message, error) {
	msg := broker.Message{}
	subjectInfo := m.createSubjectInfoIfNotExistsAndLock(subject)
	defer subjectInfo.mx.Unlock()
	if m.close {
		log.WithFields(log.Fields{
			"request": subject,
			"error":   broker.ErrUnavailable,
		}).Error("can't fetch from  topic: ", subject)
		return msg, broker.ErrUnavailable
	}
	if subjectInfo.expirations[id] {
		log.WithFields(log.Fields{
			"request": subject,
			"error":   broker.ErrExpiredID,
		}).Error("can't fetch from  topic: ", subject)
		return msg, broker.ErrExpiredID
	}
	return subjectInfo.chats[id], nil
}
