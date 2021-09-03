package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"

	"github.com/gopartyparrot/goparrot/airdrop"
)

type JSONStreamer struct {
	*bufio.Scanner
}

func NewJSONStreamer(r io.Reader) JSONStreamer {
	return JSONStreamer{bufio.NewScanner(r)}
}

func (s *JSONStreamer) Next(v interface{}) error {
	line := s.Bytes()

	err := json.Unmarshal(line, v)
	if err != nil {
		return fmt.Errorf("json decode error: %w\n%s", err, line)
	}

	return nil
}

func processJSONFlie(file string, cfg AirdropArgs) error {
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, os.Interrupt)

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			<-signalC
			log.Println("SIGINT. Wait for tasks to exit gracefully")
			cancel()
		}
	}()

	bs, err := airdrop.NewBatchSender(ctx, airdrop.BatchSenderConfig{
		VerifyConfirm: cfg.Verify,
		RPCEndpoint:   cfg.RPC,
		StorePath:     cfg.Store,
		Wallet:        cfg.Wallet,
		Concurrency:   cfg.Concurrency,
	})
	if err != nil {
		return err
	}

	s := NewJSONStreamer(f)
	var lineno int
loop:
	for s.Scan() {
		lineno++
		var req airdrop.TransferRequest
		err := s.Next(&req)
		if err != nil {
			log.Printf("%s:%d: %s\n", file, lineno, err)
			continue
		}

		err = bs.AddTransfer(req)
		if err != nil {
			log.Println("add transfer error:", err)
			break loop
		}

		// check interrupt
		select {
		case <-ctx.Done():
			break loop
		default:
		}
	}

	bs.Wait()

	err = s.Err()
	if err != nil {
		return err
	}

	return nil
}
