package main

import (
	"log"
	"time"

	"github.com/openumi/umi"
)

func main() {
	// 模拟加载人类友好的配置文件
	mockConfig := []umi.InstanceConfig{
		{ID: "dns-main", Type: "dns", Config: "nameserver: 1.1.1.1"},
		{ID: "dns-backup", Type: "dns", Config: "nameserver: 8.8.8.8"}, // 同 module 多实例测试
		{ID: "flux-inbound", Type: "flux", Config: "listen: :1080"},
	}

	log.Println("--- UMI Runtime Bootstrapping ---")
	runtime, err := umi.Bootstrap(mockConfig)
	if err != nil {
		log.Fatalf("Bootstrap failed: %v", err)
	}

	if err := runtime.Start(); err != nil {
		log.Fatalf("Runtime start failed: %v", err)
	}

	// 维持运行 2 秒看观察交互结果
	time.Sleep(2 * time.Second)

	log.Println("--- UMI Runtime Shutting Down ---")
	runtime.Stop()
}
