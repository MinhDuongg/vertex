package services

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/vertex-center/vertex/logger"
	"github.com/vertex-center/vertex/repository"
	"github.com/vertex-center/vertex/storage"
	"github.com/vertex-center/vertex/types"
)

var (
	ErrContainerStillRunning  = errors.New("the container is still running")
	ErrInstanceAlreadyRunning = errors.New("the instance is already running")
	ErrInstanceNotRunning     = errors.New("the instance is not running")
)

type InstanceService struct {
	repo       repository.InstanceRepository
	dockerRepo repository.DockerRepository
}

func NewInstanceService() InstanceService {
	return InstanceService{
		repo:       repository.NewInstanceRepository(),
		dockerRepo: repository.NewDockerRepository(),
	}
}

func (s *InstanceService) Unload() {
	s.repo.Unload()
}

func (s *InstanceService) Get(uuid uuid.UUID) (*types.Instance, error) {
	return s.repo.Get(uuid)
}

func (s *InstanceService) GetAll() map[uuid.UUID]*types.Instance {
	return s.repo.GetAll()
}

func (s *InstanceService) Delete(uuid uuid.UUID) error {
	i, err := s.repo.Get(uuid)
	if err != nil {
		return err
	}

	if i.IsRunning() {
		return ErrContainerStillRunning
	}

	if i.UseDocker {
		containerID, err := s.dockerRepo.GetContainerID(i.DockerContainerName())
		if err == repository.ErrContainerNotFound {
			logger.Warn(err.Error()).Print()
		} else if err != nil {
			return err
		} else {
			err = s.dockerRepo.RemoveContainer(containerID)
			if err != nil {
				return err
			}
		}
	}

	return s.repo.Delete(uuid)
}

func (s *InstanceService) AddListener(channel chan types.InstanceEvent) uuid.UUID {
	return s.repo.AddListener(channel)
}

func (s *InstanceService) RemoveListener(uuid uuid.UUID) {
	s.repo.RemoveListener(uuid)
}

func (s *InstanceService) Start(uuid uuid.UUID) error {
	i, err := s.repo.Get(uuid)
	if err != nil {
		return err
	}

	s.repo.WriteLogLine(i, &types.LogLine{
		Kind:    types.LogKindVertexOut,
		Message: "Starting instance...",
	})

	logger.Log("starting instance").
		AddKeyValue("uuid", uuid).
		Print()

	if i.IsRunning() {
		s.repo.WriteLogLine(i, &types.LogLine{
			Kind:    types.LogKindVertexErr,
			Message: ErrInstanceAlreadyRunning.Error(),
		})
		return ErrInstanceAlreadyRunning
	}

	if i.UseDocker {
		err = s.startWithDocker(i)
	} else {
		err = s.startManually(i)
	}

	if err != nil {
		i.SetStatus(types.InstanceStatusError)
	} else {
		s.repo.WriteLogLine(i, &types.LogLine{
			Kind:    types.LogKindVertexOut,
			Message: "Instance started.",
		})

		logger.Log("instance started").
			AddKeyValue("uuid", uuid).
			Print()
	}

	return err
}

func (s *InstanceService) Stop(uuid uuid.UUID) error {
	i, err := s.repo.Get(uuid)
	if err != nil {
		return err
	}

	s.repo.WriteLogLine(i, &types.LogLine{
		Kind:    types.LogKindVertexOut,
		Message: "Stopping instance...",
	})

	logger.Log("stopping instance").
		AddKeyValue("uuid", uuid).
		Print()

	if !i.IsRunning() {
		s.repo.WriteLogLine(i, &types.LogLine{
			Kind:    types.LogKindVertexErr,
			Message: ErrInstanceNotRunning.Error(),
		})
		return ErrInstanceNotRunning
	}

	if i.UseDocker {
		err = s.stopWithDocker(i)
	} else {
		err = s.stopManually(i)
	}

	if err == nil {
		s.repo.WriteLogLine(i, &types.LogLine{
			Kind:    types.LogKindVertexOut,
			Message: "Instance stopped.",
		})

		logger.Log("instance stopped").
			AddKeyValue("uuid", uuid).
			Print()

		i.SetStatus(types.InstanceStatusOff)
	}

	return err
}

