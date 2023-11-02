class Event {
    constructor(type, payload){
        this.type = type;
        this.payload = payload;
    }
}

class SendMessageEvent {
    constructor(message, from, color) {
        this.message = message;
        this.from    = from;
        this.color   = color;
    }
}

class NewMessageEvent {
    constructor(message, from, color, sentDate) {
        this.message = message;
        this.from = from;
        this.color = color;
        this.sentDate = sentDate;
    }
}

class ChangeChatRoomEvent {
    constructor(username, roomname) {
        this.username = username;
        this.roomname = roomname;
    }
}

class NewGameEvent {
    constructor(wordsToAlignment, sentTime) {
        this.sentTime = sentTime;
        this.wordsToAlignment = wordsToAlignment;
    }
}

class GuessEvent {
    constructor(guess) {
        this.guess = guess;
        this.guesser = username;
    }
}

class GuessResponseEvent {
    constructor(guess, guesserTeamColor, correctColor, correct) {
        this.guess = guess;
        this.guesserTeamColor = guesserTeamColor;
        this.correctColor = correctColor;
        this.correct = correct;
    }
}

const numCards = 25;
const colors = ["red", "darkorange", "blue", "dodgerblue", "green",
                "brown", "purple", "hotpink", "black", "gray"];
const defaultRoom = "General";
const defaultRole = "guesser";
const defaultTeam = "red";

var selectedChat = defaultRoom;
var username;
var usercolor;
var userTeam = defaultTeam;
var userRole = defaultRole;
var currentGame;


usercolor = colors[Math.floor(Math.random() * colors.length)];

colorContainer = document.getElementById("color-container");
for (let i = 0; i < colors.length; i++) {
    const colorItem = document.createElement("div");
    colorItem.className = "color-item";
    colorItem.style = `background-color:${colors[i]}`
    colorItem.onclick = function () {
      changeColor(this);
    };
    colorContainer.appendChild(colorItem);
}

const gameBoard = document.getElementById("gameboard");
for (let i = 0; i < numCards; i++) {
    const cardItem = document.createElement("div");
    cardItem.className = "card";
    cardItem.id = `card-${i}`;
    gameBoard.appendChild(cardItem);
}

function abortGame() {
    currentGame = null;
    for (let i = 0; i < numCards; i++) {
        const card = document.getElementById(`card-${i}`)
        card.className = "card";
        card.innerText = "";
    }
    document.getElementById("abort-button").hidden = true;
    document.getElementById("newgame-button").hidden = false;
    document.getElementById("sort-cards").disabled = true;

    const team = document.getElementById("team");
    team.value = defaultTeam;
    if (userTeam !== defaultTeam) {
        changeTeam();
    }
    team.disabled = false;
    
    const role = document.getElementById("role");
    role.value = defaultRole;
    if (userRole !== defaultRole) {
        changeRole();
    }
    role.disabled = false;
}

function setupBoard(payload) {
    // Set global variable
    currentGame = Object.assign(new NewGameEvent, payload);

    let i = 0;
    for (const [word, alignment] of Object.entries(currentGame.wordsToAlignment)) {
        setupCard(i, word, alignment);
        i += 1;
    }
    if (userRole !== defaultRole) {
        disableAllCardEvents();
        document.getElementById("sort-cards").value = "alphabetical";
        document.getElementById("sort-cards").disabled = false;
    }

    setupScoreboard();

    document.getElementById("role").disabled = true;
    document.getElementById("team").disabled = true;
    document.getElementById("newgame-button").hidden = true;
    document.getElementById("abort-button").hidden = false;
}

function sortCards(how) {
    let i = 0;
    switch (how.value) {
        case "alignment":
            var align = new Object();
            for (const [word, alignment] of Object.entries(currentGame.wordsToAlignment)) {
                if (alignment in align) {
                    align[alignment].push(word);
                } else {
                    align[alignment] = [word];
                }
            }
            const alignmentOrder = ["red", "blue", "assassin", "neutral", "white"];
            alignmentOrder.forEach(function (alignment) {
                if (align.hasOwnProperty(alignment)) {
                    align[alignment].forEach(function (word) {
                        setupCard(i, word, alignment);
                        i++;
                    })
                }
            });
            break;
        case "alphabetical":
            for (const [word, alignment] of Object.entries(currentGame.wordsToAlignment).sort()) {
                setupCard(i, word, alignment);
                i++;
            }
            break;
    }
}

