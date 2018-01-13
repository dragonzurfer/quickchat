$(document).ready(function() {

    // adjust the height of textarea using autoresize library
    autosize($('textarea'));

    var ChatID = $("#chatID").text(),
        key = "",
        ENCRYPTED_KEY = "",
        ENCRYPTED_USERKEY = "",
        messageArea = $("#msg"),
        user = "",
        userkey = "",
        socket,
        $container = $(".container"),
        $page = $('html,body'),
        amountOfColors = 18,
        csrf = document.getElementById("csrf").value ,
        month = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];


    // Get key for User and Chat
    var keyquery = "Chat key"
    while (!key) {
        key = prompt("Chat key");
    }
    ENCRYPTED_KEY = CryptoJS.SHA512(chatName + key.trim());

    var usernamequery = "Your username", userkeyquery = "User Passcode";
    while (!userkey && !user) {
        user = prompt(usernamequery);
        userkey = prompt(userkeyquery);
    }
    ENCRYPTED_USERKEY = CryptoJS.SHA512(chatName + userkey.trim());

    // utility function to show pop-up notification
    function showNotif(notif, color) {
        $(".notification").text(notif).addClass("show").css("background-color", color);
        setTimeout(function() {
            $(".notification").removeClass("show");
        }, 4000);
    };

    // utility function to generate random color
    function getColor() {
        let hue = Math.floor(Math.random() * amountOfColors) * (360 / amountOfColors);
        return `hsl(${hue}, 90%, 50%)`;
    }

    // Loads message
    function renderMessage(name, time, msg) {
        let newMessage = `
          <div class="chat-container">
            <div class="avatar" style="background-color: ${getColor()}">
              <span>${name.match(/\b(\w)/g).slice(0, 2).join("")}</span>
            </div>
            <div class="chat-text">
              <div class="chat-datetime">
                <div class="name">${name}</div>
                <div class="time">${time}</div>
              </div>
              <div class="chat-message">${msg.replace(/\n/g, "<br>")}</div>
            </div>
          </div>
        `;
        $container.append(newMessage);
    }

    // Connect to backend
    if (window["WebSocket"]) {
        url = `ws://${document.location.host}/ws/${ChatID}/${user}`;
        url += `?userkey=${encodeURIComponent(ENCRYPTED_USERKEY)}&key=${encodeURIComponent(ENCRYPTED_KEY)}`;
        socket = new WebSocket(url);
        socket.onclose = function(evt) {
            showNotif("Disconnected", "red");
        };

        socket.onopen = function(evt) {
            showNotif("Connected", "green");
            // showNotif("Connected", "red");
            for (let encMsg of EncMessages) {
                let name = encMsg[0];
                let time = encMsg[1];
                let stime = parseInt(time.split(":")[1])
                if (stime < 10) {
                    time = time.split(":")[0] + " : 0" + stime;
                }
                let monthstime = time.split(`"`);
                time = monthstime[0] + monthstime[1] + monthstime[2];
                let msg = CryptoJS.AES.decrypt(encMsg[2], key).toString(CryptoJS.enc.Utf8);
                renderMessage(name, time, msg);
            }
            $page.scrollTop($(document).height() - $(window).height());
        };

        socket.onmessage = function(evt) {
            var res = evt.data.split(":");
            var decrypted = CryptoJS.AES.decrypt(res[1], key).toString(CryptoJS.enc.Utf8);
            let curdate = new Date();
            let strminutes = ""
            let strmonth = month[curdate.getMonth()];
            if (curdate.getMinutes() < 10) {
                strminutes = "0" + curdate.getMinutes();
            } else {
                strminutes = curdate.getMinutes();
            }
            renderMessage(res[0], `${curdate.getDate()}-${strmonth} ${curdate.getHours()}:${strminutes}`, decrypted);
            $page.animate({
                scrollTop: $page.height()
            }, 300);
        };
    } else {
        showNotif("No websocket :(", "red");
    }


    function enterMessage() {
        if (!socket) {
            return;
        }
        let msg = messageArea.val().trim();
        if (msg.trim() == "") {
            return;
        }

        var encrypted = user + ":" + CryptoJS.AES.encrypt(msg, key);
        socket.send(encrypted);
        messageArea.val("").attr("rows", 1);
    }

    messageArea.on("keydown", function(event) {
        let keycode = (event.keyCode ? event.keyCode : event.which);
        if(event.keyCode == 13 && !event.shiftKey) {
            enterMessage();
        }
    });

    $(".btn-send").on("click", function() {
        enterMessage();
    });

});