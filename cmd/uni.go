package cmd

import (
	"fmt"
	"reflect"

	umi "github.com/openumi/umi"
)

var AppController umi.Controller

func PrintLiveTopology(rt *umi.Runtime) {
	fmt.Println("\n==================================================")
	fmt.Println("       🌐 UNI FRAMEWORK LIVE FLAT TOPOLOGY       ")
	fmt.Println("==================================================")
	if rt == nil || len(rt.Instances) == 0 {
		fmt.Println("(Empty Pool)")
		return
	}

	for tag, inst := range rt.Instances {
		val := reflect.ValueOf(inst).Elem()
		fmt.Printf("⚫ Component Tag: [%s]\n", tag)
		fmt.Printf("   ├── Module ID : %s\n", inst.UniModule().ID)

		if tag == "log" {
			fmt.Printf("   └── Level     : %v\n", val.FieldByName("Level").Interface())
		} else if field := val.FieldByName("EnableCache"); field.IsValid() {
			fmt.Printf("   └── Cache     : %v\n", field.Interface())
		} else if field := val.FieldByName("DnsProvider"); field.IsValid() {
			fmt.Printf("   └── Resolves Via ──▶ [%s] (Phase 2 Linked)\n", field.Interface())
		} else if field := val.FieldByName("Addr"); field.IsValid() {
			fmt.Printf("   └── Remote IP : %v\n", field.Interface())
		}
	}
	fmt.Println("==================================================")
}
