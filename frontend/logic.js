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
    constructor(message, from, color, sentTime) {
        this.message = message;
        this.from = from;
        this.color = color;
        this.sentTime = sentTime;
    }
}

class ChangeChatRoomEvent {
    constructor(username, roomname) {
        this.username = username;
        this.roomname = roomname;
    }
}

class NewGameEvent {
    constructor(cards, sentTime) {
        this.sentTime = sentTime;
        this.cards = cards;
    }
}

class AbortGameEvent {
    constructor(name, color) {
        this.username = name;
        this.teamColor = color; 
    }
}

class GiveClueEvent {
    constructor(clue, numCards) {
        this.clue = clue;
        this.numCards = numCards;
        this.from = username;
        this.teamColor = userTeam;
    }
}

class GuessEvent {
    constructor(guess, numCards) {
        this.guesser = username;
        this.guess = guess;
        this.numCards = numCards;
    }
}

class GuessResponseEvent {
    constructor(guess, numCards, teamColor, cardColor,
                correct, teamTurn, roleTurn) {
        this.guesser = username;
        this.guess = guess;
        this.numCards = numCards;
        this.teamColor = teamColor;
        this.cardColor = cardColor;
        this.correct = correct;
        this.teamTurn = teamTurn;
        this.roleTurn = roleTurn;
    }
}

class EndTurnEvent {
    constructor(teamTurn, roleTurn) {
        this.teamTurn = teamTurn;
        this.roleTurn = roleTurn;
    }
}

const totalNumCards = 25;
const colors = ["red", "darkorange", "blue", "dodgerblue", "green",
                "brown", "purple", "hotpink", "black", "gray"];
const defaultRoom = "General";
const defaultRole = "guesser";
const defaultTeam = "red";
const deathCard = "black";

var selectedChat = defaultRoom;
var username;
var usercolor = colors[Math.floor(Math.random() * colors.length)];
var userTeam = defaultTeam;
var userRole = defaultRole;
var currentGame;


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
for (let i = 0; i < totalNumCards; i++) {
    const cardItem = document.createElement("div");
    cardItem.className = "card";
    cardItem.id = `card-${i}`;
    gameBoard.appendChild(cardItem);
}

function abortGame() {
    currentGame = null;

    sendEvent("abort_game", null);

    resetCards();
    resetClueNotification();

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
    for (const [word, color] of Object.entries(currentGame.cards)) {
        setupCard(i, word, color);
        i += 1;
    }

    setupScoreboard();

    document.getElementById("clueheader").innerHTML = "";
    document.getElementById("cluebox").hidden = true;
    if (userRole !== defaultRole) {
        document.getElementById("cluebox").hidden = false;
    }

    document.getElementById("sort-cards").value = "alphabetical";
    document.getElementById("sort-cards").disabled = false;
    document.getElementById("role").disabled = true;
    document.getElementById("team").disabled = true;
    document.getElementById("newgame-button").hidden = true;
    document.getElementById("abort-button").hidden = false;

    whoseTurn(defaultTeam, "cluegiver");
}

function sortCards(how) {
    let i = 0;
    switch (how) {
        case "alphabetical":
            for (const [word, color] of Object.entries(currentGame.cards).sort()) {
                setupCard(i, word, color);
                i++;
            }
            break;

        case "keep-sorted":
        case "color":
            var align = new Object();
            for (const [word, color] of Object.entries(currentGame.cards)) {
                if (color in align) {
                    align[color].push(word);
                } else {
                    align[color] = [word];
                }
            }
            const colorOrder = ["white", "red", "blue", deathCard, "neutral",
                                "guessed", "guessed-red", "guessed-blue", "guessed-neutral"];
            colorOrder.forEach(function (color) {
                if (align.hasOwnProperty(color)) {
                    align[color].forEach(function (word) {
                        setupCard(i, word, color);
                        i++;
                    })
                }
            });
            break;
    }
}

function setupCard(cardNum, word, color) {
    const card = document.getElementById(`card-${cardNum}`);
    card.innerText = word;
    card.className = `card ${color}`;
    if (userRole === defaultRole) {
        if (color.includes("guessed")) {
            card.removeEventListener("click", this.makeGuess, false);
        } else {
            card.addEventListener("click", this.makeGuess, false);
        }
    }
}

function resetCards() {
    for (let i = 0; i < totalNumCards; i++) {
        const card = document.getElementById(`card-${i}`)
        card.className = "card";
        card.innerText = "";
        card.removeEventListener("click", this.makeGuess, false);
    }
}

function resetClueNotification() {
    document.getElementById("clueheader").innerHTML = "";
    document.getElementById("guesses-remaining").innerText = "";
    document.getElementById("number-input").value = 2;
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
    const message = `<br><span style="font-weight:bold;">${roomChange.username} has entered the room.</span><br>`;
    appendToChat(message);
}

function notifyAbortGame(payload) {
    const {teamColor, username} = Object.assign(new AbortGameEvent, payload);
    message = `<br><span style="font-weight:bold;color:${teamColor}">${username} has left the game.</span><br>`;
    appendToChat(message);
}

function guessResponseHandler(payload) {
    guessResponse = Object.assign(new GuessResponseEvent, payload);

    markGuessedCard(guessResponse);
    notifyChatroom(guessResponse);
    updateScoreboard(guessResponse);
    notifyGuessRemaining(guessResponse);

    if (document.getElementById("sort-cards").value === "keep-sorted") {
        sortCards("color");
    }

    const {teamTurn, roleTurn} = guessResponse;
    whoseTurn(teamTurn, roleTurn);
}

