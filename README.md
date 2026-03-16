Project: Go-based educational blockchain (PoS-inspired)

Overview
- A Go-based, educational blockchain prototype focused on teaching concepts such as transactions, blocks, signatures, and a simplified PoS reward model with a Bitcoin-like halving schedule. All code, comments, and API text are maintained in English.

Key concepts implemented
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

How to run
- Prerequisites: Go installed (same version used in CI environments)
- Build: go build -o blockchain-app
- Run: ./blockchain-app
- Tests: go test ./... -v

Endpoints (English naming preserved)
- POST /wallets: create a new wallet
- GET /wallets/{address}/balance: view balance for an address
- GET /blocks: fetch blocks
- POST /transactions: submit a new transaction
- GET /validators: view validators and stakes
- POST /validators: register a validator (with stake)
- Others can be added as the API evolves

Persistence details
- State is saved synchronously to disk (block data and balances/nonces)
- State is loaded on startup to resume from previous runs

Tests
- Unit tests cover transaction creation, block processing, rewards, and persistence
- A test script is provided to automate build, run, and quick checks

Contributing
- Follow the existing conventions (English only, consistent naming)
- Add tests for any new feature or behavior
- Keep code style clean and well-documented

Roadmap (high level)
- Improve API coverage and error responses
- Add bonding/unbonding for validators and more realistic PoS logic
- Expand persistence to support snapshots and more robust state management
- Introduce more tests, including integration tests and simple network simulations

For any questions or to propose changes, reach out and we can discuss the best approach to maintain the project as a learning resource.

Implementation Details
- Genesis and initial supply: Genesis block mints a fixed reserve of 50 units to a synthetic genesis address (0x0000000000000000000000000000000000000000). This creates an initial state from which further transfers occur.
- Persisted state: Blockchain data and balances/nonces are saved to disk synchronously after each block, and loaded at startup to resume the chain state.
- Signature verification: Every normal transaction must include a signature (Sig) and the sender's public key must be registered in the blockchain state. Signatures are verified using ECDSA (secp256k1-like curve, using Go's crypto APIs).
- Public key registry: Wallets created via the API register their public keys with the blockchain so signatures can be validated across nodes.
- Validation flow: The state updates on a block are applied only after the transactions have valid signatures and sufficient balances. Nonces are checked per address to prevent replays.
- Rewards and halving: The PoS reward schedule mimics Bitcoin's halving, with an initial reward and a halving interval. Rewards are minted via system transactions to the selected validator in a given block height.
- API stability: API naming remains in English and follows the existing structure. Endpoints are designed for wallet creation, balance checks, transaction submission, and block/validator inspection.
- Testing: Test suites cover transaction creation, block processing, reward calculation, and persistence, with automated scripts to run unit tests and end-to-end checks.
- How to extend: The README now includes guidance for adding features like bonding/unbonding of stake, advanced consensus behaviors, and more robust persistence mechanisms as next milestones.
