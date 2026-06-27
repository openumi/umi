package flux

import (
	"context"
	"log"
	"time"

	"github.com/openumi/fx"
	"github.com/openumi/umi"
)

// ---- Flux Proxy Module ----
func init() {
	umi.RegisterModule("flux", func(id string, cfg []byte, bus *umi.ExchangeBus) *fx.App {
		return fx.New(
			fx.Invoke(func(lc fx.Lifecycle) {
				lc.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						log.Printf("[%s] Flux Proxy outbound module started", id)

						// 模拟触发一个运行时依赖调用 (Instance-addressed)
						go func() {
							time.Sleep(1 * time.Second)
							replyCh := make(chan string, 1)

							// 显式直接寻址 "dns-main" 实例，不走 capability-routing
							env := umi.Envelope{
								Type:   umi.TypeQuery,
								Source: id,
								Target: "dns-main",
								Payload: umi.QueryPayload{
									Question: "google.com",
									ReplyCh:  replyCh,
								},
							}

							log.Printf("[%s] Sending Query to [dns-main]...", id)
							if err := bus.Send(env); err != nil {
								log.Printf("[%s] Send error: %v", id, err)
								return
							}

							answer := <-replyCh
							log.Printf("[%s] Got response: %s", id, answer)
						}()
						return nil
					},
				})
			}),
			fx.NopLogger,
		)
	})
}
