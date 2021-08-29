package airdrop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/gopartyparrot/parrot/spl"
	"github.com/portto/solana-go-sdk/client"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/types"
)

type TransferRequest struct {
	Memo   string
	Mint   common.PublicKey
	To     common.PublicKey
	Amount uint64
}

type TransferStatus struct {
	TXID string
	TransferRequest
	// ConnfirmedSlot the slot where this tx had been confirmed. 0 means unconfirmed.
	ConnfirmedSlot uint64
}

type TransferStatusStore struct {
	store *JSONStore
}

func (s *TransferStatusStore) Set(key string, status TransferStatus) error {
	return s.store.Set(key, &status)
}

func (s *TransferStatusStore) Get(key string, status *TransferStatus) (bool, error) {
	return s.store.Get(key, status)

}

type BatchSenderConfig struct {
	RPCEndpoint string
	StorePath   string
	Wallet      types.Account
	Concurrency uint
}

func NewBatchSender(cfg BatchSenderConfig) (*BatchSender, error) {
	store, err := OpenJSONStore(cfg.StorePath)
	if err != nil {
		return nil, err
	}

	ctx := context.TODO()

	c := client.NewClient(cfg.RPCEndpoint)
	tp := spl.TokenProgram{Ctx: ctx, Client: c}
	atp := &spl.AssociatedTokenProgram{TokenProgram: tp}

	concurrency := cfg.Concurrency
	if concurrency == 0 {
		concurrency = 1
	}

	return &BatchSender{
		store:  &TransferStatusStore{store},
		atp:    atp,
		wallet: cfg.Wallet,
		wtk:    make(chan struct{}, concurrency),
	}, nil

}

// BatchSender stores the transfer statuses of tranfser requests in a JSON file to make
// ensure that each transfer request is sent & confirmed once-only.
type BatchSender struct {
	store  *TransferStatusStore
	atp    *spl.AssociatedTokenProgram
	wallet types.Account

	wg sync.WaitGroup
	// concurrency work tokens
	wtk chan struct{}
}

func (s *BatchSender) Wait() {
	s.wg.Wait()
}

func (s *BatchSender) AddTransfer(req TransferRequest) {

	s.wtk <- struct{}{}
	s.wg.Add(1)
	go func() {
		err := s.Transfer(req)
		s.wg.Done()
		<-s.wtk
		if err != nil {
			log.Println("transfer error:", req, err)
		}
	}()
}

// Transfer makes a token transfer once-only (according to request key)
func (s *BatchSender) Transfer(req TransferRequest) error {
	if req.Memo == "" {
		return errors.New("empty transfer request memo is invalid")
	}

	if req.Amount == 0 {
		return nil
	}

	key := fmt.Sprintf("%s:%s:%s:%d", req.Memo, req.To, req.Mint, req.Amount)

	var status TransferStatus
	found, err := s.store.Get(key, &status)
	if err != nil {
		return err
	}

	if !found {
		log.Println("submit:", req)
		txid, err := s.atp.Transfer(req.Mint, s.wallet, req.To, req.Amount)
		if err != nil {
			return err
		}

		status = TransferStatus{
			TXID:            txid,
			TransferRequest: req,
		}

		err = s.store.Set(key, status)
		if err != nil {
			return err
		}
	}

	if status.ConnfirmedSlot > 0 {
		// already confirmed
		return nil
	}

	log.Println("confirming:", req)
	txres, err := s.atp.ConfirmTx(status.TXID)
	if err != nil {
		return err
	}

	status.ConnfirmedSlot = txres.Slot
	err = s.store.Set(key, status)
	if err != nil {
		return err
	}

	return nil
}
