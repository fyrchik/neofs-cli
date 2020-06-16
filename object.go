package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/nspcc-dev/neofs-api-go/hash"
	"github.com/nspcc-dev/neofs-api-go/object"
	"github.com/nspcc-dev/neofs-api-go/query"
	"github.com/nspcc-dev/neofs-api-go/refs"
	"github.com/nspcc-dev/neofs-api-go/service"
	"github.com/nspcc-dev/neofs-api-go/session"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type (
	grpcState interface {
		GRPCStatus() *status.Status
	}

	connectionParams struct {
		ctx context.Context

		cmd *cli.Context

		conn *grpc.ClientConn
	}

	sessionParams struct {
		connectionParams

		prm session.CreateParamsSource

		res interface {
			service.TokenIDContainer
			service.SessionKeyContainer
		}
	}

	tokenParams struct {
		connectionParams

		addr refs.Address

		verb service.Token_Info_Verb
	}
)

const (
	fullHeadersFlag = "full-headers"
	saltFlag        = "salt"
	verifyFlag      = "verify"
	rootFlag        = "root"
	userHeaderFlag  = "user"
	rawFlag         = "raw"
	copiesNumFlag   = "copies"

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
			&cli.Uint64Flag{
				Name:  copiesNumFlag,
				Usage: "set number of copies to store",
			},
			bearer,
		},
	}
	getObjectAction = &action{
		Action: get,
		Flags: []cli.Flag{
			containerID,
			objectID,
			filePath,
			permissions,
			bearer,
		},
	}
	delObjectAction = &action{
		Action: del,
		Flags: []cli.Flag{
			containerID,
			objectID,
			bearer,
		},
	}
	headObjectAction = &action{
		Action: head,
		Flags: []cli.Flag{
			containerID,
			objectID,
			fullHeaders,
			bearer,
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
			bearer,
		},
	}
	getRangeObjectAction = &action{
		Action: getRange,
		Flags: []cli.Flag{
			containerID,
			objectID,
			bearer,
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
			bearer,
		},
	}
)

func addBearerToken(c *cli.Context, req *service.RequestVerificationHeader) error {
	sBearer := c.String(bearerFlag)
	if sBearer == "" {
		return nil
	}

	key := getKey(c)

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	bearerRules, err := hex.DecodeString(sBearer)
	if err != nil {
		return errors.Wrap(err, "could not decode bearer ACL rules")
	}

	bearer := new(service.BearerTokenMsg)
	bearer.SetExpirationEpoch(math.MaxUint64)
	bearer.SetACLRules(bearerRules)
	bearer.SetOwnerID(owner)

	req.SetBearer(bearer)

	return service.AddSignatureWithKey(key, service.NewSignedBearerToken(bearer))
}

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
	} else if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	if err = objID.Parse(objArg); err != nil {
		return errors.Wrapf(err, "can't parse object id '%s'", objArg)
	}

	addr := refs.Address{
		ObjectID: objID,
		CID:      cid,
	}

	token, err := createToken(tokenParams{
		connectionParams: connectionParams{
			ctx:  ctx,
			cmd:  c,
			conn: conn,
		},

		addr: addr,

		verb: service.Token_Info_Delete,
	})
	if err != nil {
		return errors.Wrap(err, "could not create session token")
	}

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	req := &object.DeleteRequest{
		Address: addr,
		OwnerID: owner,
	}

	if err := addBearerToken(c, &req.RequestVerificationHeader); err != nil {
		return errors.Wrap(err, "could not attach Bearer token")
	}

	req.SetHeaders(parseRequestHeaders(c.StringSlice(extHdrFlag)))
	req.SetToken(token)
	setTTL(c, req)
	setRaw(c, req)
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
	} else if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	if err = objID.Parse(objArg); err != nil {
		return errors.Wrapf(err, "can't parse object id '%s'", objArg)
	}

	addr := refs.Address{
		ObjectID: objID,
		CID:      cid,
	}

	token, err := createToken(tokenParams{
		connectionParams: connectionParams{
			ctx:  ctx,
			cmd:  c,
			conn: conn,
		},

		addr: addr,

		verb: service.Token_Info_Head,
	})
	if err != nil {
		return errors.Wrap(err, "could not create session token")
	}

	req := &object.HeadRequest{
		Address:     addr,
		FullHeaders: fh,
	}
	req.SetToken(token)

	if err := addBearerToken(c, &req.RequestVerificationHeader); err != nil {
		return errors.Wrap(err, "could not attach Bearer token")
	}

	req.SetHeaders(parseRequestHeaders(c.StringSlice(extHdrFlag)))
	setTTL(c, req)
	setRaw(c, req)
	signRequest(c, req)

	resp, err := object.NewServiceClient(conn).Head(ctx, req)
	if err != nil {
		return errors.Wrap(err, "can't perform HEAD request")
	}

	return objectStringify(os.Stdout, resp.Object)
}

