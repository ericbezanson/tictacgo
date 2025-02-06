document.addEventListener("DOMContentLoaded", () => {
    const createLobbyBtn = document.getElementById("createLobbyBtn");
    const submitUsernameBtn = document.getElementById("submitUsernameBtn");
    const usernameInput = document.getElementById("username");
    const lobbyList = document.getElementById("lobby-list");
    const nameSubmit = document.getElementById("nameSubmit")
    const usernameDisplay = document.createElement("p");
    usernameDisplay.id = "usernameDisplay";
    document.body.insertBefore(usernameDisplay, nameSubmit);

    let username = "";
    let userProfile = null; // Store the user profile data

    function validateUsername(name) {
        const regex = /^[a-zA-Z0-9]{1,15}$/;
        return regex.test(name);
    }

    const userProfileJSON = localStorage.getItem('TTTprofile');

    if (userProfileJSON) {
        const userProfile = JSON.parse(userProfileJSON);
        if (userProfile) {
            username = userProfile.Name; // Set username from cookie if available
            usernameDisplay.innerHTML = `You are playing as: <strong>${username}</strong>`;
            createLobbyBtn.disabled = false;
            usernameInput.style.display = "none"; // Hide username input
            submitUsernameBtn.disabled = true;
            submitUsernameBtn.style.display = "none" // Hide submit button
        } else {
            console.log("Local Storage Data not found");
        }
    } else {
        console.log("Object not found in local storage.");
        usernameInput.style.display = "inline"; // Show username input
        submitUsernameBtn.disabled = false; // Show submit button
    }

    submitUsernameBtn.addEventListener("click", () => {
        const inputUsername = usernameInput.value.trim();
        if (validateUsername(inputUsername)) {
            username = inputUsername;
            usernameDisplay.innerHTML = `You are playing as: <strong>${username}</strong>`;
            createLobbyBtn.disabled = false;
            usernameInput.style.display = "none"; // Hide username input
            submitUsernameBtn.style.display = "none"; // Hide submit button
            // Create the user profile object
            const userProfile = {
                Name: username,
            };

     
            const userProfileJSON = JSON.stringify(userProfile);

            localStorage.setItem('TTTprofile', userProfileJSON);

        } else {
            alert("Invalid username. Please use only letters and numbers (up to 15 characters).");
        }
    });

    if (createLobbyBtn) {
        createLobbyBtn.addEventListener("click", () => {
            // Construct query parameters from userProfile (or username if cookie not available)
            let params = {};
            if (userProfile) {
                params = {
                    ID: userProfile.ID,
                    Name: userProfile.Name,
                };
            } else {
                params = {
                    Name: username, // Fallback to username if cookie not available
                };
            }

            const queryString = new URLSearchParams(params).toString();
            window.location.href = `/create-lobby?${queryString}`;
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
        let params = {};
        if (userProfile) {
            params = {
                ID: userProfile.id,
                Name: userProfile.Name,
            };
        } else {
            params = {
                Name: username, // Fallback to username if cookie not available
            };
        }
        const queryString = new URLSearchParams(params).toString();

        window.location.href = `/lobby/${id}?${queryString}`;
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

        ws.onclose = () => console.log("Disconnected from WebSocket");
    }

    document.addEventListener("DOMContentLoaded", () => {
        const lobbyID = new URLSearchParams(window.location.search).get("lobby");
        if (lobbyID) connectToLobby(lobbyID);
    });
});


