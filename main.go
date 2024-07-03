package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Paddle struct {
	Y      float64 `json:"y"`
	Height float64 `json:"height"`
}

type Ball struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	VelocityX float64 `json:"velocityX"`
	VelocityY float64 `json:"velocityY"`
}

type GameState struct {
	Ball       Ball   `json:"ball"`
	UserPaddle Paddle `json:"userPaddle"`
	AIPaddle   Paddle `json:"aiPaddle"`
}

// Global variables to manage connected clients and game state

var broadcast = make(chan *GameState) // Broadcast channel

func gameLoop(gameState *GameState) {
	ticker := time.NewTicker(20 * time.Millisecond)
	for range ticker.C {
		// Update game state
		gameState.Ball.X += gameState.Ball.VelocityX
		gameState.Ball.Y += gameState.Ball.VelocityY

		// Collision detection (simplified) for top and bottom boundaries
		if gameState.Ball.Y <= 0 || gameState.Ball.Y >= 400 {
			gameState.Ball.VelocityY = -gameState.Ball.VelocityY
		}

		// Collision detection for player paddle
		if gameState.Ball.X <= 20 { // Assuming the left edge is at X=0 and the paddle width is 20
			if gameState.Ball.Y >= gameState.UserPaddle.Y && gameState.Ball.Y <= (gameState.UserPaddle.Y+gameState.UserPaddle.Height) {
				gameState.Ball.VelocityX = -gameState.Ball.VelocityX // Reverse X velocity to bounce
			}
		}

		// Collision detection for AI paddle
		if gameState.Ball.X >= 780 { // Assuming the right edge is at X=800 and the paddle width is 20
			if gameState.Ball.Y >= gameState.AIPaddle.Y && gameState.Ball.Y <= (gameState.AIPaddle.Y+gameState.AIPaddle.Height) {
				gameState.Ball.VelocityX = -gameState.Ball.VelocityX // Reverse X velocity to bounce
			}
		}

		log.Printf("Ball position: (%f, %f)", gameState.Ball.X, gameState.Ball.Y)
		broadcast <- gameState
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Implement origin checking logic here
		// For example, allow all origins:
		return true
	},
}

func handleMessage(_ *websocket.Conn, msg []byte, gameState *GameState) {
	// Assuming msg is a []byte containing the JSON message
	var update GameUpdate
	err := json.Unmarshal(msg, &update)
	if err != nil {
		log.Printf("Error parsing game update: %v", err)
		return
	}

	gameState.UserPaddle.Y = update.UserPaddle.Y
}

func serveWs(localGameState *GameState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		go func() {
			for gameState := range broadcast {
				err := conn.WriteJSON(gameState)
				if err != nil {
					log.Printf("Error writing game state to client: %v", err)
					return
				}
			}
		}()
		// Continuously read messages from the client
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading message from client: %v", err)
				break
			}
			handleMessage(conn, msg, localGameState)
		}
	}
}

type GameUpdate struct {
	Ball struct {
		X         float64 `json:"x"`
		Y         float64 `json:"y"`
		Radius    float64 `json:"radius"`
		VelocityX float64 `json:"velocityX"`
		VelocityY float64 `json:"velocityY"`
		Speed     float64 `json:"speed"`
		Color     string  `json:"color"`
	} `json:"ball"`
	UserPaddle struct {
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
		Score  int     `json:"score"`
		Color  string  `json:"color"`
	} `json:"userPaddle"`
	AIPaddle struct {
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
		Score  int     `json:"score"`
		Color  string  `json:"color"`
	} `json:"aiPaddle"`
}

func main() {
	// Serve static files from the current directory
	http.Handle("/", http.FileServer(http.Dir(".")))

	localGameState := &GameState{
		Ball:       Ball{X: 400, Y: 200, VelocityX: 5, VelocityY: 5},
		UserPaddle: Paddle{Y: 150, Height: 100},
		AIPaddle:   Paddle{Y: 150, Height: 100},
	}

	go gameLoop(localGameState)

	// WebSocket handler
	http.HandleFunc("/ws", serveWs(localGameState))
	// Listen on port 80
	log.Fatal(http.ListenAndServe(":80", nil))
}
