package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nspcc-dev/neofs-proto/hash"
	"github.com/nspcc-dev/neofs-proto/object"
	"github.com/nspcc-dev/neofs-proto/query"
	"github.com/nspcc-dev/neofs-proto/refs"
	"github.com/nspcc-dev/neofs-proto/session"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type (
	grpcState interface {
		GRPCStatus() *status.Status
	}

	sessionParams struct {
		cmd   *cli.Context
		token *session.Token
		conn  *grpc.ClientConn
		key   *ecdsa.PrivateKey
	}
)

const (
	fullHeadersFlag = "full-headers"
	saltFlag        = "salt"
	verifyFlag      = "verify"
	rootFlag        = "root"
	userHeaderFlag  = "user"

	dataChunkSize = 3 * object.UnitsMB
)

var (
	objectAction    = &action{}
	putObjectAction = &action{
		Action: put,
		Flags: []cli.Flag{
			containerID,
			filesPath,
			permissions,
			&cli.BoolFlag{
				Name:  verifyFlag,
				Usage: "verify checksum after put",
			},
			&cli.StringSliceFlag{
				Name:  userHeaderFlag,
				Usage: "provide optional user headers",
			},
		},
	}
	getObjectAction = &action{
		Action: get,
		Flags: []cli.Flag{
			containerID,
			objectID,
			filePath,
			permissions,
		},
	}
	delObjectAction = &action{
		Action: del,
		Flags: []cli.Flag{
			containerID,
			objectID,
		},
	}
	headObjectAction = &action{
		Action: head,
		Flags: []cli.Flag{
			containerID,
			objectID,
			fullHeaders,
		},
	}
	searchObjectAction = &action{
		Action: search,
		Flags: []cli.Flag{
			containerID,
			storageGroup,
			&cli.BoolFlag{
				Name:  rootFlag,
				Usage: "search only user's objects",
			},
		},
	}
	getRangeObjectAction = &action{
		Action: getRange,
		Flags: []cli.Flag{
			containerID,
			objectID,
		},
	}
	getRangeHashObjectAction = &action{
		Action: getRangeHash,
		Flags: []cli.Flag{
			containerID,
			objectID,
			&cli.StringFlag{
				Name:  saltFlag,
				Usage: "salt to hash with",
			},
			&cli.BoolFlag{
				Name:  verifyFlag,
				Usage: "verify hash",
			},
			filePath,
			permissions,
		},
	}
)