function setupCard(cardIdNum, word, alignment) {
    const card = document.getElementById(`card-${cardIdNum}`);
    card.innerText = word;
    card.className = `card ${alignment}`;
    if (!card.className.includes("guessed") && userRole == defaultRole) {
        card.addEventListener("click", this.makeGuess);
    }
}

function setupScoreboard() {
    document.getElementById("redscore").innerText = 9;
    document.getElementById("bluescore").innerText = 8;
}

function makeGuess() {
    sendEvent("guess_event", new GuessEvent(this.innerText));
    return false;
}

function requestNewGame() {
    sendEvent("new_game", null);
    return false;
}

function changeRole() {
    userRole = document.getElementById("role").value;
    sendEvent("change_role", null);
    return false;
}

function changeTeam() {
    userTeam = document.getElementById("team").value;
    sendEvent("change_team", null);
    return false;
}

function changeColor(element) {
    usercolor = getComputedStyle(element).backgroundColor;
}

// Pad single-digit numbers with a leading zero
function padZero(number) {
    return number < 10 ? `0${number}` : `${number}`;
}

// Get formatted time string from a Date object
function fmtTimeFromDate(date) {
    const hours = date.getHours();
    const minutes = date.getMinutes();
    const seconds = date.getSeconds();

    return `${padZero(hours)}:${padZero(minutes)}:${padZero(seconds)}`;
}

function changeChatRoom() {
    var newchat = document.getElementById("chatroom");
    if (newchat != null && newchat.value != selectedChat) {
        selectedChat = newchat.value;
        header = document.getElementById("chat-header").innerHTML = "Currently in chatroom: " + selectedChat;

        let changeEvent = new ChangeChatRoomEvent(username, selectedChat);
        sendEvent("change_room", changeEvent);
    }
    // if you don't return false, it will redirect
    return false;
}

function notifyRoomEntry(payload) {
    roomChange = Object.assign(new ChangeChatRoomEvent, payload);
    const textarea = document.getElementById("chatmessages");
    textarea.innerHTML += `<br><span style="font-weight:bold;">${roomChange.username} has entered the room.</span><br>`;
    textarea.scrollTop = textarea.scrollHeight;
}

function guessResponseHandler(payload) {
    guessResponse = Object.assign(new GuessResponseEvent, payload);
    const guesser = guessResponse.guesser;
    const guessWord = guessResponse.guess;
    const guesserColor = guessResponse.guesserTeamColor;
    const cardColor = guessResponse.cardAlignment;

    markGuessedCard(guessWord, cardColor);
    notifyChatroom(guessWord, guesser, guesserColor, cardColor);
    updateScoreboard(guesserColor, cardColor);
}

function capitalize(word) {
    return word.charAt(0).toUpperCase() + word.substring(1);
}

function notifyChatroom(guess, guesser, guesserColor, cardColor) {
    const teamName = capitalize(guesserColor);
    const textarea = document.getElementById("chatmessages");
    const msg = `<br><span style="font-weight:bold; color:${guesserColor}">${guesser} uncovers ${guess}: `;
    if (guesserColor === cardColor) {
        textarea.innerHTML += `${msg} CORRECT. A point for ${teamName}.</span><br>`;
    } else {
        textarea.innerHTML += `${msg} incorrect. Card is ${cardColor}.</span><br>`;
    }
    textarea.scrollTop = textarea.scrollHeight;
    return false;
}

function updateScoreboard(guesserColor, cardColor) {
    if (cardColor == "assassin") {
        const teamName = capitalize(guesserColor);
        alert(`${teamName} Team uncovers the Assassin. ${teamName} Team loses!`)
        disableAllCardEvents();
        return false;
    }
    if (cardColor === "red" || cardColor === "blue") {
        const score = document.getElementById(`${cardColor}score`);
        score.innerText -= 1;
        if (score.innerText == 0) {
            const teamName = capitalize(cardColor);
            alert(`${teamName} Team wins!`)
            disableAllCardEvents();
        }
    }
    return false;
}

function disableCardEvents(word) {
    for (var i = 0; i < numCards; i++) {
        const card = document.getElementById(`card-${i}`)
        if (card.innerText === word) {
            card.removeEventListener("click", this.makeGuess);
            return false;
        }
    }
    return false;
}

function disableAllCardEvents() {
    for (var i = 0; i < numCards; i++) {
        document.getElementById(`card-${i}`).removeEventListener("click", this.makeGuess);
    }
}

