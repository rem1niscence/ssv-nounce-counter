# SSV Nounce Counter

This project is designed to interact with the Ethereum blockchain for managing and tracking nonce values for specific addresses, based on contract events. It achieves this using the Go Ethereum (Geth) library and primarily focuses on the following core functionalities:

### 1. Event Monitoring
- The project scans the Ethereum blockchain for specific events emitted by a smart contract, such as `ValidatorAdded`. This event signifies the addition of a validator with detailed metadata, including associated owner addresses, public keys, and account balances.

### 2. Nonce Management
- It maintains a mapping (`addressToNonce`) that tracks nonce values for selected Ethereum addresses in a thread-safe manner. When a relevant event is identified in the blockchain logs, the corresponding nonce for the address is incremented.

### 3. Efficient Blockchain Querying
- The implementation supports querying blockchain logs in batches (`blockBatchSize`), ensuring it efficiently processes block data without exceeding resource limits.
- Concurrency is managed with a configurable semaphore mechanism, enabling simultaneous log processing without race conditions.

### 4. Configurable & Validated Setup
- The project includes a `Config` structure which allows user customization like concurrency limits, starting block number, target contract address/ABI, event names, and batch sizes. This configuration is validated before initializing the system.

### 5. Scalability and Thread-Safety
- With the use of synchronization primitives like `sync.Mutex` and goroutines, the implementation ensures thread-safe nonce updates while processing logs in parallel.

---

### Main Components:
- **`main.go`**: Entry point that initializes the Ethereum client, processes blockchain logs, and parses contract events continuously.
- **`nonce_counter.go`**: Defines the `NonceCounter` and core logic for tracking events, querying logs, processing batches, and updating nonces.
- **`event.go`**: Provides a `ValidatorAddedEvent` definition and utilities for decoding and parsing blockchain events.

---

### Instructions to Run the Project:

1. **Install Dependencies**:
   - Ensure you have Go installed (version 1.23 or later).
   - Download and install project dependencies by running:
     ```bash
     go mod tidy
     ```

2. **Set Configuration**:
   - **Optional** Update the constants in `cmd/main.go` (like `rpcURL`, `contractAddress`, `eventName`, and `startBlockDecimal`) to match your specific Ethereum network and smart contract details.

3. **Compile and Run**:
   - Run the project by executing:
     ```bash
     go run .
     ```

4. **Monitor Output**:
   - Once running, the program will continuously listen for logs from the specified Ethereum smart contract and process the `ValidatorAdded` events.

Note: A functioning binary has been added for convenience
