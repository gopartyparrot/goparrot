package main

import (
	"fmt"
	"log"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
	"github.com/portto/solana-go-sdk/client/rpc"
	"github.com/portto/solana-go-sdk/types"
)

type AirdropArgs struct {
	InputFiles []string `arg:"positional"`

	Concurrency uint          `arg:"-c" help:"concurrent send" default:"1"`
	Wallet      types.Account `arg:"required,env,-w" help:"wallet private key (hex)"`
	RPC         string        `arg:"env,-r" help:"rpc url" default:"devnet"`
	Store       string        `arg:"env,-s" help:"airdrop request statuses" default:"./airdrop.store.json"`
}

func run() error {
	err := godotenv.Load()
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("loading environment: %w", err)
		}
	}

	var args AirdropArgs
	arg.MustParse(&args)

	switch args.RPC {
	case "devnet", "dev":
		args.RPC = rpc.DevnetRPCEndpoint
	case "mainnet", "main":
		args.RPC = rpc.MainnetRPCEndpoint
	}

	spew.Dump("args", args)

	for _, file := range args.InputFiles {
		err := processJSONFlie(file, args)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	err := run()

	if err != nil {
		log.Fatalln(err)
	}
}
