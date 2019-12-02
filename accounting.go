package main

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/nspcc-dev/neofs-proto/accounting"
	"github.com/nspcc-dev/neofs-proto/decimal"
	"github.com/nspcc-dev/neofs-proto/refs"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

var (
	accountingAction = &action{}
	getBalanceAction = &action{
		Action: getBalance,
	}
)

func getBalance(c *cli.Context) error {
	var (
		err  error
		key  = getKey(c)
		host = getHost(c)
		conn *grpc.ClientConn
		ctx  = gracefulContext()
	)

	if conn, err = connect(ctx, c); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return err
	}

	req := &accounting.BalanceRequest{OwnerID: owner}
	setTTL(c, req)
	signRequest(c, req)

	resp, err := accounting.NewAccountingClient(conn).Balance(ctx, req)
	if err != nil {
		return errors.Wrap(err, "could not request balance")
	}

	return displayBalance(os.Stdout, resp)
}

func displayBalance(wr io.Writer, resp *accounting.BalanceResponse) error {
	tw := tabwriter.NewWriter(wr, 1, 8, 3, ' ', 0)

	if _, err := fmt.Fprintln(tw, "Balance info:"); err != nil {
		return err
	}

	if resp == nil {
		_, err := fmt.Fprintln(tw, "- empty")
		return err
	}

	var balance = decimal.Zero.Copy()

	if resp.GetBalance() != nil {
		balance = resp.GetBalance()
	}

	if _, err := fmt.Fprintf(tw, "- Active balance: %s\n", balance); err != nil {
		return err
	}

	if len(resp.LockAccounts) > 0 {
		funds := accounting.SumFunds(resp.LockAccounts)
		if _, err := fmt.Fprintf(tw, "- Locked funds:   %s\n", funds); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(tw, "Amount\tTarget\tLifetime\tUnit"); err != nil {
			return err
		}
		for _, lf := range resp.LockAccounts {
			_, err := fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n",
				lf.ActiveFunds, lf.LockTarget, lf.Lifetime.Value, lf.Lifetime.Unit)
			if err != nil {
				return err
			}
		}
	}

	return tw.Flush()
}
