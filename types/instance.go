package types

import (
	"os/exec"

	"github.com/google/uuid"
	"github.com/vertex-center/vertex/logger"
)

const (
	InstanceStatusOff      = "off"
	InstanceStatusBuilding = "building"
	InstanceStatusStarting = "starting"
	InstanceStatusRunning  = "running"
	InstanceStatusError    = "error"

	InstanceEventStatusChange = "status_change"
)

type InstanceMetadata struct {
	UseDocker   bool `json:"use_docker"`
	UseReleases bool `json:"use_releases"`
}

type InstanceEvent struct {
	Name string
	Data string
}

type Instance struct {
	Service
	InstanceMetadata

	Status       string       `json:"status"`
	Logs         Logs         `json:"logs"`
	EnvVariables EnvVariables `json:"env"`

	UUID uuid.UUID `json:"uuid"`
	Cmd  *exec.Cmd `json:"-"`

	Listeners map[uuid.UUID]chan InstanceEvent `json:"-"`
}

type InstanceRepository interface {
	Get(uuid uuid.UUID) (*Instance, error)
	GetAll() map[uuid.UUID]*Instance
	Delete(uuid uuid.UUID) error
	Exists(uuid uuid.UUID) bool
	Create(uuid uuid.UUID, i *Instance)

	AddListener(channel chan InstanceEvent) uuid.UUID
	RemoveListener(uuid uuid.UUID)
}

func (i *Instance) DockerImageName() string {
	return "vertex_image_" + i.UUID.String()
}

func (i *Instance) DockerContainerName() string {
	return "VERTEX_CONTAINER_" + i.UUID.String()
}

func (i *Instance) IsRunning() bool {
	return i.Status != InstanceStatusOff && i.Status != InstanceStatusError
}

func (i *Instance) Register(channel chan InstanceEvent) uuid.UUID {
	id := uuid.New()
	i.Listeners[id] = channel

	logger.Log("registered to instance").
		AddKeyValue("channel", id).
		AddKeyValue("instance_uuid", i.UUID).
		Print()

	return id
}

func (i *Instance) Unregister(uuid uuid.UUID) {
	delete(i.Listeners, uuid)

	logger.Log("unregistered from instance").
		AddKeyValue("channel", uuid).
		AddKeyValue("instance_uuid", i.UUID).
		Print()
}

func (i *Instance) SetStatus(status string) {
	i.Status = status

	for _, listener := range i.Listeners {
		listener <- InstanceEvent{
			Name: InstanceEventStatusChange,
			Data: status,
		}
	}
}

func (i *Instance) NotifyListeners(event InstanceEvent) {
	for _, listener := range i.Listeners {
		listener <- event
	}
}
