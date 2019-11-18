package main

import (
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
		Usage:  "path to user key",
		EnvVar: KeyEnvValue,
	}

	ttlF = cli.UintFlag{
		Name:  ttlFlag,
		Usage: "request ttl",
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

func getTTL(c *cli.Context) uint32 {
	if ttl := c.GlobalUint(ttlFlag); ttl > 0 {
		return uint32(ttl)
	}
	return defaultTTL
}
