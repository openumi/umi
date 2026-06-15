package httpclient

import (
	"encoding/json"
	"fmt"

	umi "github.com/openumi/umi"
)

func init() {
	umi.RegisterModule(new(HttpClientModule))
}

type HttpClientModule struct {
	Tag         string `json:"tag"`
	DnsProvider string `json:"dns_provider"` // stores scheme A string reference alias
}

func (h *HttpClientModule) UniModule() umi.ModuleInfo {
	return umi.ModuleInfo{
		ID:  "http_client",
		New: func() umi.Module { return new(HttpClientModule) },
	}
}

func (h *HttpClientModule) Configure(ctx *umi.ConfigContext, raw json.RawMessage) error {
	return umi.ConfigureModule(ctx, h, raw)
}

// Provision resolves module references in Phase 2
func (h *HttpClientModule) Provision(ctx *umi.Context) error {
	if h.DnsProvider != "" {
		// namespace guard: verify the target belongs to "dns.forwarder"
		inst, err := ctx.FindInstance("dns.forwarder", h.DnsProvider)
		if err != nil {
			return fmt.Errorf("http_client [%s] dependency bind error: %w", h.Tag, err)
		}
		fmt.Printf("[HttpClient][%s] -> scheme A deferred binding success! target [%s] actual ID: (%s)\n",
			h.Tag, h.DnsProvider, inst.UniModule().ID)
	}
	return nil
}
