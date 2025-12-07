package main

import (
	"image/color"
	"math"
)

var (
	screenWidth  int
	screenHeight int
)

// Gameplay constants
const (
	angularAccel             = math.Pi * 3 // radians per second^2
	angularDampingAccel      = math.Pi * 8 // radians per second^2 (for S key)
	maxAngularSpeed          = math.Pi * 4 // maximum angular speed (radians per second)
	thrustAccel              = 500.0       // pixels per second^2
	sideThrustAccel          = 77.0        // pixels per second^2 (side thruster acceleration)
	dustCount                = 70
	dustBaseSpeed            = 20.0
	retroAlignTolerance      = 20 * math.Pi / 180 // radians
	retroVelocityStopEpsilon = 5.0                // px/s, consider ship stopped
	retroMinSpeedForTurn     = 1.0                // px/s, minimum speed to compute heading
	retroBurnAlignWindow     = 8 * math.Pi / 180  // radians, must be within this to burn
	radarRange               = 5000
	radarMargin              = 14.0
	indicatorMargin          = 18.0
	indicatorArrowLen        = 18.0
	radarTrailMaxAge         = 3.0         // seconds
	radarTrailUpdateInterval = 0.1         // seconds between trail points
	radarTrailMaxPoints      = 30          // maximum trail points per ship
	radarStackThreshold      = 10.0        // pixels - dots closer than this will be stacked
	radarStackSpacing        = 8.0         // pixels - vertical spacing between stacked dots
	rockCount                = 50          // target number of rocks to maintain
	rockRadius               = 12.0        // collision radius for rocks
	shipCollisionRadius      = 18.0        // collision radius for ships (based on ship geometry)
	collisionCourseLookAhead = 5.0         // seconds to look ahead for collision course detection
	rockPathDistance         = 800.0       // distance from player path to spawn/keep rocks
	rockDespawnDistance      = 1500.0      // distance from player to despawn rocks
	rockSpawnInterval        = 0.2         // seconds between rock spawn attempts
	rockMinSpawnDistance     = 600.0       // minimum distance from player to spawn rocks (outside view range)
	bulletSpeed              = 800.0       // pixels per second
	bulletLifetime           = 3.0         // seconds before bullet despawns
	bulletRadius             = 2.0         // collision radius for bullets
	turretFireRate           = 0.5         // seconds between shots per turret
	turretRange              = 2000.0      // maximum range for turret targeting
	turretFireAngleThreshold = math.Pi / 6 // 30 degrees - turret can fire if target is within this angle
	bulletDamage             = 10.0        // damage dealt by regular bullets
	homingMissileSpeed       = 600.0       // pixels per second (slower than bullets)
	homingMissileTurnRate    = math.Pi * 2 // radians per second - how fast missiles can turn
	homingMissileDamage      = 25.0        // damage dealt by homing missiles
	homingMissileLifetime    = 5.0         // seconds before homing missile despawns
	shipCollisionDamage      = 30.0        // damage from ship-ship collisions
	rockCollisionDamage      = 50.0        // damage from ship-rock collisions
	maxHealth                = 100.0       // maximum health for ships
	waveSpawnInterval        = 8.0         // seconds between enemy waves
	waveSpawnDistance        = 1200.0      // distance from player to spawn enemies
	enemiesPerWave           = 3           // number of enemies per wave
	waveSizeIncrease         = 0.1         // increase enemies per wave over time (multiplier)
)

// Color constants
var (
	colorBackground       = color.NRGBA{R: 3, G: 5, B: 16, A: 255}
	colorDust             = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	colorVelocityVector   = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	colorRadarBackdrop    = color.NRGBA{R: 10, G: 16, B: 32, A: 230}
	colorRadarRing        = color.NRGBA{R: 24, G: 48, B: 96, A: 255}
	colorRadarHeading     = color.NRGBA{R: 120, G: 210, B: 255, A: 255}
	colorRadarPlayer      = color.NRGBA{R: 180, G: 255, B: 200, A: 255}
	colorRadarSpeedVector = color.NRGBA{R: 120, G: 220, B: 255, A: 220}
	colorRadarFlame       = color.NRGBA{R: 255, G: 180, B: 60, A: 255}
	colorRockCollision    = color.NRGBA{R: 255, G: 100, B: 100, A: 255} // Bright red for rocks on collision course
)

// Ship geometry constants
const (
	shipNoseOffsetY     = -18.0
	shipLeftOffsetX     = -12.0
	shipLeftOffsetY     = 12.0
	shipRightOffsetX    = 12.0
	shipRightOffsetY    = 12.0
	shipBackOffsetY     = 12.0
	flameBaseLength     = 28.0
	flameVarLength      = 8.0
	sideFlameBaseLen    = 15.0
	sideFlameVarLen     = 5.0
	sideThrusterX       = 10.0
	velocityVectorScale = 0.1
	turretLeftX         = -8.0 // left turret X offset (local space)
	turretLeftY         = 8.0  // left turret Y offset (local space)
	turretRightX        = 8.0  // right turret X offset (local space)
	turretRightY        = 8.0  // right turret Y offset (local space)
	turretSize          = 3.0  // visual size of turret point

	predictiveTrailDuration     = 3.0 // seconds into future to predict
	predictiveTrailSegmentCount = 30  // number of segments in the trail
	predictiveTrailUpdateRate   = 0.1 // seconds per segment
)

// Radar geometry constants
const (
	radarHeadingOffset    = 8.0
	radarCenterDotSize    = 2.0
	radarBlipSize         = 3.0
	radarEdgeMargin       = 4.0
	radarLabelOffsetX     = 8.0
	radarLabelOffsetY     = 8.0
	radarOffRadarDist     = 10.0
	radarSpeedVectorScale = 0.18
	radarSpeedVectorMax   = 0.45
)

// UI constants
const (
	hudLabelMarginX    = 64
	hudLabelMarginY    = 12
	indicatorLabelX    = 8
	indicatorLabelY    = 8
	windowedSizeRatio  = 0.9
	dustSpanMultiplier = 1.5
	trailOpacityMax    = 0.6
)

// getRadarRadius returns the radar radius as 80% of screen height (radius = 40% of height)
func getRadarRadius() float64 {
	return float64(screenHeight) * 0.4
}
