package contexts

import (
	"encoding/json"
	"os"
	"path"
	"unsafe"

	"github.com/ebitengine/purego"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/multiversx/mx-chain-vm-v1_4-go/vmhost"
)

var vmPluginLog = logger.GetOrCreate("vm/plugins")

type vmPluginInitResult struct {
	Name    string
	Methods []string
}

func NewPluginsContext(
	host vmhost.VMHost,
) *vmhost.PluginsContext {

	return &vmhost.PluginsContext{
		Plugins: loadPlugins(host),
		Host:    host,
	}
}

func loadPlugins(host vmhost.VMHost) []vmhost.VmPlugin {
	list := make([]vmhost.VmPlugin, 0)
	vmPluginsPath := os.Getenv("MX_VM_PLUGINS_PATH")
	if vmPluginsPath == "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return list
		}

		vmPluginsPath = path.Join(homedir, ".mx-vm", "plugins")
	}

	vmPluginLog.Info("loading VM plugins from: ", "pluginsPath", vmPluginsPath)

	dirEntries, err := os.ReadDir(vmPluginsPath)
	if err != nil {
		return list
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}

		pluginPath := path.Join(vmPluginsPath, dirEntry.Name())
		_, err := os.Stat(pluginPath)

		if err != nil {
			continue
		}

		pluginso, err := purego.Dlopen(pluginPath, purego.RTLD_NOW)
		if err != nil {
			vmPluginLog.Warn("error loading VM plugin: ", "pluginPath", pluginPath)
			continue
		}

		var init func() string
		purego.RegisterLibFunc(&init, pluginso, "mx_plug_init")

		initResultStr := init()
		var initResult vmPluginInitResult

		jsonErr := json.Unmarshal([]byte(initResultStr), &initResult)
		if jsonErr != nil {
			vmPluginLog.Warn("error loading VM plugin: ", "pluginPath", pluginPath)
		}
		vmPluginLog.Info("initialized VM plugin: ", "name", initResult.Name)

		var call func(callCtx string, methodName string, args []byte) unsafe.Pointer
		purego.RegisterLibFunc(&call, pluginso, "mx_plug_call")

		methods := make([]vmhost.VmPluginMethod, len(initResult.Methods))
		for i, methodName := range initResult.Methods {
			methods[i] = vmhost.VmPluginMethod{
				Name: methodName,
			}
		}

		list = append(list, vmhost.VmPlugin{
			Name:    initResult.Name,
			Methods: methods,
			CallFn:  call,
		})

		vmPluginLog.Info("completed loading VM plugin: ", "name", initResult.Name, "methods", initResult.Methods)
	}

	return list
}
