package main

var countries = map[string]Country{
	"USA": {
		Name: "USA",
		Cities: map[string]*City{
			"New York": {
				Name:               "New York",
				StartingPopulation: 50000000,
				Population:         50000000,
				Coordinates: GPSCoordinates{
					Latitude:  40.7128,
					Longitude: -74.0060,
				},
				Radius: 10,
			},
			"Los Angeles": {
				Name:               "Los Angeles",
				StartingPopulation: 7000000,
				Population:         7000000,
				Coordinates: GPSCoordinates{
					Latitude:  34.0522,
					Longitude: -118.2437,
				},
				Radius: 10,
			},
		},
		MissileBatteries: []*MissileBattery{
			{
				Coordinates: GPSCoordinates{
					Latitude:  40.7128,
					Longitude: -74.0060,
				},
				MissileCount: 10,
			},
			{
				Coordinates: GPSCoordinates{
					Latitude:  34.0522,
					Longitude: -118.2437,
				},
				MissileCount: 5,
			},
		},
	},
	"Russia": {
		Name: "Russia",
		Cities: map[string]*City{
			"Moscow": {
				Name:               "Moscow",
				StartingPopulation: 9500000,
				Population:         9500000,
				Coordinates: GPSCoordinates{
					Latitude:  55.7558,
					Longitude: 37.6176,
				},
				Radius: 10,
			},
			"Saint Petersburg": {
				Name:               "Saint Petersburg",
				StartingPopulation: 8000000,

				Population: 8000000,
				Coordinates: GPSCoordinates{
					Latitude:  59.9343,
					Longitude: 30.3351,
				},
				Radius: 10,
			},
		},
		MissileBatteries: []*MissileBattery{
			{
				Coordinates: GPSCoordinates{
					Latitude:  55.7558,
					Longitude: 37.6176,
				},
			},
		},
	},
}
