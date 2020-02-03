package main

import (
	"fmt"
	"math"

	"github.com/nspcc-dev/neofs-api/object"
	"github.com/nspcc-dev/neofs-api/refs"
	"github.com/nspcc-dev/neofs-api/session"
	"github.com/nspcc-dev/neofs-api/storagegroup"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

var (
	sgAction = &action{}

	getSGAction = &action{
		Action: getSG,
		Flags:  append(headObjectAction.Flags, storagegroupID),
	}

	listSGAction = &action{
		Action: listSG,
		Flags:  searchObjectAction.Flags,
	}

	delSGAction = &action{
		Action: delSG,
		Flags:  append(delObjectAction.Flags, storagegroupID),
	}

	putSGAction = &action{
		Action: putSG,
		Flags: []cli.Flag{
			containerID,
			objectIDs,
		},
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

	token, err := establishSession(ctx, sessionParams{
		cmd:  c,
		key:  key,
		conn: conn,
		token: &session.Token{
			// FirstEpoch: 0,
			ObjectID:  []refs.ObjectID{objID},
			LastEpoch: math.MaxUint64,
		},
	})
	if err != nil {
		return errors.Wrap(err, "can't establish session")
	}

	client := object.NewServiceClient(conn)
	putClient, err := client.Put(ctx)
	if err != nil {
		return errors.Wrap(err, "put command failed on client creation")
	}

	req := object.MakePutRequestHeader(sg, token)
	setTTL(c, req)
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
