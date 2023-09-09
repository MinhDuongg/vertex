package repository

import (
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/vertex-center/vertex/pkg/log"
	"github.com/vertex-center/vertex/pkg/storage"
	"github.com/vertex-center/vertex/types"
)

const bufferSize = 50

var (
	ErrLoggerNotFound = errors.New("instance logger not found")
)

type InstanceLogger struct {
	file *os.File

	buffer []types.LogLine

	currentLine int
}

type InstanceLogsFSRepository struct {
	loggers map[uuid.UUID]*InstanceLogger
}

func NewInstanceLogsFSRepository() InstanceLogsFSRepository {
	r := InstanceLogsFSRepository{
		loggers: map[uuid.UUID]*InstanceLogger{},
	}
	r.startCron()
	return r
}

func (r *InstanceLogsFSRepository) Open(uuid uuid.UUID) error {
	dir := path.Join(storage.Path, "instances", uuid.String(), ".vertex", "logs")
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("logs_%s.txt", time.Now().Format(time.DateOnly))
	filepath := path.Join(dir, filename)

	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}

	l := InstanceLogger{
		buffer: []types.LogLine{},
	}
	l.file = file

	r.loggers[uuid] = &l
	return nil
}

func (r *InstanceLogsFSRepository) Close(uuid uuid.UUID) error {
	l, err := r.getLogger(uuid)
	if err != nil {
		return err
	}
	return l.Close()
}

func (r *InstanceLogsFSRepository) Push(uuid uuid.UUID, line types.LogLine) {
	l, err := r.getLogger(uuid)
	if err != nil {
		log.Default.Error(err)
		return
	}
	l.currentLine += 1
	l.buffer = append(l.buffer, line)
	if len(l.buffer) > bufferSize {
		l.buffer = l.buffer[1:]
	}

	_, err = fmt.Fprintf(l.file, "%s\n", line.Message)
	if err != nil {
		log.Default.Error(err)
	}
}

func (r *InstanceLogsFSRepository) CloseAll() error {
	var errs []error
	for id := range r.loggers {
		err := r.Close(id)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *InstanceLogsFSRepository) LoadBuffer(uuid uuid.UUID) ([]types.LogLine, error) {
	l, err := r.getLogger(uuid)
	if err != nil {
		return nil, err
	}
	return l.buffer, nil
}

func (l *InstanceLogger) Close() error {
	return l.file.Close()
}

func (r *InstanceLogsFSRepository) getLogger(uuid uuid.UUID) (*InstanceLogger, error) {
	l, ok := r.loggers[uuid]
	if !ok {
		return nil, ErrLoggerNotFound
	}
	return l, nil
}

func (r *InstanceLogsFSRepository) startCron() {
	s := gocron.NewScheduler(time.Local)
	_, err := s.Every(1).Day().At("00:00").Do(func() {
		for id := range r.loggers {
			err := r.Close(id)
			if err != nil {
				log.Default.Error(err)
				continue
			}
			err = r.Open(id)
			if err != nil {
				log.Default.Error(err)
			}
		}
	})
	if err != nil {
		log.Default.Error(err)
		return
	}
	s.StartAsync()
}
