"use strict";

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
    constructor(name, roomName, participants, gameInProgress) {
        this.name = name;
        this.roomName = roomName;
        this.participants = participants;
        this.gameInProgress = gameInProgress;
    }
}

class NewGameRequestEvent {
    constructor(bots) {
        this.bots = bots;
    }
}

class NewGameResponseEvent {
    constructor(cards, teamTurn) {
        this.cards = cards;
        this.teamTurn = teamTurn;
    }
}

class AbortGameEvent {
    constructor(name, color) {
        this.name = name;
        this.teamColor = color; 
    }
}

class GiveClueEvent {
    constructor(clue, numCards) {
        this.clue = clue;
        this.numCards = numCards;
        this.from = userName;
        this.teamColor = userTeam;
    }
}

class GuessEvent {
    constructor(guess, numCards) {
        this.guesser = userName;
        this.guess = guess;
        this.numCards = numCards;
    }
}

class GuessResponseEvent {
    constructor(guess, numCards, teamColor, cardColor,
                correct, teamTurn, roleTurn) {
        this.guesser = userName;
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
const colors = ["#ff0000", "#ff8c00", "#0000ff", "#005a9c", "#00ff00",
                "#964b00", "#800080", "#ff69b4", "#000000", "#808080"];
const defaultRoom = "lobby";
const guesserRole = "guesser";
const cluegiverRole = "cluegiver";
const defaultTeam = "red";
const defaultRole = guesserRole;
const deathCard = "black";
const botWaitMsg = "Waiting for ChatBot...";

let conn;  // websocket connection
let userName;
let userColor = colors[Math.floor(Math.random() * colors.length)];
let userTeam = defaultTeam;
let userRole = guesserRole;
let selectedChat = "";
let currentGame = null;
let teamTurn;
let roleTurn;


const gameBoard = document.getElementById("gameboard");
for (let i = 0; i < totalNumCards; i++) {
    const cardItem = document.createElement("div");
    cardItem.className = "card";
    cardItem.id = `card-${i}`;
    gameBoard.appendChild(cardItem);
}
document.getElementById("gameboard-container").display = "none";

function changeUserColor(event) {
    userColor = event.target.value;
}

function disableBotCheckboxes(boolean) {
    let bots = document.getElementById("bots");
    let boxes = bots.querySelectorAll('input[type="checkbox"]');
    for (let i = 0; i < boxes.length; i++) {
        boxes[i].disabled = boolean;
        if (boolean === false) {
            boxes[i].checked = false;
        }
    }
}

function abortGame() {
    currentGame = null;

    sendEvent("abort_game", null);

    resetCards();
    resetClueNotification();
    resetScoreboard();

    document.getElementById("cluebox").hidden = true;
    document.getElementById("abort-button").hidden = true;
    document.getElementById("newgame-button").hidden = false;
    document.getElementById("sort-cards").disabled = true;

    disableBotCheckboxes(false);

    const team = document.getElementById("team");
    team.value = defaultTeam;
    if (userTeam !== defaultTeam) {
        changeTeam();
    }
    team.disabled = false;
    
    const role = document.getElementById("role");
    role.value = guesserRole;
    if (userRole !== guesserRole) {
        changeRole();
    }
    role.disabled = false;
}

function setupBoard(payload) {
    // Set global variable
    currentGame = Object.assign(new NewGameResponseEvent, payload);
    document.getElementById("gameboard-container").hidden = false;

    let i = 0;
    for (const [word, color] of Object.entries(currentGame.cards)) {
        setupCard(i, word, color);
        i += 1;
    }

    resetScoreboard();

    document.getElementById("clue").innerHTML = "";
    if (userRole === cluegiverRole) {
        document.getElementById("cluebox").hidden = false;
    } else {
        document.getElementById("cluebox").hidden = true;
    }

    document.getElementById("sort-cards").value = "alphabetical";
    document.getElementById("sort-cards").disabled = false;
    document.getElementById("role").disabled = true;
    document.getElementById("team").disabled = true;
    document.getElementById("game-setup").hidden = true;
    document.getElementById("newgame-button").hidden = true;
    document.getElementById("abort-button").hidden = false;

    disableBotCheckboxes(true);

    teamTurn = currentGame.teamTurn;
    roleTurn = cluegiverRole;
    whoseTurn(teamTurn, roleTurn);
}

function sortCards(how) {
    if (typeof how === "object") {
        // Assume it's an event.
        how = how.target.value;
    }  // Now, "how" is a string.
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
            let align = new Object();
            for (const [word, color] of Object.entries(currentGame.cards)) {
                if (color in align) {
                    align[color].push(word);
                } else {
                    align[color] = [word];
                }
            }
            const colorOrder = ["white", "red", "blue", deathCard, "neutral",
                                "guessed", "guessed red", "guessed blue", "guessed neutral"];
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
    if (userRole === guesserRole) {
        if (color.includes("guessed")) {
            card.removeEventListener("click", makeGuess, false);
        } else {
            card.addEventListener("click", makeGuess, false);
        }
    }
}

function resetCards() {
    for (let i = 0; i < totalNumCards; i++) {
        const card = document.getElementById(`card-${i}`)
        card.className = "card";
        card.innerText = "";
        card.removeEventListener("click", makeGuess, false);
    }
}

function resetClueNotification() {
    document.getElementById("turn").innerHTML = "";
    document.getElementById("clue").innerHTML = "";
    document.getElementById("numguess").innerText = "";
    document.getElementById("number-input").value = 2;
}

function resetScoreboard() {
    document.getElementById("redscore").innerText = 9;
    document.getElementById("bluescore").innerText = 8;
}

function makeGuess() {
    sendEvent("guess_event", new GuessEvent(this.innerText));
    return false;
}

function requestNewGame() {
    const game = new NewGameRequestEvent({
        "cluegiver": {
            "red":  document.getElementById("AIRedClue").checked,
            "blue": document.getElementById("AIBlueClue").checked,
        },
        "guesser": {
            "red":  document.getElementById("AIRedGuess").checked,
            "blue": document.getElementById("AIBlueGuess").checked,
        },
    });
    sendEvent("new_game", game);
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

function appendParticipantDiv(participant) {
    const name = participant.name;
    const container = document.getElementById("participants");
    const child = document.createElement("div");
    child.className = "participant";
    child.id = `participant-${name}`;
    child.setAttribute("data-testid", `participant-${name}`);
    container.appendChild(child);
    updateParticipant(participant);
}

function addParticipant(name) {
    appendParticipantDiv({name: name, teamColor: defaultTeam, role: defaultRole});

    const container = document.getElementById("participants");
    Array.from(container.children)
         .sort(function (a, b) {
            if (a.id < b.id) {
                return -1;
            }
            if (a.id > b.id) {
                return 1;
            }
            return 0;
         })
         .forEach(element => container.append(element));
}

function addParticipantsList(participantsList) {
    /* Expect particpantsList to be sorted already. */
    for (let i = 0; i < participantsList.length; i++) {
        appendParticipantDiv(participantsList[i]);
    }
}

function updateParticipant({name, teamColor, role}) {
    const participant = document.getElementById(`participant-${name}`);
    if (selectedChat === defaultRoom) {
        participant.innerHTML = name;
        return;
    }
    participant.innerHTML = `${name} <span style="color:${teamColor}">${teamColor} ${role}</span>`;
}

function removeParticipant(name) {
    const container = document.getElementById("participants");
    const child = document.getElementById(`participant-${name}`);
    if (child != null) {
        container.removeChild(child);
    }
}

function removeAllParticipants() {
    const container = document.getElementById("participants");
    container.innerHTML = "";
}

function changeChatRoom() {
    const newchat = document.getElementById("chatroom");
    if (!goToRoom(newchat.value)) {
        // failed to change chat room
        newchat.value = selectedChat;
        return false;
    }
    removeAllParticipants();
    if (currentGame !== null) {
        abortGame();
    }
    return false;
}

function goToRoom(room) {
    const whitespace = new RegExp(/^\s*$/);
    if (typeof room !== "undefined" && !whitespace.test(room) && room !== selectedChat) {
        selectedChat = room;
        let changeEvent = new ChangeChatRoomEvent(userName, selectedChat);
        sendEvent("enter_room", changeEvent);
        return true;
    }
    return false;
}

function notifyRoomEntry(payload) {
    let roomChange = Object.assign(new ChangeChatRoomEvent, payload);

    if (roomChange.participants != null && roomChange.participants.length > 0) {
        addParticipantsList(roomChange.participants);
    } else {
        addParticipant(roomChange.name);
    }

    let message = `${roomChange.name} has entered `;
    if (userName === roomChange.name) {
        const welcome = document.getElementById("welcome-header");

        if (roomChange.roomName === defaultRoom) {
            document.getElementById("game-setup").hidden = true;
            welcome.innerText = `Welcome to the lobby, ${userName}. Go to any chat room to play a game.`;
            message += `the lobby.`;
        } else {
            document.getElementById("game-setup").hidden = false;
            welcome.innerText = `Welcome to ${selectedChat}, ${userName}.`;
            message += `room "${roomChange.roomName}".`;
        }

        document.getElementById("participants-title").innerText = `Participants in ${selectedChat}`;
        if (roomChange.gameInProgress) {
            // TOOD: consider adding a "join game" option
            document.getElementById("team").disabled = true;
            document.getElementById("role").disabled = true;
            disableBotCheckboxes(true);
            document.getElementById("newgame-button").disabled = true;
            document.getElementById("newgame-button").hidden = true;
            appendToChat(`** Game in progress **`);
        }
    } else {
        message += `the room.`;
    }
    appendToChat(message);
}

function notifyRoomExit(payload) {
    const roomChange = Object.assign(new ChangeChatRoomEvent, payload);
    removeParticipant(roomChange.name);
    let message = `${roomChange.name} has left the room.`;
    appendToChat(message);
}

function abortGameHandler(payload) {
    const {name, teamColor} = Object.assign(new AbortGameEvent, payload);
    const message = `<span style="color:${teamColor}">${name} has left the game.</span>`;
    appendToChat(message);
    if (selectedChat !== defaultRoom) {
        document.getElementById("game-setup").hidden = false;
        document.getElementById("gameboard-container").hidden = true;
    }
}

function guessResponseHandler(payload) {
    const guessResponse = Object.assign(new GuessResponseEvent, payload);
    markGuessedCard(guessResponse);
    notifyChatRoom(guessResponse);
    updateScoreboard(guessResponse);
    notifyGuessRemaining(guessResponse);

    if (document.getElementById("sort-cards").value === "keep-sorted") {
        sortCards("color");
    }

    const {teamTurn, roleTurn} = guessResponse;
    whoseTurn(teamTurn, roleTurn);
}

function whoseTurn(teamTurn, roleTurn) {
    document.getElementById("turn").innerHTML = `${capitalize(teamTurn)}<br>${capitalize(roleTurn)}`;
    document.getElementById("turn").style.color = teamTurn;
    document.getElementById("end-turn").style.visibility = "hidden";
    setMaxGuessLimit(teamTurn);
    if (userTeam !== teamTurn) {
        disableAllCardEvents();
        document.getElementById("clue-input").disabled = true;
        document.getElementById("cluebox").querySelector("input[type=submit]").disabled = true;
        return;
    }
    if (roleTurn === cluegiverRole) {
        // clear the previous clue from the clue box
        document.getElementById("clue").innerText = "";
        document.getElementById("clue-input").disabled = false;
        document.getElementById("cluebox").querySelector("input[type=submit]").disabled = false;
        return;
    }
    if (userRole === guesserRole) {
        enableCardEvents();
        document.getElementById("end-turn").style.visibility = "visible";
    }
}

function notifyGuessRemaining({guessRemaining}) {
    const remaining = document.getElementById("numguess");
    if (guessRemaining === 0) {
        remaining.innerText = "";
    } else if (guessRemaining < totalNumCards) {
        remaining.innerText = guessRemaining;
    }
}

function endTurn() {
    sendEvent("end_turn", null);
}

function endTurnHandler(payload) {
    const {teamTurn, roleTurn} = Object.assign(new EndTurnEvent, payload);
    const cluebox = document.getElementById("clue");
    if (cluebox.innerText === botWaitMsg) {
        cluebox.innerText = "";
    }
    document.getElementById("numguess").innerText = "";
    whoseTurn(teamTurn, roleTurn);
}

function capitalize(word) {
    return word.charAt(0).toUpperCase() + word.substring(1);
}

function notifyChatRoom({guess, guesser, teamColor, cardColor}) {
    let msg = `<span style="color:${teamColor}">${guesser} uncovers ${guess}:</span> `;
    if (teamColor === cardColor) {
        msg += `CORRECT.`;
    } else {
        msg += `Incorrect. Card is ${cardColor}.`;
    }
    appendToChat(msg);
}

function notifyBotWait() {
    if (roleTurn === cluegiverRole && (
            (teamTurn === "red" && document.getElementById("AIRedClue").checked) ||
            (teamTurn === "blue" && document.getElementById("AIBlueClue").checked)
        )) {
        document.getElementById("clue").innerText = botWaitMsg;
    }
}

function updateScoreboard({score}) {
    for (const color of ["red", "blue"]) {
        const loc = document.getElementById(`${color}score`);
        loc.innerText = score[color];
    }
}

function setMaxGuessLimit(teamTurn) {
    const loc = document.getElementById(`${teamTurn}score`);
    const val = parseInt(loc.innerText);
    document.getElementById("number-input").setAttribute("max", val);
}

function disableCardEvents(word) {
    for (let i = 0; i < totalNumCards; i++) {
        const card = document.getElementById(`card-${i}`);
        if (card.innerText === word) {
            card.removeEventListener("click", makeGuess, false);
            return false;
        }
    }
    return false;
}

function disableAllCardEvents() {
    for (let i = 0; i < totalNumCards; i++) {
        document.getElementById(`card-${i}`).removeEventListener("click", makeGuess, false);
    }
}

function enableCardEvents() {
    for (let i = 0; i < totalNumCards; i++) {
        const card = document.getElementById(`card-${i}`);
        if (!card.className.includes("guessed")) {
            card.addEventListener("click", makeGuess, false);
        }
    }
    return false;
}

function markGuessedCard({guess, cardColor}) {
    currentGame.cards[guess] = `guessed ${cardColor}`;

    for (let i = 0; i < totalNumCards; i++) {
        const card = document.getElementById(`card-${i}`);
        if (card.innerText === guess) {
            card.className = `card ${cardColor} guessed`;
            if (userRole === guesserRole) {
                card.removeEventListener("click", makeGuess, false);
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
        case "update_participant":
            updateParticipant(event.payload);
            break;
        case "guess_event":
            guessResponseHandler(event.payload);
            break;
        case "enter_room":
            notifyRoomEntry(event.payload);
            break;
        case "exit_room":
            notifyRoomExit(event.payload);
            break;
        case "give_clue":
            clueHandler(event.payload);
            break;
        case "end_turn":
            endTurnHandler(event.payload);
            break;
        case "abort_game":
            abortGameHandler(event.payload);
            break;
        case "bot_wait":
            notifyBotWait();
            break;
        case "game_over":
            gameOverHandler(event.payload);
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
    const formattedMsg = `${time} <span style="font-weight:bold;color:${color}">${from}</span>: ${msg}<br>`;
    const textarea = document.getElementById("chatlog");
    textarea.innerHTML += formattedMsg;
    textarea.scrollTop = textarea.scrollHeight;
}

function appendToChat(message) {
    const textarea = document.getElementById("chatlog");
    textarea.innerHTML += `<span style="font-weight:bold;">${message}</span><br>`;
    textarea.scrollTop = textarea.scrollHeight;
}

function sendEvent(eventName, payload) {
    const event = new Event(eventName, payload);
    conn.send(JSON.stringify(event));
}

function sendMessage() {
    let newmessage = document.getElementById("message");
    if (newmessage != null) {
        let outgoingEvent = new SendMessageEvent(newmessage.value, userName, userColor);
        sendEvent("send_message", outgoingEvent);
        newmessage.value = "";
    }
    return false;
}

function giveClue() {
    const clue = document.getElementById("clue-input");
    const numCards = document.getElementById("number-input").value;
    const whitespace = new RegExp(/^\s*$/);
    if (!(clue.value === null || whitespace.test(clue.value))) {
        let outgoingEvent = new GiveClueEvent(clue.value, numCards);
        sendEvent("give_clue", outgoingEvent);
        clue.disabled = true;
        document.getElementById("cluebox").querySelector("input[type=submit]").disabled = true;
    }
    clue.value = "";
    return false;
}

function clueHandler(payload) {
    const clueEvent = Object.assign(new GiveClueEvent, payload);
    const {teamColor, clue, numCards} = clueEvent;
    const numguess = document.getElementById("numguess");

    let msg = clue;
    if (numCards > 0) {
        msg += `<br>(applies to ${numCards} card${numCards > 1 ? 's' : ''})`;
        numguess.innerText = `${+numCards + 1}`;
    } else {
        numguess.innerText = `\u221E`;  /* infinity */
    }

    document.getElementById("clue").innerHTML = msg;

    whoseTurn(teamColor, guesserRole);
}

function login() {
    let formData = {
        "username": document.getElementById("username").value,
    }
    /* Do not allow whitespace in username. This would break participants list,
       etc., where the username becomes part of a CSS identifier. */
    const whitespace = new RegExp(/\s+/);
    if (typeof formData.username === 'undefined' || formData.username === "") {
        alert("Username may not be empty");
        return;
    }
    if (whitespace.test(formData.username)) {
        alert("Username may not contain whitespace");
        return;
    }
    let room = document.getElementById("gotoroom").value;
    if (room === "") {
        room = defaultRoom;
    }

    fetch("login", {
        method: 'post',
        body: JSON.stringify(formData),
        mode: 'cors'
    }).then((response) => {
        if (response.ok) {
            return response.json();
        } else {
            return response.text().then(
                (text) => {throw new Error(text)}
            );
        }
    }).then((data) => {
        // check that username is not taken and we were issued an OTP
        if (data.otp === "") {
            // expect the backend to return the error message
            document.getElementById("welcome-header").innerHTML = data.message
            return false;
        }
        // user is authenticated
        userName = formData.username;
        connectWebsocket(data.otp, room);

        // clear and hide the login form
        const loginForm = document.getElementById("login-form");
        loginForm.reset();
        loginForm.querySelector("input[type=submit]").disabled = true;
        document.getElementById("login-div").style.display = "none";
    }).catch((error) => { alert(error) });

    return false;
}

function connectWebsocket(otp, room) {
    if (window["WebSocket"]) {
        console.log("supports websockets");
        // connect to ws
        conn = new WebSocket("wss://" + document.location.host + "/ws?otp=" + otp);

        conn.onopen = function (evt) {
            document.getElementById("onconnect").hidden = false;
            const whitespace = new RegExp(/^\s*$/);
            if (!whitespace.test(room)) {
                selectedChat = room;
                let changeEvent = new ChangeChatRoomEvent(userName, selectedChat);
                sendEvent("enter_room", changeEvent);
            }
            return false;
        }
        conn.onclose = function (evt) {
            document.getElementById("welcome-header").innerHTML = "Disconnected";
            // TODO: handle automatic reconnection
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

function gameOverHandler(message) {
    if (message !== null && message !== "") {
        alert(message);
    }
    document.getElementById("end-turn").style.visibility = "hidden";
    const turnElement = document.getElementById("turn");
    turnElement.innerHTML = "Game Over";
    turnElement.style.color = "black";
    if (currentGame === null) {
        /* A non-player client in the chat room, waiting for the game to end. */
        document.getElementById("team").disabled = false;
        document.getElementById("role").disabled = false;
        disableBotCheckboxes(false);
        document.getElementById("newgame-button").disabled = false;
        document.getElementById("newgame-button").hidden = false;
    }
    appendToChat("** Game Over **");
}

window.onload = function() {
    const colorpicker = document.getElementById("colorpicker");
    colorpicker.value = userColor;
    colorpicker.addEventListener("input", changeUserColor, false);
    document.getElementById("chatroom-selection").onsubmit = changeChatRoom;
    document.getElementById("send-message").onsubmit = sendMessage;
    document.getElementById("login-form").onsubmit = login;
    document.getElementById("newgame-button").onclick = requestNewGame;
    document.getElementById("abort-button").onclick = abortGame;
    document.getElementById("cluebox").onsubmit = giveClue;
    document.getElementById("end-turn").onclick = endTurn;
    document.getElementById("sort-cards").addEventListener("change", sortCards, false);
    document.getElementById("role").addEventListener("change", changeRole, false);
    document.getElementById("team").addEventListener("change", changeTeam, false);
}