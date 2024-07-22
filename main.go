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
	"golang.org/x/exp/rand"
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
	Coordinates  GPSCoordinates `json:"coordinates"`
	Range        int            `json:"range"`
	MissileCount int            `json:"missileCount"`
}

type Country struct {
	Name             string            `json:"name"`
	Cities           map[string]*City  `json:"cities"`
	MissileBatteries []*MissileBattery `json:"missileBatteries"`
}

type Missile struct {
	LaunchSite       GPSCoordinates `json:"launchSite"`
	Destination      *City          `json:"destination"`
	AltitudeMeters   int            `json:"altitude"`
	SpeedMach        float64        `json:"speedMach"`
	CountryOfOrigin  string         `json:"countryOfOrigin"`
	PositionInFlight GPSCoordinates `json:"positionInFlight"`
	Active           bool           `json:"active"`
}

type Player struct {
	ID            uuid.UUID       `json:"id"`
	Channel       chan *GameState `json:"-"`
	WebsocketConn *websocket.Conn `json:"-"`
	Country       string          `json:"country"`
	gameInstance  *GameState      `json:"-"`
}

func NewPlayer(gameState *GameState, websocket *websocket.Conn, country string) *Player {

	return &Player{
		ID:            uuid.New(),
		Channel:       make(chan *GameState, 1000),
		Country:       country,
		WebsocketConn: websocket,
		gameInstance:  gameState,
	}
}

func (p *Player) LaunchMissile(target *City, silo *MissileBattery) error {
	if silo.MissileCount == 0 {
		return fmt.Errorf("0 balance on missiles in silo '%v'", silo.Coordinates)
	}
	silo.MissileCount--
	missile := Missile{
		LaunchSite:      silo.Coordinates,
		Destination:     target,
		AltitudeMeters:  1000,
		SpeedMach:       2.5,
		CountryOfOrigin: p.Country,
		Active:          true,
	}
	p.gameInstance.Missiles = append(p.gameInstance.Missiles, &missile)
	slog.Info("Missile launched", "missile", missile, "missilesRemaining", silo.MissileCount)
	return nil
}

type GameState struct {
	ID        uuid.UUID           `json:"id"`
	Missiles  []*Missile          `json:"missiles"`
	events    chan *GameState     `json:"-"`
	Countries map[string]*Country `json:"countries"`
	Players   []*Player           `json:"players"`
}

func moveMissiles(gameState *GameState) (destinationsHit []*City) {
	for i := range gameState.Missiles {
		missile := gameState.Missiles[i]
		if !missile.Active {
			continue
		}
		// Assuming a simple parabolic trajectory for demonstration purposes.
		// In reality, missile trajectories are more complex and depend on various factors.

		// Calculate the direction vector from current position to destination.
		directionLat := missile.Destination.Coordinates.Latitude - missile.PositionInFlight.Latitude
		directionLon := missile.Destination.Coordinates.Longitude - missile.PositionInFlight.Longitude

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
		if math.Abs(missile.PositionInFlight.Latitude-missile.Destination.Coordinates.Latitude) < 0.05 &&
			math.Abs(missile.PositionInFlight.Longitude-missile.Destination.Coordinates.Longitude) < 0.2 {
			missile.Active = false
			slog.Info("SPLASH! Missile reached its destination.", "origin", missile.CountryOfOrigin, "origin", missile.LaunchSite, "target", missile.Destination)
			destinationsHit = append(destinationsHit, missile.Destination)
		}
	}
	return destinationsHit
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
		newPlayer := NewPlayer(gameState, conn, "Russia")
		go newPlayer.subscribeClientToToGameEvents(ctx)
		newPlayer.LaunchMissile(gameState.Countries["Russia"].Cities["Moscow"], gameState.Countries["USA"].MissileBatteries[0])
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
		destinationsHit := moveMissiles(gameState)
		// loop through all players, and all their cities, and update their cities population
		if len(destinationsHit) > 0 {
			for _, country := range gameState.Countries {
				for _, city := range country.Cities {
					for _, hitCity := range destinationsHit {
						if city.Name == hitCity.Name {
							newPopulation := city.Population - rand.Intn(4500000)
							originalPopulation := city.Population
							if newPopulation > 0 {
								city.Population = newPopulation
							} else {
								city.Population = 0
								newPopulation = 0
							}
							slog.Info("City hit by missile", "city", city.Name, "beforeStrikePopulation", originalPopulation, "afterStrikePop", newPopulation)
						}
					}
				}
			}
		}
		// slog.Debug("Tick", "gameID", gameState)
		// Broadcast game state to all connected clients
		broadcastGameState(gameState)

	}
}

func serveGame(port int, wg *sync.WaitGroup) {
	// make a copy of countries
	countryList := map[string]*Country{}
	for name, country := range countries {
		countryList[name] = &Country{
			Name:             country.Name,
			Cities:           country.Cities,
			MissileBatteries: country.MissileBatteries,
		}
	}

	gameState := &GameState{
		ID:        uuid.New(),
		events:    make(chan *GameState, 1000),
		Countries: countryList,
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
