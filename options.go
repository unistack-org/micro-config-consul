package consul

import (
	"time"

	"github.com/hashicorp/consul/api"
	"go.unistack.org/micro/v4/options"
)

type configKey struct{}

func Config(cfg *api.Config) options.Option {
	return options.ContextOption(configKey{}, cfg)
}

type tokenKey struct{}

func Token(token string) options.Option {
	return options.ContextOption(tokenKey{}, token)
}

type addrKey struct{}

func Address(addr string) options.Option {
	return options.ContextOption(addrKey{}, addr)
}

type pathKey struct{}

func Path(path string) options.Option {
	return options.ContextOption(pathKey{}, path)
}

type timeoutKey struct{}

func Timeout(td time.Duration) options.Option {
	return options.ContextOption(timeoutKey{}, td)
}

/*
type tlsConfigKey struct{}

func TLSConfig(t *tls.Config) options.Option {
	return options.ContextOption(tlsConfigKey{}, t)
}
*/
