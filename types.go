package main

import "image/color"

// vec2 represents a 2D vector
type vec2 struct {
	x float64
	y float64
}

// ShipInput represents the control inputs for a ship
type ShipInput struct {
	TurnLeft       bool // Turn left (A/Left arrow)
	TurnRight      bool // Turn right (D/Right arrow)
	ThrustForward  bool // Thrust forward (W/Up arrow)
	RetrogradeBurn bool // Retrograde burn (S/Down arrow)
}

// Ship represents a single spacecraft in the world.
type Ship struct {
	pos                 vec2
	vel                 vec2
	angle               float64
	angularVel          float64
	health              float64
	faction             string
	thrustThisFrame     bool
	turningThisFrame    bool
	turnDirection       float64 // -1 for left, 1 for right, 0 for none
	dampingAngularSpeed bool    // true when S key is pressed to dampen angular speed
	retrogradeMode      bool    // true when performing retrograde burn maneuver
	retrogradeTurnDir   float64 // chosen turn direction for retrograde (-1 or 1)
	isPlayer            bool
}

// dust represents a single dust particle
type dust struct {
	pos    vec2
	speed  float64
	radius float64
}

// RadarTrailPoint represents a single point in a ship's radar trail
type RadarTrailPoint struct {
	pos vec2    // world coordinates
	age float64 // age in seconds
}

// Game holds the minimal state required for a simple arcade-feel spaceship demo.
type Game struct {
	ships            []Ship
	playerIndex      int
	dust             []dust
	factionColors    map[string]color.NRGBA
	alliances        map[string]map[string]bool
	radarTrails      map[int][]RadarTrailPoint // ship index -> trail points
	radarTrailTimers map[int]float64           // ship index -> time since last trail point
	npcStates        map[int]NPCState          // ship index -> NPC state
	npcInputs        map[int]ShipInput         // ship index -> current NPC input (for predictive trails)
	rockSpawnTimer   float64                   // timer for rock spawning
	initialized      bool                      // track if screen size has been initialized
	prevAltEnter     bool                      // track previous Alt+Enter state for toggle
}
