module github.com/gopartyparrot/goparrot

go 1.15

require (
	github.com/alexflint/go-arg v1.4.2
	github.com/davecgh/go-spew v1.1.1
	github.com/joho/godotenv v1.3.0
	github.com/portto/solana-go-sdk v1.0.0
)

// replace github.com/portto/solana-go-sdk v1.0.0 => ./solana-go-sdk
replace github.com/portto/solana-go-sdk v1.0.0 => github.com/gopartyparrot/solana-go-sdk v1.3.1-0.20210829082613-23da27539114