// objectStringify converts object into string format.
func objectStringify(dst io.Writer, obj *object.Object) error {
	// put object line
	if _, err := fmt.Fprintln(dst, "Object:"); err != nil {
		return err
	}

	// put system headers
	if _, err := fmt.Fprintln(dst, "\tSystemHeader:"); err != nil {
		return err
	}

	sysHeaders := []string{"ID", "CID", "OwnerID", "Version", "PayloadLength", "CreatedAt"}
	v := reflect.ValueOf(obj.SystemHeader)
	for _, key := range sysHeaders {
		var val interface{}

		switch key {
		case "CreatedAt":
			val = fmt.Sprintf(`{UnixTime=%d Epoch=%d}`,
				obj.SystemHeader.CreatedAt.UnixTime,
				obj.SystemHeader.CreatedAt.Epoch)
		default:
			if !v.FieldByName(key).IsValid() {
				return errors.Errorf("invalid system header key: %q", key)
			}

			val = v.FieldByName(key).Interface()

		}

		if _, err := fmt.Fprintf(dst, "\t\t- %s=%v\n", key, val); err != nil {
			return err
		}
	}

	// put user headers
	if _, err := fmt.Fprintln(dst, "\tExtendedHeaders:"); err != nil {
		return err
	}

	for _, header := range obj.Headers {
		var (
			typ = reflect.ValueOf(header.Value)
			key string
			val interface{}
		)

		switch t := typ.Interface().(type) {
		case *object.Header_Link:
			key = "Link"
			val = fmt.Sprintf(`{Type=%s ID=%s}`, t.Link.Type, t.Link.ID)
		case *object.Header_Redirect:
			key = "Redirect"
			val = fmt.Sprintf(`{CID=%s OID=%s}`, t.Redirect.CID, t.Redirect.ObjectID)
		case *object.Header_UserHeader:
			key = "UserHeader"
			val = fmt.Sprintf(`{Key=%s Val=%s}`, t.UserHeader.Key, t.UserHeader.Value)
		case *object.Header_Transform:
			key = "Transform"
			val = t.Transform.Type.String()
		case *object.Header_Tombstone:
			key = "Tombstone"
			val = "MARKED"
		case *object.Header_HomoHash:
			key = "HomoHash"
			val = hex.EncodeToString(t.HomoHash[:])
		case *object.Header_PayloadChecksum:
			key = "PayloadChecksum"
			val = hex.EncodeToString(t.PayloadChecksum)
		case *object.Header_Integrity:
			key = "Integrity"
			val = fmt.Sprintf(`{Checksum=%02x Signature=%02x}`,
				t.Integrity.HeadersChecksum,
				t.Integrity.ChecksumSignature)
		case *object.Header_StorageGroup:
			key = "StorageGroup"
			buf := new(strings.Builder)
			if sg := t.StorageGroup; sg != nil {
				buf.WriteByte('{')
				buf.WriteString("DataSize=" + strconv.FormatUint(sg.ValidationDataSize, 10))
				buf.WriteString(" Hash=" + hex.EncodeToString(sg.ValidationHash[:]))
				if lt := sg.Lifetime; lt != nil {
					buf.WriteString(" Lifetime={")
					buf.WriteString("Unit=" + lt.Unit.String())
					buf.WriteString(" Value=" + strconv.FormatInt(lt.Value, 10))
					buf.WriteByte('}')
				}
				buf.WriteByte('}')
				val = buf.String()
			}
		case *object.Header_Token:
			key = "Token"
			val = fmt.Sprintf("{ID=%s Verb=%s}",
				t.Token.GetID(),
				t.Token.GetVerb())
		case *object.Header_PublicKey:
			key = "PublicKey"
			val = hex.EncodeToString(t.PublicKey.Value)
		default:
			key = fmt.Sprintf("Unknown(%T)", t)
			val = t
		}

		if _, err := fmt.Fprintf(dst, "\t\t- Type=%s\n\t\t  Value=%v\n", key, val); err != nil {
			return err
		}
	}

	// put payload
	if len(obj.Payload) > 0 {
		if _, err := fmt.Fprintf(dst, "\tPayload: %#v\n", obj.Payload); err != nil {
			return err
		}
	}

	return nil
}

