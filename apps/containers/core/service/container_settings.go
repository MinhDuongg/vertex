package service

import (
	"github.com/google/uuid"
	"github.com/vertex-center/vertex/apps/containers/core/port"
	"github.com/vertex-center/vertex/apps/containers/core/types"
)

type ContainerSettingsService struct {
	adapter port.ContainerSettingsAdapter
}

func NewContainerSettingsService(adapter port.ContainerSettingsAdapter) *ContainerSettingsService {
	return &ContainerSettingsService{
		adapter: adapter,
	}
}

func (s *ContainerSettingsService) Save(inst *types.Container, settings types.ContainerSettings) error {
	inst.ContainerSettings = settings
	return s.adapter.Save(inst.UUID, settings)
}

func (s *ContainerSettingsService) Load(inst *types.Container) error {
	settings, err := s.adapter.Load(inst.UUID)
	if err != nil {
		return err
	}
	if settings.DisplayName == "" {
		settings.DisplayName = inst.Service.Name
	}
	inst.ContainerSettings = settings
	return nil
}

func (s *ContainerSettingsService) SetLaunchOnStartup(inst *types.Container, value bool) error {
	inst.ContainerSettings.LaunchOnStartup = &value
	return s.adapter.Save(inst.UUID, inst.ContainerSettings)
}

func (s *ContainerSettingsService) SetDisplayName(inst *types.Container, value string) error {
	inst.ContainerSettings.DisplayName = value
	return s.adapter.Save(inst.UUID, inst.ContainerSettings)
}

func (s *ContainerSettingsService) SetDatabases(inst *types.Container, databases map[string]uuid.UUID) error {
	inst.Databases = databases
	return s.adapter.Save(inst.UUID, inst.ContainerSettings)
}

func (s *ContainerSettingsService) SetVersion(inst *types.Container, value string) error {
	inst.Version = &value
	return s.adapter.Save(inst.UUID, inst.ContainerSettings)
}

func (s *ContainerSettingsService) SetTags(inst *types.Container, tags []string) error {
	inst.Tags = tags
	return s.adapter.Save(inst.UUID, inst.ContainerSettings)
}
