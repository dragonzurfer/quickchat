function send() {
    var key = document.getElementById("key");
    var chatName = document.getElementById("chatname").value;
    if(key.value && chatName) {
        var str = chatName + key.value.trim();
        debugger;
        var encrypted = CryptoJS.SHA512(str);
        key.value = encrypted;
        return true;
    }
    return false;
}
function getColor() {
    let hue = Math.floor(Math.random() * 18) * (360 / 18);
    return `hsl(${hue}, 90%, 50%)`;
}
document.getElementById("btn-submit").style.background = getColor();