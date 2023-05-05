package consul

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/consul/api"
	"go.unistack.org/micro/v4/config"
	"go.unistack.org/micro/v4/util/jitter"
	rutil "go.unistack.org/micro/v4/util/reflect"
)

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
			if len(dstmp) > 0 {
				w.vchan <- dstmp
				src = dst
			}
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
