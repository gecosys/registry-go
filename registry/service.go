package registry

import (
	"context"
	fmt "fmt"
	"log"
	"sync"
	"time"

	config "github.com/gecosys/registry-go/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var onceRegister sync.Once
var onceInstance sync.Once
var instance *service

type Registry interface {
	RegisterService(env Environment, name string, conn *Connection)
	GetService(env Environment, name string) (*Connection, error)
}

func Get() Registry {
	if instance != nil {
		return instance
	}

	onceInstance.Do(func() {
		instance = new(service)
	})
	return instance
}

type service struct {
}

func (s *service) RegisterService(env Environment, name string, conn *Connection) {
	onceRegister.Do(func() {
		go s.loopRegisterService(env, name, conn)
	})
}

func (s *service) GetService(env Environment, name string) (*Connection, error) {
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
			Env:  env,
			Name: name,
		},
	)
}

func (s *service) loopRegisterService(env Environment, name string, conn *Connection) {
	var timer = time.NewTimer(0)
	for range timer.C {
		s.doRegisterService(env, name, conn)
		timer.Reset(1 * time.Second)
	}
}

func (s *service) doRegisterService(env Environment, name string, conn *Connection) {
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
				Env:  env,
				Name: name,
			},
			Connection: conn,
		},
	)
}
