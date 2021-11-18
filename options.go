package consul

import (
	"time"

	"github.com/hashicorp/consul/api"
	"go.unistack.org/micro/v3/config"
)

type configKey struct{}

func Config(cfg *api.Config) config.Option {
	return config.SetOption(configKey{}, cfg)
}

type tokenKey struct{}

func Token(token string) config.Option {
	return config.SetOption(tokenKey{}, token)
}

type addrKey struct{}

func Address(addr string) config.Option {
	return config.SetOption(addrKey{}, addr)
}

type pathKey struct{}

func Path(path string) config.Option {
	return config.SetOption(pathKey{}, path)
}

/*
type tlsConfigKey struct{}

func TLSConfig(t *tls.Config) config.Option {
	return config.SetOption(tlsConfigKey{}, t)
}
*/

type timeoutKey struct{}

func Timeout(td time.Duration) config.Option {
	return config.SetOption(timeoutKey{}, td)
}

func LoadPath(path string) config.LoadOption {
	return config.SetLoadOption(pathKey{}, path)
}

func SavePath(path string) config.SaveOption {
	return config.SetSaveOption(pathKey{}, path)
}

func WatchPath(path string) config.WatchOption {
	return config.SetWatchOption(pathKey{}, path)
}
