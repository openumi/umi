package dns

import (
	"encoding/json"

	umi "github.com/openumi/umi"
)

func init() {
	umi.RegisterModule(new(DnsModule))
	umi.RegisterModule(new(DohForwarder))
}

type DnsModule struct {
	Tag         string            `json:"tag"`
	EnableCache bool              `json:"enable_cache"`
	Forwarder   []json.RawMessage `json:"forwarder" umi:"namespace=dns.forwarder inline_key=type"`
}

func (d *DnsModule) UniModule() umi.ModuleInfo {
	return umi.ModuleInfo{
		ID:  "dns",
		New: func() umi.Module { return new(DnsModule) },
	}
}

func (d *DnsModule) Configure(ctx *umi.ConfigContext, raw json.RawMessage) error {
	// Task 0 & Task 2: 一行流自动开箱并打平提升
	return umi.ConfigureModule(ctx, d, raw)
}

// DohForwarder 真实子元件
type DohForwarder struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`
	Addr string `json:"addr"`
}

func (d *DohForwarder) UniModule() umi.ModuleInfo {
	return umi.ModuleInfo{
		ID:  "dns.forwarder.doh",
		New: func() umi.Module { return new(DohForwarder) },
	}
}

func (d *DohForwarder) Configure(ctx *umi.ConfigContext, raw json.RawMessage) error {
	return umi.ConfigureModule(ctx, d, raw)
}
