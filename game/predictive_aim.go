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
