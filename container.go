package main

import (
	"context"
	"crypto/ecdsa"
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
			keyFile,
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
		Flags: []cli.Flag{
			keyFile,
		},
	}
)

func putContainer(c *cli.Context) error {
	var (
		err    error
		key    *ecdsa.PrivateKey
		conn   *grpc.ClientConn
		plRule *netmap.PlacementRule
		msgID  refs.MessageID

		host   = c.Parent().String(hostFlag)
		keyArg = c.String(keyFlag)
		cCap   = c.Uint64(capFlag)
		sRule  = c.String(ruleFlag)
	)

	if host == "" || keyArg == "" || sRule == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	// Try to receive key from file
	if key, err = parseKeyValue(keyArg); err != nil {
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
		TTL:       getTTL(c),
	}

	if err = service.SignRequest(req, key); err != nil {
		return errors.Wrap(err, "could not sign request")
	}

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

			resp, err := client.List(ctx, &container.ListRequest{
				OwnerID: owner,
				TTL:     getTTL(c),
			})
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

		host = c.Parent().String(hostFlag)
		sCID = c.String(cidFlag)
	)

	if cid, err = refs.CIDFromString(sCID); err != nil {
		return errors.Wrapf(err, "can't parse CID %s", sCID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure()); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	resp, err := container.NewServiceClient(conn).Get(ctx, &container.GetRequest{CID: cid, TTL: getTTL(c)})
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

		host = c.Parent().String(hostFlag)
		sCID = c.String(cidFlag)
	)

	if cid, err = refs.CIDFromString(sCID); err != nil {
		return errors.Wrapf(err, "can't parse CID %s", sCID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure()); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	_, err = container.NewServiceClient(conn).Delete(ctx, &container.DeleteRequest{CID: cid, TTL: getTTL(c)})

	return errors.Wrap(err, "can't perform request")
}

func listContainers(c *cli.Context) error {
	var (
		err  error
		key  *ecdsa.PrivateKey
		conn *grpc.ClientConn

		host   = c.Parent().String(hostFlag)
		keyArg = c.String(keyFlag)
	)

	if host == "" || keyArg == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	// Try to receive key from file
	if key, err = parseKeyValue(keyArg); err != nil {
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

	resp, err := container.NewServiceClient(conn).List(ctx, &container.ListRequest{
		OwnerID: owner,
		TTL:     getTTL(c),
	})
	if err != nil {
		return errors.Wrapf(err, "can't complete request")
	}

	fmt.Println("Container ID")
	for i := range resp.CID {
		fmt.Println(resp.CID[i])
	}

	return nil
}
