Project: Go-based educational blockchain (PoS-inspired)

Overview
- A Go-based educational blockchain prototype focusing on teaching concepts such as transactions, blocks, signatures, and a simplified PoS reward model with a Bitcoin-like halving schedule.
- Genesis supplies 50 units to a genesis address 0x0000000000000000000000000000000000000000.

What’s implemented
- Genesis block mints a fixed reserve of 50 units to a genesis address.
- Transactions include From, To, Amount, Nonce, Fee, Payload, and Signature (Sig).
- Signatures are verified against the sender's public key registered in the blockchain state.
- State updates occur when blocks are added: balances, nonces, and total supply.
- Block rewards follow a Bitcoin-like halving schedule (initial 50, halving interval 210000 blocks).
- Persistence: blockchain and state are saved synchronously to disk and loaded on startup.
- Multi-wallet capability via API: wallets can be created and used to send transactions.
- Logs are in English and structured for readability.

Architecture
- internal/core: core data models (Block, Transaction, Blockchain, Mempool, ValidatorSet)
- internal/wallet: wallet/keypair management and signing utilities
- Main API and node orchestration in main.go (Node, HTTP handlers, block production loop)
- internal/consensus: simplified PoS components (validator selection, stake updates)
- README, scripts, and tests accompany the code for testing and validation

Getting started
- Prerequisites: Go installed (same version used in CI environments)
- Build: go build -o blockchain-app
- Run: ./blockchain-app
- Tests: go test ./... -v

Endpoints (summary)
- POST /wallets: create a new wallet
- GET /wallets/{address}/balance: view balance for an address
- GET /blocks: fetch blocks
- POST /transactions: submit a new transaction
- GET /validators: view validators and stakes
- POST /validators: register a validator (with stake)
- Others can be added as the API evolves

Persistence
- State is saved synchronously to disk (block data and balances/nonces)
- State is loaded on startup to resume from previous runs

Testing
- Unit tests cover genesis, persistence, signing, etc.
- A test script is provided to automate build, run, and quick checks

Roadmap 
- Improve API coverage and error handling
- Add validator bonding/unbonding and more realistic PoS
- Add snapshots/persist/more robust state management
- Expand tests including integration and network simulations
- Add CI pipeline

Contributing
- Follow existing conventions
- Add tests for new features
- Keep code style clean and documented
