.PHONY: airdrop

airdrop:
	mkdir -p build
	go build -o ./build/airdrop ./airdrop/airdrop