func del(c *cli.Context) error {
	var (
		err   error
		key   = getKey(c)
		host  = getHost(c)
		cid   refs.CID
		objID refs.ObjectID
		conn  *grpc.ClientConn

		cidArg = c.String(cidFlag)
		objArg = c.String(objFlag)
		ctx    = gracefulContext()
	)

	if cidArg == "" || objArg == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(cidArg); err != nil {
		return errors.Wrapf(err, "can't parse CID '%s'", cidArg)
	}

	if err = objID.Parse(objArg); err != nil {
		return errors.Wrapf(err, "can't parse object id '%s'", objArg)
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

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

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	req := &object.DeleteRequest{
		Address: refs.Address{
			CID:      cid,
			ObjectID: objID,
		},
		OwnerID: owner,
		Token:   token,
	}
	setTTL(c, req)
	signRequest(c, req)

	_, err = object.NewServiceClient(conn).Delete(ctx, req)
	if err != nil {
		return errors.Wrap(err, "can't perform DELETE request")
	}

	return nil
}

func head(c *cli.Context) error {
	var (
		err   error
		conn  *grpc.ClientConn
		cid   refs.CID
		objID refs.ObjectID

		host   = getHost(c)
		cidArg = c.String(cidFlag)
		objArg = c.String(objFlag)
		fh     = c.Bool(fullHeadersFlag)
		ctx    = gracefulContext()
	)

	if cidArg == "" || objArg == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(cidArg); err != nil {
		return errors.Wrapf(err, "can't parse CID '%s'", cidArg)
	}

	if err = objID.Parse(objArg); err != nil {
		return errors.Wrapf(err, "can't parse object id '%s'", objArg)
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	req := &object.HeadRequest{
		Address: refs.Address{
			CID:      cid,
			ObjectID: objID,
		},
		FullHeaders: fh,
	}
	setTTL(c, req)
	signRequest(c, req)

	resp, err := object.NewServiceClient(conn).Head(ctx, req)
	if err != nil {
		return errors.Wrap(err, "can't perform HEAD request")
	}

	fmt.Println("System headers:")
	fmt.Printf("  Object ID   : %s\n", resp.Object.SystemHeader.ID)
	fmt.Printf("  Owner ID    : %s\n", resp.Object.SystemHeader.OwnerID)
	fmt.Printf("  Container ID: %s\n", resp.Object.SystemHeader.CID)
	fmt.Printf("  Payload Size: %s\n", object.ByteSize(resp.Object.SystemHeader.PayloadLength))
	fmt.Printf("  Version     : %d\n", resp.Object.SystemHeader.Version)
	fmt.Printf("  Created at  : epoch #%d, %s\n", resp.Object.SystemHeader.CreatedAt.Epoch, time.Unix(resp.Object.SystemHeader.CreatedAt.UnixTime, 0))
	if len(resp.Object.Headers) != 0 {
		fmt.Println("Other headers:")
		for i := range resp.Object.Headers {
			fmt.Println("  " + resp.Object.Headers[i].String())
		}
	}

	return nil
}

func search(c *cli.Context) error {
	var (
		err  error
		conn *grpc.ClientConn
		cid  refs.CID
		q    query.Query

		host   = getHost(c)
		cidArg = c.String(cidFlag)
		qArgs  = c.Args()
		isRoot = c.Bool(rootFlag)
		sg     = c.Bool(sgFlag)
		ctx    = gracefulContext()
	)

	if cidArg == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if c.NArg()%2 != 0 {
		return errors.Errorf("number of positional arguments must be event\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(cidArg); err != nil {
		return errors.Wrapf(err, "can't parse CID '%s'", cidArg)
	}

	for i := 0; i < qArgs.Len(); i += 2 {
		q.Filters = append(q.Filters, query.Filter{
			Type:  query.Filter_Regex,
			Name:  qArgs.Get(i),
			Value: qArgs.Get(i + 1),
		})
	}
	if isRoot {
		q.Filters = append(q.Filters, query.Filter{
			Type: query.Filter_Exact,
			Name: object.KeyRootObject,
		})
	}
	if sg {
		q.Filters = append(q.Filters, query.Filter{
			Type: query.Filter_Exact,
			Name: object.KeyStorageGroup,
		})
	}

	data, err := q.Marshal()
	if err != nil {
		return errors.Wrap(err, "can't marshal query")
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	req := &object.SearchRequest{
		ContainerID:  cid,
		Query:        data,
		QueryVersion: 1,
	}
	setTTL(c, req)
	signRequest(c, req)

	resp, err := object.NewServiceClient(conn).Search(ctx, req)
	if err != nil {
		return errors.Wrap(err, "can't perform SEARCH request")
	}

	fmt.Println("Container ID: Object ID")
	for i := range resp.Addresses {
		fmt.Println(resp.Addresses[i].CID.String() + ": " + resp.Addresses[i].ObjectID.String())
	}

	return nil
}

func getRange(c *cli.Context) error {
	var (
		err    error
		conn   *grpc.ClientConn
		cid    refs.CID
		objID  refs.ObjectID
		ranges []object.Range

		host   = getHost(c)
		cidArg = c.String(cidFlag)
		objArg = c.String(objFlag)
		rngArg = c.Args()
		ctx    = gracefulContext()
	)

	if cidArg == "" || objArg == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(cidArg); err != nil {
		return errors.Wrapf(err, "can't parse CID '%s'", cidArg)
	}

	if err = objID.Parse(objArg); err != nil {
		return errors.Wrapf(err, "can't parse object id '%s'", objArg)
	}

	ranges, err = parseRanges(rngArg)
	if err != nil {
		return errors.Wrap(err, "can't parse ranges")
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	req := &object.GetRangeRequest{
		Address: refs.Address{
			ObjectID: objID,
			CID:      cid,
		},
		Ranges: ranges,
	}
	setTTL(c, req)
	signRequest(c, req)

	resp, err := object.NewServiceClient(conn).GetRange(ctx, req)
	if err != nil {
		return errors.Wrap(err, "can't perform GETRANGE request")
	}

	// TODO process response
	_ = resp

	return nil
}

func getRangeHash(c *cli.Context) error {
	var (
		err    error
		conn   *grpc.ClientConn
		cid    refs.CID
		objID  refs.ObjectID
		ranges []object.Range
		salt   []byte

		host    = getHost(c)
		cidArg  = c.String(cidFlag)
		objArg  = c.String(objFlag)
		saltArg = c.String(saltFlag)
		verify  = c.Bool(verifyFlag)
		fPath   = c.String(fileFlag)
		perm    = c.Int(permFlag)
		rngArg  = c.Args()
		ctx     = gracefulContext()
	)

	if cidArg == "" || objArg == "" || saltArg == "" || len(fPath) == 0 {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(cidArg); err != nil {
		return errors.Wrapf(err, "can't parse CID '%s'", cidArg)
	}

	if err = objID.Parse(objArg); err != nil {
		return errors.Wrapf(err, "can't parse object id '%s'", objArg)
	}

	if salt, err = hex.DecodeString(saltArg); err != nil {
		return errors.Wrapf(err, "can't decode salt")
	}

	ranges, err = parseRanges(rngArg)
	if err != nil {
		return errors.Wrap(err, "can't parse ranges")
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	req := &object.GetRangeHashRequest{
		Address: refs.Address{
			ObjectID: objID,
			CID:      cid,
		},
		Ranges: ranges,
		Salt:   salt,
	}
	setTTL(c, req)
	signRequest(c, req)

	resp, err := object.NewServiceClient(conn).GetRangeHash(ctx, req)
	if err != nil {
		return errors.Wrap(err, "can't perform GETRANGEHASH request")
	}

	var fd *os.File
	if verify {
		if fd, err = os.OpenFile(fPath, os.O_RDONLY, os.FileMode(perm)); err != nil {
			return errors.Wrap(err, "could not open file")
		}
	}

	for i := range resp.Hashes {
		if verify {
			d := make([]byte, ranges[i].Length)
			if _, err = fd.ReadAt(d, int64(ranges[i].Offset)); err != nil && err != io.EOF {
				return errors.Wrap(err, "could not read range from file")
			}

			xor := hash.SaltXOR(d[:ranges[i].Length], salt)

			fmt.Print("(")
			if !hash.Sum(xor).Equal(resp.Hashes[i]) {
				fmt.Print("in")
			}
			fmt.Print("valid) ")
		}
		fmt.Printf("%s\n", resp.Hashes[i])
	}

	return nil
}

func parseRanges(rng cli.Args) (ranges []object.Range, err error) {
	ranges = make([]object.Range, rng.Len())
	for i := 0; i < rng.Len(); i++ {
		var (
			t     uint64
			items = strings.Split(rng.Get(i), ":")
		)
		if len(items) != 2 {
			return nil, errors.New("range must have form 'offset:length'")
		}
		t, err = strconv.ParseUint(items[0], 10, 32)
		if err != nil {
			return nil, errors.Wrap(err, "can't parse offset")
		}
		ranges[i].Offset = t

		t, err = strconv.ParseUint(items[1], 10, 32)
		if err != nil {
			return nil, errors.Wrap(err, "can't parse length")
		}
		ranges[i].Length = t
	}
	return
}

func put(c *cli.Context) error {
	var (
		err   error
		cid   refs.CID
		conn  *grpc.ClientConn
		fd    *os.File
		fSize int64

		key    = getKey(c)
		host   = getHost(c)
		sCID   = c.String(cidFlag)
		fPaths = c.StringSlice(fileFlag)
		perm   = c.Int(permFlag)
		verify = c.Bool(verifyFlag)
		userH  = c.StringSlice(userHeaderFlag)
		ctx    = gracefulContext()
	)

	if sCID == "" || len(fPaths) == 0 {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(sCID); err != nil {
		return errors.Wrapf(err, "can't parse CID %s", sCID)
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	for i := range fPaths {
		fPath := fPaths[i]

		if fd, err = os.OpenFile(fPath, os.O_RDONLY, os.FileMode(perm)); err != nil {
			return errors.Wrapf(err, "can't open file %s", fPath)
		}

		fi, err := fd.Stat()
		if err != nil {
			return errors.Wrap(err, "can't get file info")
		}

		fSize = fi.Size()

		objID, err := refs.NewObjectID()
		if err != nil {
			return errors.Wrap(err, "can't generate new object ID")
		}

		token, err := establishSession(ctx, sessionParams{
			cmd:  c,
			key:  key,
			conn: conn,
			token: &session.Token{
				ObjectID:   []refs.ObjectID{objID},
				FirstEpoch: 0,
				LastEpoch:  math.MaxUint64,
			},
		})
		if st, ok := err.(grpcState); ok {
			state := st.GRPCStatus()
			return errors.Errorf("%s (%s): %s", host, state.Code(), state.Message())
		} else if err != nil {
			return errors.Wrap(err, "can't establish session")
		}

		client := object.NewServiceClient(conn)
		putClient, err := client.Put(ctx)
		if err != nil {
			return errors.Wrap(err, "put command failed on client creation")
		}

		var (
			n      int
			curOff int64
			data   = make([]byte, dataChunkSize)
			obj    = &object.Object{
				SystemHeader: object.SystemHeader{
					ID:            objID,
					OwnerID:       owner,
					CID:           cid,
					PayloadLength: uint64(fSize),
				},
				Headers: parseUserHeaders(userH),
			}
		)

		fmt.Printf("[%s] Sending header...\n", fPath)

		req := object.MakePutRequestHeader(obj, token)
		setTTL(c, req)
		signRequest(c, req)

		if err = putClient.Send(req); err != nil {
			return errors.Wrap(err, "put command failed on Send object origin")
		}

		fmt.Printf("[%s] Sending data...\n", fPath)
		h := hash.Sum(nil)
		for ; err != io.EOF; curOff += int64(n) {
			if n, err = fd.ReadAt(data, curOff); err != nil && err != io.EOF {
				return errors.Wrap(err, "put command failed on file read")
			}

			if n > 0 {
				if verify {
					h, _ = hash.Concat([]hash.Hash{h, hash.Sum(data[:n])})
				}

				req := object.MakePutRequestChunk(data[:n])
				setTTL(c, req)
				signRequest(c, req)

				if err := putClient.Send(req); err != nil && err != io.EOF {
					return errors.Wrap(err, "put command failed on Send")
				}
			}
		}

		resp, err := putClient.CloseAndRecv()
		if err != nil {
			return errors.Wrap(err, "put command failed on CloseAndRecv")
		}

		fmt.Printf("[%s] Object successfully stored\n", fPath)
		fmt.Printf("  ID: %s\n  CID: %s\n", resp.Address.ObjectID, resp.Address.CID)
		if verify {
			result := "success"
			req := &object.GetRangeHashRequest{
				Address: refs.Address{
					ObjectID: resp.Address.ObjectID,
					CID:      resp.Address.CID,
				},
				Ranges: []object.Range{{Offset: 0, Length: obj.SystemHeader.PayloadLength}},
			}

			setTTL(c, req)
			signRequest(c, req)

			if r, err := client.GetRangeHash(ctx, req); err != nil {
				result = "can't perform GETRANGEHASH request"
			} else if len(r.Hashes) == 0 {
				result = "empty hash list received"
			} else if !r.Hashes[0].Equal(h) {
				result = "hashes are not equal"
			}
			fmt.Printf("Verification result: %s.\n", result)
		}
	}

	return nil
}

func parseUserHeaders(userH []string) (headers []object.Header) {
	headers = make([]object.Header, len(userH))
	for i := range userH {
		kv := strings.SplitN(userH[i], "=", 2)
		uh := &object.UserHeader{Key: kv[0]}
		if len(kv) > 1 {
			uh.Value = kv[1]
		}
		headers[i].Value = &object.Header_UserHeader{UserHeader: uh}
	}
	return
}

func establishSession(ctx context.Context, p sessionParams) (*session.Token, error) {
	client, err := session.NewSessionClient(p.conn).Create(ctx)
	if err != nil {
		return nil, err
	}

	owner, err := refs.NewOwnerID(&p.key.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute owner ID")
	}

	token := &session.Token{
		OwnerID:    owner,
		ObjectID:   p.token.ObjectID,
		FirstEpoch: p.token.FirstEpoch,
		LastEpoch:  p.token.LastEpoch,
	}
	token.SetPublicKeys(&p.key.PublicKey)

	req := session.NewInitRequest(token)
	setTTL(p.cmd, req)
	signRequest(p.cmd, req)

	if err := client.Send(req); err != nil {
		return nil, err
	}

	resp, err := client.Recv()
	if err != nil {
		return nil, err
	}

	// receive first response and check than nothing was changed
	unsigned := resp.GetUnsigned()
	if unsigned == nil {
		return nil, errors.New("expected unsigned token")
	}

	same := unsigned.FirstEpoch == token.FirstEpoch && unsigned.LastEpoch == token.LastEpoch &&
		unsigned.OwnerID == token.OwnerID && len(unsigned.ObjectID) == len(token.ObjectID)
	if same {
		for i := range unsigned.ObjectID {
			if !unsigned.ObjectID[i].Equal(token.ObjectID[i]) {
				same = false
				break
			}
		}
	}

	if !same {
		return nil, errors.New("received token differ")
	} else if unsigned.Header.PublicKey == nil {
		return nil, errors.New("received nil public key")
	} else if err = unsigned.Sign(p.key); err != nil {
		return nil, errors.Wrap(err, "can't sign token")
	}

	req = session.NewSignedRequest(unsigned)
	setTTL(p.cmd, req)
	signRequest(p.cmd, req)
	if err = client.Send(req); err != nil {
		return nil, err
	} else if resp, err = client.Recv(); err != nil {
		return nil, err
	} else if result := resp.GetResult(); result != nil {
		return result, nil
	}
	return nil, errors.New("expected result token")
}

func get(c *cli.Context) error {
	var (
		err  error
		fd   *os.File
		cid  refs.CID
		oid  refs.ObjectID
		conn *grpc.ClientConn

		host  = getHost(c)
		sCID  = c.String(cidFlag)
		sOID  = c.String(objFlag)
		fPath = c.String(fileFlag)
		perm  = c.Int(permFlag)
		ctx   = gracefulContext()
	)

	if sCID == "" || sOID == "" || len(fPath) == 0 {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(sCID); err != nil {
		return errors.Wrapf(err, "can't parse CID %s", sCID)
	}

	if err = oid.Parse(sOID); err != nil {
		return errors.Wrapf(err, "can't parse Object ID %s", sOID)
	}

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	req := &object.GetRequest{
		Address: refs.Address{
			ObjectID: oid,
			CID:      cid,
		},
	}
	setTTL(c, req)
	signRequest(c, req)

	getClient, err := object.NewServiceClient(conn).Get(ctx, req)
	if err != nil {
		return errors.Wrap(err, "get command failed on client creation")
	}

	fmt.Println("Waiting for data...")

	var objectOriginReceived bool

	for {
		resp, err := getClient.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "get command received error")
		}

		if !objectOriginReceived {
			obj := resp.GetObject()

			if _, hdr := obj.LastHeader(object.HeaderType(object.TombstoneHdr)); hdr != nil {
				if err := obj.Verify(); err != nil {
					fmt.Println("Object corrupted")
					return err
				}
				fmt.Println("Object removed")
				return nil
			}

			fmt.Printf("Object origin received: %s\n", resp.GetObject().SystemHeader.ID)

			if fd, err = os.OpenFile(fPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.FileMode(perm)); err != nil {
				return errors.Wrapf(err, "can't open file %s", fPath)
			}

			if _, err := fd.Write(obj.Payload); err != nil && err != io.EOF {
				return errors.Wrap(err, "get command failed on file write")
			}
			objectOriginReceived = true
			fmt.Print("receiving chunks: ")
			continue
		}

		chunk := resp.GetChunk()

		fmt.Print("#")

		if _, err := fd.Write(chunk); err != nil && err != io.EOF {
			return errors.Wrap(err, "get command failed on file write")
		}
	}

	fmt.Println("\nObject successfully fetched")

	return nil
}
