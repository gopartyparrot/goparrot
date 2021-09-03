# Go Parrot

Solana Golang tools by [Parrot](https://parrot.fi)

# Airdrop

A CLI airdrop tool that

- Always send to receiver's associated token address
- Processes token transfers concurrently
- Ensures that each unique transfer request is processed only once
- Confirms transfer transactions
- Tolerates program restarts and RPC errors

## Airdrop Input File

This airdrop tool will read transfer requests from an input file, where each
line is a JSON object:

- memo: a string that distinguishes between different airdrop campaigns
- amount: the amount of tokens to send
- to: the main wallet address of receiver
- mint: to tokene to send

```json
{"memo":"abcd","amount":"1000000","to":"C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM","mint":"4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT"}
{"memo":"abcd","amount":"2000000","to":"C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM","mint":"4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT"}
{"memo":"abcd","amount":"3000000","to":"C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM","mint":"4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT"}
{"memo":"abcd","amount":"4000000","to":"C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM","mint":"4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT"}
{"memo":"abcd","amount":"5000000","to":"C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM","mint":"4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT"}
{"memo":"abcd","amount":"6000000","to":"C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM","mint":"4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT"}
{"memo":"abcd","amount":"7000000","to":"C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM","mint":"4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT"}
{"memo":"abcd","amount":"8000000","to":"C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM","mint":"4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT"}
{"memo":"abcd","amount":"9000000","to":"C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM","mint":"4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT"}
```

These fields are appended together to form a unique key to make sure that
each transfer request is processed exactly once.

## Running Airdrop

Configure your wallet and RPC in `.env`:

```
# private key in hex.
# 6ewXcXuD1LWVr5qZWgzs3vjEQfY6EJQocWA3p46zqRth
WALLET=8cff95874f5212047774c0d1437c1f0d2c04ca400a3b2f706edee7b179392fd0540294506b9309e39f1a6ddd928903d32d3ecdac7ed355da139837ea1f821b86

# solana RPC node
RPC=https://api.devnet.solana.com
```

```
airdrop -c 5 airdrop.100.json
```

- `airdrop.100.json` is the input file of airdrop requests
- `-c 5` specifies a concurrency of 5 when processing requests

(Note: the public RPC has stringent rate-limit, so use a lower concurrency.)

The progress of the airdrop would be saved in `airdrop.store.json`:

```json
{
  "abcd:C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM:4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT:1000000": {
    "TXID": "4CR8CKoXRmnGTKews84X4x3uJZNTHZoykPHsaHvCjrm7xxhNdCzJfc4jUSn9sCTzPi3h7MMrZPvJCKywxrRy2Czb",
    "Memo": "abcd",
    "Mint": "4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT",
    "To": "C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM",
    "Amount": "1000000",
    "ConnfirmedSlot": "78421840"
  },
  "abcd:C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM:4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT:2000000": {
    "TXID": "3Z1YohqR1TU4NMDaJmSaE6F2L8tBrNebLaxV2xsdfZRqQPAUUS9W2xAvnv77awMcaBZusDJ8DLf6NdgMKzepV5k8",
    "Memo": "abcd",
    "Mint": "4ZqfC84c1qgMnLgjeoi5tPfpmvHhinvhQSYjfYGLjaDT",
    "To": "C23a3jPhw84hy6mRAhBbNhjgzSuGFcGeNExCCBe1bHSM",
    "Amount": "2000000",
    "ConnfirmedSlot": "78421877"
  }
}
```

The `airdrop.store.json` file ensures that the airdrop are processed exactly once, and verify that the transactions have gone through.

If the `airdrop` program is interrupted half-way through, you can restart it
again, and the program will load history from `airdrop.store.json` to pick up from where it left off.

## Running Airdrop (Advanced)

The parrot needs to process many airdrops simultaneously when distributing +earn profits to users. The airdrop tool can process multiple input files:

```
airdrop -c 20 \
  airdrop-MERLP-USDC.USDT.UST+earn.json \
  airdrop-SBRLP-USDT.USDC+earn.json \
  airdrop-SBRLP-USD.UST+earn.json \
  airdrop-SBRLP-SOL.mSOL+earn.json
```
