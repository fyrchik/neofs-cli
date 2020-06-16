module github.com/nspcc-dev/neofs-cli

go 1.14

require (
	github.com/mr-tron/base58 v1.2.0
	github.com/nspcc-dev/neofs-api-go v1.0.1-0.20200615181008-cd6f628b5b56
	github.com/nspcc-dev/neofs-crypto v0.3.0
	github.com/nspcc-dev/netmap v1.7.0
	github.com/nspcc-dev/netmap-ql v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.10.0
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.0
	github.com/urfave/cli/v2 v2.2.0
	google.golang.org/grpc v1.29.1
)

// Temporary, before we move repo to github:
// replace github.com/nspcc-dev/neofs-api-go => ../neofs-api
