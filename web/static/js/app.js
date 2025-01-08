document.addEventListener("DOMContentLoaded", () => {
    const createLobbyBtn = document.getElementById("createLobbyBtn");
    const lobbyList = document.getElementById("lobby-list");

    // Redirects the user to a new lobby creation page when they click "Create Lobby."
    if (createLobbyBtn) {
        createLobbyBtn.addEventListener("click", () => {
            window.location.href = "/create-lobby";
        });
    }
    // Fetches available lobbies from the server and displays them dynamically.
    function fetchLobbies() {
        fetch("/lobbies")
            .then((response) => response.json())
            .then((data) => {
                console.log("DATA", data);
                lobbyList.innerHTML = "";
    
                // Check the structure of the data to prevent errors
                if (!Array.isArray(data)) {
                    console.error("Expected an array of lobbies, but got:", data);
                    return;
                }
    
                data.forEach((lobby) => {
                    if (lobby && lobby.Name && Array.isArray(lobby.Players)) {
                        const playerCount = lobby.Players.length;
                        const playerNames = lobby.Players.map(player => player.Name).join(", ");
                        const li = document.createElement("li");
                        li.innerHTML = `${lobby.Name} (${playerNames}/${lobby.MaxPlayers}) 
                        <button onclick="joinLobby('${lobby.ID}')">Join</button>`;
                        lobbyList.appendChild(li);
                    }
                });
            })
            .catch((error) => {
                console.error("Error fetching lobbies:", error);
                alert(`Error fetching lobbies: ${error.message}`);
            });
    }
    

    window.joinLobby = function (id) {
        window.location.href = `/lobby/${id}`;
    };

    if (lobbyList) {
        fetchLobbies();
        setInterval(fetchLobbies, 3000); // Refresh lobbies every 3 seconds
    }
});

let ws;

// Opens a WebSocket connection to the server.
function connectToLobby(lobbyID) {
    ws = new WebSocket(`ws://localhost:8080/ws?lobby=${lobbyID}`);

    ws.onopen = () => console.log("Connected to WebSocket");

    ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        console.log("Message received:", message);

        switch (message.type) {

            case "updatePlayers":
                const playerList = message.players.map(p => `${p.Name} (${p.Mark})`).join(", ");
                playerInfo.innerHTML = `Current Players: <b>${playerList}</b>`;
                break;
            // more than two connections 
            case "lobbyFull":
                userName = message.userName;
                playerInfo.innerHTML = `YOU ARE SPECTATING AS <b>${userName}</b>`;
                alert(message.text);
                break;

            // player move message, triggered by what handleCellClick() sends to server
            case "move":
                // Ensure message.position is a valid number
                if (typeof message.position === "number" && message.position >= 0) {
                    console.log("mesg", message)
                    const cell = gameBoard.children[message.position];
                    console.log("Accessing Position:", message.position);

                    if (!cell) {
                        console.error("Invalid tile position", message.position);
                        return;
                    }
                    cell.textContent = message.text;  // Update the tile
                } else {
                    console.error("Unexpected message format", message);
                }
                break;

            // switch whos turn it is based on who server deems is the active player
            case "updateTurn":
                activePlayer = message.text;
                displaySystemMessage(`It's ${activePlayer}'s turn.`);
                break;

            // used for painting system chat messages
            case "system":
                messagesDiv.innerHTML += `<p class="system-msg">GAMEMASTER: ${message.text}</p>`;
                break;

            // used for painting user chat messages
            case "chat":
                messagesDiv.innerHTML += `<p>${message.sender}: ${message.text}</p>`;
                break;

            // player assignment
            case "assignPlayer":
                userName = message.userName;
                playerSymbol = message.symbol;
                playerInfo.innerHTML = `YOU ARE PLAYING AS <b>${userName} (${playerSymbol})</b>`;
                break;

            // alert when game is won, reset board
            case "gameOver":
                console.log("game over!", message)
                alert(message.text);  // Show the winner
                resetBoard();  // Reset the game
                break;

            // reset
            case "reset":
                resetBoard();
                break;

            default:
                console.error("Unknown message type:", message);
        }

        // Auto-scroll chat
        messagesDiv.scrollTop = messagesDiv.scrollHeight;
    };
    
    ws.onclose = () => console.log("Disconnected from WebSocket");
}

function updateLobbyState(data) {
    const { players } = data;
    const playerList = document.getElementById("player-list");
    playerList.innerHTML = "";

    players.forEach((player) => {
        const li = document.createElement("li");
        li.textContent = `Player ${player.Mark} (ID: ${player.ID})`;
        playerList.appendChild(li);
    });
}

document.addEventListener("DOMContentLoaded", () => {
    const lobbyID = new URLSearchParams(window.location.search).get("lobby");
    if (lobbyID) connectToLobby(lobbyID);
});
