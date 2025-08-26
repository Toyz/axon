package main

import (
	"github.com/toyz/axon/examples/simple-app/internal/config"
	"github.com/toyz/axon/examples/simple-app/internal/logging"
	"github.com/toyz/axon/examples/simple-app/internal/services"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(config.LoadConfig),

		logging.AutogenModule,
		services.AutogenModule,
	).Run()
}
