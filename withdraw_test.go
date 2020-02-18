package main

import (
	"bytes"
	"testing"

	"github.com/nspcc-dev/neofs-api/accounting"
	"github.com/nspcc-dev/neofs-api/decimal"
	"github.com/nspcc-dev/neofs-api/refs"
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
Owner ID      ALYeYC41emF6MrmUMc4a8obEPdgFhq9ran
Amount        100.7
Height        100
              
Signatures:

hash   8f6edd0831737ff1c84c066b91042dd8f1ae73588765bc56d16aace35eae2f9fea6d34e1ce276e40b93e8e2cd1f62b3beabd83aaaa046bb22a3329a5ef95e441
key    22ZpK5iv7SryYD1aWEHL6oz61egP7nKGmR99veZL7QnGe

hash   695c430cf5a4ac7f432be2f5128890bde8ca7a0676d9789e9504286299b7bceb20a90164c3fda946d2fe5718de7ab031007bc306c7c31b14b8b44d7aa6611f45
key    gwumsMmZgigrcxeWkuCexbJNqhyGC1BbANyMHDeCvGPT

hash   1926d91ca6a0e045b28c05cfb6c8ba6f19e4c089c66320e3c26f5a72545773a7bfc39f4d705eb1946cc18890cceb993884edea6970207247c8f18e518900a719
key    phkHKba8mn2jqn7XaKzdJCb1MfPo5wan4HJJZ1tCWXk2

hash   ce12e5c3164fbdaff6de38c83f639f6633138d09f3ee325d3a86d6e11374855e6a18ec6138267f67040b7cff0cfdccb9e6a817ff12f58a55b5ea020dd50e6822
key    hGJTNg9aBefWJSjeNEVrzAQCuPLjRhxvyp8zRrPo3gfh

Cheque data:
00000000000000000000000000000000000000000000000000173458475ec75d98b0be634d334a77cc44f529eab6330805cb8001385802000000640000000000000004000375099c302b77664a2508bec1cae47903857b762c62713f190e8d99912ef767378f6edd0831737ff1c84c066b91042dd8f1ae73588765bc56d16aace35eae2f9fea6d34e1ce276e40b93e8e2cd1f62b3beabd83aaaa046bb22a3329a5ef95e441025188d33a3113ac77fea0c17137e434d704283c234400b9b70bcdf4829094374a695c430cf5a4ac7f432be2f5128890bde8ca7a0676d9789e9504286299b7bceb20a90164c3fda946d2fe5718de7ab031007bc306c7c31b14b8b44d7aa6611f4502c4c574d1bbe7efb2feaeed99e6c03924d6d3c9ad76530437d75c07bff3ddcc0f1926d91ca6a0e045b28c05cfb6c8ba6f19e4c089c66320e3c26f5a72545773a7bfc39f4d705eb1946cc18890cceb993884edea6970207247c8f18e518900a71902563eece0b9035e679d28e2d548072773c43ce44a53cb7f30d3597052210dbb70ce12e5c3164fbdaff6de38c83f639f6633138d09f3ee325d3a86d6e11374855e6a18ec6138267f67040b7cff0cfdccb9e6a817ff12f58a55b5ea020dd50e6822
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
