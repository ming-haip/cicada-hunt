package generation

import (
	"math"
	"math/rand"
	"time"

	"github.com/cicada-hunt/server/internal/models"
	"github.com/cicada-hunt/server/internal/spatial"
)

// CicadaAI implements the behavior state machine for adult cicadas.
//
// State transitions:
//
//	RESTING ──(player < alertDist)──→ ALERT
//	RESTING ──(spontaneous)─────────→ FLYING (casual tree swap)
//	ALERT ────(player < fleeDist)───→ FLYING (evasive)
//	ALERT ────(player recedes)──────→ RESTING (calm down)
//	FLYING ───(reached destination)─→ RESTING
//	ANY ──────(net swing nearby)────→ STARTLED (panic flight)
//	STARTLED ─(after cooldown)──────→ RESTING
type CicadaAI struct {
	behaviorTimer map[string]time.Time  // cicadaID → next behavior check
	startledUntil map[string]time.Time  // cicadaID → startled cooldown end
}

// NewCicadaAI creates a new cicada AI manager.
func NewCicadaAI() *CicadaAI {
	return &CicadaAI{
		behaviorTimer: make(map[string]time.Time),
		startledUntil: make(map[string]time.Time),
	}
}

// CicadaAIContext encapsulates all information the AI needs to decide behavior.
type CicadaAIContext struct {
	Cicada       *models.CicadaSpawn
	NearestPlayer *PlayerInfo
	Now          time.Time
}

// PlayerInfo is the minimal player data the AI needs.
type PlayerInfo struct {
	PlayerID    string
	Lat, Lng    float64
	SpeedMS     float64 // estimated movement speed
	IsSwinging  bool    // player just swung a net
}

// AIEvent is emitted when a cicada's behavior changes.
type AIEvent struct {
	CicadaID      string
	OldState      models.CicadaState
	NewState      models.CicadaState
	FlightType    models.FlightType
	TargetLat     float64
	TargetLng     float64
	TargetAltM    float64
	DurationSec   float64
}

// Update evaluates AI for a cicada and returns state change events.
func (ai *CicadaAI) Update(ctx *CicadaAIContext) *AIEvent {
	// Check startled cooldown
	if until, ok := ai.startledUntil[ctx.Cicada.ID]; ok {
		if ctx.Now.Before(until) {
			// Still in startled recovery — don't process further
			return nil
		}
		// Cooldown expired — cicada calms down
		delete(ai.startledUntil, ctx.Cicada.ID)
		return ai.transition(ctx, models.CicadaStateResting, models.FlightTypeCasual)
	}

	// Don't re-evaluate too frequently
	if lastCheck, ok := ai.behaviorTimer[ctx.Cicada.ID]; ok {
		if ctx.Now.Before(lastCheck.Add(1 * time.Second)) {
			return nil
		}
	}
	ai.behaviorTimer[ctx.Cicada.ID] = ctx.Now

	// Switch on current state
	switch ctx.Cicada.CurrentState {

	case models.CicadaStateResting, models.CicadaStateRestingSinging:
		return ai.updateResting(ctx)

	case models.CicadaStateAlert:
		return ai.updateAlert(ctx)

	case models.CicadaStateFlying:
		return ai.updateFlying(ctx)

	case models.CicadaStateStartled:
		return ai.updateStartled(ctx)
	}

	return nil
}

// ================================================================
// State update handlers
// ================================================================

func (ai *CicadaAI) updateResting(ctx *CicadaAIContext) *AIEvent {
	cicada := ctx.Cicada
	player := ctx.NearestPlayer

	if player == nil {
		// No player nearby — spontaneous behavior
		return ai.maybeSpontaneousFly(ctx)
	}

	// Calculate distance and risk
	distM := spatial.HaversineDistance(cicada.Lat, cicada.Lng, player.Lat, player.Lng)

	// Player swinging a net nearby → STARTLED
	if player.IsSwinging && distM < 3.0 {
		return ai.transition(ctx, models.CicadaStateStartled, models.FlightTypePanic)
	}

	// Player too close → FLEE directly (skip alert for fast approach)
	if distM < cicada.AlertDistM*0.3 && player.SpeedMS > 3.0 {
		return ai.transition(ctx, models.CicadaStateFlying, models.FlightTypeEvasive)
	}

	// Player within alert range → become ALERT
	if distM < cicada.AlertDistM {
		return ai.transition(ctx, models.CicadaStateAlert, models.FlightTypeCasual)
	}

	// Player far away — spontaneous behavior
	return ai.maybeSpontaneousFly(ctx)
}

