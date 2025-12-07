package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// Particle represents a single particle in a particle system
type Particle struct {
	pos      vec2   // world position
	vel      vec2   // velocity vector
	age      float64 // age in seconds
	lifetime float64 // total lifetime in seconds
	color    color.NRGBA
	size     float64
}

// IsAlive returns true if the particle is still alive
func (p *Particle) IsAlive() bool {
	return p.age < p.lifetime
}

// ParticleSystem represents a particle emitter system
type ParticleSystem struct {
	particles            []Particle
	maxParticles         int
	emissionRate         float64 // particles per second
	emissionTimer        float64 // time since last emission
	emitterPos           vec2    // world position of emitter
	emitterAngle         float64 // angle of emission direction
	emitterVelocity      vec2    // velocity of emitter (ship velocity)
	emitterOffset        vec2    // offset from ship center in local space
	emissionDirectionOffset float64 // angle offset for emission direction (0 = forward, Pi = backward)
	velocityMin          float64 // minimum particle velocity
	velocityMax          float64 // maximum particle velocity
	spreadAngle          float64 // spread angle in radians (half-angle)
	lifetimeMin          float64 // minimum particle lifetime
	lifetimeMax          float64 // maximum particle lifetime
	sizeMin              float64 // minimum particle size
	sizeMax              float64 // maximum particle size
	colorBase            color.NRGBA // base color
	colorVariation       color.NRGBA // color variation range
	active               bool    // whether the system is currently emitting
}

// Update updates the particle system
func (ps *ParticleSystem) Update(dt float64, emitterWorldPos vec2, emitterAngle float64, emitterVelocity vec2) {
	// Update emitter position, angle, and velocity
	ps.emitterPos = emitterWorldPos
	ps.emitterAngle = emitterAngle
	ps.emitterVelocity = emitterVelocity

	// Emit new particles if active
	if ps.active {
		ps.emissionTimer += dt
		particlesToEmit := int(ps.emissionRate * ps.emissionTimer)
		if particlesToEmit > 0 {
			ps.emissionTimer -= float64(particlesToEmit) / ps.emissionRate
			for i := 0; i < particlesToEmit && len(ps.particles) < ps.maxParticles; i++ {
				ps.emitParticle()
			}
		}
	}

	// Update existing particles
	for i := len(ps.particles) - 1; i >= 0; i-- {
		p := &ps.particles[i]
		p.age += dt
		p.pos.x += p.vel.x * dt
		p.pos.y += p.vel.y * dt

		// Remove dead particles
		if !p.IsAlive() {
			ps.particles = append(ps.particles[:i], ps.particles[i+1:]...)
		}
	}
}

// emitParticle creates a new particle
func (ps *ParticleSystem) emitParticle() {
	// Calculate emitter position in world space
	emitterLocal := rotatePoint(ps.emitterOffset, ps.emitterAngle)
	emitterWorld := vec2{
		x: ps.emitterPos.x + emitterLocal.x,
		y: ps.emitterPos.y + emitterLocal.y,
	}

	// Random angle within spread, relative to emission direction
	angleOffset := (rand.Float64() - 0.5) * ps.spreadAngle * 2
	particleAngle := ps.emitterAngle + ps.emissionDirectionOffset + angleOffset

	// Random velocity relative to emitter direction
	// Use same coordinate system as ship: forward = (sin(angle), -cos(angle))
	velocity := ps.velocityMin + rand.Float64()*(ps.velocityMax-ps.velocityMin)
	vel := vec2{
		x: math.Sin(particleAngle) * velocity,
		y: -math.Cos(particleAngle) * velocity,
	}
	// Add emitter velocity so particles inherit ship's motion
	vel.x += ps.emitterVelocity.x
	vel.y += ps.emitterVelocity.y

	// Random lifetime
	lifetime := ps.lifetimeMin + rand.Float64()*(ps.lifetimeMax-ps.lifetimeMin)

	// Random size
	size := ps.sizeMin + rand.Float64()*(ps.sizeMax-ps.sizeMin)

	// Random color variation
	color := color.NRGBA{
		R: uint8(clamp(float64(ps.colorBase.R)+rand.Float64()*float64(ps.colorVariation.R)*2-float64(ps.colorVariation.R), 0, 255)),
		G: uint8(clamp(float64(ps.colorBase.G)+rand.Float64()*float64(ps.colorVariation.G)*2-float64(ps.colorVariation.G), 0, 255)),
		B: uint8(clamp(float64(ps.colorBase.B)+rand.Float64()*float64(ps.colorVariation.B)*2-float64(ps.colorVariation.B), 0, 255)),
		A: ps.colorBase.A,
	}

	ps.particles = append(ps.particles, Particle{
		pos:      emitterWorld,
		vel:      vel,
		age:      0,
		lifetime: lifetime,
		color:    color,
		size:     size,
	})
}

