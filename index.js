const canvas = document.getElementById('pongCanvas');
const context = canvas.getContext('2d');

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
    x: 0, // left side of canvas
    y: (canvas.height - 100) / 2, // -100 the height of paddle
    width: 10,
    height: 100,
    score: 0,
    color: "WHITE"
};

let aiPaddle = {
    x: canvas.width - 10, // - width of paddle
    y: (canvas.height - 100) / 2, // -100 the height of paddle
    width: 10,
    height: 100,
    score: 0,
    color: "WHITE"
};

function drawRect(x, y, w, h, color) {
    context.fillStyle = color;
    context.fillRect(x, y, w, h);
}

function drawCircle(x, y, r, color) {
    context.fillStyle = color;
    context.beginPath();
    context.arc(x, y, r, 0, Math.PI*2, false);
    context.closePath();
    context.fill();
}

function drawText(text, x, y, color) {
    context.fillStyle = color;
    context.font = "75px fantasy";
    context.fillText(text, x, y);
}

function render() {
    // Clear the canvas
    drawRect(0, 0, canvas.width, canvas.height, "BLACK");
    
    // Draw the user and AI paddles
    drawRect(userPaddle.x, userPaddle.y, userPaddle.width, userPaddle.height, userPaddle.color);
    drawRect(aiPaddle.x, aiPaddle.y, aiPaddle.width, aiPaddle.height, aiPaddle.color);
    
    // Draw the ball
    drawCircle(ball.x, ball.y, ball.radius, ball.color);
    
    // Draw the scores
    drawText(userPaddle.score, canvas.width / 4, canvas.height / 5, "WHITE");
    drawText(aiPaddle.score, 3 * canvas.width / 4, canvas.height / 5, "WHITE");
}
function update() {
    ball.x += ball.velocityX;
    ball.y += ball.velocityY;
    
    // Simple AI to control the aiPaddle (to be improved)
    aiPaddle.y += ((ball.y - (aiPaddle.y + aiPaddle.height / 2))) * 0.1;
    
    // Ball collision with top/bottom walls
    if(ball.y - ball.radius < 0 || ball.y + ball.radius > canvas.height){
        ball.velocityY = -ball.velocityY;
    }
    
    // Ball collision with paddles
    let player = (ball.x < canvas.width / 2) ? userPaddle : aiPaddle;
    if(collision(ball, player)){
        // Reverse the ball's direction
        ball.velocityX = -ball.velocityX;
    }
    
    // Update scores (to be implemented)
}


function collision(b, p) {
    b.top = b.y - b.radius;
    b.bottom = b.y + b.radius;
    b.left = b.x - b.radius;
    b.right = b.x + b.radius;
    
    p.top = p.y;
    p.bottom = p.y + p.height;
    p.left = p.x;
    p.right = p.x + p.width;
    
    return b.right > p.left && b.bottom > p.top && b.left < p.right && b.top < p.bottom;
}

function game() {
    update();
    render();
}

// Control the user paddle
canvas.addEventListener('mousemove', movePaddle);

function movePaddle(evt) {
    let rect = canvas.getBoundingClientRect();
    
    userPaddle.y = evt.clientY - rect.top - userPaddle.height / 2;
}

// Loop the game
let framePerSecond = 50;
setInterval(game, 1000 / framePerSecond);