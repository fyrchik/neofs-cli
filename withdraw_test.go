package main

import (
	"bytes"
	"testing"

	"github.com/nspcc-dev/neofs-api-go/accounting"
	"github.com/nspcc-dev/neofs-api-go/decimal"
	"github.com/nspcc-dev/neofs-api-go/refs"
	"github.com/nspcc-dev/neofs-crypto/test"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func mockedCheque(t *testing.T) []byte {
	var (
		err      error
		result   []byte
		ownerKey = test.DecodeKey(0)
	)

	ownerID, err := refs.NewOwnerID(&ownerKey.PublicKey)
	require.NoError(t, err)

	cheque := &accounting.Cheque{
		ID:     "test-cheque-id",
		Owner:  ownerID,
		Height: 100,
		Amount: decimal.New(1.007e10),
	}

	// Signatures:
	require.NoError(t, cheque.Sign(ownerKey))
	require.NoError(t, cheque.Sign(test.DecodeKey(1)))
	require.NoError(t, cheque.Sign(test.DecodeKey(2)))
	require.NoError(t, cheque.Sign(test.DecodeKey(3)))

	// Marshaling:
	result, err = cheque.MarshalBinary()
	require.NoError(t, err)

	return result
}

func Test_displayWithdrawal(t *testing.T) {
	mockedResult := `Withdrawal info:
              
Withdraw ID   1111111111111111111111111
Owner ID      NQHKh7fKGieCPrPuiEkY58ucRFwWMyU1Mc
Amount        100.7
Height        100
              
Signatures:

hash   f8dceeafdce381ff387b97c8a71acf6da7760bd01f24fc44991ce4974bc503d198b79fa03ee7c5b1f89e8b6d46ef6f8e3064de999321ea1c9fb7962c08e2702e
key    22ZpK5iv7SryYD1aWEHL6oz61egP7nKGmR99veZL7QnGe

hash   7dd7913d1fdc278690ef26cfa5115956f823213467db0d5f212d3a29108305c55b70a271a9654ba994273f656fdde090cb2838b2306aa52b8ea96306c1be848f
key    gwumsMmZgigrcxeWkuCexbJNqhyGC1BbANyMHDeCvGPT

hash   9ec8c174ab9ef3b84d2547171404b50b56b9951157288eba6e55b0007cdca11157c70b1482c9b7e50507a7faa26a6f632df1367499bdc69621271333404e58e3
key    phkHKba8mn2jqn7XaKzdJCb1MfPo5wan4HJJZ1tCWXk2

hash   4fcd782dde6968d95fc90dc33e862270141572b40fe90f9eb0f1946e5a64b131a9d25f46e2293b45438c20cb749ebb7605ecb819815c32df67651259f16dad92
key    hGJTNg9aBefWJSjeNEVrzAQCuPLjRhxvyp8zRrPo3gfh

Cheque data:
00000000000000000000000000000000000000000000000000352fea5b25f49d5ab7c9982acb5374ec10c8e41db2db00b9238001385802000000640000000000000004000375099c302b77664a2508bec1cae47903857b762c62713f190e8d99912ef76737f8dceeafdce381ff387b97c8a71acf6da7760bd01f24fc44991ce4974bc503d198b79fa03ee7c5b1f89e8b6d46ef6f8e3064de999321ea1c9fb7962c08e2702e025188d33a3113ac77fea0c17137e434d704283c234400b9b70bcdf4829094374a7dd7913d1fdc278690ef26cfa5115956f823213467db0d5f212d3a29108305c55b70a271a9654ba994273f656fdde090cb2838b2306aa52b8ea96306c1be848f02c4c574d1bbe7efb2feaeed99e6c03924d6d3c9ad76530437d75c07bff3ddcc0f9ec8c174ab9ef3b84d2547171404b50b56b9951157288eba6e55b0007cdca11157c70b1482c9b7e50507a7faa26a6f632df1367499bdc69621271333404e58e302563eece0b9035e679d28e2d548072773c43ce44a53cb7f30d3597052210dbb704fcd782dde6968d95fc90dc33e862270141572b40fe90f9eb0f1946e5a64b131a9d25f46e2293b45438c20cb749ebb7605ecb819815c32df67651259f16dad92
`

	tests := []struct {
		name   string
		data   []byte
		result string
		error  error
	}{
		{
			name:  "empty data",
			error: accounting.ErrWrongChequeData,
		},
		{
			name:   "mocked cheque",
			data:   mockedCheque(t),
			result: mockedResult,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wr := new(bytes.Buffer)

			err := displayWithdrawal(wr, tt.data)
			if err != nil {
				require.Errorf(t, tt.error, err.Error(), tt.name)
				require.EqualErrorf(t, errors.Cause(err), tt.error.Error(), tt.name)
				return
			}

			require.Equalf(t, tt.result, wr.String(), tt.name)
		})
	}
}
