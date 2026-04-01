package utils

import (
	"context"
	"strconv"

	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/loadbalance"
	"github.com/stathat/consistent"
)

type consistentHashLoadBalancer struct{}

func NewConsistentHashLoadBalancer() loadbalance.Loadbalancer {
	return &consistentHashLoadBalancer{}
}

func (c *consistentHashLoadBalancer) GetPicker(result discovery.Result) loadbalance.Picker {
	instances := result.Instances
	ch := consistent.New()
	for _, ins := range instances {
		ch.Add(ins.Address().String())
	}
	return &consistentHashPicker{ch: ch, instances: instances}
}

func (c *consistentHashLoadBalancer) Rebalance(change discovery.Change) {}

func (c *consistentHashLoadBalancer) Delete(change discovery.Change) {}

func (c *consistentHashLoadBalancer) Name() string {
	return "consistent_hash"
}

type consistentHashPicker struct {
	ch        *consistent.Consistent
	instances []discovery.Instance
}

func (p *consistentHashPicker) Next(ctx context.Context, request interface{}) discovery.Instance {
	lessonID := ctx.Value("lessonID")
	if lessonID == nil {
		if len(p.instances) > 0 {
			return p.instances[0]
		}
		return nil
	}

	var key string
	switch v := lessonID.(type) {
	case int64:
		key = strconv.FormatInt(v, 10)
	case string:
		key = v
	default:
		if len(p.instances) > 0 {
			return p.instances[0]
		}
		return nil
	}

	addr, _ := p.ch.Get(key)
	for _, ins := range p.instances {
		if ins.Address().String() == addr {
			return ins
		}
	}

	if len(p.instances) > 0 {
		return p.instances[0]
	}
	return nil
}
