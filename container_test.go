package main

import (
	"testing"

	query "github.com/nspcc-dev/netmap-ql"
	"github.com/stretchr/testify/require"
)

func Test_PlacementStringify(t *testing.T) {
	cases := []struct {
		Actual string
		Expect string
		Error  error
	}{
		{Actual: "SELECT 2 Node", Expect: "RF 2 SELECT 2 Node"},
		{Actual: "RF 1 SELECT 2 Country", Expect: "RF 1 SELECT 2 Country"},
		{Actual: "RF 0 SELECT 2 Country", Expect: "RF 0 SELECT 2 Country"},
		{Actual: "SELECT 2 Node FILTER Country NE Russia", Expect: "RF 2 SELECT 2 Node FILTER Country NE Russia"},
	}

	for i := range cases {
		tt := cases[i]
		t.Run(tt.Actual, func(t *testing.T) {
			res, err := query.ParseQuery(tt.Actual)
			if tt.Error != nil {
				require.EqualError(t, err, tt.Error.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.Expect, placementStringify(res))
		})
	}

}
