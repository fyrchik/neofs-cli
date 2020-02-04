module github.com/nspcc-dev/neofs-cli

go 1.13

require (
	github.com/mr-tron/base58 v1.1.3
	github.com/nspcc-dev/neofs-api v0.2.14
	github.com/nspcc-dev/neofs-crypto v0.2.3
	github.com/nspcc-dev/netmap v1.6.1
	github.com/nspcc-dev/netmap-ql v1.2.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/common v0.7.0
	github.com/spf13/viper v1.6.1
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli/v2 v2.0.0
	google.golang.org/grpc v1.25.1
)

// Temporary, before we move repo to github:
// replace github.com/nspcc-dev/neofs-api => ../neofs-api
