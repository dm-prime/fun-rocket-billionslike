package main

import (
	"sync/atomic"
)

// EntityID is a unique identifier for any entity in the game.
// Using unique IDs instead of array indices eliminates fragile index remapping
// when entities are added or removed, making the codebase more robust.
type EntityID uint64

// InvalidEntityID represents an invalid or null entity reference.
// Use this to indicate that an entity reference is not set or has been invalidated.
const InvalidEntityID EntityID = 0

var nextEntityID uint64 = 1

// generateEntityID creates a new unique entity ID.
// This function is thread-safe and uses atomic operations to ensure uniqueness.
func generateEntityID() EntityID {
	return EntityID(atomic.AddUint64(&nextEntityID, 1))
}

// Entity is the base interface for all game entities
type Entity interface {
	ID() EntityID
	Position() vec2
	IsAlive() bool
}

// Collidable represents entities that can collide with others
type Collidable interface {
	Entity
	CollisionRadius() float64
	OnCollision(other Entity, damage float64)
}

// Updatable represents entities that need per-frame updates
type Updatable interface {
	Entity
	Update(dt float64)
}

// NewShip creates a new ship with a unique ID
func NewShip(pos vec2, vel vec2, angle float64, faction string, isPlayer bool) *Ship {
	return &Ship{
		id:           generateEntityID(),
		pos:          pos,
		vel:          vel,
		angle:        angle,
		health:       maxHealth,
		faction:      faction,
		isPlayer:     isPlayer,
		turretPoints: make([]vec2, 0),
	}
}

// NewRock creates a new rock with a unique ID
func NewRock(pos vec2, vel vec2, angle float64) *Rock {
	return &Rock{
		id:     generateEntityID(),
		pos:    pos,
		vel:    vel,
		angle:  angle,
		health: maxHealth,
	}
}

// NewBullet creates a new bullet with a unique ID
func NewBullet(pos vec2, vel vec2, faction string, ownerID EntityID, isHoming bool, targetID EntityID, damage float64) *Bullet {
	return &Bullet{
		id:       generateEntityID(),
		pos:      pos,
		vel:      vel,
		age:      0,
		faction:  faction,
		ownerID:  ownerID,
		isHoming: isHoming,
		targetID: targetID,
		damage:   damage,
	}
}

