# messagerelayer

## Description

This project simulates a message relaying system where messages are broadcast from a network endpoint to other systems or subscribers within the codebase. It's designed to demonstrate handling real-time messaging with constraints on memory usage and subscriber responsiveness.

## Technical Decisions

### Architecture

### MessageRelayer

- Network Socket Listening: The MessageRelayer continuously listens for incoming messages from the network socket in a dedicated goroutine. Upon receiving a new message, it performs two main actions:
 - Channel Dispatch: Immediately sends the received message to a channel that's monitored for broadcasting messages to subscribers.
 - RelayBuffer Update: Updates the RelayBuffer with the new message, ensuring that the buffer maintains only the most relevant messages according to predefined rules.
- New subscriber are added to the subscriber lists and receive the most recent messages from the RelayBuffer.

### RelayBuffer

- Manages the storage and prioritization of messages within the MessageRelayer, adhering to memory constraints and ensuring efficient message broadcasting.
- Stores up to two most recent StartNewRound messages and the most recent ReceivedAnswer message, prioritizing StartNewRound messages when broadcasting.

### MockNetworkSocket

- Two versions of `MockNetworkSocket` are used to simulate network behavior:
  - `ChannelMockNetworkSocket`: Utilizes a Go channel to dynamically receive and send messages.
  - `StaticMockNetworkSocket`: Preloads messages from an array, simulating a fixed set of messages to be relayed.

### Message Types

- Supports two message types, `StartNewRound` and `ReceivedAnswer`, with prioritization logic for broadcasting.

### Subscriber Management

- Dynamically manage subscribers, allowing runtime subscription and processing of messages according to subscriber interest.

## Commands and How To

### Running the Project

1. **Clone the repository**: Ensure you have Git and Go installed on your system.
```
git clone https://github.com/0xnogo/messagerelayer.git
```

2. **Navigate to the project directory**:
```
cd messagerelayer
```

3. **Build the project** (optional):
```
go build .
```

4. **Run the tests**:
```
go test -v ./...
```

5. **Run the project**:
- In normal mode:

  ```
  go run .
  ```

- In interactive mode:

  ```
  go run . --interactive
  ```

### Interactive Commands

- **Add a message**: Press 'm' to add a dynamic message to the system.
- **Add a subscriber**: Press 'u' to add a new subscriber with random interest.
- **List subscribers**: Press 'l' to display a list of current subscribers and their interests.
- **Quit**: Use 'Ctrl+C' to exit the interactive mode.