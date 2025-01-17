package types

import "github.com/vertex-center/vertex/pkg/router"

const (
	ErrCodeContainerUuidInvalid           router.ErrCode = "container_uuid_invalid"
	ErrCodeContainerUuidMissing           router.ErrCode = "container_uuid_missing"
	ErrCodeContainerNotFound              router.ErrCode = "container_not_found"
	ErrCodeContainerAlreadyRunning        router.ErrCode = "container_already_running"
	ErrCodeContainerStillRunning          router.ErrCode = "container_still_running"
	ErrCodeContainerNotRunning            router.ErrCode = "container_not_running"
	ErrCodeFailedToGetContainer           router.ErrCode = "failed_to_get_container"
	ErrCodeFailedToStartContainer         router.ErrCode = "failed_to_start_container"
	ErrCodeFailedToStopContainer          router.ErrCode = "failed_to_stop_container"
	ErrCodeFailedToDeleteContainer        router.ErrCode = "failed_to_delete_container"
	ErrCodeFailedToGetContainerLogs       router.ErrCode = "failed_to_get_logs"
	ErrCodeFailedToUpdateServiceContainer router.ErrCode = "failed_to_update_service_container"
	ErrCodeFailedToGetVersions            router.ErrCode = "failed_to_get_versions"
	ErrCodeFailedToWaitContainer          router.ErrCode = "failed_to_wait_container"
	ErrCodeFailedToSetLaunchOnStartup     router.ErrCode = "failed_to_set_launch_on_startup"
	ErrCodeFailedToSetDisplayName         router.ErrCode = "failed_to_set_display_name"
	ErrCodeFailedToSetDatabase            router.ErrCode = "failed_to_set_database"
	ErrCodeFailedToSetVersion             router.ErrCode = "failed_to_set_version"
	ErrCodeFailedToSetTags                router.ErrCode = "failed_to_set_tags"
	ErrCodeFailedToSetEnv                 router.ErrCode = "failed_to_set_env"
	ErrCodeFailedToCheckForUpdates        router.ErrCode = "failed_to_check_for_updates"

	ErrCodeServiceIdMissing       router.ErrCode = "service_id_missing"
	ErrCodeServiceNotFound        router.ErrCode = "service_not_found"
	ErrCodeFailedToInstallService router.ErrCode = "failed_to_install_service"
)
