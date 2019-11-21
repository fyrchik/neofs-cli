package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/mr-tron/base58"
	crypto "github.com/nspcc-dev/neofs-crypto"
	"github.com/nspcc-dev/neofs-proto/accounting"
	"github.com/nspcc-dev/neofs-proto/decimal"
	"github.com/nspcc-dev/neofs-proto/refs"
	"github.com/nspcc-dev/neofs-proto/service"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

var (
	withdrawAction = &action{
		Flags: []cli.Flag{
			hostAddr,
		},
	}
	putWithdrawAction = &action{
		Action: putWithdraw,
		Flags: []cli.Flag{
			keyFile,
			blockHeight,
			amount,
		},
	}
	getWithdrawAction = &action{
		Action: getWithdraw,
		Flags: []cli.Flag{
			keyFile,
			withdrawID,
		},
	}
	delWithdrawAction = &action{
		Action: delWithdraw,
		Flags: []cli.Flag{
			keyFile,
			withdrawID,
		},
	}

	listWithdrawAction = &action{
		Action: listWithdraw,
		Flags: []cli.Flag{
			keyFile,
		},
	}
)

func putWithdraw(c *cli.Context) error {
	var (
		err   error
		key   *ecdsa.PrivateKey
		conn  *grpc.ClientConn
		msgID refs.MessageID

		host        = c.Parent().String(hostFlag)
		keyArg      = c.String(keyFlag)
		amount      = c.Float64(amountFlag)
		blockHeight = c.Uint64(heightFlag)
	)

	if host == "" || keyArg == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	// Try to receive key from file
	if key, err = parseKeyValue(keyArg); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure()); err != nil {
		return errors.Wrapf(err, "could not connect to host %s", host)
	}

	if msgID, err = refs.NewMessageID(); err != nil {
		return errors.Wrap(err, "could not create message ID")
	}

	dec := decimal.ParseFloat(amount)
	fmt.Printf("Will be used precision: %d\n", decimal.GASPrecision)

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	req := &accounting.PutRequest{
		OwnerID:   owner,
		Amount:    dec,
		Height:    blockHeight,
		MessageID: msgID,
	}
	req.SetTTL(getTTL(c))

	if err = service.SignRequest(req, key); err != nil {
		return errors.Wrap(err, "could not sign request")
	}

	resp, err := accounting.NewWithdrawClient(conn).Put(ctx, req)
	if err != nil {
		return errors.Wrap(err, "put request failed")
	}

	fmt.Printf("Withdrawal created: %s\n", resp.ID)

	return nil
}

func getWithdraw(c *cli.Context) error {
	var (
		err  error
		key  *ecdsa.PrivateKey
		conn *grpc.ClientConn

		keyArg = c.String(keyFlag)
		host   = c.Parent().String(hostFlag)
		wid    = c.String(widFlag)
	)

	if host == "" || keyArg == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	// Try to receive key from file
	if key, err = parseKeyValue(keyArg); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure()); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	req := &accounting.GetRequest{
		ID:      accounting.ChequeID(wid),
		OwnerID: owner,
	}
	req.SetTTL(getTTL(c))
	resp, err := accounting.NewWithdrawClient(conn).Get(ctx, req)
	if err != nil {
		return errors.Wrap(err, "can't perform request")
	}

	return displayWithdrawal(os.Stdout, resp.Withdraw.Payload)
}

func displayWithdrawal(wr io.Writer, data []byte) error {
	var (
		ch = new(accounting.Cheque)
		tw = tabwriter.NewWriter(wr, 1, 8, 3, ' ', 0)
	)

	if err := ch.UnmarshalBinary(data); err != nil {
		return errors.Wrap(err, "could not unmarshal cheque")
	}

	if _, err := fmt.Fprintln(tw, "Withdrawal info:"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(tw, "\t"); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(tw, "Withdraw ID\t%s\n", ch.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "Owner ID\t%s\n", ch.Owner); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "Amount\t%s\n", ch.Amount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "Height\t%d\n", ch.Height); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(tw, "\t"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(tw, "Signatures:"); err != nil {
		return err
	}

	for i := range ch.Signatures {
		if _, err := fmt.Fprintf(
			tw, "\nhash\t%s\nkey\t%s\n",
			hex.EncodeToString(ch.Signatures[i].Hash),
			base58.Encode(crypto.MarshalPublicKey(ch.Signatures[i].Key)),
		); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(tw, "\nCheque data:\n%s\n", hex.EncodeToString(data)); err != nil {
		return err
	}

	return tw.Flush()
}

func delWithdraw(c *cli.Context) error {
	var (
		err   error
		key   *ecdsa.PrivateKey
		conn  *grpc.ClientConn
		msgID refs.MessageID

		keyArg = c.String(keyFlag)
		host   = c.Parent().String(hostFlag)
		wid    = c.String(widFlag)
	)

	if host == "" || keyArg == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	// Try to receive key from file
	if key, err = parseKeyValue(keyArg); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure()); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	if msgID, err = refs.NewMessageID(); err != nil {
		return errors.Wrap(err, "could not create message ID")
	}

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	req := &accounting.DeleteRequest{
		ID:        accounting.ChequeID(wid),
		OwnerID:   owner,
		MessageID: msgID,
	}
	req.SetTTL(getTTL(c))

	if err = service.SignRequest(req, key); err != nil {
		return errors.Wrap(err, "could not sign request")
	}

	_, err = accounting.NewWithdrawClient(conn).Delete(ctx, req)

	return errors.Wrap(err, "can't perform request")
}

func listWithdraw(c *cli.Context) error {
	var (
		err  error
		key  *ecdsa.PrivateKey
		conn *grpc.ClientConn

		keyArg = c.String(keyFlag)
		host   = c.Parent().String(hostFlag)
	)

	if host == "" || keyArg == "" {
		return errors.Errorf("invalid input\nUsage: %s", c.Command.UsageText)
	} else if host, err = parseHostValue(host); err != nil {
		return err
	}

	// Try to receive key from file
	if key, err = parseKeyValue(keyArg); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if conn, err = grpc.DialContext(ctx, host, grpc.WithInsecure()); err != nil {
		return errors.Wrapf(err, "can't connect to host '%s'", host)
	}

	owner, err := refs.NewOwnerID(&key.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not compute owner ID")
	}

	req := &accounting.ListRequest{OwnerID: owner}
	req.SetTTL(getTTL(c))
	resp, err := accounting.NewWithdrawClient(conn).List(ctx, req)
	if err != nil {
		return errors.Wrapf(err, "can't complete request")
	}

	if len(resp.Items) == 0 {
		fmt.Println("No active withdrawals")
	}

	for _, item := range resp.Items {
		fmt.Println(fmt.Sprintf("amount: %sGAS, height: %d, ID: %s, owner ID: %s",
			item.Amount,
			item.Height,
			item.ID,
			item.OwnerID))
	}

	return nil
}
