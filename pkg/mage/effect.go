package mage

import (
	. "github.com/mage/mage/pkg/mage/core"
	"fmt"

	"github.com/google/uuid"
)

// Effect represents a one-shot effect that resolves.
type Effect interface {
	Apply(g GameMutator, sourceID uuid.UUID, controller uuid.UUID, targets []uuid.UUID) error
	Text() string
	Properties() EffectProperties
}

// funcEffect wraps an anonymous function as an Effect. Use FuncEffect to create
// one-off effects inline in card definitions without needing a dedicated struct.
type funcEffect struct {
	text  string
	props EffectProperties
	fn    func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error
}

// FuncEffect creates an Effect from an anonymous function. This is ideal for
// card-specific effects that are used by only one card and don't warrant a
// dedicated type. The text parameter is used for Text() (rules text display).
// The props parameter allows callers to declare AI-visible properties (outcome, damage, etc.).
func FuncEffect(text string, props EffectProperties, fn func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error) Effect {
	return &funcEffect{text: text, props: props, fn: fn}
}

func (e *funcEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	return e.fn(g, sourceID, controller, targets)
}

func (e *funcEffect) Text() string                   { return e.text }
func (e *funcEffect) Properties() EffectProperties   { return e.props }

// compositeEffect applies multiple effects in sequence.
type compositeEffect struct {
	effects []Effect
	text    string
}

// CompositeEffects creates an effect that applies multiple effects in sequence.
func CompositeEffects(text string, effects ...Effect) Effect {
	return &compositeEffect{effects: effects, text: text}
}

func (e *compositeEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, eff := range e.effects {
		if err := eff.Apply(g, sourceID, controller, targets); err != nil {
			return err
		}
	}
	return nil
}

func (e *compositeEffect) Text() string { return e.text }
func (e *compositeEffect) Properties() EffectProperties { return EffectProperties{} }

// Outcome describes whether an effect is beneficial or detrimental to its primary target.
// The AI uses this to choose appropriate targets.
type Outcome int

const (
	OutcomeUnknown   Outcome = iota // context-dependent or not classifiable
	OutcomeBenefit                  // good for the target (buff, heal, protect, untap)
	OutcomeDetriment                // bad for the target (damage, destroy, discard, steal)
)

// EffectProperties describes the AI-visible shape of an Effect.
// Effects declare their own properties at construction time; the AI reads
// them without type-switching.
type EffectProperties struct {
	Outcome     Outcome     // benefit / detriment / unknown (from target's perspective)
	DamageValue ValueSource // non-nil if this effect deals damage; call Resolve for amount
	DrawCount   int         // fixed cards drawn; 0 for X or non-draw effects
	Mass        bool        // true if effect is board-wide (wrath, earthquake, etc.)

	// AI-search properties (used by search.go for simplified spell resolution):
	LifeGain       int // life gained by controller; 0 if not a life-gain effect
	PowerBoost     int // power boost for target; 0 if not a boost effect
	ToughnessBoost int // toughness boost for target; 0 if not a boost effect
	IsBounce       bool // true if this effect bounces a permanent to hand
	TokenPower     int  // token creature power; 0 if not a token-creation effect
	TokenToughness int  // token creature toughness; 0 if not a token-creation effect
}

// IsDamageEffect returns true if the given effect is a damage-dealing effect.
func IsDamageEffect(e Effect) bool {
	return e.Properties().DamageValue != nil
}

// EffectOutcome classifies a single effect from the primary target's perspective.
func EffectOutcome(e Effect) Outcome {
	return e.Properties().Outcome
}

// SpellOutcome returns the dominant outcome of a spell's effect list.
// It returns the first non-unknown outcome found, or OutcomeUnknown if none is classifiable.
func SpellOutcome(effects []Effect) Outcome {
	for _, e := range effects {
		if o := EffectOutcome(e); o != OutcomeUnknown {
			return o
		}
	}
	return OutcomeUnknown
}

// --- ValueSource, PlayerSelector, PermanentSelector ---

// ValueSource resolves a dynamic integer value for an effect.
// It receives a GameReader (read-only view) since value resolution never mutates state.
type ValueSource interface {
	Resolve(g GameReader, sourceID, controller uuid.UUID) int
	Text() string
}

// PlayerSelector picks one or more players for an effect.
// It receives a GameReader (read-only view) since player selection never mutates state.
type PlayerSelector interface {
	Select(g GameReader, sourceID, controller uuid.UUID, targets []uuid.UUID) []uuid.UUID
	Text() string
}

// PermanentSelector chooses between the target permanent and the source permanent.
type PermanentSelector int

const (
	SelectTarget PermanentSelector = iota
	SelectSource
)

// fixedValue is a ValueSource that always returns a constant.
type fixedValue struct{ n int }

// Fixed creates a ValueSource that always returns the constant n.
func Fixed(n int) ValueSource                            { return fixedValue{n: n} }
func (v fixedValue) Resolve(_ GameReader, _, _ uuid.UUID) int { return v.n }
func (v fixedValue) Text() string                        { return fmt.Sprintf("%d", v.n) }

// xValue is a ValueSource that reads g.CurrentX.
type xValue struct{}

// XValue creates a ValueSource that reads the X value from the current spell/ability (g.CurrentX).
func XValue() ValueSource                            { return xValue{} }
func (v xValue) Resolve(g GameReader, _, _ uuid.UUID) int { return g.XValue() }
func (v xValue) Text() string                        { return "X" }

// countBattlefieldValue is a ValueSource that counts permanents on the battlefield.
// If who is nil, all permanents are counted regardless of controller.
type countBattlefieldValue struct {
	who    PlayerSelector
	filter PermanentFilter
}

