package main

import (
	"crypto/ecdsa"
	"fmt"
	"os"

	"github.com/nspcc-dev/neofs-api-go/service"
	crypto "github.com/nspcc-dev/neofs-crypto"
	"github.com/urfave/cli/v2"
)

const (
	hostFlag    = "host"
	objFlag     = "oid"
	cidFlag     = "cid"
	keyFlag     = "key"
	permFlag    = "perm"
	fileFlag    = "file"
	sgidFlag    = "sgid"
	ttlFlag     = "ttl"
	widFlag     = "wid"
	heightFlag  = "height"
	amountFlag  = "amount"
	sgFlag      = "sg"
	verboseFlag = "verbose"
	beautyFlag  = "beauty"
	stateFlag   = "state"

	ConfigFlag = "config"

	defaultPermission = 0600
)

var (
	verbose = &cli.BoolFlag{
		Name:  verboseFlag,
		Usage: "verbose gRPC connection",
	}

	keyFile = &cli.StringFlag{
		Name:    keyFlag,
		EnvVars: []string{KeyEnvValue},
		Usage:   "user private key in hex, wif formats or path to file",
	}

	ttlF = &cli.UintFlag{
		Name:  ttlFlag,
		Usage: "request ttl",
		Value: service.SingleForwardingTTL,
	}

	cfgF = &cli.StringFlag{
		Name:    ConfigFlag,
		Usage:   "config",
		EnvVars: []string{ConfigEnvValue},
		Value:   DefaultConfig,
	}

	hostAddr = &cli.StringFlag{
		Name:    hostFlag,
		Usage:   "host net address",
		EnvVars: []string{HostEnvValue},
	}

	containerID = &cli.StringFlag{
		Name:     cidFlag,
		Required: true,
		Usage:    "user container ID",
	}

	objectID = &cli.StringFlag{
		Name:     objFlag,
		Required: true,
		Usage:    "object ID to receive",
	}

	objectIDs = &cli.StringSliceFlag{
		Name:     objFlag,
		Required: true,
		Usage:    "object IDs to receive",
	}

	storagegroupID = &cli.StringFlag{
		Name:     sgidFlag,
		Required: true,
		Usage:    "storage group id",
	}

	storageGroup = &cli.BoolFlag{
		Name:  sgFlag,
		Usage: "storage group",
	}

	filePath = &cli.StringFlag{
		Name:  fileFlag,
		Usage: "path to output file",
	}

	filesPath = &cli.StringSliceFlag{
		Name:  fileFlag,
		Usage: "path to output file",
	}

	permissions = &cli.UintFlag{
		Name:  permFlag,
		Usage: "output file permissions",
		Value: defaultPermission,
	}

	withdrawID = &cli.StringFlag{
		Name:  widFlag,
		Usage: "withdrawal ID",
	}

	amount = &cli.Float64Flag{
		Name:  amountFlag,
		Usage: "withdrawal amount",
	}

	blockHeight = &cli.Uint64Flag{
		Name:  heightFlag,
		Usage: "block blockHeight",
	}

	fullHeaders = &cli.BoolFlag{
		Name:  fullHeadersFlag,
		Usage: "return all headers",
	}

	rawQuery = &cli.BoolFlag{
		Name:  rawFlag,
		Usage: "send request with raw flag set",
	}
)

func signRequest(c *cli.Context, req service.DataWithTokenSignAccumulator) {
	key := getKey(c)
	if err := service.SignDataWithSessionToken(key, req); err != nil {
		fmt.Printf("%T could not sign request\n", req)
		fmt.Println(err.Error())
		os.Exit(2)
	}
}

func getHost(c *cli.Context) string {
	var (
		err  error
		host string
	)

	if arg := c.String(hostFlag); arg == "" {
		fmt.Println("host cannot be empty (--host)")
		fmt.Println("provide <host>:<port> or <ip>:<port>")
		os.Exit(2)
	} else if host, err = parseHostValue(arg); err != nil {
		fmt.Printf("could not parse host from: %s\n", arg)
		fmt.Println(err.Error())
		os.Exit(2)
	}

	return host
}

func getKey(c *cli.Context) *ecdsa.PrivateKey {
	var (
		err error
		key *ecdsa.PrivateKey
	)

	if arg := c.String(keyFlag); arg == "" {
		fmt.Println("private key cannot be empty (--key)")
		fmt.Println("provide hex-string, wif or path")
		os.Exit(2)
	} else if key, err = crypto.LoadPrivateKey(arg); err != nil {
		fmt.Printf("could not load private key: %s\n", arg)
		fmt.Println(err.Error())
		os.Exit(2)
	}
	return key
}

func setTTL(c *cli.Context, req service.TTLContainer) {
	ttl := c.Uint(ttlFlag)
	req.SetTTL(uint32(ttl))
}

func setRaw(c *cli.Context, req service.RawContainer) {
	req.SetRaw(
		c.Bool(rawFlag),
	)
}