func search(c *cli.Context) error {
	var (
		err    error
		conn   *grpc.ClientConn
		cid    refs.CID
		q      query.Query
		result []refs.Address

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
	} else if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
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

	token, err := createToken(tokenParams{
		connectionParams: connectionParams{
			ctx:  ctx,
			cmd:  c,
			conn: conn,
		},

		addr: refs.Address{
			CID: cid,
		},

		verb: service.Token_Info_Search,
	})
	if err != nil {
		return errors.Wrap(err, "could not create session token")
	}

	req := &object.SearchRequest{
		ContainerID:  cid,
		Query:        data,
		QueryVersion: 1,
	}
	req.SetToken(token)

	if err := addBearerToken(c, &req.RequestVerificationHeader); err != nil {
		return errors.Wrap(err, "could not attach Bearer token")
	}

	req.SetHeaders(parseRequestHeaders(c.StringSlice(extHdrFlag)))
	setTTL(c, req)
	setRaw(c, req)
	signRequest(c, req)

	searchClient, err := object.NewServiceClient(conn).Search(ctx, req)
	if err != nil {
		return errors.Wrap(err, "search command failed on client creation")
	}

	for {
		resp, err := searchClient.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "search command received error")
		}
		result = append(result, resp.Addresses...)
	}

	fmt.Println("Container ID: Object ID")
	for i := range result {
		fmt.Println(result[i].CID.String() + ": " + result[i].ObjectID.String())
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
	} else if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	if err = objID.Parse(objArg); err != nil {
		return errors.Wrapf(err, "can't parse object id '%s'", objArg)
	}

	ranges, err = parseRanges(rngArg)
	if err != nil {
		return errors.Wrap(err, "can't parse ranges")
	}

	if len(ranges) != 1 {
		return errors.New("specify one range")
	}

	addr := refs.Address{
		ObjectID: objID,
		CID:      cid,
	}

	token, err := createToken(tokenParams{
		connectionParams: connectionParams{
			ctx:  ctx,
			cmd:  c,
			conn: conn,
		},

		addr: addr,

		verb: service.Token_Info_Range,
	})
	if err != nil {
		return errors.Wrap(err, "could not create session token")
	}

	req := &object.GetRangeRequest{
		Address: addr,
		Range:   ranges[0],
	}
	req.SetToken(token)

	if err := addBearerToken(c, &req.RequestVerificationHeader); err != nil {
		return errors.Wrap(err, "could not attach Bearer token")
	}

	req.SetHeaders(parseRequestHeaders(c.StringSlice(extHdrFlag)))
	setTTL(c, req)
	setRaw(c, req)
	signRequest(c, req)

	rangeClient, err := object.NewServiceClient(conn).GetRange(ctx, req)
	if err != nil {
		return errors.Wrap(err, "can't perform get-range request")
	}

	var result []byte
	for {
		resp, err := rangeClient.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "get-range command received error")
		}
		result = append(result, resp.Fragment...)
	}
	fmt.Println(hex.EncodeToString(result))

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
	} else if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
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

	addr := refs.Address{
		ObjectID: objID,
		CID:      cid,
	}

	token, err := createToken(tokenParams{
		connectionParams: connectionParams{
			ctx:  ctx,
			cmd:  c,
			conn: conn,
		},

		addr: addr,

		verb: service.Token_Info_RangeHash,
	})
	if err != nil {
		return errors.Wrap(err, "could not create session token")
	}

	req := &object.GetRangeHashRequest{
		Address: addr,
		Ranges:  ranges,
		Salt:    salt,
	}
	req.SetToken(token)

	if err := addBearerToken(c, &req.RequestVerificationHeader); err != nil {
		return errors.Wrap(err, "could not attach Bearer token")
	}

	req.SetHeaders(parseRequestHeaders(c.StringSlice(extHdrFlag)))
	setTTL(c, req)
	setRaw(c, req)
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
		cpNum  = c.Uint64(copiesNumFlag)
	)

	if sCID == "" || len(fPaths) == 0 {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	}

	if cid, err = refs.CIDFromString(sCID); err != nil {
		return errors.Wrapf(err, "can't parse CID %s", sCID)
	} else if conn, err = connect(ctx, c); err != nil {
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

		req := &object.PutRequest{
			R: &object.PutRequest_Header{
				Header: &object.PutRequest_PutHeader{
					Object:       obj,
					CopiesNumber: uint32(cpNum),
				},
			},
		}
		req.SetToken(token)

		if err := addBearerToken(c, &req.RequestVerificationHeader); err != nil {
			return errors.Wrap(err, "could not attach Bearer token")
		}

		req.SetHeaders(parseRequestHeaders(c.StringSlice(extHdrFlag)))
		setTTL(c, req)
		setRaw(c, req)
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
				setRaw(c, req)
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

		addr := resp.GetAddress()

		fmt.Printf("[%s] Object successfully stored\n", fPath)
		fmt.Printf("  ID: %s\n  CID: %s\n", addr.ObjectID, addr.CID)
		if verify {
			result := "success"

			token, err := createToken(tokenParams{
				connectionParams: connectionParams{
					ctx:  ctx,
					cmd:  c,
					conn: conn,
				},

				addr: addr,

				verb: service.Token_Info_RangeHash,
			})
			if err != nil {
				return errors.Wrap(err, "could not create session token")
			}

			req := &object.GetRangeHashRequest{
				Address: addr,
				Ranges:  []object.Range{{Offset: 0, Length: obj.SystemHeader.PayloadLength}},
			}
			req.SetToken(token)
			setTTL(c, req)
			setRaw(c, req)
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

func parseRequestHeaders(reqH []string) []service.RequestExtendedHeader_KV {
	headers := make([]service.RequestExtendedHeader_KV, 0, len(reqH))

	for i := range reqH {
		kv := strings.SplitN(reqH[i], "=", 2)

		h := service.RequestExtendedHeader_KV{}

		h.SetK(kv[0])

		if len(kv) > 1 {
			h.SetV(kv[1])
		}

		headers = append(headers, h)
	}

	return headers
}

func establishSession(p sessionParams) error {
	creator, err := session.NewGRPCCreator(
		p.conn,
		getKey(p.cmd),
	)
	if err != nil {
		return err
	}

	res, err := creator.Create(p.ctx, p.prm)
	if err != nil {
		return err
	}

	p.res.SetID(res.GetID())
	p.res.SetSessionKey(res.GetSessionKey())

	return nil
}

func createToken(p tokenParams) (*service.Token, error) {
	key := getKey(p.cmd)

	ownerID, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return nil, err
	}

	token := new(service.Token)
	token.SetOwnerID(ownerID)
	token.SetCreationEpoch(0)
	token.SetExpirationEpoch(math.MaxUint64)
	token.SetVerb(p.verb)

	// open a new session
	if err := establishSession(sessionParams{
		connectionParams: p.connectionParams,

		prm: token,
		res: token,
	}); err != nil {
		return nil, err
	}

	// sign token message
	if err := service.AddSignatureWithKey(
		key,
		service.NewSignedSessionToken(token),
	); err != nil {
		return nil, err
	}

	return token, nil
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
	} else if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	if err = oid.Parse(sOID); err != nil {
		return errors.Wrapf(err, "can't parse Object ID %s", sOID)
	}

	addr := refs.Address{
		ObjectID: oid,
		CID:      cid,
	}

	token, err := createToken(tokenParams{
		connectionParams: connectionParams{
			ctx:  ctx,
			cmd:  c,
			conn: conn,
		},

		addr: addr,

		verb: service.Token_Info_Get,
	})
	if err != nil {
		return errors.Wrap(err, "could not create session token")
	}

	req := &object.GetRequest{
		Address: addr,
	}
	req.SetToken(token)

	if err := addBearerToken(c, &req.RequestVerificationHeader); err != nil {
		return errors.Wrap(err, "could not attach Bearer token")
	}

	req.SetHeaders(parseRequestHeaders(c.StringSlice(extHdrFlag)))
	setTTL(c, req)
	setRaw(c, req)
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
				return errors.New("Object removed")
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
