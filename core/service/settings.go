package service

import (
	"github.com/vertex-center/vertex/core/port"
	"github.com/vertex-center/vertex/core/types"
)

type SettingsService struct {
	settingsAdapter port.SettingsAdapter
}

func NewSettingsService(settingsAdapter port.SettingsAdapter) port.SettingsService {
	return &SettingsService{
		settingsAdapter: settingsAdapter,
	}
}

func (s *SettingsService) Get() types.Settings {
	return s.settingsAdapter.GetSettings()
}

func (s *SettingsService) Update(settings types.Settings) error {
	if settings.Notifications != nil {
		notifs := settings.Notifications
		if notifs.Webhook != nil {
			err := s.settingsAdapter.SetNotificationsWebhook(*notifs.Webhook)
			if err != nil {
				return err
			}
		}
	}

	if settings.Updates != nil {
		updates := settings.Updates
		if updates.Channel != nil {
			err := s.settingsAdapter.SetChannel(*updates.Channel)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SettingsService) GetNotificationsWebhook() *string {
	return s.settingsAdapter.GetNotificationsWebhook()
}

func (s *SettingsService) SetNotificationsWebhook(webhook string) error {
	return s.settingsAdapter.SetNotificationsWebhook(webhook)
}

func (s *SettingsService) GetChannel() types.SettingsUpdatesChannel {
	channel := s.settingsAdapter.GetChannel()
	if channel == nil {
		return types.SettingsUpdatesChannelStable
	}
	return *channel
}

func (s *SettingsService) SetChannel(channel types.SettingsUpdatesChannel) error {
	return s.settingsAdapter.SetChannel(channel)
}
