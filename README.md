# goChatSocket - basic websocket chat applicationG

A simple real-time chat application built with Go (using WebSockets) and a basic HTML front end.

## Prerequisites

To run this application, you need the following installed on your system:
- **Go** (version 1.19 or later)
- **Git**

## Getting Started

Follow these steps to clone and run the application:

### 1. Clone the Repository
```bash
git clone https://github.com/ericbezanson/tictacgo.git
cd tictacgo
```

### 2. Install Deps 

```
go mod tidy
```

### 3. Run the Server

```
go run cmd/main.go
// you should see Chat server started on :8080
```

### 4. In Browser

Navigate to `http://localhost:8080` in two different tabs, type message and click send!


### 5. Known Issues!

| **Type**  | **Description**                                                                 |
|-----------|---------------------------------------------------------------------------------|
| [cleanup] | Move JS out of `lobby.html`                                                     |
| [cleanup] | Clean up `app.js`                                                                |
| [cleanup] | Separate spectator and player roles (e.g., spectators should not see a ready button) |
| [bug]     | Sometimes GAMEMASTER chat is not red                                            |
| [bug]     | Unexpected message format error in console with `startGame` message             |
| [bug]     | Game ending (win/stalemate) not starting a new game after adding ready-up system |
| [bug]     | Fix stalemate logic; player X wins in the event of a stalemate                   |
| [bug]     | Ensure that “game hasn't started” message is sent over “not your turn”          |



