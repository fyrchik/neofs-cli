package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nspcc-dev/neofs-api-go/service"
	"github.com/nspcc-dev/neofs-api-go/state"
	"github.com/pkg/errors"
	"github.com/prometheus/common/expfmt"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

var (
	statusAction = &action{}

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

	configAction = &action{
		Action: getConfig,
	}

	dumpVarsAction = &action{
		Action: getVars,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "beauty"},
		},
	}

	changeStateAction = &action{
		Action: changeState,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: stateFlag},
		},
	}
)

func changeState(c *cli.Context) error {
	var (
		err  error
		host = getHost(c)
		conn *grpc.ClientConn
		req  = new(state.ChangeStateRequest)
		ctx  = gracefulContext()
	)

	switch st := c.String(stateFlag); st {
	case "online":
		req.State = state.ChangeStateRequest_Online
	case "offline":
		req.State = state.ChangeStateRequest_Offline
	default:
		return errors.Errorf("unknown state: %q", st)
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	req.SetTTL(service.NonForwardingTTL)
	signRequest(c, req)

	if _, err = state.NewStatusClient(conn).ChangeState(ctx, req); err != nil {
		return errors.Wrap(err, "status command failed on remote call")
	}

	fmt.Println("DONE")

	return nil
}

func getVars(c *cli.Context) error {
	var (
		err  error
		host = getHost(c)
		conn *grpc.ClientConn
		req  = new(state.DumpVarsRequest)
		ctx  = gracefulContext()
	)

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	req.SetTTL(service.NonForwardingTTL)
	signRequest(c, req)

	res, err := state.NewStatusClient(conn).DumpVars(ctx, req)
	if err != nil {
		return errors.Wrap(err, "status command failed on remote call")
	}

	if !c.Bool(beautyFlag) {
		_, err = os.Stdout.Write(res.Variables)
		return err
	}

	dump := make(map[string]interface{})
	if err = json.Unmarshal(res.Variables, &dump); err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "\t")
	return enc.Encode(dump)
}

func getConfig(c *cli.Context) error {
	var (
		err  error
		host = getHost(c)
		conn *grpc.ClientConn
		req  = new(state.DumpRequest)
		ctx  = gracefulContext()
	)

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	req.SetTTL(service.NonForwardingTTL)
	signRequest(c, req)

	res, err := state.NewStatusClient(conn).DumpConfig(ctx, req)
	if err != nil {
		return errors.Wrap(err, "status command failed on remote call")
	}

	_, err = os.Stdout.Write(res.Config)

	return nil
}

func getMetrics(c *cli.Context) error {
	var (
		err  error
		host = getHost(c)
		conn *grpc.ClientConn
		req  = new(state.MetricsRequest)
		ctx  = gracefulContext()
	)

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	setTTL(c, req)
	signRequest(c, req)

	res, err := state.NewStatusClient(conn).Metrics(ctx, req)
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
		host = getHost(c)
		conn *grpc.ClientConn
		req  = new(state.HealthRequest)
		ctx  = gracefulContext()
	)

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	setTTL(c, req)
	signRequest(c, req)

	res, err := state.NewStatusClient(conn).HealthCheck(ctx, req)
	if err != nil {
		return errors.Wrap(err, "status command failed on remote call")
	}

	fmt.Printf("Healthy: %t\nStatus: %s\n", res.Healthy, res.Status)

	return nil
}

func getEpoch(c *cli.Context) error {
	var (
		err  error
		host = getHost(c)
		conn *grpc.ClientConn
		req  = new(state.NetmapRequest)
		ctx  = gracefulContext()
	)

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	setTTL(c, req)
	signRequest(c, req)

	nm, err := state.NewStatusClient(conn).Netmap(ctx, req)
	if err != nil {
		return errors.Wrap(err, "status command failed on remote call")
	}
	fmt.Println(nm.Epoch)

	return nil
}

func getNetmap(c *cli.Context) error {
	var (
		err  error
		host = getHost(c)
		conn *grpc.ClientConn
		req  = new(state.NetmapRequest)
		ctx  = gracefulContext()
	)

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	setTTL(c, req)
	signRequest(c, req)

	nm, err := state.NewStatusClient(conn).Netmap(ctx, req)
	if err != nil {
		return errors.Wrap(err, "status command failed on remote call")
	}

	if err := json.NewEncoder(os.Stdout).Encode(nm); err != nil {
		return errors.Wrap(err, "can't marshall network map to json")
	}
	return nil
}
