package spl

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/portto/solana-go-sdk/client"
	"github.com/portto/solana-go-sdk/client/rpc"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/program/assotokenprog"
	"github.com/portto/solana-go-sdk/program/tokenprog"
	"github.com/portto/solana-go-sdk/types"
)

type TokenAccountInfo struct {
	Mint    *tokenprog.MintAccount
	Account *tokenprog.TokenAccount
}

func (c *TokenAccountInfo) AmountFloat() float64 {
	return float64(c.Account.Amount) / math.Pow10(int(c.Mint.Decimals))
}

type AssociatedTokenProgram struct {
	TokenProgram
}

// Transfer a mint token between two wallets using their associated token accounts
func (c *AssociatedTokenProgram) Transfer(mint common.PublicKey, from types.Account, dest common.PublicKey, amount uint64) (txid string, err error) {
	// check if associated token account exists

	var ixs []types.Instruction
	sender, _, err := common.FindAssociatedTokenAddress(from.PublicKey, mint)
	if err != nil {
		return
	}

	receiver, _, err := common.FindAssociatedTokenAddress(dest, mint)
	if err != nil {
		return
	}

	_, err = c.TokenProgram.Account(receiver)
	if errors.Is(err, ErrorTokenAccountNotFound) {
		ixs = append(ixs, assotokenprog.CreateAssociatedTokenAccount(from.PublicKey, dest, mint))
	} else if err != nil {
		return
	}

	ixs = append(ixs, tokenprog.Transfer(sender, receiver, from.PublicKey, nil, amount))

	return c.sendTx(types.CreateRawTransactionParam{
		Instructions: ixs,
		Signers: []types.Account{
			from,
		},
		FeePayer: from.PublicKey,
	})
}

func (c *AssociatedTokenProgram) AccountInfo(mint, addr common.PublicKey) (*TokenAccountInfo, error) {
	tokenAddr, _, err := common.FindAssociatedTokenAddress(addr, mint)
	if err != nil {
		return nil, err
	}

	return c.TokenProgram.AccountInfo(tokenAddr)
}

func (c *AssociatedTokenProgram) Account(mint, addr common.PublicKey) (*tokenprog.TokenAccount, error) {
	tokenAddr, _, err := common.FindAssociatedTokenAddress(addr, mint)
	if err != nil {
		return nil, err
	}

	return c.TokenProgram.Account(tokenAddr)
}

func (c *AssociatedTokenProgram) Balance(mint, addr common.PublicKey) (uint64, error) {
	tokenAddr, _, err := common.FindAssociatedTokenAddress(addr, mint)
	if err != nil {
		return 0, err
	}

	// TODO: should return 0 if account is not found
	return c.TokenProgram.Balance(tokenAddr)
}

// TODO token transfer.

type TokenBalance struct {
	Decimals uint8
	Amount   uint64
}

//  might be useful to have read-only and signer APIs?
type TokenProgram struct {
	Ctx    context.Context
	Client *client.Client
	// pubkey  common.PublicKey
	// privkey ed25519.PrivateKey // could be nil
}

var ErrorTokenAccountNotFound = errors.New("token account not found")

func (c *TokenProgram) AccountInfo(addr common.PublicKey) (*TokenAccountInfo, error) {
	account, err := c.Account(addr)
	if err != nil {
		return nil, err
	}

	// TODO: cache mint data
	mintData, err := c.accountData(account.Mint)
	if err != nil {
		return nil, err
	}

	mint, err := tokenprog.MintAccountFromData(mintData)
	if err != nil {
		return nil, err
	}

	return &TokenAccountInfo{
		Account: account,
		Mint:    mint,
	}, nil
}

func (c *TokenProgram) Account(addr common.PublicKey) (*tokenprog.TokenAccount, error) {
	data, err := c.accountData(addr)
	if err != nil {
		return nil, err
	}

	if data == nil {
		// uninitialized token account
		return nil, fmt.Errorf("spl token account %s: %w", addr, ErrorTokenAccountNotFound)
	}

	account, err := tokenprog.TokenAccountFromData(data)
	if err != nil {
		return nil, fmt.Errorf("decode spl token account: %w", err)
	}

	return account, nil
}

func (c *TokenProgram) Balance(addr common.PublicKey) (uint64, error) {
	account, err := c.Account(addr)
	if err != nil {
		return 0, err
	}

	return account.Amount, nil
}

func (c *TokenProgram) ConfirmTx(ctx context.Context, txid string) (*rpc.GetConfirmedTransactionResponse, error) {
	// TODO: have a way to specify option
	// TODO: be able tto just pollig interval?

	for {
		// TODO: handle RPC error
		// log.Println("confirming:", txid)
		res, err := c.Client.GetConfirmedTransaction(ctx, txid)
		// res, err := c.Client.GetTransaction(ctx, txid, rpc.GetTransactionWithLimitConfig{})

		if err != nil {
			return nil, err
		}

		if res.Slot == 0 {
			time.Sleep(time.Millisecond * 5000)
			continue
		}

		return &res, nil
	}

}

func (c *TokenProgram) sendTx(tx types.CreateRawTransactionParam) (string, error) {
	res, err := c.Client.GetRecentBlockhash(context.Background())
	if err != nil {
		return "", fmt.Errorf("send tx get recent block hash error: %w", err)
	}

	tx.RecentBlockHash = res.Blockhash

	txdata, err := types.CreateRawTransaction(tx)
	if err != nil {
		return "", err
	}

	// https://docs.solana.com/developing/clients/jsonrpc-api#sendtransaction
	// Should simulate? Looking at the doc, the RPC node seems like it would
	// simulate first before submitting to the network.
	return c.Client.SendRawTransaction(c.Ctx, txdata)
}

func (c *TokenProgram) accountData(addr common.PublicKey) ([]byte, error) {

	accountRes, err := c.Client.GetAccountInfo(c.Ctx, addr.String(), rpc.GetAccountInfoConfig{
		Encoding: rpc.GetAccountInfoConfigEncodingBase64,
	})

	if err != nil {
		return nil, err
	}

	// A non-existeet account returns:
	//
	// (rpc.GetAccountInfoResponse) {
	// 	Lamports: (uint64) 0,
	// 	Owner: (string) "",
	// 	Excutable: (bool) false,
	// 	RentEpoch: (uint64) 0,
	// 	Data: (interface {}) <nil>
	//  }

	if accountRes.Data == nil {
		return nil, nil
	}

	// todo: handle case where account data is nil
	base64String := accountRes.Data.([]interface{})[0].(string)

	data, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return nil, fmt.Errorf("account data base64 decode error: %w", err)
	}

	return data, nil

}
