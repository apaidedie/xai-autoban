package action

const (
	Ban     = "ban"
	Disable = "disable"
	Delete  = "delete"
	Reauth  = "reauth"

	SuccessNone             = "none"
	SuccessUnban            = "unban"
	SuccessReenable         = "reenable"
	SuccessUnbanAndReenable = "unban_and_reenable"

	DisableViaHostAuth         = "host_auth"
	DisableViaManagementAPI    = "management_api"
	DefaultManagementURL       = "http://127.0.0.1:8317"
	DefaultManagementKeyEnv    = "CPA_MANAGEMENT_KEY"
	DefaultMgmtTimeoutSec      = 10
	DefaultMgmtAuthCooldownSec = 600
)
