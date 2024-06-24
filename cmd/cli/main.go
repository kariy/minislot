// cmd/cli/main.go
package main

import (
	"fmt"
	"os"

	"minislot/pkg/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Println(err)
		
		os.Exit(1)
	}
}