package repository

import (
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/vertex-center/vertex/pkg/logger"
	"github.com/vertex-center/vertex/pkg/storage"
	"github.com/vertex-center/vertex/types"
)

type SettingsFSRepository struct {
	settingsPath string
	settings     types.Settings
}

type SettingsRepositoryParams struct {
	settingsPath string
}

func NewSettingsFSRepository(params *SettingsRepositoryParams) SettingsFSRepository {
	if params == nil {
		params = &SettingsRepositoryParams{}
	}
	if params.settingsPath == "" {
		params.settingsPath = path.Join(storage.Path, "settings")
	}

	err := os.MkdirAll(params.settingsPath, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		logger.Error(err).
			AddKeyValue("message", "failed to create directory").
			AddKeyValue("path", params.settingsPath).
			Print()
		os.Exit(1)
	}

	repo := SettingsFSRepository{
		settingsPath: params.settingsPath,
	}

	err = repo.read()
	if err != nil {
		logger.Error(err).Print()
	}

	return repo
}

func (r *SettingsFSRepository) GetSettings() types.Settings {
	return r.settings
}

func (r *SettingsFSRepository) GetNotificationsWebhook() *string {
	if r.settings.Notifications == nil {
		return nil
	}
	return r.settings.Notifications.Webhook
}

func (r *SettingsFSRepository) SetNotificationsWebhook(webhook *string) error {
	if r.settings.Notifications == nil {
		r.settings.Notifications = &types.SettingsNotifications{}
	}
	r.settings.Notifications.Webhook = webhook
	return r.write()
}

func (r *SettingsFSRepository) read() error {
	p := path.Join(r.settingsPath, "settings.json")
	file, err := os.ReadFile(p)

	if errors.Is(err, os.ErrNotExist) {
		logger.Log("settings.json doesn't exists or could not be found").Print()
		return nil
	} else if err != nil {
		return err
	}

	err = json.Unmarshal(file, &r.settings)
	if err != nil {
		return err
	}

	return nil
}

func (r *SettingsFSRepository) write() error {
	p := path.Join(r.settingsPath, "settings.json")

	bytes, err := json.MarshalIndent(r.settings, "", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(p, bytes, os.ModePerm)
}