package pluginabi

const (
	ABIVersion    uint32 = 1
	SchemaVersion uint32 = 1
)

const (
	MethodPluginRegister    = "plugin.register"
	MethodPluginReconfigure = "plugin.reconfigure"
	MethodPluginShutdown    = "plugin.shutdown"

	MethodSchedulerPick = "scheduler.pick"
	MethodUsageHandle   = "usage.handle"

	MethodManagementRegister = "management.register"
	MethodManagementHandle   = "management.handle"

	MethodHostHTTPDo         = "host.http.do"
	MethodHostLog            = "host.log"
	MethodHostAuthList       = "host.auth.list"
	MethodHostAuthGet        = "host.auth.get"
	MethodHostAuthGetRuntime = "host.auth.get_runtime"
	MethodHostAuthSave       = "host.auth.save"
)