// Draw renders all particles in the system
func (ps *ParticleSystem) Draw(screen *ebiten.Image, cameraOffset vec2, cameraAngle float64) {
	for _, p := range ps.particles {
		// Transform particle position to screen space
		offset := vec2{p.pos.x - cameraOffset.x, p.pos.y - cameraOffset.y}
		rotated := rotatePoint(offset, -cameraAngle)
		screenX := float64(screenWidth)*0.5 + rotated.x
		screenY := float64(screenHeight)*0.5 + rotated.y

		// Calculate alpha based on age, with reduced base alpha for transparency
		ageAlpha := 1.0 - (p.age / p.lifetime)
		ageAlpha = math.Max(0, math.Min(1, ageAlpha))
		baseAlpha := 0.5 // Reduce base alpha to 50% for semi-transparent particles
		finalAlpha := baseAlpha * ageAlpha
		particleColor := color.NRGBA{
			R: p.color.R,
			G: p.color.G,
			B: p.color.B,
			A: uint8(float64(p.color.A) * finalAlpha),
		}

		// Draw particle as a filled circle
		drawFilledCircle(screen, screenX, screenY, p.size, particleColor)
	}
}

// SetActive sets whether the particle system is actively emitting
func (ps *ParticleSystem) SetActive(active bool) {
	ps.active = active
	if !active {
		ps.emissionTimer = 0
	}
}

// NewThrustParticleSystem creates a particle system for forward thrust
func NewThrustParticleSystem() *ParticleSystem {
	return &ParticleSystem{
		maxParticles:          50,
		emissionRate:          60.0, // particles per second
		emitterOffset:         vec2{0, shipBackOffsetY},
		emissionDirectionOffset: math.Pi, // emit backward (opposite to ship forward)
		velocityMin:           80.0,
		velocityMax:           150.0,
		spreadAngle:           math.Pi / 6, // 30 degrees
		lifetimeMin:           0.2,
		lifetimeMax:           0.5,
		sizeMin:               1.5,
		sizeMax:               3.0,
		colorBase:             color.NRGBA{R: 255, G: 200, B: 0, A: 255},
		colorVariation:        color.NRGBA{R: 55, G: 100, B: 0, A: 0},
		active:                false,
	}
}

// NewReverseThrustParticleSystem creates a particle system for reverse thrust
func NewReverseThrustParticleSystem() *ParticleSystem {
	return &ParticleSystem{
		maxParticles:          50,
		emissionRate:          60.0,
		emitterOffset:         vec2{0, shipNoseOffsetY},
		emissionDirectionOffset: 0, // emit forward (same as ship forward direction)
		velocityMin:           80.0,
		velocityMax:           150.0,
		spreadAngle:           math.Pi / 6,
		lifetimeMin:           0.2,
		lifetimeMax:           0.5,
		sizeMin:               1.5,
		sizeMax:               3.0,
		colorBase:             color.NRGBA{R: 100, G: 150, B: 255, A: 255},
		colorVariation:        color.NRGBA{R: 50, G: 50, B: 55, A: 0},
		active:                false,
	}
}

// NewSideThrusterParticleSystem creates a particle system for side thrusters
func NewSideThrusterParticleSystem(leftSide bool) *ParticleSystem {
	offsetX := sideThrusterX
	emissionDirOffset := math.Pi / 2 // right side: 90 degrees (perpendicular right)
	if leftSide {
		offsetX = -sideThrusterX
		emissionDirOffset = -math.Pi / 2 // left side: -90 degrees (perpendicular left)
	}
	return &ParticleSystem{
		maxParticles:          30,
		emissionRate:          40.0,
		emitterOffset:         vec2{offsetX, shipBackOffsetY},
		emissionDirectionOffset: emissionDirOffset, // emit perpendicular to ship forward
		velocityMin:           60.0,
		velocityMax:           120.0,
		spreadAngle:           math.Pi / 4, // 45 degrees
		lifetimeMin:           0.15,
		lifetimeMax:           0.35,
		sizeMin:               1.0,
		sizeMax:               2.5,
		colorBase:             color.NRGBA{R: 255, G: 150, B: 0, A: 255},
		colorVariation:        color.NRGBA{R: 55, G: 80, B: 0, A: 0},
		active:                false,
	}
}

// updateShipParticles updates all particle systems for a ship based on its state
func (g *Game) updateShipParticles(ship *Ship, dt float64) {
	// Update forward thrust particles
	ship.thrustParticles.SetActive(ship.thrustThisFrame)
	ship.thrustParticles.Update(dt, ship.pos, ship.angle, ship.vel)

	// Update reverse thrust particles
	ship.reverseParticles.SetActive(ship.reverseThrustFrame)
	ship.reverseParticles.Update(dt, ship.pos, ship.angle, ship.vel)

	// Determine which side thrusters should be active
	leftThrusterActive := false
	rightThrusterActive := false

	// Active turning
	if ship.turningThisFrame {
		if ship.turnDirection > 0 {
			rightThrusterActive = true
		} else {
			leftThrusterActive = true
		}
	}

	// Angular damping (automatic or manual)
	if math.Abs(ship.angularVel) > 0.1 {
		if ship.angularVel > 0 {
			// Rotating right, fire left thruster to counter
			leftThrusterActive = true
		} else {
			// Rotating left, fire right thruster to counter
			rightThrusterActive = true
		}
	}

	// Update side thruster particles
	ship.leftThrusterParticles.SetActive(leftThrusterActive)
	ship.leftThrusterParticles.Update(dt, ship.pos, ship.angle, ship.vel)

	ship.rightThrusterParticles.SetActive(rightThrusterActive)
	ship.rightThrusterParticles.Update(dt, ship.pos, ship.angle, ship.vel)
}

