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
    constructor(name) {
        this.name = name;
    }
}

class NewGameEvent {
    constructor(words, sentTime) {
        this.words = words;
        this.sentTime = sentTime;
    }
}

var selectedChat = "general";
var username = "";
var usercolor;

const numCards = 25;

const colors = ["red", "darkorange", "blue", "dodgerblue", "green",
                "brown", "purple", "hotpink", "black", "gray"];

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

function newGame(gameEvent) {
    let i = 0;
    for (const [word, alignment] of Object.entries(gameEvent.words)) {
        const cardItem = document.getElementById(`card-${i}`);
        cardItem.className = `card ${alignment}`;
        cardItem.innerHTML = `${word}`;
        i += 1;
    }
}

function requestNewGame() {
    sendEvent("new_game", null);
    return false;
}

function changerole() {
    sendEvent("change_role", null);
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

        let changeEvent = new ChangeChatRoomEvent(selectedChat);
        sendEvent("change_room", changeEvent);
        textarea = document.getElementById("chatmessages");
        textarea.innerHTML = `You changed room into: ${selectedChat}`;
        textarea.scrollTop = textarea.scrollHeight;
    }
    // if you don't return false, it will redirect
    return false;
}

function routeEvent(event) {
    if (event.type === undefined) {
        alert("no type field in the event");
    }

    switch(event.type) {
        case "new_message":
            const messageEvent = Object.assign(new NewMessageEvent, event.payload);
            appendChatMessage(messageEvent);
            break;
        case "new_game":
            const gameEvent = Object.assign(new NewGameEvent, event.payload);
            newGame(gameEvent);
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

function appendChatMessage(messageEvent) {
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
            document.getElementById("connection-header").innerHTML = "Connected to Websocket: true";
        }
        conn.onclose = function (evt) {
            document.getElementById("connection-header").innerHTML = "Connected to Websocket: false";
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
}
