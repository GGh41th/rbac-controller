package main

import (
	"os"

	"github.com/GGh41th/rbac-controller/cmd/controller-manager/app"
)

func main() {
	cmd := app.NewControllerManagerCommand()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
