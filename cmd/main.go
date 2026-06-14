package cmd

import (
	"fmt"

	_ "github.com/openumi/umi/modules/standard"
)

const finalFlatConfig = `
{
    "log": {
        "level": "INFO"
    },
    "dns": {
        "enable_cache": true,
        "forwarder": [
            {
                "type": "doh",
                "tag": "dns.01",
                "addr": "1.1.1.1"
            }
        ]
    },
    "http_client": [
        {
            "tag": "httpclient.01",
            "dns_provider": "dns.01"
        },
        {
            "tag": "httpclient.02",
            "dns_provider": "dns.01"
        }
    ]
}
`

func Main() {
	fmt.Println("[Uni Engine] Commencing Time-Two-Phase Bus Booting...")

	err := AppController.Reload([]byte(finalFlatConfig))
	if err != nil {
		panic(fmt.Sprintf("[Uni Engine] Critical Error: Boot Failed -> %v", err))
	}

	rt := AppController.GetRuntime()
	PrintLiveTopology(rt)
}
