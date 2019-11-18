package main

import (
	"bytes"
	"testing"

	"github.com/nspcc-dev/neofs-proto/accounting"
	"github.com/nspcc-dev/neofs-proto/decimal"
	"github.com/stretchr/testify/require"
)

func mockedBalance(target *accounting.LockTarget, amount int64, locks ...int64) *accounting.BalanceResponse {
	var items = make([]*accounting.Account, 0, len(locks))

	for i := range locks {
		items = append(items, &accounting.Account{
			LockTarget:  target,
			ActiveFunds: decimal.New(locks[i]),
			Lifetime: accounting.Lifetime{
				Unit:  accounting.Lifetime_NeoBlock,
				Value: int64(i),
			},
		})
	}
	return &accounting.BalanceResponse{
		Balance:      decimal.New(amount),
		LockAccounts: items,
	}
}

func Test_displayBalance(t *testing.T) {
	target := &accounting.LockTarget{
		Target: &accounting.LockTarget_WithdrawTarget{
			WithdrawTarget: &accounting.WithdrawTarget{
				Cheque: "cheque",
			},
		},
	}

	tests := []struct {
		error    error
		name     string
		result   string
		response *accounting.BalanceResponse
	}{
		{
			name:   "empty response",
			result: "Balance info:\n- empty\n",
		},

		{
			name:     "empty response",
			result:   "Balance info:\n- Active balance: 0.000001\n",
			response: mockedBalance(target, 100),
		},

		{
			name: "empty response",
			result: `Balance info:
- Active balance: 0.000001
- Locked funds:   0.00001368
Amount       Target                               Lifetime   Unit
0.00000123   WithdrawTarget:<Cheque:"cheque" >    0          NeoBlock
0.00000456   WithdrawTarget:<Cheque:"cheque" >    1          NeoBlock
0.00000789   WithdrawTarget:<Cheque:"cheque" >    2          NeoBlock
`,
			response: mockedBalance(target, 100, 123, 456, 789),
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				wr := new(bytes.Buffer)
				err := displayBalance(wr, tt.response)
				if err != nil {
					require.Error(t, tt.error, err.Error())
					require.EqualError(t, err, tt.error.Error())
					return
				}

				require.Equal(t, tt.result, wr.String())
			}, "step %d (%s)", i, tt.name)
		})
	}
}
