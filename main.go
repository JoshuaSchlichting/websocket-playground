package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type GPSCoordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type City struct {
	Name               string         `json:"name"`
	StartingPopulation int            `json:"startingPopulation"`
	Population         int            `json:"population"`
	Coordinates        GPSCoordinates `json:"coordinates"`
	Radius             int            `json:"radius"`
}

type MissileBattery struct {
	Name         string         `json:"name"`
	Coordinates  GPSCoordinates `json:"coordinates"`
	Range        int            `json:"range"`
	MissileCount int            `json:"missileCount"`
}

type Country struct {
	Name             string           `json:"name"`
	Cities           []City           `json:"cities"`
	MissileBatteries []MissileBattery `json:"missileBatteries"`
}
type Missile struct {
	LaunchSite       GPSCoordinates `json:"launchSite"`
	Destination      GPSCoordinates `json:"destination"`
	AltitudeMeters   int            `json:"altitude"`
	SpeedMach        float64        `json:"speedMach"`
	CountryOfOrigin  Country        `json:"countryOfOrigin"`
	PositionInFlight GPSCoordinates `json:"positionInFlight"`
}
type Player struct {
	ID            uuid.UUID       `json:"id"`
	Channel       chan *GameState `json:"-"`
	WebsocketConn *websocket.Conn `json:"-"`
	Countries     []Country       `json:"countries"`
}
type GameState struct {
	ID       uuid.UUID       `json:"id"`
	Missiles []Missile       `json:"missiles"`
	events   chan *GameState `json:"-"`
	Players  []Player        `json:"players"`
}

func moveMissiles(gameState *GameState) {
	for i := range gameState.Missiles {
		missile := &gameState.Missiles[i]
		// Move missile towards destination
		// Calculate the distance between the launch site and the destination
		distance := calculateDistance(missile.LaunchSite, missile.Destination)

		// Calculate the time it takes for the missile to reach the destination
		time := calculateTime(distance, missile.SpeedMach)

		// Calculate the velocity vector of the missile
		velocity := calculateVelocity(missile.LaunchSite, missile.Destination, time)

		// Calculate the new position of the missile
		newPosition := calculateNewPosition(missile.LaunchSite, velocity, time)

		// Update the missile's position
		missile.PositionInFlight = newPosition
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

func addTypeToJSON(jsonBytes []byte, eventType string) []byte {
	var data map[string]interface{}
	err := json.Unmarshal(jsonBytes, &data)
	if err != nil {
		slog.Error("Error unmarshalling JSON", "error", err)
		return jsonBytes
	}
	data["type"] = eventType
	newJSON, err := json.Marshal(data)
	if err != nil {
		slog.Error("Error marshalling JSON", "error", err)
		return jsonBytes
	}
	return newJSON
}
func subscribeClientToToGameEvents(gameState *GameState, conn *websocket.Conn, ctx context.Context) {
	slog.Debug("Subscribing client to game events")
	subscriber := Player{
		ID:      uuid.New(),
		Channel: make(chan *GameState, 1000),
	}
	gameState.Players = append(gameState.Players, subscriber)
	go func() {
		// Broadcast event to the gamestate events channel
		for event := range subscriber.Channel {
			slog.Debug("Sending game state to client", "gameState", event)
			// make sure context is still valid
			select {
			case <-ctx.Done():
				slog.Debug("Context is done, ending client subscription!")
				// remove subscriber
				for i, sub := range gameState.Players {
					if sub.ID == subscriber.ID {
						gameState.Players = append(gameState.Players[:i], gameState.Players[i+1:]...)
						break
					}
				}
				return
			default:
				// context is still valid
			}
			slog.Debug("Sending game state to client", "gameState", event)
			// create slimmed down version of the game state without channels

			eventJSON, err := json.Marshal(event)
			if err != nil {
				slog.Error("Error marshalling game state to JSON", "error", err)
				continue
			}

			// Add "type" key-value pair to the JSON
			eventJSON = addTypeToJSON(eventJSON, "gameStateBroadcast")

			err = conn.WriteJSON(eventJSON)
			if err != nil {
				slog.Error("Error writing game state to client", "error", err)
				return
			}

			err = conn.WriteJSON(eventJSON)
			if err != nil {
				slog.Error("Error writing game state to client", "error", err)
				return
			}
		}
	}()

}

func handleMessage(_ *websocket.Conn, msg []byte, gameState *GameState) {
	// Assuming msg is a []byte containing the JSON message
	var update ClientUpdate
	err := json.Unmarshal(msg, &update)
	if err != nil {
		slog.Debug("Error parsing game update", "error", err)
		return
	}

}
func serveWs(localGameState *GameState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Debug(err.Error())
			return
		}
		defer conn.Close()
		slog.Debug("Client connected")
		ctx := r.Context()
		go subscribeClientToToGameEvents(localGameState, conn, ctx)
		// Continuously read messages from the client
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if err.Error() == "websocket: close 1001 (going away)" {
					slog.Debug("Client disconnected")
					// update context to cancel
					ctx.Done()
					break
				}
				slog.Debug("Error reading message from client", "error", err)
				break
			}
			slog.Debug("Received message from client", "msg", string(msg))
			handleMessage(conn, msg, localGameState)
		}
	}
}

type ClientUpdate struct {
	MissileLaunch struct {
		Missile Missile `json:"missile"`
	}
}

func broadcastGameState(gameState *GameState) {
	select {
	case gameState.events <- gameState:
		// Successfully sent
		// send next event to each subscriber
		event := <-gameState.events
		slog.Debug("Broadcasting game state to clients", "gameState", event)
		for _, subscriber := range gameState.Players {
			slog.Debug("Sending game state to subscriber", "subscriberID", subscriber.ID)
			subscriber.Channel <- event
		}

		slog.Debug("Game state broadcasted", "gameID", gameState.ID)
	default:
		slog.Debug("No receiver ready", "gameID", gameState.ID)
		// No receiver ready, skip or handle accordingly
	}

}
func gameLoop(gameState *GameState, wg *sync.WaitGroup) {
	slog.Debug("Starting game loop for game ID", "gameID", gameState.ID)
	defer wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer slog.Debug("Ending game", "gameID", gameState.ID)
	for range ticker.C {
		// Update game state
		moveMissiles(gameState)
		slog.Debug("Tick", "gameID", gameState)
		// Broadcast game state to all connected clients
		broadcastGameState(gameState)

	}
}

func serveGame(port int, wg *sync.WaitGroup) {
	gameState := &GameState{
		ID:     uuid.New(),
		events: make(chan *GameState, 1000),
	}
	slog.Debug("Starting game", "id", gameState.ID)
	wg.Add(1)
	go gameLoop(gameState, wg)

	http.HandleFunc("/ws", serveWs(gameState))
	http.ListenAndServe(":"+strconv.Itoa(port), nil)
}

func main() {

	// set slog debug level to DEBUG

	slog.SetLogLoggerLevel(slog.LevelDebug)
	// Serve static files from the current directory
	http.Handle("/", http.FileServer(http.Dir(".")))

	runningServersWaitGroup := &sync.WaitGroup{}

	slog.Debug("Starting game server")
	defer slog.Debug("Game server stopped")
	// Listen on port 80
	serveGame(80, runningServersWaitGroup)
	runningServersWaitGroup.Wait()
}
