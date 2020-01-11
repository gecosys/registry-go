package registry

import (
	"context"
	fmt "fmt"
	"log"
	"sync/atomic"
	"time"

	config "github.com/gecosys/registry-go/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var mtxRegistryFlag uint32
var instance *service

type Registry interface {
	RegisterService(code string, env Environment, name string, conn *Connection)
	GetService(code string, env Environment, name string) (*Connection, error)
}

func Get() Registry {
	if atomic.SwapUint32(&mtxRegistryFlag, 1) == 1 {
		return instance
	}
	instance = new(service)
	instance.services = make(map[string]chan bool)
	return instance
}

type service struct {
	services map[string]chan bool
}

func (s *service) RegisterService(code string, env Environment, name string, conn *Connection) {
	key := fmt.Sprintf("%s_%s_%s", code, env, name)
	done, ok := s.services[key]
	if ok {
		done <- true
	} else {
		done = make(chan bool)
		s.services[key] = done
	}
	go s.loopRegisterService(code, env, name, conn, done)
}

func (s *service) GetService(code string, env Environment, name string) (*Connection, error) {
	creds, err := credentials.NewClientTLSFromFile("keys/registry-server.crt", "")
	if err != nil {
		log.Fatal(err)
	}

	dial, err := grpc.Dial(config.Get().Registry.Address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	defer dial.Close()

	return NewRegistryClient(dial).GetService(
		context.Background(),
		&Service{
			Code: code,
			Env:  env,
			Name: name,
		},
	)
}

func (s *service) loopRegisterService(code string, env Environment, name string, conn *Connection, done chan bool) {
	var timer = time.NewTimer(0)
	for {
		select {
		case <-done:
			return
		case <-timer.C:
			s.doRegisterService(code, env, name, conn)
			timer.Reset(1 * time.Second)
		}
	}
}

func (s *service) doRegisterService(code string, env Environment, name string, conn *Connection) {
	var (
		err  error
		dial *grpc.ClientConn
	)

	defer func() {
		if err != nil {
			fmt.Println(err)
		}
	}()

	creds, err := credentials.NewClientTLSFromFile("keys/registry-server.crt", "")
	if err != nil {
		log.Fatal(err)
	}

	dial, err = grpc.Dial(config.Get().Registry.Address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return
	}
	defer dial.Close()

	_, err = NewRegistryClient(dial).RegisterService(
		context.Background(),
		&RegistrationForm{
			Service: &Service{
				Code: code,
				Env:  env,
				Name: name,
			},
			Connection: conn,
		},
	)
}
