package main

import (
	"bytes"
	"testing"

	"github.com/nspcc-dev/neofs-api-go/object"
	"github.com/nspcc-dev/neofs-api-go/refs"
	"github.com/nspcc-dev/neofs-api-go/service"
	"github.com/nspcc-dev/neofs-api-go/storagegroup"
	"github.com/nspcc-dev/neofs-crypto/test"
	"github.com/stretchr/testify/require"
)

func TestStringify(t *testing.T) {
	res := `Object:
	SystemHeader:
		- ID=7e0b9c6c-aabc-4985-949e-2680e577b48b
		- CID=11111111111111111111111111111111
		- OwnerID=NQHKh7fKGieCPrPuiEkY58ucRFwWMyU1Mc
		- Version=1
		- PayloadLength=1
		- CreatedAt={UnixTime=1 Epoch=1}
	ExtendedHeaders:
		- Type=Link
		  Value={Type=Child ID=7e0b9c6c-aabc-4985-949e-2680e577b48b}
		- Type=Redirect
		  Value={CID=11111111111111111111111111111111 OID=7e0b9c6c-aabc-4985-949e-2680e577b48b}
		- Type=UserHeader
		  Value={Key=test_key Val=test_value}
		- Type=Transform
		  Value=Split
		- Type=Tombstone
		  Value=MARKED
		- Type=Token
		  Value={ID=01020304-0506-0000-0000-000000000000 Verb=Put}
		- Type=HomoHash
		  Value=00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
		- Type=PayloadChecksum
		  Value=010203040506
		- Type=Integrity
		  Value={Checksum=010203040506 Signature=010203040506}
		- Type=StorageGroup
		  Value={DataSize=5 Hash=00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000 Lifetime={Unit=UnixTime Value=555}}
		- Type=PublicKey
		  Value=010203040506
	Payload: []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7}
`

	key := test.DecodeKey(0)

	uid, err := refs.NewOwnerID(&key.PublicKey)
	require.NoError(t, err)

	var oid refs.UUID

	require.NoError(t, oid.Parse("7e0b9c6c-aabc-4985-949e-2680e577b48b"))

	obj := &object.Object{
		SystemHeader: object.SystemHeader{
			Version:       1,
			PayloadLength: 1,
			ID:            oid,
			OwnerID:       uid,
			CID:           refs.CID{},
			CreatedAt: object.CreationPoint{
				UnixTime: 1,
				Epoch:    1,
			},
		},
		Payload: []byte{1, 2, 3, 4, 5, 6, 7},
	}

	// *Header_Link
	obj.Headers = append(obj.Headers, object.Header{
		Value: &object.Header_Link{
			Link: &object.Link{ID: oid, Type: object.Link_Child},
		},
	})

	// *Header_Redirect
	obj.Headers = append(obj.Headers, object.Header{
		Value: &object.Header_Redirect{
			Redirect: &object.Address{ObjectID: oid, CID: refs.CID{}},
		},
	})

	// *Header_UserHeader
	obj.Headers = append(obj.Headers, object.Header{
		Value: &object.Header_UserHeader{
			UserHeader: &object.UserHeader{
				Key:   "test_key",
				Value: "test_value",
			},
		},
	})

	// *Header_Transform
	obj.Headers = append(obj.Headers, object.Header{
		Value: &object.Header_Transform{
			Transform: &object.Transform{
				Type: object.Transform_Split,
			},
		},
	})

	// *Header_Tombstone
	obj.Headers = append(obj.Headers, object.Header{
		Value: &object.Header_Tombstone{
			Tombstone: &object.Tombstone{},
		},
	})

	// *Header_Token
	token := new(service.Token)
	token.SetID(service.TokenID{1, 2, 3, 4, 5, 6})
	token.SetVerb(service.Token_Info_Put)

	obj.Headers = append(obj.Headers, object.Header{
		Value: &object.Header_Token{
			Token: token,
		},
	})

	// *Header_HomoHash
	obj.Headers = append(obj.Headers, object.Header{
		Value: &object.Header_HomoHash{
			HomoHash: object.Hash{},
		},
	})

	// *Header_PayloadChecksum
	obj.Headers = append(obj.Headers, object.Header{
		Value: &object.Header_PayloadChecksum{
			PayloadChecksum: []byte{1, 2, 3, 4, 5, 6},
		},
	})

	// *Header_Integrity
	obj.Headers = append(obj.Headers, object.Header{
		Value: &object.Header_Integrity{
			Integrity: &object.IntegrityHeader{
				HeadersChecksum:   []byte{1, 2, 3, 4, 5, 6},
				ChecksumSignature: []byte{1, 2, 3, 4, 5, 6},
			},
		},
	})

	// *Header_StorageGroup
	obj.Headers = append(obj.Headers, object.Header{
		Value: &object.Header_StorageGroup{
			StorageGroup: &storagegroup.StorageGroup{
				ValidationDataSize: 5,
				ValidationHash:     storagegroup.Hash{},
				Lifetime: &storagegroup.StorageGroup_Lifetime{
					Unit:  storagegroup.StorageGroup_Lifetime_UnixTime,
					Value: 555,
				},
			},
		},
	})

	// *Header_PublicKey
	obj.Headers = append(obj.Headers, object.Header{
		Value: &object.Header_PublicKey{
			PublicKey: &object.PublicKey{Value: []byte{1, 2, 3, 4, 5, 6}},
		},
	})

	buf := new(bytes.Buffer)

	require.NoError(t, objectStringify(buf, obj))
	require.Equal(t, res, buf.String())
}
