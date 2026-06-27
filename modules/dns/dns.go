package dns

import (
	"context"
	"fmt"
	"log"

	"github.com/openumi/fx"
	"github.com/openumi/umi"
)

// ---- DNS Module ----
func init() {
	umi.RegisterModule("dns", func(id string, cfg []byte, bus *umi.ExchangeBus) *fx.App {
		// 每一个实例拥有自己独立的 Fx 容器
		return fx.New(
			fx.Provide(func() chan umi.Envelope {
				ch := make(chan umi.Envelope, 100)
				bus.Register(id, ch) // 将自己的信箱挂到总线上
				return ch
			}),
			fx.Invoke(func(lc fx.Lifecycle, mailbox chan umi.Envelope) {
				lc.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						go func() {
							log.Printf("[%s] DNS service started, listening for messages...", id)
							for env := range mailbox {
								switch env.Type {
								case umi.TypeQuery:
									req := env.Payload.(umi.QueryPayload)
									log.Printf("[%s] received query: %s from %s", id, req.Question, env.Source)
									req.ReplyCh <- fmt.Sprintf("resolved by %s", id)
								}
							}
						}()
						return nil
					},
					OnStop: func(ctx context.Context) error {
						bus.Unregister(id)
						close(mailbox)
						return nil
					},
				})
			}),
			// 压制 Fx 自身的日志输出，保持 UMI 控制台干净
			fx.NopLogger,
		)
	})
}
