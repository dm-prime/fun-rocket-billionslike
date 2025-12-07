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
	id                  EntityID
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
	turretPoints        []vec2  // turret positions relative to ship center (local space)
	lastFireTime        float64 // time since last bullet was fired
}

// ID returns the entity ID of the ship
func (s *Ship) ID() EntityID {
	return s.id
}

// Position returns the position of the ship
func (s *Ship) Position() vec2 {
	return s.pos
}

// IsAlive returns true if the ship has health remaining
func (s *Ship) IsAlive() bool {
	return s.health > 0
}

// CollisionRadius returns the collision radius for the ship
func (s *Ship) CollisionRadius() float64 {
	return shipCollisionRadius
}

// OnCollision handles collision response for ships
func (s *Ship) OnCollision(other Entity, damage float64) {
	s.health -= damage
	if s.health < 0 {
		s.health = 0
	}
}

// Rock represents a space rock obstacle
type Rock struct {
	id     EntityID
	pos    vec2
	vel    vec2
	angle  float64
	health float64
}

// ID returns the entity ID of the rock
func (r *Rock) ID() EntityID {
	return r.id
}

// Position returns the position of the rock
func (r *Rock) Position() vec2 {
	return r.pos
}

// IsAlive returns true if the rock has health remaining
func (r *Rock) IsAlive() bool {
	return r.health > 0
}

// CollisionRadius returns the collision radius for the rock
func (r *Rock) CollisionRadius() float64 {
	return rockRadius
}

// OnCollision handles collision response for rocks
func (r *Rock) OnCollision(other Entity, damage float64) {
	r.health -= damage
	if r.health < 0 {
		r.health = 0
	}
}

// Bullet represents a projectile fired from a ship's turret
type Bullet struct {
	id       EntityID // unique ID for this bullet
	pos      vec2     // world position
	vel      vec2     // velocity vector
	age      float64  // age in seconds
	faction  string   // faction that fired the bullet
	ownerID  EntityID // ID of entity that fired it
	isHoming bool     // true if this is a homing missile
	targetID EntityID // ID of target entity (for homing missiles)
	damage   float64  // damage this bullet deals on hit
}

// ID returns the entity ID of the bullet
func (b *Bullet) ID() EntityID {
	return b.id
}

// Position returns the position of the bullet
func (b *Bullet) Position() vec2 {
	return b.pos
}

// IsAlive returns true if the bullet hasn't expired
func (b *Bullet) IsAlive() bool {
	lifetime := bulletLifetime
	if b.isHoming {
		lifetime = homingMissileLifetime
	}
	return b.age < lifetime
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
	ships            map[EntityID]*Ship             // all ships by ID
	rocks            map[EntityID]*Rock             // all rocks by ID
	bullets          map[EntityID]*Bullet           // all bullets by ID
	playerID         EntityID                       // ID of the player's ship
	dust             []dust                         // dust particles for visual effect
	factionColors    map[string]color.NRGBA         // faction color mapping
	alliances        map[string]map[string]bool     // faction alliance relationships
	radarTrails      map[EntityID][]RadarTrailPoint // entity ID -> trail points
	radarTrailTimers map[EntityID]float64           // entity ID -> time since last trail point
	npcStates        map[EntityID]NPCState          // entity ID -> NPC state
	npcInputs        map[EntityID]ShipInput         // entity ID -> current NPC input (for predictive trails)
	rockSpawnTimer   float64                        // timer for rock spawning
	gameTime         float64                        // total game time in seconds
	initialized      bool                           // track if screen size has been initialized
	prevAltEnter     bool                           // track previous Alt+Enter state for toggle
	gameOver         bool                           // true when player is dead
	prevRestartKey   bool                           // track previous R key state for restart
	prevSpaceKey     bool                           // track previous Space key state for shooting
	waveSpawnTimer   float64                        // timer for enemy wave spawning
	waveNumber       int                            // current wave number
}

// GetShip retrieves a ship by ID, returning nil if not found
func (g *Game) GetShip(id EntityID) *Ship {
	return g.ships[id]
}

// GetRock retrieves a rock by ID, returning nil if not found
func (g *Game) GetRock(id EntityID) *Rock {
	return g.rocks[id]
}

// GetBullet retrieves a bullet by ID, returning nil if not found
func (g *Game) GetBullet(id EntityID) *Bullet {
	return g.bullets[id]
}

// PlayerShip returns the player's ship, or nil if not found
func (g *Game) PlayerShip() *Ship {
	return g.ships[g.playerID]
}

// RemoveShip removes a ship and cleans up all associated state
func (g *Game) RemoveShip(id EntityID) {
	delete(g.ships, id)
	delete(g.radarTrails, id)
	delete(g.radarTrailTimers, id)
	delete(g.npcStates, id)
	delete(g.npcInputs, id)
}

// RemoveRock removes a rock from the game
func (g *Game) RemoveRock(id EntityID) {
	delete(g.rocks, id)
}

// RemoveBullet removes a bullet from the game
func (g *Game) RemoveBullet(id EntityID) {
	delete(g.bullets, id)
}
