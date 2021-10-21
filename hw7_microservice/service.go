package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const (
	bizAdmin = "biz_admin"
	bizUser  = "biz_user"
	logger   = "logger"
)

var aclStorage map[string]json.RawMessage

type service struct {
	m                    *sync.RWMutex
	incomingLogsCh       chan *logMsg
	closeListenersCh     chan struct{}
	listeners            []*listener
	aclStorage           map[string][]string
	statListeners        []*statListener
	incomingStatCh       chan *statMsg
	closeStatListenersCh chan struct{}
}

type logMsg struct {
	methodName   string
	consumerName string
}

type listener struct {
	logsCh  chan *logMsg
	closeCh chan struct{}
}

type statMsg struct {
	methodName   string
	consumerName string
}

type statListener struct {
	statCh  chan *statMsg
	closeCh chan struct{}
}

func getConsumerNameFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", grpc.Errorf(codes.Unauthenticated, "can not get metadata")
	}
	consumer, ok := md["consumer"]
	if !ok || len(consumer) != 1 {
		return "", grpc.Errorf(codes.Unauthenticated, "can not get metadata")
	}

	return consumer[0], nil
}

func (srv *service) checkBizPermission(consumer, method string) error {
	allowedMethods, ok := srv.aclStorage[consumer]
	if !ok {
		return grpc.Errorf(codes.Unauthenticated, "permission denied")
	}

	for _, m := range allowedMethods {
		//check if everything allowed
		splitted := strings.Split(m, "/")
		if len(splitted) == 3 && splitted[2] == "*" {
			return nil
		}

		if m == method {
			return nil
		}
	}

	return grpc.Errorf(codes.Unauthenticated, "permission denied")
}

func parseACL(acl string) (map[string][]string, error) {
	var aclParsed map[string]*json.RawMessage
	result := make(map[string][]string)

	err := json.Unmarshal([]byte(acl), &aclParsed)
	if err != nil {
		return nil, err
	}

	for k, v := range aclParsed {
		var val []string
		err := json.Unmarshal(*v, &val)
		if err != nil {
			return nil, err
		}

		result[k] = val
	}

	return result, nil
}

func (srv *service) addListener(l *listener) {
	srv.m.Lock()
	srv.listeners = append(srv.listeners, l)
	srv.m.Unlock()
}

func (srv *service) logsSender() {
	for {
		select {
		case log := <-srv.incomingLogsCh:
			srv.m.RLock()
			for _, l := range srv.listeners {
				l.logsCh <- log
			}
			srv.m.RUnlock()

		case <-srv.closeListenersCh:
			srv.m.RLock()
			for _, l := range srv.listeners {
				l.closeCh <- struct{}{}
			}
			srv.m.RUnlock()

			return
		}
	}
}

func (srv *service) statsSender() {
	for {
		select {
		case statMsg := <-srv.incomingStatCh:
			srv.m.RLock()
			for _, l := range srv.statListeners {
				l.statCh <- statMsg
			}
			srv.m.RUnlock()

		case <-srv.closeStatListenersCh:
			srv.m.RLock()
			for _, l := range srv.statListeners {
				l.closeCh <- struct{}{}
			}
			srv.m.RUnlock()
			return
		}
	}
}

func (srv *service) addStatListener(sl *statListener) {
	srv.m.Lock()
	srv.statListeners = append(srv.statListeners, sl)
	srv.m.Unlock()
}

