const chatID = "123456"; // В реальном проекте будет подставляться динамически
const socket = new WebSocket(`ws://localhost:8080/ws?chat_id=${chatID}`);

const chatBox = document.getElementById("chatBox");
const messageInput = document.getElementById("messageInput");
const sendBtn = document.getElementById("sendBtn");
const statusIndicator = document.getElementById("status");

// Обновление статуса WebSocket
socket.onopen = () => {
    statusIndicator.innerHTML = "🟢 Онлайн";
};

socket.onclose = () => {
    statusIndicator.innerHTML = "🔴 Офлайн";
};

// Получение сообщений
socket.onmessage = (event) => {
    const message = JSON.parse(event.data);
    displayMessage(message.sender, message.content);
};

// Отправка сообщений
sendBtn.addEventListener("click", sendMessage);
messageInput.addEventListener("keypress", (e) => {
    if (e.key === "Enter") sendMessage();
});

function sendMessage() {
    const content = messageInput.value.trim();
    if (!content) return;

    const messageData = {
        chat_id: chatID,
        sender: "user", // Или "admin", если это администратор
        content: content
    };

    socket.send(JSON.stringify(messageData));
    displayMessage("user", content);
    messageInput.value = "";
}

// Отображение сообщений в чате
function displayMessage(sender, content) {
    const messageDiv = document.createElement("div");
    messageDiv.classList.add("message", sender);
    messageDiv.textContent = content;

    chatBox.appendChild(messageDiv);
    chatBox.scrollTop = chatBox.scrollHeight; // Автопрокрутка вниз
}
