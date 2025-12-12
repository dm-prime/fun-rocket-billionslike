package game

import "math"

const (
	// ProjectileSpeed is the speed of projectiles in pixels per second
	ProjectileSpeed = 500.0
)

// AimResult contains the result of an aiming calculation
type AimResult struct {
	TargetX, TargetY     float64 // Predicted target position
	AimPointX, AimPointY float64 // Position from which to aim (turret or ship center)
	HasTarget            bool    // Whether a valid target was found
}

// GetAimPoint calculates the position from which an entity should aim
// For entities with active turrets, returns turret position; otherwise returns ship center
func GetAimPoint(entity *Entity) (aimX, aimY float64, hasTurret bool) {
	shipConfig := GetShipTypeConfig(entity.ShipType)

	// Check for active turret mount
	var activeMount *TurretMountPoint
	for i := range shipConfig.TurretMounts {
		if shipConfig.TurretMounts[i].Active {
			activeMount = &shipConfig.TurretMounts[i]
			break
		}
	}

	if activeMount != nil {
		// Calculate turret position in world space
		cosRot := math.Cos(entity.Rotation)
		sinRot := math.Sin(entity.Rotation)
		mountX := activeMount.OffsetX*cosRot - activeMount.OffsetY*sinRot
		mountY := activeMount.OffsetX*sinRot + activeMount.OffsetY*cosRot
		return entity.X + mountX, entity.Y + mountY, true
	}

	// No turret, use ship center
	return entity.X, entity.Y, false
}

// CalculatePredictiveAim calculates the predicted target position accounting for target velocity
// shooterX, shooterY: Position from which the shot will be fired
// target: The target entity
// Returns the predicted position where the shooter should aim
func CalculatePredictiveAim(shooterX, shooterY float64, target *Entity) (predictedX, predictedY float64) {
	return PredictiveAim(
		shooterX, shooterY,
		target.X, target.Y,
		target.VX, target.VY,
		ProjectileSpeed,
	)
}

// PredictiveAim calculates the predicted target position accounting for target velocity and projectile speed
// Returns the predicted position where the shooter should aim
func PredictiveAim(shooterX, shooterY, targetX, targetY, targetVX, targetVY, projectileSpeed float64) (predictedX, predictedY float64) {
	// Calculate relative position and velocity
	dx := targetX - shooterX
	dy := targetY - shooterY

	// If target is not moving, just return current position
	if math.Abs(targetVX) < 0.1 && math.Abs(targetVY) < 0.1 {
		return targetX, targetY
	}

	// Calculate distance to target
	distance := math.Sqrt(dx*dx + dy*dy)
	if distance < 1.0 {
		return targetX, targetY
	}

	// Use iterative approach to solve for interception point
	// We need to find time t such that:
	// distance(shooter, target + targetVelocity * t) = projectileSpeed * t

	// Start with initial guess: time to reach current target position
	t := distance / projectileSpeed

	// Iterate a few times to refine the solution
	for i := 0; i < 5; i++ {
		// Predict where target will be at time t
		predictedTargetX := targetX + targetVX*t
		predictedTargetY := targetY + targetVY*t

		// Calculate distance to predicted position
		predictedDx := predictedTargetX - shooterX
		predictedDy := predictedTargetY - shooterY
		predictedDistance := math.Sqrt(predictedDx*predictedDx + predictedDy*predictedDy)

		// Calculate time to reach predicted position
		if predictedDistance > 0 && projectileSpeed > 0 {
			newT := predictedDistance / projectileSpeed
			// If change is very small, we've converged
			if math.Abs(newT-t) < 0.001 {
				break
			}
			t = newT
		} else {
			break
		}
	}

	// Return predicted position
	predictedX = targetX + targetVX*t
	predictedY = targetY + targetVY*t

	return predictedX, predictedY
}

// RotateTowardsTarget smoothly rotates a rotation value towards a target angle
// currentRotation: Current rotation in radians
// targetRotation: Desired rotation in radians
// maxAngularVelocity: Maximum rotation speed in radians per second
// deltaTime: Time step in seconds
// Returns the new rotation value
func RotateTowardsTarget(currentRotation, targetRotation, maxAngularVelocity, deltaTime float64) float64 {
	angleDiff := targetRotation - currentRotation

	// Normalize angle difference to [-π, π]
	for angleDiff > math.Pi {
		angleDiff -= 2 * math.Pi
	}
	for angleDiff < -math.Pi {
		angleDiff += 2 * math.Pi
	}

	// Calculate rotation step
	rotationStep := angleDiff
	maxStep := maxAngularVelocity * deltaTime
	if math.Abs(rotationStep) > maxStep {
		if rotationStep > 0 {
			rotationStep = maxStep
		} else {
			rotationStep = -maxStep
		}
	}

	return currentRotation + rotationStep
}