func (ai *CicadaAI) updateAlert(ctx *CicadaAIContext) *AIEvent {
	cicada := ctx.Cicada
	player := ctx.NearestPlayer

	if player == nil {
		// Player disappeared — calm down
		return ai.transition(ctx, models.CicadaStateResting, models.FlightTypeCasual)
	}

	distM := spatial.HaversineDistance(cicada.Lat, cicada.Lng, player.Lat, player.Lng)

	// Player swung net → STARTLED
	if player.IsSwinging && distM < 3.0 {
		return ai.transition(ctx, models.CicadaStateStartled, models.FlightTypePanic)
	}

	// Player very close → FLEE
	if distM < cicada.AlertDistM*0.3 {
		return ai.transition(ctx, models.CicadaStateFlying, models.FlightTypeEvasive)
	}

	// Player approaching quickly → high chance to flee
	detectionRisk := calculateDetectionRisk(distM, player.SpeedMS, cicada.Agility, cicada.AlertDistM)
	if detectionRisk > 0.7 {
		return ai.transition(ctx, models.CicadaStateFlying, models.FlightTypeEvasive)
	}

	// Player receded beyond alert range → calm down after delay
	if distM > cicada.AlertDistM*1.5 {
		if ctx.Now.After(ai.behaviorTimer[cicada.ID].Add(3 * time.Second)) {
			return ai.transition(ctx, models.CicadaStateResting, models.FlightTypeCasual)
		}
	}

	// Stay alert — no transition
	return nil
}

func (ai *CicadaAI) updateFlying(ctx *CicadaAIContext) *AIEvent {
	cicada := ctx.Cicada

	// Simulate flight completion
	// In production: track flight progress, check if destination reached
	// For now: after 3-8 seconds, land on a new tree
	lastCheck, exists := ai.behaviorTimer[cicada.ID+"_flight_start"]
	if !exists {
		ai.behaviorTimer[cicada.ID+"_flight_start"] = ctx.Now
		return nil
	}

	flightDuration := 3 + rand.Float64()*5 // 3-8 seconds
	if ctx.Now.After(lastCheck.Add(time.Duration(flightDuration * float64(time.Second)))) {
		delete(ai.behaviorTimer, cicada.ID+"_flight_start")
		return ai.transition(ctx, models.CicadaStateResting, models.FlightTypeCasual)
	}

	return nil
}

func (ai *CicadaAI) updateStartled(ctx *CicadaAIContext) *AIEvent {
	cicada := ctx.Cicada

	// Startled state: rapid panic flight, then long cooldown
	cooldown := 60 + rand.Float64()*60 // 60-120 second cooldown
	ai.startledUntil[cicada.ID] = ctx.Now.Add(time.Duration(cooldown * float64(time.Second)))

	// Emit flight event
	event := ai.transition(ctx, models.CicadaStateFlying, models.FlightTypePanic)
	return event
}

// ================================================================
// Helpers
// ================================================================

// maybeSpontaneousFly randomly decides if a resting cicada flies to a new tree.
func (ai *CicadaAI) maybeSpontaneousFly(ctx *CicadaAIContext) *AIEvent {
	// 15% chance per evaluation to spontaneously fly
	if rand.Float64() < 0.15 {
		// Pick random direction within 50m
		angle := rand.Float64() * 2 * math.Pi
		dist := 20 + rand.Float64()*30
		targetLat := ctx.Cicada.Lat + (dist/111320.0)*math.Cos(angle)
		targetLng := ctx.Cicada.Lng + (dist/(111320.0*math.Cos(ctx.Cicada.Lat*math.Pi/180.0)))*math.Sin(angle)

		event := ai.transition(ctx, models.CicadaStateFlying, models.FlightTypeCasual)
		if event != nil {
			event.TargetLat = targetLat
			event.TargetLng = targetLng
			event.TargetAltM = ctx.Cicada.AltitudeM
			event.DurationSec = 3 + rand.Float64()*5
		}
		return event
	}
	return nil
}

// transition executes a state change and returns the event.
func (ai *CicadaAI) transition(ctx *CicadaAIContext, newState models.CicadaState, flightType models.FlightType) *AIEvent {
	oldState := ctx.Cicada.CurrentState
	ctx.Cicada.CurrentState = newState

	return &AIEvent{
		CicadaID:   ctx.Cicada.ID,
		OldState:   oldState,
		NewState:   newState,
		FlightType: flightType,
	}
}

