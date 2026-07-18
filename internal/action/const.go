package action

const (
	Ban     = "ban"
	Disable = "disable"
	Delete  = "delete"
	Reauth  = "reauth"
	// UsingAPI enables CPA "使用 API 模式" (using_api) for xAI OAuth files.
	UsingAPI = "using_api"
	// UsingAPIOff disables using_api (bulk cleanup; temporary ops helper).
	UsingAPIOff = "using_api_off"

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
