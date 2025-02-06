const lobbyID = window.location.pathname.split("/")[2];  // Extract lobby ID from URL

const ws = new WebSocket(`ws://localhost:8080/ws?lobby=${lobbyID}`);


// Interface vars
const gameBoard = document.getElementById("tic-tac-toe");
const messagesDiv = document.getElementById("messages");
const playerInfo = document.getElementById("player-info");
const user = document.getElementById("user")
const role = document.getElementById("role")
const readyToggle = document.getElementById("ready-toggle");

// Initial game values
let currentPlayer = "X";
let username = "";
let playerSymbol = "";
let activePlayer = "X";
let gameStarted = false;
let playerTotal = 0;
let isReady = false;  // Track the player's readiness
// Local chatMessages array
let chatMessages = [];

// WebSocket connection opened
ws.onopen = () => {

    const userProfileJSON = localStorage.getItem('TTTprofile');

    if (userProfileJSON) {
        const userProfile = JSON.parse(userProfileJSON);
        if (userProfile) {
            // Send the cookie value to the server
            ws.send(JSON.stringify({
                type: "setUsername",
                username: userProfile.Name,
                id: userProfile.ID
            }));
        } else {
            console.log("Local Storage Data not found");
        }
    } else {
        console.log("Object not found in local storage.");
    }
};

// WebSocket connection error
ws.onerror = (error) => {
    console.log("WebSocket error:", error);
};

// WebSocket connection closed
ws.onclose = () => {
    console.log("WebSocket connection closed");
};

// Handler for messages received from server
ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    switch (message.type) {

        case "initialState":
            // Populate game board
            message.gameBoard.forEach((symbol, index) => {
                if (symbol) {
                    gameBoard.children[index].textContent = symbol;
                    gameBoard.children[index].style.pointerEvents = "none";
                }
            });

            if (message.chatMessages && Array.isArray(message.chatMessages)) {
                updateChatMessages(message.chatMessages);
            }

            break;


        case "updatePlayers":
            // Handle chat messages
            if (message.chatMessages && Array.isArray(message.chatMessages)) {
                updateChatMessages(message.chatMessages);
            }

            // Other state updates (e.g., game board, players)
            // Update gameBoard or other UI elements here
            break;


        case "lobbyFull":
            username = message.usermame;
            user.innerHTML = `YOU ARE SPECTATING AS <b>${username}</b>`;
            alert(message.text);
            break;

        case "assignPlayer":
            username = message.username;
            playerSymbol = message.symbol;
            user.innerHTML = `YOU ARE PLAYING AS <b>${username}</b>`;
            role.innerHTML = `Playing as: <b>${playerSymbol}<b>`

            // Create the user profile object
            const userProfile = {
                ID: message.id, // Assuming the server sends the ID
                Name: username,
            };

            const userProfileJSON = JSON.stringify(userProfile);

            localStorage.setItem('TTTprofile', userProfileJSON);
            break;

        case "startGame":
            gameStarted = true

        // Handler for player moves
        case "move":
            if (typeof message.position === "number" && message.position >= 0) {
                const cell = gameBoard.children[message.position];
                if (!cell) {
                    console.error("Invalid tile position", message.position);
                    return;
                }
                cell.textContent = message.symbol;
                cell.style.pointerEvents = "none";
                handleNext(message); // Call handleNext *here*
            } else {
                console.error("Unexpected message format", message);
            }
            break; // Break is essential here

        case "chat":
            updateChatMessages(message.chatMessages)

            break;

        default:
            console.error("Unknown message type:", message);
    }

    // Auto-scroll chat
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
};

// Sends message to server when submit button is clicked
function sendMessage() {
    const input = document.getElementById("message");
    ws.send(JSON.stringify({ type: "chat", sender: username, text: input.value }));
    input.value = "";
}