// calculateDetectionRisk computes how likely the cicada is to detect the player.
// Factors: distance, player speed, cicada agility, alert distance.
func calculateDetectionRisk(distM, playerSpeedMS, agility, alertDistM float64) float64 {
	risk := 0.0

	// Distance factor (exponential decay)
	risk += math.Exp(-distM / alertDistM)

	// Speed factor
	if playerSpeedMS > 4.0 {
		risk += 0.5
	} else if playerSpeedMS > 2.0 {
		risk += 0.3
	} else if playerSpeedMS > 0.5 {
		risk += 0.1
	}

	// Agile cicadas detect sooner
	risk *= (0.8 + agility*0.4)

	return math.Min(risk, 1.0)
}

// ================================================================
// Flight path generation
// ================================================================

// CicadaFlightPath represents a generated flight trajectory.
type CicadaFlightPath struct {
	StartLat  float64
	StartLng  float64
	StartAlt  float64
	EndLat    float64
	EndLng    float64
	EndAlt    float64
	Control1  [3]float64 // bezier control point 1
	Control2  [3]float64 // bezier control point 2
	Duration  float64     // seconds
	FlightType models.FlightType
}

// GenerateFlightPath creates a Bezier flight path for a cicada.
func GenerateFlightPath(
	startLat, startLng, startAlt float64,
	endLat, endLng, endAlt float64,
	flightType models.FlightType,
) *CicadaFlightPath {

	path := &CicadaFlightPath{
		StartLat:   startLat,
		StartLng:   startLng,
		StartAlt:   startAlt,
		EndLat:     endLat,
		EndLng:     endLng,
		EndAlt:     endAlt,
		FlightType: flightType,
	}

	// Convert to local meter-space for control point computation
	dx := (endLng - startLng) * 111320.0 * math.Cos(startLat*math.Pi/180.0)
	dy := (endLat - startLat) * 111320.0
	dz := endAlt - startAlt

	switch flightType {
	case models.FlightTypeCasual:
		// High arc, meandering
		path.Control1 = [3]float64{dx * 0.3, dy * 0.2, dz + 3}
		path.Control2 = [3]float64{dx * 0.7, dy * 0.8, dz + 2}
		path.Duration = 3 + rand.Float64()*5

	case models.FlightTypeEvasive:
		// Direct, fast, slight upward
		path.Control1 = [3]float64{dx * 0.5, dy * 0.3, dz + 5}
		path.Control2 = [3]float64{dx * 0.8, dy * 0.9, dz + 1}
		path.Duration = 1.5 + rand.Float64()*1.5

	case models.FlightTypePanic:
		// Sharp upward, then zigzag
		path.Control1 = [3]float64{dx * 0.2, dy * 0.1, dz + 8}
		path.Control2 = [3]float64{dx * 0.6, dy * 0.7, dz + 4}
		path.Duration = 1.0 + rand.Float64()*1.0
	}

	return path
}

// EvaluatePosition returns the cicada position at normalized time t (0-1)
// along the Bezier flight path.
func (p *CicadaFlightPath) EvaluatePosition(t float64) (lat, lng, alt float64) {
	// Cubic bezier: B(t) = (1-t)³P0 + 3(1-t)²tP1 + 3(1-t)t²P2 + t³P3
	u := 1.0 - t

	x := u*u*u*0 +
		3*u*u*t*p.Control1[0] +
		3*u*t*t*p.Control2[0] +
		t*t*t*(p.EndLng-p.StartLng)*111320.0*math.Cos(p.StartLat*math.Pi/180.0)

	y := u*u*u*0 +
		3*u*u*t*p.Control1[1] +
		3*u*t*t*p.Control2[1] +
		t*t*t*(p.EndLat-p.StartLat)*111320.0

	z := u*u*u*0 +
		3*u*u*t*p.Control1[2] +
		3*u*t*t*p.Control2[2] +
		t*t*t*(p.EndAlt-p.StartAlt)

	// Convert back to geo
	lng = p.StartLng + x/(111320.0*math.Cos(p.StartLat*math.Pi/180.0))
	lat = p.StartLat + y/111320.0
	alt = p.StartAlt + z

	return lat, lng, alt
}
