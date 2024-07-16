package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
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
	gameInstance  *GameState      `json:"-"`
}

func NewPlayer(gameState *GameState, websocket *websocket.Conn, countries []Country) *Player {
	return &Player{
		ID:            uuid.New(),
		Channel:       make(chan *GameState, 1000),
		Countries:     countries,
		WebsocketConn: websocket,
		gameInstance:  gameState,
	}
}

func (p *Player) AddCountry(country Country) {
	p.Countries = append(p.Countries, country)
}

func (p *Player) RemoveCountry(country Country) {
	for i, c := range p.Countries {
		if c.Name == country.Name {
			p.Countries = append(p.Countries[:i], p.Countries[i+1:]...)
			break
		}
	}
}

func (p *Player) LaunchMissile(target GPSCoordinates, silo *MissileBattery) error {
	if silo.MissileCount == 0 {
		return fmt.Errorf("0 balance on missiles in silo '%s'", silo.Name)
	}
	silo.MissileCount--
	missile := Missile{
		LaunchSite:      silo.Coordinates,
		Destination:     target,
		AltitudeMeters:  1000,
		SpeedMach:       2.5,
		CountryOfOrigin: p.Countries[0],
	}
	p.gameInstance.Missiles = append(p.gameInstance.Missiles, missile)
	slog.Info("Missile launched", "missile", missile)
	return nil
}

type GameState struct {
	ID       uuid.UUID       `json:"id"`
	Missiles []Missile       `json:"missiles"`
	events   chan *GameState `json:"-"`
	Players  []*Player       `json:"players"`
}

func moveMissiles(gameState *GameState) {
	for i := range gameState.Missiles {
		missile := &gameState.Missiles[i]
		// Assuming a simple parabolic trajectory for demonstration purposes.
		// In reality, missile trajectories are more complex and depend on various factors.

		// Calculate the direction vector from current position to destination.
		directionLat := missile.Destination.Latitude - missile.PositionInFlight.Latitude
		directionLon := missile.Destination.Longitude - missile.PositionInFlight.Longitude

		// Normalize the direction vector (so its length is 1).
		distance := math.Sqrt(directionLat*directionLat + directionLon*directionLon)
		directionLat /= distance
		directionLon /= distance

		// Assuming SpeedMach as a simple scalar for movement per time unit.
		// Convert Mach speed to a distance unit relevant to GPS coordinates. This is a simplification.
		// In reality, you would convert the speed to a distance per time unit based on altitude, etc.
		speed := missile.SpeedMach * 0.1 // Placeholder conversion factor

		// Calculate the maximum height of the parabolic trajectory.
		maxHeight := distance / 2

		// Calculate the current height of the missile based on its position in the trajectory.
		currentHeight := maxHeight - math.Pow(distance/2, 2)

		// Update position in flight.
		missile.PositionInFlight.Latitude += directionLat * speed
		missile.PositionInFlight.Longitude += directionLon * speed

		// Update altitude based on the current height of the missile.
		missile.AltitudeMeters = int(currentHeight)

		// Check if the rocket has reached its destination (or close enough).
		if math.Abs(missile.PositionInFlight.Latitude-missile.Destination.Latitude) < 0.01 &&
			math.Abs(missile.PositionInFlight.Longitude-missile.Destination.Longitude) < 0.01 {
			fmt.Println("Rocket has reached its destination.")
		}
		slog.Debug("Missile moved", "origin", missile.CountryOfOrigin.Name, "position", missile.PositionInFlight, "target", missile.Destination)
	}
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

func (p *Player) subscribeClientToToGameEvents(ctx context.Context) {
	slog.Debug("Subscribing client to game events")
	p.gameInstance.Players = append(p.gameInstance.Players, p)
	go func() {
		// Broadcast event to the gamestate events channel
		for event := range p.Channel {
			// slog.Debug("Sending game state to client", "gameState", event)
			// make sure context is still valid
			select {
			case <-ctx.Done():
				slog.Debug("Context is done, ending client subscription!")
				// remove subscriber
				for i, sub := range p.gameInstance.Players {
					if sub.ID == p.ID {
						p.gameInstance.Players = append(p.gameInstance.Players[:i], p.gameInstance.Players[i+1:]...)
						break
					}
				}
				return
			default:
				// context is still valid
			}

			eventJSON, err := json.Marshal(event)
			if err != nil {
				slog.Error("Error marshalling game state to JSON", "error", err)
				continue
			}

			// Add "type" key-value pair to the JSON
			eventJSON = addTypeToJSON(eventJSON, "gameStateBroadcast")

			err = p.WebsocketConn.WriteJSON(eventJSON)
			if err != nil {
				slog.Error("Error writing game state to client", "error", err)
				return
			}

			err = p.WebsocketConn.WriteJSON(eventJSON)
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

func serveWs(gameState *GameState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Implement origin checking logic here
				// For example, allow all origins:
				return true
			},
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Debug(err.Error())
			return
		}
		defer conn.Close()
		slog.Debug("Client connected")
		ctx := r.Context()
		newPlayer := NewPlayer(gameState, conn, []Country{
			{
				Name: "Test",
				Cities: []City{
					{
						Name:               "Test",
						StartingPopulation: 1000,
						Population:         1000,
						Coordinates: GPSCoordinates{
							Latitude:  0,
							Longitude: 0,
						},
						Radius: 100,
					},
				},
				MissileBatteries: []MissileBattery{
					{
						Name:         "Test",
						Coordinates:  GPSCoordinates{Latitude: 0, Longitude: 0},
						Range:        1000,
						MissileCount: 10,
					},
				},
			},
		})

		go newPlayer.subscribeClientToToGameEvents(ctx)
		newPlayer.LaunchMissile(GPSCoordinates{Latitude: 20.29136348359818, Longitude: -78.61084102637136}, &MissileBattery{
			Name:         "Test",
			Coordinates:  GPSCoordinates{Latitude: 40.716690215399325, Longitude: -74.20734855505961},
			Range:        1000,
			MissileCount: 10,
		})
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
			handleMessage(conn, msg, gameState)
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
		// slog.Debug("Broadcasting game state to clients", "gameState", event)
		for _, subscriber := range gameState.Players {
			// slog.Debug("Sending game state to subscriber", "subscriberID", subscriber.ID)
			subscriber.Channel <- event
		}

	default:
		slog.Debug("No receiver ready", "gameID", gameState.ID)
		// No receiver ready, skip or handle accordingly
	}

}
func gameLoop(gameState *GameState, wg *sync.WaitGroup) {
	slog.Debug("Starting game loop for game ID", "gameID", gameState.ID)
	defer wg.Done()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer slog.Debug("Ending game", "gameID", gameState.ID)
	for range ticker.C {
		// Update game state
		moveMissiles(gameState)
		// slog.Debug("Tick", "gameID", gameState)
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