func (s *InstanceService) startWithDocker(i *types.Instance) error {
	imageName := i.DockerImageName()
	containerName := i.DockerContainerName()

	i.SetStatus(types.InstanceStatusBuilding)

	instancePath := s.repo.GetPath(i)

	// Build
	err := s.dockerRepo.BuildImage(instancePath, imageName)
	if err != nil {
		return err
	}

	// Create
	id, err := s.dockerRepo.GetContainerID(containerName)
	if err == repository.ErrContainerNotFound {
		logger.Log("container doesn't exists, create it.").
			AddKeyValue("container_name", containerName).
			Print()

		id, err = s.dockerRepo.CreateContainer(imageName, containerName)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	i.SetStatus(types.InstanceStatusStarting)

	// Start
	err = s.dockerRepo.StartContainer(id)
	if err != nil {
		return err
	}

	i.SetStatus(types.InstanceStatusRunning)
	return nil
}

func (s *InstanceService) startManually(i *types.Instance) error {
	if i.Cmd != nil {
		logger.Error(errors.New("runner already started")).
			AddKeyValue("name", i.Name).
			Print()
	}

	dir := s.repo.GetPath(i)
	executable := i.ID
	command := "./" + i.ID

	// Try to find the executable
	// For a service of ID=vertex-id, the executable can be:
	// - vertex-id
	// - vertex-id.sh
	_, err := os.Stat(path.Join(dir, executable))
	if errors.Is(err, os.ErrNotExist) {
		_, err = os.Stat(path.Join(dir, executable+".sh"))
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("the executable %s (or %s.sh) was not found at path", i.ID, i.ID)
		} else if err != nil {
			return err
		}
		command = fmt.Sprintf("./%s.sh", i.ID)
	} else if err != nil {
		return err
	}

	i.Cmd = exec.Command(command)
	i.Cmd.Dir = dir

	i.Cmd.Stdin = os.Stdin

	stdoutReader, err := i.Cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderrReader, err := i.Cmd.StderrPipe()
	if err != nil {
		return err
	}

	stdoutScanner := bufio.NewScanner(stdoutReader)
	go func() {
		for stdoutScanner.Scan() {
			s.repo.WriteLogLine(i, &types.LogLine{
				Kind:    types.LogKindOut,
				Message: stdoutScanner.Text(),
			})
		}
	}()

	stderrScanner := bufio.NewScanner(stderrReader)
	go func() {
		for stderrScanner.Scan() {
			s.repo.WriteLogLine(i, &types.LogLine{
				Kind:    types.LogKindErr,
				Message: stderrScanner.Text(),
			})
		}
	}()

	i.SetStatus(types.InstanceStatusRunning)

	err = i.Cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		err := i.Cmd.Wait()
		if err != nil {
			logger.Error(err).
				AddKeyValue("name", i.Service.Name).
				Print()
		}
		i.SetStatus(types.InstanceStatusOff)
	}()

	return nil
}

func (s *InstanceService) stopWithDocker(i *types.Instance) error {
	id, err := s.dockerRepo.GetContainerID(i.DockerContainerName())
	if err != nil {
		return err
	}

	return s.dockerRepo.StopContainer(id)
}

func (s *InstanceService) stopManually(i *types.Instance) error {
	err := i.Cmd.Process.Signal(os.Interrupt)
	if err != nil {
		return err
	}

	// TODO: Force kill if the process continues

	i.Cmd = nil

	return nil
}

func (s *InstanceService) WriteEnv(uuid uuid.UUID, environment map[string]string) error {
	i, err := s.Get(uuid)
	if err != nil {
		return err
	}

	return s.repo.WriteEnv(i, environment)
}

func (s *InstanceService) Install(repo string, useDocker *bool, useReleases *bool) (*types.Instance, error) {
	serviceUUID := uuid.New()
	basePath := path.Join(storage.PathInstances, serviceUUID.String())

	forceClone := (useDocker != nil && *useDocker) || (useReleases == nil || !*useReleases)

	var err error
	if strings.HasPrefix(repo, "marketplace:") {
		err = s.repo.Download(basePath, repo, forceClone)
	} else if strings.HasPrefix(repo, "localstorage:") {
		err = s.repo.Symlink(basePath, repo)
	} else if strings.HasPrefix(repo, "git:") {
		err = s.repo.Download(basePath, repo, forceClone)
	} else {
		return nil, fmt.Errorf("this protocol is not supported")
	}

	if err != nil {
		return nil, err
	}

	i, err := s.repo.Instantiate(serviceUUID)
	if err != nil {
		return nil, err
	}

	if useDocker != nil {
		i.InstanceMetadata.UseDocker = *useDocker
	}
	if useReleases != nil {
		i.InstanceMetadata.UseReleases = *useReleases
	}

	err = s.repo.SaveMetadata(i)
	if err != nil {
		return nil, err
	}

	return i, nil
}
