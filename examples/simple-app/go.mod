module github.com/toyz/axon/examples/simple-app

go 1.25

require go.uber.org/fx v1.24.0

replace github.com/toyz/axon => ../../

require (
	github.com/stretchr/testify v1.10.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)
