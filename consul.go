package consul // import "go.unistack.org/micro-config-consul/v3"

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/imdario/mergo"
	"go.unistack.org/micro/v3/config"
	rutil "go.unistack.org/micro/v3/util/reflect"
)

var DefaultStructTag = "consul"

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
		c.opts.Logger.Errorf(c.opts.Context, "consul init err: %v", err)
		if !c.opts.AllowFail {
			return err
		}
	}

	c.cli = cli
	c.path = path

	return nil
}

func (c *consulConfig) Load(ctx context.Context, opts ...config.LoadOption) error {
	path := c.path
	options := config.NewLoadOptions(opts...)
	if options.Context != nil {
		if v, ok := options.Context.Value(pathKey{}).(string); ok && v != "" {
			path = v
		}
	}

	if err := config.DefaultBeforeLoad(ctx, c); err != nil {
		return err
	}

	pair, _, err := c.cli.KV().Get(path, nil)
	if err != nil {
		err = fmt.Errorf("consul path %s load error: %w", path, err)
	} else if pair == nil {
		err = fmt.Errorf("consul path %s load error: not found", path)
	}

	if err != nil {
		c.opts.Logger.Error(c.opts.Context, err)
		if !c.opts.AllowFail {
			return err
		}
		return config.DefaultAfterLoad(ctx, c)
	}

	mopts := []func(*mergo.Config){mergo.WithTypeCheck}
	if options.Override {
		mopts = append(mopts, mergo.WithOverride)
	}
	if options.Append {
		mopts = append(mopts, mergo.WithAppendSlice)
	}

	dst := c.opts.Struct
	if options.Struct != nil {
		dst = options.Struct
	}

	src, err := rutil.Zero(dst)
	if err == nil {
		err = c.opts.Codec.Unmarshal(pair.Value, src)
		if err == nil {
			err = mergo.Merge(dst, src, mopts...)
		}
	}

	if err != nil {
		c.opts.Logger.Errorf(c.opts.Context, "consul load err: %v", err)
		if !c.opts.AllowFail {
			return err
		}
	}

	if err := config.DefaultAfterLoad(ctx, c); err != nil {
		return err
	}

	return nil
}

func (c *consulConfig) Save(ctx context.Context, opts ...config.SaveOption) error {
	path := c.path
	options := config.NewSaveOptions(opts...)
	if options.Context != nil {
		if v, ok := options.Context.Value(pathKey{}).(string); ok && v != "" {
			path = v
		}
	}

	if err := config.DefaultBeforeSave(ctx, c); err != nil {
		return err
	}

	buf, err := c.opts.Codec.Marshal(c.opts.Struct)
	if err == nil {
		_, err = c.cli.KV().Put(&api.KVPair{Key: path, Value: buf}, nil)
	}

	if err != nil {
		c.opts.Logger.Errorf(c.opts.Context, "consul path %s save error: %v", path, err)
		if !c.opts.AllowFail {
			return fmt.Errorf("consul path %s save error: %v", path, err)
		}
	}

	if err := config.DefaultAfterSave(ctx, c); err != nil {
		return err
	}

	return nil
}

func (c *consulConfig) String() string {
	return "consul"
}

func (c *consulConfig) Name() string {
	return c.opts.Name
}

func (c *consulConfig) Watch(ctx context.Context, opts ...config.WatchOption) (config.Watcher, error) {
	path := c.path
	options := config.NewWatchOptions(opts...)
	if options.Context != nil {
		if v, ok := options.Context.Value(pathKey{}).(string); ok && v != "" {
			path = v
		}
	}

	w := &consulWatcher{
		cli:   c.cli,
		path:  path,
		opts:  c.opts,
		wopts: config.NewWatchOptions(opts...),
		done:  make(chan struct{}),
		vchan: make(chan map[string]interface{}),
		echan: make(chan error),
	}

	go w.run()

	return w, nil
}

func NewConfig(opts ...config.Option) config.Config {
	options := config.NewOptions(opts...)
	if len(options.StructTag) == 0 {
		options.StructTag = DefaultStructTag
	}
	return &consulConfig{opts: options}
}
