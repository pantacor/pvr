package templates

var (
	Handlers = map[string]func(map[string]interface{}) (map[string][]byte, error){
		"builtin-lxc-docker": BuiltinLXCDockerHandler,
	}
)