// CalculateInterceptDirection calculates the optimal acceleration direction for a homing rocket
// to intercept a moving target, accounting for both rocket and target velocities
// Uses proportional navigation with predictive intercept calculation
// rocketX, rocketY: Rocket position
// rocketVX, rocketVY: Rocket velocity
// targetX, targetY: Target position
// targetVX, targetVY: Target velocity
// rocketAcceleration: Maximum acceleration of the rocket
// deltaTime: Time step
// Returns: directionX, directionY (normalized direction vector for acceleration)
func CalculateInterceptDirection(rocketX, rocketY, rocketVX, rocketVY, targetX, targetY, targetVX, targetVY, rocketAcceleration, deltaTime float64) (directionX, directionY float64) {
	// Calculate relative position and velocity
	relX := targetX - rocketX
	relY := targetY - rocketY
	relVX := targetVX - rocketVX
	relVY := targetVY - rocketVY

	// Calculate distance
	distance := math.Sqrt(relX*relX + relY*relY)
	if distance < 0.1 {
		// Already very close, maintain current direction
		currentSpeed := math.Sqrt(rocketVX*rocketVX + rocketVY*rocketVY)
		if currentSpeed > 0.1 {
			return rocketVX / currentSpeed, rocketVY / currentSpeed
		}
		return 1.0, 0.0 // Default direction
	}

	// Calculate relative speed
	relativeSpeed := math.Sqrt(relVX*relVX + relVY*relVY)

	// Use iterative approach to find intercept point
	// We need to solve: where will target be when rocket can intercept?
	// Rocket can accelerate, so we need to account for that

	// Initial estimate: time to intercept based on current positions and velocities
	// Use a reasonable estimate considering acceleration
	rocketSpeed := math.Sqrt(rocketVX*rocketVX + rocketVY*rocketVY)
	closingSpeed := relativeSpeed
	if closingSpeed < 1.0 {
		closingSpeed = rocketSpeed + rocketAcceleration*deltaTime*5 // Estimate with acceleration
	}

	t := distance / math.Max(closingSpeed, 1.0)

	// Iterate to find intercept point
	for i := 0; i < 15; i++ {
		// Predict target position at time t
		predictedTargetX := targetX + targetVX*t
		predictedTargetY := targetY + targetVY*t

		// Calculate vector from rocket to predicted target
		dx := predictedTargetX - rocketX
		dy := predictedTargetY - rocketY
		predictedDistance := math.Sqrt(dx*dx + dy*dy)

		if predictedDistance < 0.1 {
			// Very close, use direction to target
			return dx / math.Max(predictedDistance, 0.1), dy / math.Max(predictedDistance, 0.1)
		}

		// Calculate required velocity to reach predicted position in time t
		// v_required = (predicted_pos - rocket_pos) / t
		reqVX := dx / t
		reqVY := dy / t

		// Calculate velocity change needed from current rocket velocity
		deltaVX := reqVX - rocketVX
		deltaVY := reqVY - rocketVY
		deltaV := math.Sqrt(deltaVX*deltaVX + deltaVY*deltaVY)

		// Maximum velocity change possible with acceleration in time t
		maxDeltaV := rocketAcceleration * t

		// If we can achieve the required velocity change, we have our direction
		if deltaV <= maxDeltaV+0.01 {
			// Normalize and return direction
			if deltaV > 0.01 {
				return deltaVX / deltaV, deltaVY / deltaV
			}
			// Very small change, use direction to predicted target
			return dx / predictedDistance, dy / predictedDistance
		}

		// Can't intercept in time t, need more time
		// Estimate new time based on distance and acceleration capability
		// If we accelerate at max rate towards target, how long to close distance?
		// Use kinematic equation: d = v0*t + 0.5*a*t^2
		// Solve for t: t = (-v0 + sqrt(v0^2 + 2*a*d)) / a
		// Simplified: estimate based on average velocity
		avgSpeed := rocketSpeed + 0.5*rocketAcceleration*t
		newT := predictedDistance / math.Max(avgSpeed+relativeSpeed, 1.0)

		if math.Abs(newT-t) < 0.001 || newT > 100.0 {
			// Converged or timeout, use current estimate
			if deltaV > 0.01 {
				return deltaVX / deltaV, deltaVY / deltaV
			}
			return dx / predictedDistance, dy / predictedDistance
		}

		t = newT
	}

	// Fallback: use proportional navigation
	// Lead the target based on relative velocity
	// Calculate line-of-sight rate
	losRate := (relX*relVY - relY*relVX) / (distance * distance)

	// Proportional navigation: accelerate perpendicular to line of sight
	// plus some component towards target
	// Navigation constant (higher = more aggressive)
	N := 3.0

	// Desired acceleration direction combines:
	// 1. Direction to target (to close distance)
	// 2. Perpendicular component (to null line-of-sight rate)
	perpX := -relY / distance
	perpY := relX / distance

	// Combine components
	dirX := relX/distance + N*losRate*perpX
	dirY := relY/distance + N*losRate*perpY

	// Normalize
	dirLen := math.Sqrt(dirX*dirX + dirY*dirY)
	if dirLen > 0.01 {
		return dirX / dirLen, dirY / dirLen
	}

	// Final fallback: direction to target
	return relX / distance, relY / distance
}