function markGuessedCard(guessWord, cardColor) {
    for (var i = 0; i < numCards; i++) {
        const card = document.getElementById(`card-${i}`)
        if (card.innerText === guessWord) {
            if (userRole === defaultRole) {
                card.className = `card ${cardColor}`;
                card.style.textDecoration = "line-through";
                card.removeEventListener("click", this.makeGuess);
            } else {
                card.className = "card guessed";
            }
            break;
        }
    }
    return false;
}

function routeEvent(event) {
    if (event.type === undefined) {
        alert("no type field in the event");
    }

    switch(event.type) {
        case "new_message":
            appendChatMessage(event.payload);
            break;
        case "new_game":
            setupBoard(event.payload);
            break;
        case "guess_event":
            guessResponseHandler(event.payload);
            break;
        case "change_room":
            notifyRoomEntry(event.payload);
            break;
        default:
            alert("unsupported message type: " + event.type);
            break;
    }
}

function htmlEscape(str) {
    return str.replace(/&/g, '&amp;')
              .replace(/>/g, '&gt;')
              .replace(/</g, '&lt;')
              .replace(/"/g, '&quot;')
              .replace(/'/g, '&#39;')
              .replace(/`/g, '&#96;');
}

function appendChatMessage(payload) {
    const messageEvent = Object.assign(new NewMessageEvent, payload);
    var date = new Date(messageEvent.sentDate);
    const senderName = messageEvent.from;
    const senderColor = messageEvent.color;
    const formattedMsg = `${fmtTimeFromDate(date)} <span style="font-weight:bold; color:${senderColor}">${senderName}</span>: ${htmlEscape(messageEvent.message)}<br>`;
    textarea = document.getElementById("chatmessages");
    textarea.innerHTML += formattedMsg;
    textarea.scrollTop = textarea.scrollHeight;
}

function sendEvent(eventName, payload) {
    const event = new Event(eventName, payload);
    conn.send(JSON.stringify(event));
}

function sendMessage() {
    var newmessage = document.getElementById("message");
    if (newmessage != null) {
        let outgoingEvent = new SendMessageEvent(newmessage.value, username, usercolor);
        sendEvent("send_message", outgoingEvent);
        newmessage.value = "";
    }
    return false;
}

function login() {
    let formData = {
        "username": document.getElementById("username").value,
        "password": document.getElementById("password").value,
    }

    fetch("login", {
        method: 'post',
        body: JSON.stringify(formData),
        mode: 'cors'
    }).then((response) => {
        if (response.ok) {
            return response.json();
        } else {
            throw 'unauthorized';
        }
    }).then((data) => {
        // check that username is not taken and we were issued an OTP
        if (data.otp === "") {
            // expect the backend to return the error message
            document.getElementById("welcome-header").innerHTML = data.message
            return false
        }
        // user is authenticated
        connectWebsocket(data.otp);
        username = formData.username;
        const loginForm = document.getElementById("login-form");
        loginForm.reset();
        const submitButton = loginForm.querySelector("input[type=submit]");
        submitButton.disabled = true;
        document.getElementById("login-div").style.display = "none";
        document.getElementById("welcome-header").innerHTML = "Welcome, " + username;
        document.getElementById("chat-header").innerHTML = "Currently in chatroom: " + selectedChat;
        document.getElementById("team").disabled = false;
        document.getElementById("role").disabled = false;
    }).catch((e) => { alert(e) });

    return false;
}

function connectWebsocket(otp) {
    if (window["WebSocket"]) {
        console.log("supports websockets");
        // connect to ws
        conn = new WebSocket("wss://" + document.location.host + "/ws?otp=" + otp);

        conn.onopen = function (evt) {
            document.getElementById("onconnect").hidden = false;
        }
        conn.onclose = function (evt) {
            document.getElementById("connection-header").innerHTML = "Disconnected";
            // handle automatic reconnection
        }


        conn.onmessage = function(evt) {
            const eventData = JSON.parse(evt.data);

            const event = Object.assign(new Event, eventData);

            routeEvent(event);
        }
    } else {
        alert("Client does not support websockets");
    }
}

window.onload = function() {
    document.getElementById("chatroom-selection").onsubmit = changeChatRoom;
    document.getElementById("chatroom-message").onsubmit = sendMessage;
    document.getElementById("login-form").onsubmit = login;
    document.getElementById("newgame-button").onclick = requestNewGame;
    document.getElementById("abort-button").onclick = abortGame;
}