// CountBattlefield creates a ValueSource that counts permanents on the battlefield
// matching the filter, optionally restricted to those controlled by who.
// Pass who=nil to count across all players.
func CountBattlefield(who PlayerSelector, f PermanentFilter) ValueSource {
	return countBattlefieldValue{who: who, filter: f}
}
func (v countBattlefieldValue) Resolve(g GameReader, sourceID, controller uuid.UUID) int {
	if v.who == nil {
		return g.CountBattlefield(v.filter)
	}
	total := 0
	for _, pid := range v.who.Select(g, sourceID, controller, nil) {
		total += g.CountBattlefield(And(v.filter, ControlledBy(pid)))
	}
	return total
}
func (v countBattlefieldValue) Text() string {
	noun := v.filter.Text()
	if noun == "" {
		noun = "permanent"
	}
	if v.who != nil {
		return fmt.Sprintf("the number of %ss the %s controls", noun, v.who.Text())
	}
	return fmt.Sprintf("the number of %ss on the battlefield", noun)
}

// countZoneValue is a ValueSource that counts cards in a player zone (hand, graveyard, library).
type countZoneValue struct {
	zone   Zone
	who    PlayerSelector
	filter CardFilter // nil matches any card
}

// CountZone creates a ValueSource that counts cards in the given zone for the selected
// players. Pass filter=nil to count all cards in the zone.
func CountZone(zone Zone, who PlayerSelector, f CardFilter) ValueSource {
	return countZoneValue{zone: zone, who: who, filter: f}
}
func (v countZoneValue) Resolve(g GameReader, sourceID, controller uuid.UUID) int {
	total := 0
	for _, pid := range v.who.Select(g, sourceID, controller, nil) {
		p := g.GetPlayer(pid)
		if p == nil {
			continue
		}
		var cards []Card
		switch v.zone {
		case ZoneHand:
			cards = p.Hand()
		case ZoneGraveyard:
			cards = p.Graveyard()
		case ZoneLibrary:
			cards = p.Library()
		}
		for _, c := range cards {
			if v.filter.Match(c) {
				total++
			}
		}
	}
	return total
}
func (v countZoneValue) Text() string {
	zone := "zone"
	switch v.zone {
	case ZoneHand:
		zone = "hand"
	case ZoneGraveyard:
		zone = "graveyard"
	case ZoneLibrary:
		zone = "library"
	}
	noun := v.filter.Text()
	if noun == "" {
		noun = "card"
	}
	return fmt.Sprintf("the number of %ss in the %s's %s", noun, v.who.Text(), zone)
}

// selectController returns the effect's controller.
type selectController struct{}

// SelectController creates a PlayerSelector that returns the effect's controller.
func SelectController() PlayerSelector { return selectController{} }
func (s selectController) Select(_ GameReader, _, controller uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	return []uuid.UUID{controller}
}
func (s selectController) Text() string { return "controller" }

// selectActivePlayer returns the active player (whose turn it is).
type selectActivePlayer struct{}

// SelectActivePlayer creates a PlayerSelector that returns the active player (whose turn it is).
func SelectActivePlayer() PlayerSelector { return selectActivePlayer{} }
func (s selectActivePlayer) Select(g GameReader, _, _ uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	return []uuid.UUID{g.ActivePlayerObj().PlayerID()}
}
func (s selectActivePlayer) Text() string { return "active player" }

// selectEachPlayer returns all players.
type selectEachPlayer struct{}

// SelectEachPlayer creates a PlayerSelector that returns all players in the game.
func SelectEachPlayer() PlayerSelector { return selectEachPlayer{} }
func (s selectEachPlayer) Select(g GameReader, _, _ uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	players := g.AllPlayers()
	ids := make([]uuid.UUID, len(players))
	for i, p := range players {
		ids[i] = p.PlayerID()
	}
	return ids
}
func (s selectEachPlayer) Text() string { return "each player" }

// selectEachOpponent returns all players other than the controller.
type selectEachOpponent struct{}

// SelectEachOpponent creates a PlayerSelector that returns all opponents of the controller.
func SelectEachOpponent() PlayerSelector { return selectEachOpponent{} }
func (s selectEachOpponent) Select(g GameReader, _, controller uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	var ids []uuid.UUID
	for _, p := range g.AllPlayers() {
		if p.PlayerID() != controller {
			ids = append(ids, p.PlayerID())
		}
	}
	return ids
}
func (s selectEachOpponent) Text() string { return "each opponent" }

// selectAttachedController follows source → AttachedTo → Controller.
type selectAttachedController struct{}

// SelectAttachedController creates a PlayerSelector that returns the controller of the permanent
// the source is attached to (for aura-based effects like Psychic Venom).
func SelectAttachedController() PlayerSelector { return selectAttachedController{} }
func (s selectAttachedController) Select(g GameReader, sourceID, _ uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	src := g.FindPermanent(sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target == nil {
		return nil
	}
	return []uuid.UUID{target.Controller}
}
func (s selectAttachedController) Text() string { return "enchanted creature's controller" }

// selectEventController reads targets[0] as a player ID (for event-based triggers).
type selectEventController struct{}

// SelectEventController creates a PlayerSelector that reads targets[0] as a player ID,
// used for event-based triggers that pass the relevant player through the target list.
func SelectEventController() PlayerSelector { return selectEventController{} }
func (s selectEventController) Select(_ GameReader, _, _ uuid.UUID, targets []uuid.UUID) []uuid.UUID {
	if len(targets) == 0 {
		return nil
	}
	return []uuid.UUID{targets[0]}
}
func (s selectEventController) Text() string { return "that player" }
