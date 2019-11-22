package main

import (
	"crypto/ecdsa"
	"fmt"
	"os"

	crypto "github.com/nspcc-dev/neofs-crypto"
	"github.com/nspcc-dev/neofs-proto/service"
	"github.com/urfave/cli"
)

const (
	hostFlag   = "host"
	objFlag    = "oid"
	cidFlag    = "cid"
	keyFlag    = "key"
	permFlag   = "perm"
	fileFlag   = "file"
	sgidFlag   = "sgid"
	ttlFlag    = "ttl"
	widFlag    = "wid"
	heightFlag = "height"
	amountFlag = "amount"
	sgFlag     = "sg"

	ConfigFlag = "config"

	defaultPermission = 0600
)

var (
	keyFile = cli.StringFlag{
		Name:   keyFlag,
		EnvVar: KeyEnvValue,
		Usage:  "user private key in hex, wif formats or path to file",
	}

	ttlF = cli.UintFlag{
		Name:  ttlFlag,
		Usage: "request ttl",
		Value: service.SingleForwardingTTL,
	}

	cfgF = cli.StringFlag{
		Name:   ConfigFlag,
		Usage:  "config",
		EnvVar: ConfigEnvValue,
		Value:  DefaultConfig,
	}

	hostAddr = cli.StringFlag{
		Name:   hostFlag,
		Usage:  "host net address",
		EnvVar: HostEnvValue,
	}

	containerID = cli.StringFlag{
		Name:  cidFlag,
		Usage: "user container ID",
	}

	objectID = cli.StringFlag{
		Name:  objFlag,
		Usage: "object ID to receive",
	}

	objectIDs = cli.StringSliceFlag{
		Name:  objFlag,
		Usage: "object IDs to receive",
	}

	storagegroupID = cli.StringFlag{
		Name:  sgidFlag,
		Usage: "storage group id",
	}

	storageGroup = cli.BoolFlag{
		Name:  sgFlag,
		Usage: "storage group",
	}

	filePath = cli.StringFlag{
		Name:  fileFlag,
		Usage: "path to output file",
	}

	filesPath = cli.StringSliceFlag{
		Name:  fileFlag,
		Usage: "path to output file",
	}

	permissions = cli.UintFlag{
		Name:  permFlag,
		Usage: "output file permissions",
		Value: defaultPermission,
	}

	withdrawID = cli.StringFlag{
		Name:  widFlag,
		Usage: "withdrawal ID",
	}

	amount = cli.Float64Flag{
		Name:  amountFlag,
		Usage: "withdrawal amount",
	}

	blockHeight = cli.Uint64Flag{
		Name:  heightFlag,
		Usage: "block blockHeight",
	}

	fullHeaders = cli.BoolFlag{
		Name:  fullHeadersFlag,
		Usage: "return all headers",
	}
)

func signRequest(c *cli.Context, req service.VerifiableRequest) {
	key := getKey(c)
	if err := service.SignRequestHeader(key, req); err != nil {
		fmt.Printf("%T could not sign request\n", req)
		fmt.Println(err.Error())
		os.Exit(2)
	}
}

func getKey(c *cli.Context) *ecdsa.PrivateKey {
	var (
		err error
		key *ecdsa.PrivateKey
	)

	if arg := c.GlobalString(keyFlag); arg == "" {
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

func setTTL(c *cli.Context, req service.MetaHeader) {
	ttl := c.GlobalUint(ttlFlag)
	req.SetTTL(uint32(ttl))
}
