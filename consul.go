package consul

import (
	"context"
	"encoding/json"
	"errors"

	api "github.com/hashicorp/consul/api"
	"github.com/unistack-org/micro/v3/config"
)

var (
	DefaultStructTag = "consul"
	ErrInvalidStruct = errors.New("invalid struct specified")
	ErrPathNotExist  = errors.New("path is not exist")
)

type consulConfig struct {
	opts config.Options
	cli  *api.Client
	path string
}

func (c *consulConfig) Options() config.Options {
	return c.opts
}

func (c *consulConfig) Init(opts ...config.Option) error {
	for _, o := range opts {
		o(&c.opts)
	}

	cfg := api.DefaultConfig()
	path := ""

	if c.opts.Context != nil {
		if v, ok := c.opts.Context.Value(configKey{}).(*api.Config); ok {
			cfg = v
		}

		if v, ok := c.opts.Context.Value(addrKey{}).(string); ok {
			cfg.Address = v
		}

		if v, ok := c.opts.Context.Value(tokenKey{}).(string); ok {
			cfg.Token = v
		}

		if v, ok := c.opts.Context.Value(pathKey{}).(string); ok {
			path = v
		}

	}

	cli, err := api.NewClient(cfg)
	if err != nil {
		return err
	}

	c.cli = cli
	c.path = path

	return nil
}

func (c *consulConfig) Load(ctx context.Context) error {
	pair, _, err := c.cli.KV().Get(c.path, nil)
	if err != nil {
		return err
	} else if pair == nil {
		return ErrPathNotExist
	}

	return json.Unmarshal(pair.Value, c.opts.Struct)
}

func (c *consulConfig) Save(ctx context.Context) error {
	return nil
}

func (c *consulConfig) String() string {
	return "consul"
}

func NewConfig(opts ...config.Option) config.Config {
	options := config.NewOptions(opts...)
	if len(options.StructTag) == 0 {
		options.StructTag = DefaultStructTag
	}
	return &consulConfig{opts: options}
}
