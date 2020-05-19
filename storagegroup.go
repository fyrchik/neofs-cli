package main

import (
	"fmt"

	"github.com/nspcc-dev/neofs-api-go/object"
	"github.com/nspcc-dev/neofs-api-go/refs"
	"github.com/nspcc-dev/neofs-api-go/service"
	"github.com/nspcc-dev/neofs-api-go/storagegroup"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

var (
	sgAction = &action{}

	getSGAction = &action{
		Action: getSG,
		Flags: []cli.Flag{
			containerID,
			storagegroupID,
			oidHidden,
			fullHeadersHidden,
		},
	}

	listSGAction = &action{
		Action: listSG,
		Flags:  searchObjectAction.Flags,
	}

	delSGAction = &action{
		Action: delSG,
		Flags: []cli.Flag{
			containerID,
			storagegroupID,
			oidHidden,
			fullHeadersHidden,
		},
	}

	putSGAction = &action{
		Action: putSG,
		Flags: []cli.Flag{
			containerID,
			objectIDs,
		},
	}

	oidHidden = &cli.StringFlag{
		Name:   objFlag,
		Hidden: true,
	}

	fullHeadersHidden = &cli.StringFlag{
		Name:   fullHeadersFlag,
		Hidden: true,
	}
)

func listSG(c *cli.Context) error {
	if err := c.Set(sgFlag, "true"); err != nil {
		return err
	}
	return search(c)
}

func getSG(c *cli.Context) error {
	if err := c.Set(objFlag, c.String(sgidFlag)); err != nil {
		return err
	} else if err := c.Set(fullHeadersFlag, "true"); err != nil {
		return err
	}
	return head(c)
}

func delSG(c *cli.Context) error {
	if err := c.Set(objFlag, c.String(sgidFlag)); err != nil {
		return err
	}
	return del(c)
}

func putSG(c *cli.Context) error {
	var (
		err  error
		key  = getKey(c)
		host = getHost(c)
		conn *grpc.ClientConn
		cid  refs.CID
		oids []refs.ObjectID

		ctx            = gracefulContext()
		strContainerID = c.String(cidFlag)
		strObjectIDs   = c.StringSlice(objFlag)
	)

	if strContainerID == "" || len(strObjectIDs) == 0 {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	// Try to parse container id
	cid, err = refs.CIDFromString(strContainerID)
	if err != nil {
		return errors.Wrapf(err, "could not parse container id %s", strContainerID)
	}

	oids = make([]refs.ObjectID, 0, len(strObjectIDs))
	for i := range strObjectIDs {
		var oid refs.ObjectID
		if err = oid.Parse(strObjectIDs[i]); err != nil {
			return errors.Wrapf(err, "could not parse object id %s", strObjectIDs[i])
		}
		oids = append(oids, oid)
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	sg := &object.Object{
		SystemHeader: object.SystemHeader{
			OwnerID: owner,
			CID:     cid,
		},
		Headers: make([]object.Header, 0, len(oids)+1),
	}

	for i := range oids {
		sg.AddHeader(&object.Header{Value: &object.Header_Link{
			Link: &object.Link{Type: object.Link_StorageGroup, ID: oids[i]},
		}})
	}

	sg.SetStorageGroup(new(storagegroup.StorageGroup))

	objID, err := refs.NewObjectID()
	if err != nil {
		return errors.Wrap(err, "can't generate new object ID")
	}

	sg.SystemHeader.ID = objID

	token, err := createToken(tokenParams{
		connectionParams: connectionParams{
			ctx:  ctx,
			cmd:  c,
			conn: conn,
		},

		addr: refs.Address{
			ObjectID: objID,
			CID:      cid,
		},

		verb: service.Token_Info_Put,
	})
	if err != nil {
		return errors.Wrap(err, "could not create session token")
	}

	client := object.NewServiceClient(conn)
	putClient, err := client.Put(ctx)
	if err != nil {
		return errors.Wrap(err, "put command failed on client creation")
	}

	req := object.MakePutRequestHeader(sg)
	req.SetToken(token)
	setTTL(c, req)
	setRaw(c, req)
	signRequest(c, req)

	if err = putClient.Send(req); err != nil {
		return errors.Wrap(err, "storage group put command failed on Send SG origin")
	}

	resp, err := putClient.CloseAndRecv()
	if err != nil {
		return errors.Wrap(err, "storage group put command failed on CloseAndRecv")
	}

	fmt.Printf("Storage group successfully stored\n\tID: %s\n\tCID: %s\n", resp.Address.ObjectID, resp.Address.CID)

	return nil
}
