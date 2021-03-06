package consul

import (
	"github.com/hashicorp/consul/api"
	"github.com/unistack-org/micro/v3/config"
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
