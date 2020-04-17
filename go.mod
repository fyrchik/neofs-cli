module github.com/nspcc-dev/neofs-cli

go 1.14

require (
	github.com/mr-tron/base58 v1.1.3
	github.com/nspcc-dev/neofs-api-go v0.7.0
	github.com/nspcc-dev/neofs-crypto v0.3.0
	github.com/nspcc-dev/netmap v1.7.0
	github.com/nspcc-dev/netmap-ql v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.9.1
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.5.1
	github.com/urfave/cli/v2 v2.1.1
	google.golang.org/grpc v1.28.1
)

// Temporary, before we move repo to github:
// replace github.com/nspcc-dev/neofs-api-go => ../neofs-api
