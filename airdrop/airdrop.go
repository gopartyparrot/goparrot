package airdrop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/gopartyparrot/goparrot/spl"
	"github.com/portto/solana-go-sdk/client"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/types"
)

var ErrCanceledContext = errors.New("canceled context")

type TransferRequest struct {
	Memo   string
	Mint   common.PublicKey
	To     common.PublicKey
	Amount uint64 `json:",string"`
}

type TransferStatus struct {
	TXID string
	TransferRequest
	// ConnfirmedSlot the slot where this tx had been confirmed. 0 means unconfirmed.
	ConnfirmedSlot uint64 `json:",string"`
	ErrLogs        string `json:",omitempty"`
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
	RPCEndpoint   string
	StorePath     string
	Wallet        types.Account
	Concurrency   uint
	VerifyConfirm bool
	RetryError    bool
}

func NewBatchSender(ctx context.Context, cfg BatchSenderConfig) (*BatchSender, error) {
	store, err := OpenJSONStore(cfg.StorePath)
	if err != nil {
		return nil, err
	}

	clientCtx := context.TODO()
	c := client.NewClient(cfg.RPCEndpoint)
	tp := spl.TokenProgram{Ctx: clientCtx, Client: c}
	atp := &spl.AssociatedTokenProgram{TokenProgram: tp}

	concurrency := cfg.Concurrency
	if concurrency == 0 {
		concurrency = 1
	}

	return &BatchSender{
		cfg:    cfg,
		ctx:    ctx,
		store:  &TransferStatusStore{store},
		atp:    atp,
		wallet: cfg.Wallet,
		wtk:    make(chan struct{}, concurrency),
	}, nil

}

// BatchSender stores the transfer statuses of tranfser requests in a JSON file to make
// ensure that each transfer request is sent & confirmed once-only.
type BatchSender struct {
	cfg    BatchSenderConfig
	ctx    context.Context
	store  *TransferStatusStore
	atp    *spl.AssociatedTokenProgram
	wallet types.Account

	wg sync.WaitGroup
	// concurrency work tokens
	wtk chan struct{}

	// halt the batch sender if too many errors
	errCount int64
}

func (s *BatchSender) Wait() {
	s.wg.Wait()
}

func (s *BatchSender) AddTransfer(req TransferRequest) error {

	s.wtk <- struct{}{}

	errCount := atomic.LoadInt64(&s.errCount)
	if errCount >= 3 {
		return errors.New("too many transfer error")
	}

	s.wg.Add(1)

	go func() {
		defer func() {
			s.wg.Done()
			<-s.wtk
		}()

		select {
		case <-s.ctx.Done():
			// don't do the task, context cancelled
			return
		default:
		}

		err := s.Transfer(req)

		if err != nil {
			atomic.AddInt64(&s.errCount, 1)
			log.Println("transfer error:", req, err)
		}
	}()

	return nil
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

	if !found && (s.cfg.VerifyConfirm || s.cfg.RetryError) {
		// don't create new transactions in verify or retry
		return nil
	}

	var retryError bool
	if found {
		retryError = s.cfg.RetryError && status.ErrLogs != ""
	}

	if !found || retryError {
		txid, err := s.atp.Transfer(req.Mint, s.wallet, req.To, req.Amount)
		if err != nil {
			return err
		}

		log.Println("submitted tx:", key, txid)

		status = TransferStatus{
			TXID:            txid,
			TransferRequest: req,
		}

		err = s.store.Set(key, status)
		if err != nil {
			return err
		}
	}

	if status.ConnfirmedSlot > 0 && !s.cfg.VerifyConfirm {
		// already confirmed
		return nil
	}

	txres, err := s.atp.ConfirmTx(s.ctx, status.TXID)
	if err != nil {
		return fmt.Errorf("confirm tx: %s", err)
	}

	// verify mode
	if txres.Meta.Err != nil {
		status.ErrLogs = fmt.Sprintf("error: %v", txres.Meta.LogMessages)
	}

	status.ConnfirmedSlot = txres.Slot
	err = s.store.Set(key, status)
	if err != nil {
		return err
	}

	return nil
}
