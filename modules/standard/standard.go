// Package standard imports all the standard Uni modules.
//
// This file exists purely to register modules via their init() functions.
package standard

import (
	_ "github.com/openumi/umi/modules/dns"
	_ "github.com/openumi/umi/modules/httpclient"
	_ "github.com/openumi/umi/modules/log"
)
