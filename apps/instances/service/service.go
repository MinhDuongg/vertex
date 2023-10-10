package service

import (
	"github.com/vertex-center/vertex/apps/instances/adapter"
	"github.com/vertex-center/vertex/apps/instances/types"
	"github.com/vertex-center/vertex/pkg/log"
	vtypes "github.com/vertex-center/vertex/types"
)

type ServiceService struct {
	serviceAdapter types.ServiceAdapterPort
}

func NewServiceService() *ServiceService {
	return &ServiceService{
		serviceAdapter: adapter.NewServiceFSAdapter(nil),
	}
}

func (s *ServiceService) GetById(id string) (types.Service, error) {
	return s.serviceAdapter.Get(id)
}

func (s *ServiceService) GetAll() []types.Service {
	return s.serviceAdapter.GetAll()
}

func (s *ServiceService) Reload() error {
	return s.serviceAdapter.Reload()
}

func (s *ServiceService) OnEvent(e interface{}) {
	switch e.(type) {
	case vtypes.EventDependenciesUpdated:
		err := s.Reload()
		if err != nil {
			log.Error(err)
			return
		}
	}
}
