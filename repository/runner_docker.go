package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path"
	"path/filepath"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
	"github.com/vertex-center/vertex/pkg/log"
	"github.com/vertex-center/vertex/pkg/storage"
	"github.com/vertex-center/vertex/types"
	"github.com/vertex-center/vlog"
)

type RunnerDockerRepository struct {
	cli *client.Client
}

type dockerMessage struct {
	Stream string `json:"stream"`
}

func NewRunnerDockerRepository() RunnerDockerRepository {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Default.Warn("couldn't connect with the Docker cli.",
			vlog.String("error", err.Error()),
		)

		return RunnerDockerRepository{}
	}

	return RunnerDockerRepository{
		cli: cli,
	}
}

func (r RunnerDockerRepository) Delete(instance *types.Instance) error {
	id, err := r.getContainerID(*instance)
	if err != nil {
		return err
	}

	return r.cli.ContainerRemove(context.Background(), id, dockertypes.ContainerRemoveOptions{})
}

func (r RunnerDockerRepository) Start(instance *types.Instance, onLog func(msg string), onErr func(msg string), setStatus func(status string)) error {
	imageName := instance.DockerImageName()

	setStatus(types.InstanceStatusBuilding)

	instancePath := r.getPath(*instance)

	// Build
	var err error
	if instance.Methods.Docker.Dockerfile != nil {
		err = r.buildImageFromDockerfile(instancePath, imageName, onLog)
	} else if instance.Methods.Docker.Image != nil {
		err = r.buildImageFromName(*instance.Methods.Docker.Image, onLog)
	} else {
		err = errors.New("no Docker methods found")
	}

	if err != nil {
		onErr(err.Error())
		return err
	}

	// Create
	id, err := r.getContainerID(*instance)
	if errors.Is(err, ErrContainerNotFound) {
		containerName := instance.DockerContainerName()

		log.Default.Info("container doesn't exists, create it.",
			vlog.String("container_name", containerName),
		)

		options := createContainerOptions{
			imageName:     imageName,
			containerName: containerName,
			exposedPorts:  nat.PortSet{},
			portBindings:  nat.PortMap{},
			binds:         []string{},
			env:           []string{},
			capAdd:        []string{},
		}

		// exposedPorts and portBindings
		if instance.Methods.Docker.Ports != nil {
			var all []string

			for in, out := range *instance.Methods.Docker.Ports {
				for _, e := range instance.EnvDefinitions {
					if e.Type == "port" && e.Default == out {
						out = instance.EnvVariables[e.Name]
						all = append(all, out+":"+in)
						break
					}
				}
			}

			var err error
			options.exposedPorts, options.portBindings, err = nat.ParsePortSpecs(all)
			if err != nil {
				return err
			}
		}

		// binds
		if instance.Methods.Docker.Volumes != nil {
			for source, target := range *instance.Methods.Docker.Volumes {
				if !strings.HasPrefix(source, "/") {
					source, err = filepath.Abs(path.Join(instancePath, "volumes", source))
				}
				if err != nil {
					return err
				}
				options.binds = append(options.binds, source+":"+target)
			}
		}

		// env
		if instance.Methods.Docker.Environment != nil {
			for in, out := range *instance.Methods.Docker.Environment {
				value := instance.EnvVariables[out]
				options.env = append(options.env, in+"="+value)
			}
		}

		// capAdd
		if instance.Methods.Docker.Capabilities != nil {
			options.capAdd = *instance.Methods.Docker.Capabilities
		}

		// sysctls
		if instance.Methods.Docker.Sysctls != nil {
			options.sysctls = *instance.Methods.Docker.Sysctls
		}

		if instance.Methods.Docker.Dockerfile != nil {
			id, err = r.createContainer(options)
		} else if instance.Methods.Docker.Image != nil {
			options.imageName = *instance.Methods.Docker.Image
			id, err = r.createContainer(options)
		}
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Start
	err = r.cli.ContainerStart(context.Background(), id, dockertypes.ContainerStartOptions{})
	if err != nil {
		setStatus(types.InstanceStatusError)
		return err
	}
	setStatus(types.InstanceStatusRunning)

	r.watchForLogs(id, instance, onLog)
	r.watchForStatusChange(id, instance, setStatus)

	return nil
}

func (r RunnerDockerRepository) Stop(instance *types.Instance) error {
	id, err := r.getContainerID(*instance)
	if err != nil {
		return err
	}

	return r.cli.ContainerStop(context.Background(), id, container.StopOptions{})
}

func (r RunnerDockerRepository) Info(instance types.Instance) (map[string]any, error) {
	id, err := r.getContainerID(instance)
	if err != nil {
		return nil, err
	}

	info, err := r.cli.ContainerInspect(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":       info.ID,
		"name":     info.Name,
		"image":    info.Image,
		"platform": info.Platform,
	}, nil
}

func (r RunnerDockerRepository) CheckForUpdates(instance *types.Instance) error {
	if instance.Methods.Docker.Image == nil {
		// TODO: Support Dockerfile updates
		return nil
	}

	imageName := *instance.Methods.Docker.Image

	res, err := r.pullImage(imageName)
	if err != nil {
		return err
	}
	defer res.Close()

	imageInfo, _, err := r.cli.ImageInspectWithRaw(context.Background(), imageName)
	if err != nil {
		return err
	}

	latestImageID := imageInfo.ID

	currentImageID, err := r.getImageID(*instance)
	if err != nil {
		return err
	}

	if latestImageID == currentImageID {
		log.Default.Info("already up-to-date",
			vlog.String("uuid", instance.UUID.String()),
		)
		instance.Update = nil
	} else {
		log.Default.Info("a new update is available",
			vlog.String("uuid", instance.UUID.String()),
		)
		instance.Update = &types.InstanceUpdate{
			CurrentVersion: currentImageID,
			LatestVersion:  latestImageID,
		}
	}

	return nil
}

func (r RunnerDockerRepository) HasUpdateAvailable(instance types.Instance) (bool, error) {
	//TODO implement me
	return false, nil
}

func (r RunnerDockerRepository) getContainer(instance types.Instance) (dockertypes.Container, error) {
	containers, err := r.cli.ContainerList(context.Background(), dockertypes.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return dockertypes.Container{}, err
	}

	var dockerContainer *dockertypes.Container

	for _, c := range containers {
		name := c.Names[0]
		if name == "/"+instance.DockerContainerName() {
			dockerContainer = &c
			break
		}
	}

	if dockerContainer == nil {
		return dockertypes.Container{}, ErrContainerNotFound
	}

	return *dockerContainer, nil
}

func (r RunnerDockerRepository) getContainerID(instance types.Instance) (string, error) {
	c, err := r.getContainer(instance)
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

func (r RunnerDockerRepository) getImageID(instance types.Instance) (string, error) {
	c, err := r.getContainer(instance)
	if err != nil {
		return "", err
	}
	return c.ImageID, nil
}

func (r RunnerDockerRepository) pullImage(imageName string) (io.ReadCloser, error) {
	res, err := r.cli.ImagePull(context.Background(), imageName, dockertypes.ImagePullOptions{})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r RunnerDockerRepository) buildImageFromName(imageName string, onMsg func(msg string)) error {
	res, err := r.pullImage(imageName)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(res)
	for scanner.Scan() {
		if scanner.Err() != nil {
			return scanner.Err()
		}
		onMsg(scanner.Text())
	}

	return nil
}

func (r RunnerDockerRepository) buildImageFromDockerfile(instancePath string, imageName string, onMsg func(msg string)) error {
	buildOptions := dockertypes.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{imageName},
		Remove:     true,
	}

	reader, err := archive.TarWithOptions(instancePath, &archive.TarOptions{
		ExcludePatterns: []string{".git/**/*"},
	})
	if err != nil {
		return err
	}

	res, err := r.cli.ImageBuild(context.Background(), reader, buildOptions)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		if scanner.Err() != nil {
			return scanner.Err()
		}
		msg := dockerMessage{}
		err := json.Unmarshal(scanner.Bytes(), &msg)
		if err != nil {
			log.Default.Warn("Failed to parse message",
				vlog.String("message", scanner.Text()),
			)
		} else {
			if msg.Stream != "" {
				onMsg(msg.Stream)
			}
		}
	}

	log.Default.Info("Docker build: success.")
	return nil
}

