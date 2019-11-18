package main

import (
	"bytes"
	"testing"

	"github.com/nspcc-dev/neofs-crypto/test"
	"github.com/nspcc-dev/neofs-proto/accounting"
	"github.com/nspcc-dev/neofs-proto/decimal"
	"github.com/nspcc-dev/neofs-proto/refs"
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
Owner ID      ALYeYC41emF6MrmUMc4a8obEPdgFhq9ran
Amount        100.7
Height        100
              
Signatures:

hash   d404697b2b0c0bb703c384d63deb25e4619cf4f76e6bd613b915e2cf2ea5faf3459e7092aa6c5b361f3b331e9b8459c259f91758054eac7599a52eaf8906eabc
key    22ZpK5iv7SryYD1aWEHL6oz61egP7nKGmR99veZL7QnGe

hash   9da164ac8cbc45916a8faad09c7a8d97b7a70f55ab32cc5192a2cf7159f5d195051b6d0b9cfa8f8469e763ab8751165d5219de591a4637ce5e79434900a4eeee
key    gwumsMmZgigrcxeWkuCexbJNqhyGC1BbANyMHDeCvGPT

hash   55b4f03513daa4ad69c7a5c82b81ea227d39cbb6dceab42ee23566a78b4d9907798fe38bc150d31fdd1beb8299d37bf85dea74dfff9059787f53526ab30c1353
key    phkHKba8mn2jqn7XaKzdJCb1MfPo5wan4HJJZ1tCWXk2

hash   0a26009ec6590d44d632f2ace893e25fdc860da05b56bc2624f905a61f9f1351b23a1110641812c62af1ac0afefcb4e561173c41a4247bc020473887e168d10d
key    hGJTNg9aBefWJSjeNEVrzAQCuPLjRhxvyp8zRrPo3gfh

Cheque data:
00000000000000000000000000000000000000000000000000173458475ec75d98b0be634d334a77cc44f529eab6330805cb0000000258380180000000000000006400040375099c302b77664a2508bec1cae47903857b762c62713f190e8d99912ef76737d404697b2b0c0bb703c384d63deb25e4619cf4f76e6bd613b915e2cf2ea5faf3459e7092aa6c5b361f3b331e9b8459c259f91758054eac7599a52eaf8906eabc025188d33a3113ac77fea0c17137e434d704283c234400b9b70bcdf4829094374a9da164ac8cbc45916a8faad09c7a8d97b7a70f55ab32cc5192a2cf7159f5d195051b6d0b9cfa8f8469e763ab8751165d5219de591a4637ce5e79434900a4eeee02c4c574d1bbe7efb2feaeed99e6c03924d6d3c9ad76530437d75c07bff3ddcc0f55b4f03513daa4ad69c7a5c82b81ea227d39cbb6dceab42ee23566a78b4d9907798fe38bc150d31fdd1beb8299d37bf85dea74dfff9059787f53526ab30c135302563eece0b9035e679d28e2d548072773c43ce44a53cb7f30d3597052210dbb700a26009ec6590d44d632f2ace893e25fdc860da05b56bc2624f905a61f9f1351b23a1110641812c62af1ac0afefcb4e561173c41a4247bc020473887e168d10d
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
