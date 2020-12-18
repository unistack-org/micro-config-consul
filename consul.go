package consul

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/unistack-org/micro/v3/config"
)

var (
	DefaultStructTag = "consul"
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

	if c.opts.Codec == nil {
		return config.ErrCodecMissing
	}

	cfg := api.DefaultConfigWithLogger(&consulLogger{logger: c.opts.Logger})
	path := ""

	if c.opts.Context != nil {
		if v, ok := c.opts.Context.Value(configKey{}).(*api.Config); ok {
			cfg.Address = v.Address
			cfg.Scheme = v.Scheme
			cfg.Datacenter = v.Datacenter
			cfg.Transport = v.Transport
			cfg.HttpClient = v.HttpClient
			cfg.HttpAuth = v.HttpAuth
			cfg.WaitTime = v.WaitTime
			cfg.Token = v.Token
			cfg.TokenFile = v.TokenFile
			cfg.Namespace = v.Namespace
			cfg.TLSConfig = v.TLSConfig
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
		/*
			if v, ok := c.opts.Context.Value(tlsConfigKey{}).(*tls.Config); ok {
				cfg.TLSConfig = *v
			}
		*/
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
	for _, fn := range c.opts.BeforeLoad {
		if err := fn(ctx, c); err != nil {
			return err
		}
	}

	pair, _, err := c.cli.KV().Get(c.path, nil)
	if err != nil {
		return fmt.Errorf("consul path load error: %v", err)
	} else if pair == nil {
		return fmt.Errorf("consul path not found %v", ErrPathNotExist)
	}

	if err = c.opts.Codec.Unmarshal(pair.Value, c.opts.Struct); err != nil {
		return err
	}

	for _, fn := range c.opts.AfterLoad {
		if err := fn(ctx, c); err != nil {
			return err
		}
	}

	return nil
}

func (c *consulConfig) Save(ctx context.Context) error {
	for _, fn := range c.opts.BeforeSave {
		if err := fn(ctx, c); err != nil {
			return err
		}
	}

	for _, fn := range c.opts.AfterSave {
		if err := fn(ctx, c); err != nil {
			return err
		}
	}

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