type createContainerOptions struct {
	imageName     string
	containerName string
	exposedPorts  nat.PortSet
	portBindings  nat.PortMap
	binds         []string
	env           []string
	capAdd        []string
	sysctls       map[string]string
}

func (r RunnerDockerRepository) createContainer(options createContainerOptions) (string, error) {
	config := container.Config{
		Image:        options.imageName,
		ExposedPorts: options.exposedPorts,
		Env:          options.env,
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
	}

	hostConfig := container.HostConfig{
		Binds:        options.binds,
		PortBindings: options.portBindings,
		CapAdd:       options.capAdd,
		Sysctls:      options.sysctls,
	}

	res, err := r.cli.ContainerCreate(context.Background(), &config, &hostConfig, nil, nil, options.containerName)
	for _, warn := range res.Warnings {
		log.Default.Warn("warning while creating container",
			vlog.String("warning", warn),
		)
	}
	return res.ID, err
}

func (r RunnerDockerRepository) watchForStatusChange(containerID string, instance *types.Instance, setStatus func(status string)) {
	go func() {
		resChan, errChan := r.cli.ContainerWait(context.Background(), containerID, container.WaitConditionNotRunning)

		select {
		case err := <-errChan:
			if err != nil {
				log.Default.Error(err,
					vlog.String("uuid", instance.UUID.String()),
				)
			}
		case status := <-resChan:
			log.Default.Info("container exited",
				vlog.String("uuid", instance.UUID.String()),
				vlog.Int64("status", status.StatusCode),
			)
			setStatus(types.InstanceStatusOff)
		}
	}()
}

func (r RunnerDockerRepository) watchForLogs(containerID string, instance *types.Instance, onLog func(msg string)) {
	go func() {
		logs, err := r.cli.ContainerLogs(context.Background(), containerID, dockertypes.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: false,
			Follow:     true,
			Tail:       "0",
		})
		if err != nil {
			log.Default.Error(err,
				vlog.String("uuid", instance.UUID.String()),
			)
		}

		scanner := bufio.NewScanner(logs)
		for scanner.Scan() {
			onLog(scanner.Text())
		}
		_ = logs.Close()
		log.Default.Info("logs pipe closed",
			vlog.String("uuid", instance.UUID.String()),
		)
	}()
}

func (r RunnerDockerRepository) getPath(instance types.Instance) string {
	return path.Join(storage.Path, "instances", instance.UUID.String())
}
