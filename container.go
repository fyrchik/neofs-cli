package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nspcc-dev/neofs-api-go/container"
	"github.com/nspcc-dev/neofs-api-go/object"
	"github.com/nspcc-dev/neofs-api-go/refs"
	crypto "github.com/nspcc-dev/neofs-crypto"
	"github.com/nspcc-dev/netmap"
	query "github.com/nspcc-dev/netmap-ql"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

const (
	ruleFlag = "rule"
	capFlag  = "cap"
	aclFlag  = "acl"

	timeoutFlag = "timeout"

	defaultCapacity = 1

	publicContainerACLRule   = 0x1FFFFFFF
	privateContainerACLRule  = 0x18888888
	readonlyContainerACLRule = 0x1FFF88FF
)

var (
	containerAction    = &action{}
	putContainerAction = &action{
		Action: putContainer,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Required: true,
				Name:     ruleFlag,
				Usage:    "container rules",
			},
			&cli.Uint64Flag{
				Name:  capFlag,
				Usage: "container capacity in GB",
				Value: defaultCapacity,
			},
			&cli.StringFlag{
				Name:  aclFlag,
				Usage: "basic ACL: public, private, readonly or 32-bit hex",
				Value: "private",
			},
			&cli.DurationFlag{
				Name:  timeoutFlag,
				Usage: "create container timeout",
				Value: time.Minute * 2,
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

	setContainerEACLAction = &action{
		Flags: []cli.Flag{
			containerID,
			eacl,
		},
		Action: setContainerEACL,
	}

	getContainerEACLAction = &action{
		Flags: []cli.Flag{
			containerID,
		},
		Action: getContainerEACL,
	}
)

func putContainer(c *cli.Context) error {
	var (
		err      error
		basicACL uint64
		key      = getKey(c)
		host     = getHost(c)
		msgID    refs.MessageID
		conn     *grpc.ClientConn
		ctx      = gracefulContext()
		cCap     = c.Uint64(capFlag)
		sRule    = c.String(ruleFlag)
		sACL     = strings.TrimLeft(c.String(aclFlag), "0x")
		plRule   *netmap.PlacementRule

		createTimeout = c.Duration(timeoutFlag)
	)

	if sRule == "" || cCap == 0 {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if plRule, err = query.ParseQuery(sRule); err != nil {
		return errors.Wrapf(err, "placement rule parse failed %s", sRule)
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	if msgID, err = refs.NewMessageID(); err != nil {
		return errors.Wrap(err, "could not create message ID")
	}

	switch sACL {
	case "public":
		basicACL = publicContainerACLRule
	case "private":
		basicACL = privateContainerACLRule
	case "readonly":
		basicACL = readonlyContainerACLRule
	default:
		basicACL, err = strconv.ParseUint(sACL, 16, 32)
		if err != nil {
			return errors.Wrap(err, "incorrect basic ACL")
		}
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
		BasicACL:  uint32(basicACL),
	}

	setTTL(c, req)
	setRaw(c, req)
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

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

loop:
	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			fmt.Println("Timeout exceeded! Something went wrong.")
			fmt.Println("Try to find your container by command `container list` or retry in few minutes.")
			os.Exit(2)
		case <-ticker.C:
			fmt.Printf("...")

			req := &container.ListRequest{OwnerID: owner}
			setTTL(c, req)
			setRaw(c, req)
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

func fetchContainer(ctx context.Context, con *grpc.ClientConn, cid refs.CID, cli *cli.Context) (*container.GetResponse, error) {
	req := &container.GetRequest{CID: cid}
	setTTL(cli, req)
	setRaw(cli, req)
	signRequest(cli, req)
	return container.NewServiceClient(con).Get(ctx, req)
}

func sfGroupStringify(g netmap.SFGroup) string {
	w := new(strings.Builder)
	for i := range g.Selectors {
		_, _ = fmt.Fprintf(w, "SELECT %d %s", g.Selectors[i].Count, g.Selectors[i].Key)

		if len(g.Filters) > i {
			val := ""
			switch v := g.Filters[i].F.Args.(type) {
			case *netmap.SimpleFilter_Value:
				val = v.Value
			default:
				val = "UNKNOWN"
			}

			_, _ = fmt.Fprintf(w, " FILTER %s %s %v",
				g.Filters[i].Key,
				g.Filters[i].F.Op,
				val)
		}
	}

	return w.String()
}

func placementStringify(p *netmap.PlacementRule) string {
	result := new(strings.Builder)

	_, _ = fmt.Fprintf(result, "RF %d ", p.ReplFactor)

	items := make([]string, 0, len(p.SFGroups))
	for i := range p.SFGroups {
		items = append(items, sfGroupStringify(p.SFGroups[i]))
	}

	result.WriteString(strings.Join(items, "; "))

	return result.String()
}

func getContainer(c *cli.Context) error {
	var (
		err  error
		cid  refs.CID
		host = getHost(c)
		conn *grpc.ClientConn
		sCID = c.String(cidFlag)
		ctx  = gracefulContext()
	)

	if sCID == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(sCID); err != nil {
		return errors.Wrapf(err, "can't parse CID %s", sCID)
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	resp, err := fetchContainer(ctx, conn, cid, c)
	if err != nil {
		return errors.Wrap(err, "can't perform request")
	}

	fmt.Printf("Container ID: %s\n", cid)
	fmt.Printf("Owner ID    : %s\n", resp.Container.OwnerID)
	fmt.Printf("Capacity    : %s\n", object.ByteSize(resp.Container.Capacity))
	fmt.Printf("Placement   : %s\n", placementStringify(&resp.Container.Rules))
	fmt.Printf("Salt        : %s\n", resp.Container.Salt)
	fmt.Printf("BasicACL    : %08x\n", resp.Container.BasicACL)

	return nil
}

func delContainer(c *cli.Context) error {
	var (
		err  error
		cid  refs.CID
		host = getHost(c)
		conn *grpc.ClientConn
		sCID = c.String(cidFlag)
		ctx  = gracefulContext()
	)

	if sCID == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(sCID); err != nil {
		return errors.Wrapf(err, "can't parse CID %s", sCID)
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	req := &container.DeleteRequest{CID: cid}
	setTTL(c, req)
	setRaw(c, req)
	signRequest(c, req)

	_, err = container.NewServiceClient(conn).Delete(ctx, req)

	return errors.Wrap(err, "can't perform request")
}

func listContainers(c *cli.Context) error {
	var (
		err  error
		key  = getKey(c)
		host = getHost(c)
		conn *grpc.ClientConn
		ctx  = gracefulContext()
	)

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	req := &container.ListRequest{OwnerID: owner}
	setTTL(c, req)
	setRaw(c, req)
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

func setContainerEACL(c *cli.Context) error {
	var (
		err   error
		cid   refs.CID
		eacl  []byte
		sig   []byte
		key   = getKey(c)
		host  = getHost(c)
		conn  *grpc.ClientConn
		sCID  = c.String(cidFlag)
		sEACL = c.String(eaclFlag)
		ctx   = gracefulContext()
	)

	if sCID == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(sCID); err != nil {
		return errors.Wrapf(err, "can't parse CID %s", sCID)
	}

	switch sEACL {
	case "empty":
		eacl = make([]byte, 0)
	default:
		if eacl, err = hex.DecodeString(sEACL); err != nil {
			return errors.Wrap(err, "could not decode extended ACL")
		}
	}

	if sig, err = crypto.SignRFC6979(key, eacl); err != nil {
		return errors.Wrap(err, "could not sign extended ACL")
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	req := new(container.SetExtendedACLRequest)
	req.SetID(cid)
	req.SetEACL(eacl)
	req.SetSignature(sig)

	setTTL(c, req)
	setRaw(c, req)
	signRequest(c, req)

	fmt.Println("Updating ACL rules of container...")

	_, err = container.NewServiceClient(conn).SetExtendedACL(ctx, req)
	if err != nil {
		return errors.Wrapf(err, "can't complete request")
	}

	fmt.Println("Extended ACL rules was successfully updated.")

	return nil
}

func getContainerEACL(c *cli.Context) error {
	var (
		err  error
		cid  refs.CID
		key  = getKey(c)
		host = getHost(c)
		conn *grpc.ClientConn
		sCID = c.String(cidFlag)
		ctx  = gracefulContext()
	)

	if sCID == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(sCID); err != nil {
		return errors.Wrapf(err, "can't parse CID %s", sCID)
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	req := new(container.GetExtendedACLRequest)
	req.SetID(cid)

	setTTL(c, req)
	setRaw(c, req)
	signRequest(c, req)

	fmt.Println("Waiting for ACL rules of container...")

	resp, err := container.NewServiceClient(conn).GetExtendedACL(ctx, req)
	if err != nil {
		return errors.Wrapf(err, "can't complete request")
	}

	if err := crypto.VerifyRFC6979(&key.PublicKey, resp.GetEACL(), resp.GetSignature()); err != nil {
		return errors.Wrap(err, "could not verify signature")
	}

	fmt.Printf("Extended container ACL table: %s\n", hex.EncodeToString(resp.GetEACL()))

	return nil
}
