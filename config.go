package main

import (
	"fmt"
	"net"
	"os"

	crypto "github.com/nspcc-dev/neofs-crypto"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

type setMode int

const (
	DefaultConfig = ".neofs-cli.yml"

	KeyMode setMode = iota
	HostMode

	KeyEnvValue    = "NEOFS_CLI_KEY"
	KeyCfgValue    = "key"
	HostEnvValue   = "NEOFS_CLI_ADDRESS"
	HostCfgValue   = "host"
	ConfigEnvValue = "NEOFS_CLI_CONFIG"
)

func beforeAction(ctx *cli.Context) error {
	// do something before command
	cfg := ctx.GlobalString(ConfigFlag)

	viper.SetConfigFile(cfg)
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		if cfg != DefaultConfig {
			return errors.Wrapf(err, "could not read config file: %q", cfg)
		}
	}

	if value := viper.GetString(KeyCfgValue); value != "" {
		if err := os.Setenv(KeyEnvValue, value); err != nil {
			fmt.Printf("could not set ENV %q: %s", KeyEnvValue, err)
		}
	}

	if value := viper.GetString(HostCfgValue); value != "" {
		if err := os.Setenv(HostEnvValue, value); err != nil {
			fmt.Printf("could not set ENV %q: %s", HostEnvValue, err)
		}
	}

	return nil
}

func setCommand(mode setMode) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		value := ctx.Args().First()
		if value == "" {
			fmt.Println("value could not be empty")
			os.Exit(2)
		}

		switch mode {
		case KeyMode:
			if _, err := crypto.LoadPrivateKey(value); err != nil {
				fmt.Println(err.Error())
				os.Exit(2)
			}
			fmt.Printf("set new value for key: %q\n", value)
			viper.Set(KeyCfgValue, value)
			return viper.WriteConfig()
		case HostMode:
			value, err := parseHostValue(value)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(2)
			}
			fmt.Printf("set new value for host: %q\n", value)
			viper.Set(HostCfgValue, value)
			return viper.WriteConfig()
		default:
			fmt.Println("unknown setter type")
			os.Exit(2)
		}

		return nil
	}
}

func parseHostValue(val string) (string, error) {
	host, port, err := net.SplitHostPort(val)
	if err != nil {
		return "", errors.Wrapf(err, "could not fetch host/port: %q", val)
	} else if host == "" {
		host = "0.0.0.0"
	}

	addr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return "", errors.Wrapf(err, `could not resolve address: "%s:%s"`, host, port)
	}

	return net.JoinHostPort(addr.IP.String(), port), nil
}
