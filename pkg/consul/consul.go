package consul

import (
	"encoding/json"
	"fmt"
	"github.com/docker/go-units"
	"github.com/elastic/go-sysinfo"
	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/hashicorp/consul/api"
	"golang.org/x/xerrors"
	"os"
	"strings"
	"time"
)

const (
	CONSUL_RUL = "CONSUL_RUL"
)

type ServiceDiscovery struct {
	ServerName   string
	ListenAddr   string
	ListenPort   uint64
	Interval     uint64
	DeRegister   uint64
	ConsulUrl    string
	EndPoint     string
	consulClient *api.Client
}

func NewServiceDiscovery(serverName, listenAddr string, listenPort, interval, deRegister uint64, endPoint, consulUrl string) (*ServiceDiscovery, error) {
	sd := &ServiceDiscovery{
		ServerName: serverName,
		ListenAddr: listenAddr,
		ListenPort: listenPort,
		Interval:   interval,
		DeRegister: deRegister,
		EndPoint:   endPoint,
		ConsulUrl:  consulUrl,
	}
	if err := sd.InitConsul(); err != nil {
		return nil, err
	}
	return sd, nil
}

func (s *ServiceDiscovery) InitConsul() error {

	if s.ConsulUrl == "" {
		url := os.Getenv(CONSUL_RUL)
		if url == "" {
			panic("CONSUL_RUL env is required")
		}
		s.ConsulUrl = url
	}
	var err error
	s.consulClient, err = api.NewClient(&api.Config{
		Address: s.ConsulUrl})
	if err != nil {
		return err
	}
	return nil
}

func (s *ServiceDiscovery) ServiceRegistr() error {
	agent := s.consulClient.Agent()
	interval := time.Duration(s.Interval) * time.Second
	deRegister := time.Duration(s.DeRegister) * time.Minute

	var check *api.AgentServiceCheck

	check = &api.AgentServiceCheck{ // 健康检查
		Interval:                       interval.String(),                                                      // 健康检查间隔
		HTTP:                           fmt.Sprintf("http://%s:%d/%s", s.ListenAddr, s.ListenPort, s.EndPoint), // grpc 支持，执行健康检查的地址，service 会传到 Health.Check 函数中
		DeregisterCriticalServiceAfter: deRegister.String(),                                                    // 注销时间，相当于过期时间
		//Status:                         "passing",

	}

	mate := map[string]string{}
	hostname, err := os.Hostname()
	if err != nil {
		return xerrors.Errorf("os get hostname failed: %+v", err)

	}
	mate["hostname"] = hostname

	gpus, err := ffi.GetGPUDevices()
	if err != nil {
		return xerrors.Errorf("getting gpu devices failed: %+v", err)
	}
	mate["gpus"] = strings.Join(gpus, ",")
	h, err := sysinfo.Host()
	if err != nil {
		return xerrors.Errorf("getting host info: %w", err)
	}

	mem, err := h.Memory()
	if err != nil {
		return xerrors.Errorf("getting memory info: %w", err)
	}
	mate["mem"] = units.HumanSize(float64(mem.Total))

	reg := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("%s-%s-%d", s.ServerName, s.ListenAddr, s.ListenPort), // 服务节点的名称
		Name:    s.ServerName,                                                      // 服务名称
		Tags:    []string{s.ServerName},                                            // tag，可以为空
		Port:    int(s.ListenPort),                                                 // 服务端口
		Address: s.ListenAddr,                                                      // 服务 IP
		Check:   check,
		Meta:    mate,
	}

	if err := agent.ServiceRegister(reg); err != nil {
		return err
	}
	return nil
}

func (s *ServiceDiscovery) ServiceDeregister() error {
	agent := s.consulClient.Agent()
	if err := agent.ServiceDeregister(fmt.Sprintf("%s-%s-%d", s.ServerName, s.ListenAddr, s.ListenPort)); err != nil {
		return err
	}
	return nil
}

func (s *ServiceDiscovery) GetKey(key string, result interface{}) error {
	kv := s.consulClient.KV()
	kvPair, _, err := kv.Get(key, nil)
	if err != nil {
		return err
	}
	if len(kvPair.Value) == 0 {
		return fmt.Errorf("kvPair value len is 0")
	}
	if err := json.Unmarshal(kvPair.Value, &result); err != nil {
		return err
	}
	return nil
}

func (s *ServiceDiscovery) PutKey(key string, val []byte) error {
	kv := s.consulClient.KV()
	if _, err := kv.Put(&api.KVPair{Key: key, Value: val}, nil); err != nil {
		return err
	}
	return nil
}