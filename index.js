const canvas = document.getElementById('gameCanvas');
const context = canvas.getContext('2d');
const ws = new WebSocket('ws://localhost:80/ws'); // Connect to the Go server

canvas.width = 800;
canvas.height = 400;


let points = []; // Array to store points
let gameState = {}; // Object to store the game state
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
    if (parsedData['type'] === 'gameStateBroadcast') {
        gameState = parsedData;

        // add gameState['messages'] to the gameMessages textarea
        let gameMessages = document.getElementById('gameMessages');
        let messages = gameState['messages'];
        if (messages) {
            gameMessages.value = messages.join('\n');
        }
    }

};



function drawMap() {
    let mapImage = new Image();
    mapImage.src = 'vector-world-map.svg'; // Path to your SVG map
    context.drawImage(mapImage, 0, 0, canvas.width, canvas.height);
}


function drawPoints() {
    // drag points on the map for missiles, based on example:
    // "{\"countries\":{\"Russia\":{\"cities\":{\"Moscow\":{\"coordinates\":{\"latitude\":55.7558,\"longitude\":37.6176},\"name\":\"Moscow\",\"population\":6010649,\"radius\":10,\"startingPopulation\":9500000},\"Saint Petersburg\":{\"coordinates\":{\"latitude\":59.9343,\"longitude\":30.3351},\"name\":\"Saint Petersburg\",\"population\":8000000,\"radius\":10,\"startingPopulation\":8000000}},\"missileBatteries\":[{\"coordinates\":{\"latitude\":55.7558,\"longitude\":37.6176},\"missileCount\":0,\"range\":0}],\"name\":\"Russia\"},\"USA\":{\"cities\":{\"Los Angeles\":{\"coordinates\":{\"latitude\":34.0522,\"longitude\":-118.2437},\"name\":\"Los Angeles\",\"population\":7000000,\"radius\":10,\"startingPopulation\":7000000},\"New York\":{\"coordinates\":{\"latitude\":40.7128,\"longitude\":-74.006},\"name\":\"New York\",\"population\":50000000,\"radius\":10,\"startingPopulation\":50000000}},\"missileBatteries\":[{\"coordinates\":{\"latitude\":40.7128,\"longitude\":-74.006},\"missileCount\":9,\"range\":0},{\"coordinates\":{\"latitude\":34.0522,\"longitude\":-118.2437},\"missileCount\":5,\"range\":0}],\"name\":\"USA\"}},\"id\":\"09ca9f7e-7eb1-4910-8fd5-3fe5ba215ddf\",\"missiles\":[{\"active\":false,\"altitude\":0,\"countryOfOrigin\":\"Russia\",\"destination\":{\"coordinates\":{\"latitude\":55.7558,\"longitude\":37.6176},\"name\":\"Moscow\",\"population\":6010649,\"radius\":10,\"startingPopulation\":9500000},\"launchSite\":{\"latitude\":40.7128,\"longitude\":-74.006},\"positionInFlight\":{\"latitude\":55.74821613040324,\"longitude\":37.612483277202685},\"speedMach\":2.5}],\"players\":[{\"country\":\"Russia\",\"id\":\"bb227c6b-bd6c-489a-b1f1-08ce5ee0339e\"}],\"type\":\"gameStateBroadcast\"}"
    let countries = gameState['countries'];
    for (let country in countries) {
        let cities = countries[country]['cities'];
        for (let city in cities) {
            let point = cities[city];
            let x = (point.coordinates.longitude + 180) / 360 * canvas.width;
            let y = canvas.height - (point.coordinates.latitude + 90) / 180 * canvas.height;
            context.beginPath();
            context.arc(x, y, 5, 0, Math.PI * 2);
            context.fillStyle = "BLUE";
            context.fill();
            context.closePath();
        }
    }
    let missiles = gameState['missiles'];
    for (let missile of missiles) {
        let x = (missile.positionInFlight.longitude + 180) / 360 * canvas.width;
        let y = canvas.height - (missile.positionInFlight.latitude + 90) / 180 * canvas.height;
        context.beginPath();
        context.arc(x, y, 5, 0, Math.PI * 2);
        context.fillStyle = "RED";
        context.fill();
        context.closePath();
    }
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

canvas.addEventListener('click', function (evt) {
    let rect = canvas.getBoundingClientRect();
    let x = evt.clientX - rect.left;
    let y = evt.clientY - rect.top;
    let longitude = (x / canvas.width) * 360 - 180;
    let latitude = ((canvas.height - y) / canvas.height) * 180 - 90;
    console.log("Longitude:", longitude, "Latitude:", latitude);
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