func StartMyMicroservice(ctx context.Context, addr, acl string) error {
	aclParsed, err := parseACL(acl)
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Sprintf("can not start the service. %s", err.Error()))
	}

	service := &service{
		m:                    &sync.RWMutex{},
		incomingLogsCh:       make(chan *logMsg, 0),
		listeners:            make([]*listener, 0),
		aclStorage:           aclParsed,
		closeListenersCh:     make(chan struct{}),
		statListeners:        make([]*statListener, 0),
		incomingStatCh:       make(chan *statMsg, 0),
		closeStatListenersCh: make(chan struct{}),
	}

	go service.logsSender()
	go service.statsSender()

	opts := []grpc.ServerOption{grpc.UnaryInterceptor(service.unaryInterceptor),
		grpc.StreamInterceptor(service.streamInterceptor)}

	srv := grpc.NewServer(opts...)
	fmt.Println("starting server at: ", addr)

	RegisterBizServer(srv, service)
	RegisterAdminServer(srv, service)

	go func() {
		select {
		case <-ctx.Done():
			service.closeListenersCh <- struct{}{}

			service.closeStatListenersCh <- struct{}{}

			srv.Stop()
			return
		}
	}()

	go func() {
		err := srv.Serve(lis)
		if err != nil {
			panic(err)
		}
		return
	}()

	return nil
}

func (s *service) unaryInterceptor(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {
	consumer, err := getConsumerNameFromContext(ctx)
	if err != nil {
		return nil, err
	}

	err = s.checkBizPermission(consumer, info.FullMethod)
	if err != nil {
		return nil, err
	}

	logMsg := logMsg{
		consumerName: consumer,
		methodName:   info.FullMethod,
	}

	s.incomingLogsCh <- &logMsg

	statMsg := statMsg{
		consumerName: consumer,
		methodName:   info.FullMethod,
	}

	s.incomingStatCh <- &statMsg

	h, err := handler(ctx, req)
	return h, err
}

func (s *service) streamInterceptor(srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	consumer, err := getConsumerNameFromContext(ss.Context())
	if err != nil {
		return err
	}

	err = s.checkBizPermission(consumer, info.FullMethod)
	if err != nil {
		return err
	}

	if info.FullMethod == "/main.Admin/Logging" {
		msg := logMsg{
			consumerName: consumer,
			methodName:   info.FullMethod,
		}
		s.m.RLock()
		for _, l := range s.listeners {
			l.logsCh <- &msg
		}
		s.m.RUnlock()

	} else {
		msg := statMsg{
			consumerName: consumer,
			methodName:   info.FullMethod,
		}

		s.m.RLock()
		for _, l := range s.statListeners {
			l.statCh <- &msg
		}
		s.m.RUnlock()

	}

	return handler(srv, ss)
}

func (s *service) Check(ctx context.Context, n *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s *service) Add(ctx context.Context, n *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s *service) Test(ctx context.Context, n *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s *service) Logging(nothing *Nothing, srv Admin_LoggingServer) error {

	listener := listener{
		logsCh:  make(chan *logMsg),
		closeCh: make(chan struct{}),
	}
	s.addListener(&listener)

	for {
		select {
		case logMsg := <-listener.logsCh:
			event := &Event{
				Consumer: logMsg.consumerName,
				Method:   logMsg.methodName,
				Host:     "127.0.0.1:8083",
			}
			srv.Send(event)

		case <-listener.closeCh:
			return nil
		}
	}
}

func (s *service) Statistics(interval *StatInterval, srv Admin_StatisticsServer) error {

	closeCh := make(chan struct{})

	ticker := time.NewTicker(time.Second * time.Duration(interval.IntervalSeconds))

	sl := statListener{
		statCh:  make(chan *statMsg, 0),
		closeCh: make(chan struct{}, 0),
	}

	s.addStatListener(&sl)

	c := make(map[string]uint64)
	m := make(map[string]uint64)

	for {
		select {
		case <-ticker.C:
			statEvent := &Stat{
				Timestamp:  0,
				ByMethod:   m,
				ByConsumer: c,
			}

			srv.Send(statEvent)

			c = make(map[string]uint64)
			m = make(map[string]uint64)

		case statMsg := <-sl.statCh:
			_, ok := c[statMsg.consumerName]
			if !ok {
				c[statMsg.consumerName] = 1
			} else {
				c[statMsg.consumerName]++
			}

			_, ok = m[statMsg.methodName]
			if !ok {
				m[statMsg.methodName] = 1
			} else {
				m[statMsg.methodName]++
			}

		case <-closeCh:
			fmt.Println("CLOSED")
			return nil
		}
	}
}
