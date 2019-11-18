package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nspcc-dev/neofs-proto/state"
	"github.com/pkg/errors"
	"github.com/prometheus/common/expfmt"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

var (
	statusAction = &action{
		Flags: []cli.Flag{
			hostAddr,
		},
	}

	epochAction = &action{
		Action: getEpoch,
	}

	netmapAction = &action{
		Action: getNetmap,
	}

	metricsAction = &action{
		Action: getMetrics,
	}

	healthyAction = &action{
		Action: getHealthy,
	}
)

func getMetrics(c *cli.Context) error {
	var (
		err  error
		conn *grpc.ClientConn
		host = c.Parent().String(hostFlag)
	)

	if host == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	client := state.NewStatusClient(conn)
	res, err := client.Metrics(ctx, new(state.MetricsRequest))
	if err != nil {
		return errors.Wrap(err, "status command failed on remote call")
	}

	metrics, err := state.DecodeMetrics(res)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal metrics")
	}

	enc := expfmt.NewEncoder(os.Stdout, expfmt.FmtText)

	for _, mf := range metrics {
		if err := enc.Encode(mf); err != nil {
			fmt.Println("error encoding and sending metric family:", err)
			os.Exit(2)
		}
	}

	return nil
}

func getHealthy(c *cli.Context) error {
	var (
		err  error
		conn *grpc.ClientConn
		host = c.Parent().String(hostFlag)
	)

	if host == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	client := state.NewStatusClient(conn)
	res, err := client.HealthCheck(ctx, new(state.HealthRequest))
	if err != nil {
		return errors.Wrap(err, "status command failed on remote call")
	}

	fmt.Printf("Healthy: %t\nStatus: %s\n", res.Healthy, res.Status)

	return nil
}

func getEpoch(c *cli.Context) error {
	var (
		err  error
		conn *grpc.ClientConn
		host = c.Parent().String(hostFlag)
	)

	if host == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	client := state.NewStatusClient(conn)
	nm, err := client.Netmap(ctx, new(state.NetmapRequest))
	if err != nil {
		return errors.Wrap(err, "status command failed on remote call")
	}
	fmt.Println(nm.Epoch)

	return nil
}

func getNetmap(c *cli.Context) error {
	var (
		err  error
		conn *grpc.ClientConn
		host = c.Parent().String(hostFlag)
	)

	if host == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	client := state.NewStatusClient(conn)
	nm, err := client.Netmap(ctx, new(state.NetmapRequest))
	if err != nil {
		return errors.Wrap(err, "status command failed on remote call")
	}

	if err := json.NewEncoder(os.Stdout).Encode(nm); err != nil {
		return errors.Wrap(err, "can't marshall network map to json")
	}
	return nil
}
