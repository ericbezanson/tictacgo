

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
                lobbyList.innerHTML = "";
                console.log("RESP", data)
                // Check the structure of the data to prevent errors
                if (!Array.isArray(data)) {
                    console.error("Expected an array of lobbies, but got:", data);
                    return;
                }
    
                data.forEach((lobby) => {
                    console.log("lobby", lobby)
                    if (lobby && lobby.Name && Array.isArray(lobby.State.Players)) {
                        const playerNames = lobby.State.Players.map(player => player.Name).join(", ");
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