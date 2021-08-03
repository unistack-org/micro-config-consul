package consul

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/consul/api"
	"github.com/imdario/mergo"
	"github.com/unistack-org/micro/v3/config"
	"github.com/unistack-org/micro/v3/util/jitter"
	rutil "github.com/unistack-org/micro/v3/util/reflect"
)

var (
	DefaultStructTag = "consul"
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
	if err != nil && !c.opts.AllowFail {
		return err
	}

	c.cli = cli
	c.path = path

	return nil
}

func (c *consulConfig) Load(ctx context.Context, opts ...config.LoadOption) error {
	for _, fn := range c.opts.BeforeLoad {
		if err := fn(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}
	}

	pair, _, err := c.cli.KV().Get(c.path, nil)
	if err != nil && !c.opts.AllowFail {
		return fmt.Errorf("consul path %s load error: %v", c.path, err)
	} else if pair == nil && !c.opts.AllowFail {
		return fmt.Errorf("consul path %s not found", c.path)
	}

	if err == nil && pair != nil {
		options := config.NewLoadOptions(opts...)
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

		if err != nil && !c.opts.AllowFail {
			return err
		}

	}

	for _, fn := range c.opts.AfterLoad {
		if err := fn(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}
	}

	return nil
}

func (c *consulConfig) Save(ctx context.Context, opts ...config.SaveOption) error {
	for _, fn := range c.opts.BeforeSave {
		if err := fn(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}
	}

	buf, err := c.opts.Codec.Marshal(c.opts.Struct)
	if err == nil {
		_, err = c.cli.KV().Put(&api.KVPair{Key: c.path, Value: buf}, nil)
	}

	if err != nil && !c.opts.AllowFail {
		return fmt.Errorf("consul path %s save error: %v", c.path, err)
	}

	for _, fn := range c.opts.AfterSave {
		if err := fn(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}
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
	w := &consulWatcher{
		cli:   c.cli,
		path:  c.path,
		opts:  c.opts,
		wopts: config.NewWatchOptions(opts...),
		done:  make(chan struct{}),
		vchan: make(chan map[string]interface{}),
		echan: make(chan error),
	}

	go w.run()

	return w, nil
}

type consulWatcher struct {
	cli   *api.Client
	path  string
	opts  config.Options
	wopts config.WatchOptions
	done  chan struct{}
	vchan chan map[string]interface{}
	echan chan error
}

func (w *consulWatcher) run() {
	ticker := jitter.NewTicker(w.wopts.MinInterval, w.wopts.MaxInterval)
	defer ticker.Stop()

	src := w.opts.Struct
	if w.wopts.Struct != nil {
		src = w.wopts.Struct
	}

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			dst, err := rutil.Zero(src)
			if err != nil {
				w.echan <- err
				return
			}

			pair, _, err := w.cli.KV().Get(w.path, nil)
			if err != nil {
				w.echan <- err
				return
			}

			if pair == nil {
				w.echan <- fmt.Errorf("consul path %s not found", w.path)
				return
			}

			err = w.opts.Codec.Unmarshal(pair.Value, dst)
			if err != nil {
				w.echan <- err
				return
			}

			srcmp, err := rutil.StructFieldsMap(src)
			if err != nil {
				w.echan <- err
				return
			}

			dstmp, err := rutil.StructFieldsMap(dst)
			if err != nil {
				w.echan <- err
				return
			}

			for sk, sv := range srcmp {
				if reflect.DeepEqual(dstmp[sk], sv) {
					delete(dstmp, sk)
				}
			}

			w.vchan <- dstmp
			src = dst
		}
	}
}

func (w *consulWatcher) Next() (map[string]interface{}, error) {
	select {
	case <-w.done:
		break
	case err := <-w.echan:
		return nil, err
	case v, ok := <-w.vchan:
		if !ok {
			break
		}
		return v, nil
	}
	return nil, config.ErrWatcherStopped
}

func (w *consulWatcher) Stop() error {
	close(w.done)
	return nil
}

func NewConfig(opts ...config.Option) config.Config {
	options := config.NewOptions(opts...)
	if len(options.StructTag) == 0 {
		options.StructTag = DefaultStructTag
	}
	return &consulConfig{opts: options}
}
