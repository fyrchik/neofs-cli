package main

import (
	"context"
	"fmt"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/nspcc-dev/neofs-proto/container"
	"github.com/nspcc-dev/neofs-proto/object"
	"github.com/nspcc-dev/neofs-proto/refs"
	"github.com/nspcc-dev/neofs-proto/service"
	"github.com/nspcc-dev/netmap"
	query "github.com/nspcc-dev/netmap-ql"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

const (
	ruleFlag = "rule"
	capFlag  = "cap"

	defaultCapacity = 1
)

var (
	containerAction = &action{
		Flags: []cli.Flag{
			hostAddr,
		},
	}
	putContainerAction = &action{
		Action: putContainer,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  ruleFlag,
				Usage: "container rules",
			},
			cli.Uint64Flag{
				Name:  capFlag,
				Usage: "container capacity in GB",
				Value: defaultCapacity,
			},
		},
	}
	getContainerAction = &action{
		Action: getContainer,
		Flags: []cli.Flag{
			containerID,
		},
	}
	delContainerAction = &action{
		Action: delContainer,
		Flags: []cli.Flag{
			containerID,
		},
	}
	listContainersAction = &action{
		Action: listContainers,
	}
)

func putContainer(c *cli.Context) error {
	var (
		err    error
		key    = getKey(c)
		msgID  refs.MessageID
		conn   *grpc.ClientConn
		plRule *netmap.PlacementRule

		host  = c.Parent().String(hostFlag)
		cCap  = c.Uint64(capFlag)
		sRule = c.String(ruleFlag)
	)

	if host == "" || sRule == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	if plRule, err = query.ParseQuery(sRule); err != nil {
		return errors.Wrapf(err, "placement rule parse failed %s", sRule)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure()); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	if msgID, err = refs.NewMessageID(); err != nil {
		return errors.Wrap(err, "could not create message ID")
	}

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	req := &container.PutRequest{
		MessageID: msgID,
		Capacity:  cCap * uint64(object.UnitsGB),
		OwnerID:   owner,
		Rules:     *plRule,
	}

	if err = service.SignRequest(req, key); err != nil {
		return errors.Wrap(err, "could not sign request")
	}

	setTTL(c, req)
	signRequest(c, req)

	resp, err := container.NewServiceClient(conn).Put(ctx, req)
	if err != nil {
		return errors.Wrap(err, "put request failed")
	}

	fmt.Printf("Container processed: %s\n\n", resp.CID)
	fmt.Println("Trying to wait until container will be accepted on consensus...")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// store response CID
	cid := resp.CID
	client := container.NewServiceClient(conn)

loop:
	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			fmt.Println("Timeout exceeded! Something went wrong.")
			fmt.Println("Try to find your container by command `container list` or retry in few minutes.")
			break loop
		case <-ticker.C:
			fmt.Printf("...")

			req := &container.ListRequest{OwnerID: owner}
			setTTL(c, req)
			signRequest(c, req)

			resp, err := client.List(ctx, req)
			if err != nil {
				continue loop
			}

			for i := range resp.CID {
				if resp.CID[i].Equal(cid) {
					fmt.Printf("\nSuccess! Container <%s> created.\n", cid)

					break loop
				}
			}
		}
	}

	return nil
}

func getContainer(c *cli.Context) error {
	var (
		err  error
		cid  refs.CID
		conn *grpc.ClientConn

		sCID = c.String(cidFlag)
		host = c.Parent().String(hostFlag)
	)

	if cid, err = refs.CIDFromString(sCID); err != nil {
		return errors.Wrapf(err, "can't parse CID %s", sCID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure()); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	req := &container.GetRequest{CID: cid}
	setTTL(c, req)
	signRequest(c, req)

	resp, err := container.NewServiceClient(conn).Get(ctx, req)
	if err != nil {
		return errors.Wrap(err, "can't perform request")
	}

	fmt.Printf("Container ID: %s\n", cid)
	fmt.Printf("Owner ID    : %s\n", resp.Container.OwnerID)
	fmt.Printf("Capacity    : %s\n", bytefmt.ByteSize(resp.Container.Capacity))
	fmt.Printf("Placement   : %s\n", resp.Container.Rules.String())
	fmt.Printf("Salt        : %s\n", resp.Container.Salt)

	return nil
}

func delContainer(c *cli.Context) error {
	var (
		err  error
		cid  refs.CID
		conn *grpc.ClientConn

		sCID = c.String(cidFlag)
		host = c.Parent().String(hostFlag)
	)

	if cid, err = refs.CIDFromString(sCID); err != nil {
		return errors.Wrapf(err, "can't parse CID %s", sCID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure()); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	req := &container.DeleteRequest{CID: cid}
	setTTL(c, req)
	signRequest(c, req)

	_, err = container.NewServiceClient(conn).Delete(ctx, req)

	return errors.Wrap(err, "can't perform request")
}

func listContainers(c *cli.Context) error {
	var (
		err  error
		key  = getKey(c)
		conn *grpc.ClientConn
		host = c.Parent().String(hostFlag)
	)

	if host == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure()); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	req := &container.ListRequest{OwnerID: owner}
	setTTL(c, req)
	signRequest(c, req)

	resp, err := container.NewServiceClient(conn).List(ctx, req)
	if err != nil {
		return errors.Wrapf(err, "can't complete request")
	}

	fmt.Println("Container ID")
	for i := range resp.CID {
		fmt.Println(resp.CID[i])
	}

	return nil
}
