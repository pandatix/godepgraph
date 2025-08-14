package apiv1rdg

import (
	"fmt"
	"sync"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

func init() {
	reg := GetRegistry()
	reg.MustStore("kubernetes", func(in apitype.ResourceV3) *Resource {
		mod := in.Type.Module().String()
		name := in.Type.Name().String()

		switch mod {
		case "kubernetes:apps/v1":
			switch name {
			case "Deployment", "StatefulSet", "DaemonSet", "Pod":
				return &Resource{
					ID: string(in.URN),
				}
			}

		case "kubernetes:batch/v1":
			switch name {
			case "CronJob", "Job":
				return &Resource{
					ID: string(in.URN),
				}
			}
		}
		return nil
	})
}

type Transformer func(in apitype.ResourceV3) *Resource

// Register holds transformers for ecosystems, identified by their keys.
type Registry struct {
	mp map[string]Transformer
	mx sync.Mutex
}

func NewRegistry() *Registry {
	return &Registry{
		mp: map[string]Transformer{},
		mx: sync.Mutex{},
	}
}

// Store appends an ecosystem transformer.
// If one is already registered for the same key, return an error.
func (r *Registry) Store(key string, t Transformer) error {
	r.mx.Lock()
	defer r.mx.Unlock()

	if _, ok := r.mp[key]; ok {
		return fmt.Errorf("registry already has transformer registered for key %s", key)
	}
	if t == nil {
		return fmt.Errorf("nil transformer for key %s", key)
	}
	r.mp[key] = t

	return nil
}

// MustStore behaves as [*Register].Store, but panics in case of an error.
func (r *Registry) MustStore(key string, t Transformer) {
	if err := r.Store(key, t); err != nil {
		panic(err)
	}
}

func (r *Registry) Load(key string) (Transformer, bool) {
	r.mx.Lock()
	defer r.mx.Unlock()

	t, ok := r.mp[key]
	return t, ok
}

var (
	reg     *Registry
	regOnce sync.Once
)

func GetRegistry() *Registry {
	regOnce.Do(func() {
		reg = NewRegistry()
	})
	return reg
}