function whoseTurn(teamTurn, roleTurn) {
    if (userTeam !== teamTurn) {
        disableAllCardEvents();
        document.getElementById("clue-input").disabled = true;
        document.getElementById("cluebox").querySelector("input[type=submit]").disabled = true;
        document.getElementById("end-turn").hidden = true;
        return;
    }
    if (roleTurn !== defaultRole) {
        // cluegiver turn
        document.getElementById("clue-input").disabled = false;
        document.getElementById("cluebox").querySelector("input[type=submit]").disabled = false;
    }
    if (userRole === defaultRole) {
        enableCardEvents();
        document.getElementById("end-turn").hidden = false;
    }
    return;
}

function notifyGuessRemaining({guessRemaining}) {
    if (guessRemaining < totalNumCards) {
        document.getElementById("guesses-remaining").innerText = `Guesses Remaining: ${guessResponse.guessRemaining}`;
    }
}

function endTurn() {
    sendEvent("end_turn", null);
    return false;
}

function endTurnHandler(payload) {
    const {teamTurn, roleTurn} = Object.assign(new EndTurnEvent, payload);
    whoseTurn(teamTurn, roleTurn);
    return false;
}

function capitalize(word) {
    return word.charAt(0).toUpperCase() + word.substring(1);
}

function notifyChatroom({guess, guesser, teamColor, cardColor}) {
    const teamName = capitalize(teamColor);
    let msg = `<br><span style="font-weight:bold; color:${teamColor}">${guesser} uncovers ${guess}: `;
    if (teamColor === cardColor) {
        msg += `${msg} CORRECT. A point for ${teamName}.</span><br>`;
    } else {
        msg += `${msg} incorrect. Card is ${cardColor}.</span><br>`;
    }
    appendToChat(msg);
    return false;
}

function updateScoreboard({teamColor, cardColor}) {
    const teamName = capitalize(teamColor);

    if (cardColor == deathCard) {
        alert(`${teamName} Team uncovers the Black Card. ${teamName} Team loses!`)
        disableAllCardEvents();
        return false;
    }

    if (cardColor === "red" || cardColor === "blue") {
        const score = document.getElementById(`${cardColor}score`);
        score.innerText -= 1;
        if (score.innerText == 0) {
            alert(`${teamName} Team wins!`)
            disableAllCardEvents();
        }
    }
    return false;
}

function disableCardEvents(word) {
    for (var i = 0; i < totalNumCards; i++) {
        const card = document.getElementById(`card-${i}`);
        if (card.innerText === word) {
            card.removeEventListener("click", this.makeGuess, false);
            return false;
        }
    }
    return false;
}

function disableAllCardEvents() {
    for (var i = 0; i < totalNumCards; i++) {
        document.getElementById(`card-${i}`).removeEventListener("click", this.makeGuess, false);
    }
}

function enableCardEvents() {
    for (var i = 0; i < totalNumCards; i++) {
        const card = document.getElementById(`card-${i}`);
        if (!card.className.includes("guessed")) {
            card.addEventListener("click", this.makeGuess, false);
        }
    }
    return false;
}

function markGuessedCard({guess, cardColor}) {
    currentGame.cards[guess] = `guessed-${cardColor}`;

    for (var i = 0; i < totalNumCards; i++) {
        const card = document.getElementById(`card-${i}`);
        if (card.innerText === guess) {
            card.className = `card guessed-${cardColor}`;
            if (userRole === defaultRole) {
                card.removeEventListener("click", this.makeGuess, false);
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
        case "give_clue":
            clueHandler(event.payload);
            break;
        case "end_turn":
            endTurnHandler(event.payload);
            break;
        case "abort_game":
            notifyAbortGame(event.payload);
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
    const time = fmtTimeFromDate(new Date(messageEvent.sentTime));
    const {from, color} = messageEvent;
    const msg = htmlEscape(messageEvent.message);
    const formattedMsg = `${time} <span style="font-weight:bold; color:${color}">${from}</span>: ${msg}<br>`;
    appendToChat(formattedMsg);
}

function appendToChat(message) {
    const textarea = document.getElementById("chatmessages");
    textarea.innerHTML += message;
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

function giveClue() {
    const clue = document.getElementById("clue-input");
    const numCards = document.getElementById("number-input").value;
    if (clue != null) {
        let outgoingEvent = new GiveClueEvent(clue.value, numCards);
        sendEvent("give_clue", outgoingEvent);
        clue.value = "";
    }
    clue.disabled = true;
    document.getElementById("cluebox").querySelector("input[type=submit]").disabled = true;
    return false;
}

function clueHandler(payload) {
    const clueEvent = Object.assign(new GiveClueEvent, payload);
    const {from, teamColor, clue, numCards} = clueEvent;
    const teamName = capitalize(teamColor);

    var msg = `${from} gives clue for <span style="color:${teamColor};">${teamName}</span>: ${clue}<br>`;
    if (numCards > 0) {
        msg += `Applies to ${numCards} cards.`
        document.getElementById("guesses-remaining").innerText = `Guesses Remaining: ${+numCards + 1}`;
    } else {
        msg += `${from} did not specify the number of cards. ${teamName} has unlimited guesses.`
    }

    document.getElementById("clueheader").innerHTML = msg;

    whoseTurn(teamColor, defaultRole);
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
    document.getElementById("cluebox").onsubmit = giveClue;
    document.getElementById("end-turn").onclick = endTurn;
}