// Send Tic-Tac-Toe tile position data to server to be processed by server game logic
function handleCellClick(e) {
    const cell = e.target;
    const cells = Array.from(gameBoard.children);
    const position = cells.indexOf(cell);

    // Check if spectator
    if (playerSymbol === "") {
        alert("You are spectating and cannot play.");
        return;
    }

    // Prevent clicking on an already-filled cell or playing out of turn
    if (cell.textContent !== "" || activePlayer !== playerSymbol) {
        alert("It's not your turn!");
        return;
    }

    // prevent click on a tile before game has started 
    if (!gameStarted) {
        alert("Slow down! game has not started yet!")
        return;
    }

    // Send move message to the server
    ws.send(JSON.stringify({
        type: "move",
        position: position,
        username: username,
        symbol: playerSymbol
    }));
}

// Handle the "Ready Up" toggle
function toggleReady() {
    isReady = !isReady;

    // Send the readiness status to the server
    ws.send(JSON.stringify({
        type: "ready",
        ready: isReady,
        username: username
    }));
}
// Creates game board
function createTicTacToeBoard() {
    gameBoard.innerHTML = "";  // Clear previous game board
    for (let i = 0; i < 9; i++) {
        const cell = document.createElement("div");
        cell.classList.add("cell");
        cell.addEventListener("click", handleCellClick);  // Attach event
        gameBoard.appendChild(cell);
    }
}

createTicTacToeBoard();  // Call this during page load

// Reset Board with Style Reset
function resetBoard() {
    Array.from(gameBoard.children).forEach((cell) => {
        cell.textContent = "";
        cell.style.pointerEvents = "auto"; // Reset pointer-events style
        cell.style.backgroundColor = "";  // Reset cell background
    });
}


// Function to update chat messages
function updateChatMessages(serverMessages) {
    const localLength = chatMessages.length;
    const serverLength = serverMessages.length;

    // if new messages available
    if (serverLength > localLength) {
        // Add only the new messages
        const newMessages = serverMessages.slice(localLength);
        chatMessages = [...chatMessages, ...newMessages];

        // Render the new messages on the UI
        renderMessages(newMessages);
    }
}

// Function to render messages on the UI
function renderMessages(messages) {
    if (messages) {
        // Check if 'messages' is an array or a single object
        const isArray = Array.isArray(messages);

        // Create an array to handle both single object and array cases
        const messagesArray = isArray ? messages : [messages];

        messagesArray.forEach(chatMsg => {
            // Check if the message has already been added 
            // (You might want to refine this logic based on your needs)

            // Check if sender is "GAMEMASTER" and add the "system-msg" class
            const timestamp = new Date(chatMsg.Timestamp).toLocaleString('en-US', { hour: 'numeric', minute: 'numeric', second: 'numeric', hour12: true });
            if (chatMsg.Sender === "GAMEMASTER") {
                messagesDiv.innerHTML += `<p class="system-msg">${chatMsg.Sender}: ${chatMsg.Text} <span class="timestamp">(${timestamp})</span></p>`;
            } else {
                messagesDiv.innerHTML += `<p>${chatMsg.Sender}: ${chatMsg.Text} <span class="timestamp">(${timestamp})</span></p>`;
            }

        });
    }
}

function handleNext(message) {
    switch (message.next) {
        case "updateTurn":
            activePlayer = message.text;
            break;

        case "win":
            gameStarted = false
            isReady = false
            alert(message.text);  // Show the winner

            // Uncheck the "ready-toggle" checkbox
            readyToggle.checked = false;

            // TODO! - add support for unready messages
            ws.send(JSON.stringify({
                type: "ready",
                ready: false,
                username: username
            }));
            resetBoard();  // Reset the game
            break;

        case "draw":
            gameStarted = false
            isReady = false
            alert(message.text);  // Show the winner

            // Uncheck the "ready-toggle" checkbox
            readyToggle.checked = false;

            // TODO! - add support for unready messages
            ws.send(JSON.stringify({
                type: "ready",
                ready: false,
                username: username
            }));
            resetBoard();  // Reset the game
            break;
        default:
            break;
    }
}