module github.com/nspcc-dev/neofs-cli

go 1.13

require (
	github.com/mr-tron/base58 v1.1.3
	github.com/nspcc-dev/neofs-api v0.3.1
	github.com/nspcc-dev/neofs-crypto v0.2.3
	github.com/nspcc-dev/netmap v1.6.1
	github.com/nspcc-dev/netmap-ql v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.9.1
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli/v2 v2.1.1
	google.golang.org/grpc v1.27.1
)

// Temporary, before we move repo to github:
// replace github.com/nspcc-dev/neofs-api => ../neofs-api
