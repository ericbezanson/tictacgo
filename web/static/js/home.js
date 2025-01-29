document.addEventListener("DOMContentLoaded", () => {
    const createLobbyBtn = document.getElementById("createLobbyBtn");
    const submitUsernameBtn = document.getElementById("submitUsernameBtn");
    const usernameInput = document.getElementById("username");
    const lobbyList = document.getElementById("lobby-list");
    let username = "";

    function validateUsername(name) {
        const regex = /^[a-zA-Z0-9]{1,15}$/;
        return regex.test(name);
    }

    submitUsernameBtn.addEventListener("click", () => {
       const inputUsername = usernameInput.value.trim();
        console.log("fired", inputUsername)
            if (validateUsername(inputUsername)) {
                username = inputUsername;
                document.cookie = `username=${username}; path=/`; // Set a cookie for the username
                alert(`Username set to ${username}`);
                createLobbyBtn.disabled = false;
            } else {
                alert("Invalid username. Please use only letters and numbers (up to 15 characters).");
            }
       
    });

    if (createLobbyBtn) {
        createLobbyBtn.addEventListener("click", () => {
            const usernameParam = encodeURIComponent(username); // Make sure username is properly encoded
            window.location.href = `/create-lobby?username=${usernameParam}`;
        });
        
    }

    function fetchLobbies() {
        fetch("/lobbies")
            .then((response) => response.json())
            .then((data) => {
                lobbyList.innerHTML = "";

                if (!Array.isArray(data)) {
                    console.error("Expected an array of lobbies, but got:", data);
                    return;
                }

                data.forEach((lobby) => {
                    if (lobby && lobby.Name && Array.isArray(lobby.Players)) {
                        const playerNames = lobby.Players.map(player => player.Name).join(", ");
                        const li = document.createElement("li");
                        li.innerHTML = `${lobby.Name} (${playerNames}/${lobby.MaxPlayers}) 
                        <button onclick="joinLobby('${lobby.ID}')" disabled>Join</button>`;
                        lobbyList.appendChild(li);
                    }
                });

                if (username) {
                    document.querySelectorAll("#lobby-list button").forEach(button => {
                        button.disabled = false;
                    });
                }
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

    function connectToLobby(lobbyID) {
        ws = new WebSocket(`ws://localhost:8080/ws?lobby=${lobbyID}`);

        ws.onopen = () => {
            console.log("Connected to WebSocket", username);
        };

        ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            console.log("(HOMESCREEN: Message received:", message);

        };

        ws.onclose = () => console.log("Disconnected from WebSocket");
    }

    document.addEventListener("DOMContentLoaded", () => {
        const lobbyID = new URLSearchParams(window.location.search).get("lobby");
        if (lobbyID) connectToLobby(lobbyID);
    });
});
