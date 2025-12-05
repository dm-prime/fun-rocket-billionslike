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
	thrustAccel              = 350.0       // pixels per second^2
	sideThrustAccel          = 77.0        // pixels per second^2 (side thruster acceleration)
	dustCount                = 70
	dustBaseSpeed            = 20.0
	retroAlignTolerance      = 20 * math.Pi / 180 // radians
	retroVelocityStopEpsilon = 5.0                // px/s, consider ship stopped
	retroMinSpeedForTurn     = 1.0                // px/s, minimum speed to compute heading
	retroBurnAlignWindow     = 8 * math.Pi / 180  // radians, must be within this to burn
	radarRadius              = 200.0
	radarRange               = 1520.0
	radarMargin              = 14.0
	indicatorMargin          = 18.0
	indicatorArrowLen        = 18.0
	radarTrailMaxAge         = 3.0  // seconds
	radarTrailUpdateInterval = 0.1  // seconds between trail points
	radarTrailMaxPoints      = 30   // maximum trail points per ship
	radarStackThreshold      = 10.0 // pixels - dots closer than this will be stacked
	radarStackSpacing        = 8.0  // pixels - vertical spacing between stacked dots
)

// Color constants
var (
	colorBackground       = color.NRGBA{R: 3, G: 5, B: 16, A: 255}
	colorDust             = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	colorVelocityVector    = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	colorRadarBackdrop     = color.NRGBA{R: 10, G: 16, B: 32, A: 230}
	colorRadarRing         = color.NRGBA{R: 24, G: 48, B: 96, A: 255}
	colorRadarHeading      = color.NRGBA{R: 120, G: 210, B: 255, A: 255}
	colorRadarPlayer       = color.NRGBA{R: 180, G: 255, B: 200, A: 255}
	colorRadarSpeedVector  = color.NRGBA{R: 120, G: 220, B: 255, A: 220}
	colorRadarFlame        = color.NRGBA{R: 255, G: 180, B: 60, A: 255}
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

