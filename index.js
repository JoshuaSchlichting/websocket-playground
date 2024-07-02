const canvas = document.getElementById('pongCanvas');
const context = canvas.getContext('2d');
const ws = new WebSocket('ws://localhost:80/ws'); // Connect to the Go server

canvas.width = 800;
canvas.height = 400;

let ball = {
    x: canvas.width / 2,
    y: canvas.height / 2,
    radius: 10,
    velocityX: 5,
    velocityY: 5,
    speed: 7,
    color: "WHITE"
};

let userPaddle = {
    x: 0,
    y: (canvas.height - 100) / 2,
    width: 10,
    height: 100,
    score: 0,
    color: "WHITE"
};

let aiPaddle = {
    x: canvas.width - 10,
    y: (canvas.height - 100) / 2,
    width: 10,
    height: 100,
    score: 0,
    color: "WHITE"
};

// WebSocket message handler
ws.onmessage = function(event) {
    const gameState = JSON.parse(event.data);
    // Update local game state with the server's state
    ball.x = gameState.ball.x;
    ball.y = gameState.ball.y;
    userPaddle.y = gameState.userPaddle.y;
    aiPaddle.y = gameState.aiPaddle.y;
    // Handle scores and other game state updates here
};

function update() {
    // Send the current state to the server for processing
    ws.send(JSON.stringify({ball: ball, userPaddle: userPaddle, aiPaddle: aiPaddle}));
    // The server will handle collision detection and game state updates
    // Local updates related to rendering can still be done here if needed
}

function render() {
    // Clear the canvas
    context.clearRect(0, 0, canvas.width, canvas.height);

    // Render the ball
    context.beginPath();
    context.arc(ball.x, ball.y, ball.radius, 0, Math.PI * 2);
    context.fillStyle = ball.color;
    context.fill();
    context.closePath();

    // Render the user paddle
    context.fillStyle = userPaddle.color;
    context.fillRect(userPaddle.x, userPaddle.y, userPaddle.width, userPaddle.height);

    // Render the AI paddle
    context.fillStyle = aiPaddle.color;
    context.fillRect(aiPaddle.x, aiPaddle.y, aiPaddle.width, aiPaddle.height);
}

function game() {
    update(); // Now primarily sends data to the server
    render(); // Continue to render based on the updated state from the server
}

canvas.addEventListener('mousemove', movePaddle);

function movePaddle(evt) {
    let rect = canvas.getBoundingClientRect();
    userPaddle.y = evt.clientY - rect.top - userPaddle.height / 2;
}

let framePerSecond = 50;
setInterval(game, 1000 / framePerSecond);