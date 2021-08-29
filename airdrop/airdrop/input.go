package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gopartyparrot/parrot/airdrop"
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
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	bs, err := airdrop.NewBatchSender(airdrop.BatchSenderConfig{
		RPCEndpoint: cfg.RPC,
		StorePath:   cfg.Store,
		Wallet:      cfg.Wallet,
		Concurrency: cfg.Concurrency,
	})
	if err != nil {
		return err
	}

	s := NewJSONStreamer(f)
	var lineno int
	for s.Scan() {
		lineno++
		var req airdrop.TransferRequest
		err := s.Next(&req)
		if err != nil {
			log.Printf("%s:%d: %s\n", file, lineno, err)
			continue
		}

		bs.AddTransfer(req)
	}

	bs.Wait()

	err = s.Err()
	if err != nil {
		return err
	}

	return nil
}
