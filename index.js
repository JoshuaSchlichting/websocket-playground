const canvas = document.getElementById('gameCanvas');
const context = canvas.getContext('2d');
const ws = new WebSocket('ws://localhost:80/ws'); // Connect to the Go server

canvas.width = 800;
canvas.height = 400;


let points = []; // Array to store points

let entity = { // Formerly 'ball'
    x: 0,
    y: 0,
    targetIndex: 0, // Index of the next point to move towards
    speed: 1,
    color: "RED"
};
ws.onmessage = function (event) {
    // remove leading/trailing junk from base64 message
    data = event.data.slice(1, -2);
    let parsedData = JSON.parse(atob(data));
    console.log("Parsed data:", parsedData); // Enhanced logging


};



function drawMap() {
    let mapImage = new Image();
    mapImage.src = 'vector-world-map.svg'; // Path to your SVG map
    context.drawImage(mapImage, 0, 0, canvas.width, canvas.height);
}


function drawPoints() {
    points.forEach(point => {
        context.beginPath();
        context.arc(point.x, point.y, 5, 0, Math.PI * 2);
        context.fillStyle = "RED";
        context.fill();
        context.closePath();
    });
}

function moveEntity() {
    if (entity.targetIndex < points.length) {
        let targetPoint = points[entity.targetIndex];
        let dx = targetPoint.x - entity.x;
        let dy = targetPoint.y - entity.y;
        let distance = Math.sqrt(dx * dx + dy * dy);

        if (distance < entity.speed) {
            entity.x = targetPoint.x;
            entity.y = targetPoint.y;
            entity.targetIndex++;
        } else {
            entity.x += (dx / distance) * entity.speed;
            entity.y += (dy / distance) * entity.speed;
        }
    }
}

function render() {
    drawMap();
    drawPoints();
    // Render the entity
    context.beginPath();
    context.arc(entity.x, entity.y, 5, 0, Math.PI * 2);
    context.fillStyle = entity.color;
    context.fill();
    context.closePath();
}

function game() {
    moveEntity();
    render();
}

canvas.addEventListener('click', function (evt) {
    let rect = canvas.getBoundingClientRect();
    let x = evt.clientX - rect.left;
    let y = evt.clientY - rect.top;
    points.push({ x: x, y: y });
    if (points.length === 1) {
        // Place the entity at the first point
        entity.x = x;
        entity.y = y;
    } else {
        // Start moving the entity towards the new point
        entity.targetIndex = points.length - 1;
    }
});

let framePerSecond = 50;
setInterval(game, 1000 / framePerSecond);