package mage

import (
	"fmt"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// Effect represents a one-shot effect that resolves. Effects are inert data
// describing what should happen; the executor (ExecuteEffect) dispatches on
// the concrete type to perform mutations. Use ApplyEffect to run an Effect
// against a *Game.
type Effect interface {
	Text() string
	Properties() EffectProperties
}

type TargetedEffect interface {
	Text() string
	Properties() EffectProperties
	Targeting(PlayerSelector) TargetedEffect
}

// funcEffect wraps an anonymous function as an Effect. Use FuncEffect to create
// one-off effects inline in card definitions without needing a dedicated struct.
type funcEffect struct {
	text  string
	props EffectProperties
	fn    func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error
}

// FuncEffect creates an Effect from an anonymous function. This is ideal for
// card-specific effects that are used by only one card and don't warrant a
// dedicated type. The text parameter is used for Text() (rules text display).
// The props parameter allows callers to declare AI-visible properties (outcome, damage, etc.).
func FuncEffect(text string, props EffectProperties, fn func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error) Effect {
	return &funcEffect{text: text, props: props, fn: fn}
}

func (e *funcEffect) Text() string                 { return e.text }
func (e *funcEffect) Properties() EffectProperties { return e.props }

// compositeEffect applies multiple effects in sequence.
type compositeEffect struct {
	effects []Effect
	text    string
}

// CompositeEffects creates an effect that applies multiple effects in sequence.
func CompositeEffects(text string, effects ...Effect) Effect {
	return &compositeEffect{effects: effects, text: text}
}

func (e *compositeEffect) Text() string                 { return e.text }
func (e *compositeEffect) Properties() EffectProperties { return EffectProperties{} }

// ApplyEffect runs an Effect by constructing an EffectContext and dispatching
// through the executor. This is the canonical entry point for Effect resolution
// from engine code that holds a *Game directly.
func ApplyEffect(g *Game, e Effect, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	ctx := &EffectContext{
		Game:       g,
		SourceID:   sourceID,
		Controller: controller,
		Targets:    targets,
		Vars:       make(map[string]any),
	}
	return ExecuteEffect(ctx, e)
}

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
	LifeGain       int  // life gained by controller; 0 if not a life-gain effect
	PowerBoost     int  // power boost for target; 0 if not a boost effect
	ToughnessBoost int  // toughness boost for target; 0 if not a boost effect
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
// It receives a *Game (read-only view) since value resolution never mutates state.
// targets contains the resolved target IDs from the spell/ability context.
type ValueSource interface {
	Resolve(g GameReader, sourceID, controller uuid.UUID, targets []uuid.UUID) int
	Text() string
}

// PlayerSelector picks one or more players for an effect.
// It receives a *Game (read-only view) since player selection never mutates state.
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
func Fixed(n int) ValueSource                                                { return fixedValue{n: n} }
func (v fixedValue) Resolve(_ GameReader, _, _ uuid.UUID, _ []uuid.UUID) int { return v.n }
func (v fixedValue) Text() string                                            { return fmt.Sprintf("%d", v.n) }

// xValue is a ValueSource that reads g.CurrentX.
type xValue struct{}

// XValue creates a ValueSource that reads the X value from the current spell/ability (g.CurrentX).
func XValue() ValueSource                                                { return xValue{} }
func (v xValue) Resolve(g GameReader, _, _ uuid.UUID, _ []uuid.UUID) int { return g.XValue() }
func (v xValue) Text() string                                            { return "X" }

// mulValue multiplies two ValueSources.
type mulValue struct {
	a, b ValueSource
}

// Mul creates a ValueSource that returns a.Resolve() * b.Resolve().
func Mul(a, b ValueSource) ValueSource { return mulValue{a: a, b: b} }
func (v mulValue) Resolve(g GameReader, sourceID, controller uuid.UUID, targets []uuid.UUID) int {
	return v.a.Resolve(g, sourceID, controller, targets) * v.b.Resolve(g, sourceID, controller, targets)
}
func (v mulValue) Text() string { return v.a.Text() + " * " + v.b.Text() }

// addValue adds two ValueSources.
type addValue struct {
	a, b ValueSource
}

// Add creates a ValueSource that returns a.Resolve() + b.Resolve().
func Add(a, b ValueSource) ValueSource { return addValue{a: a, b: b} }
func (v addValue) Resolve(g GameReader, sourceID, controller uuid.UUID, targets []uuid.UUID) int {
	return v.a.Resolve(g, sourceID, controller, targets) + v.b.Resolve(g, sourceID, controller, targets)
}
func (v addValue) Text() string { return v.a.Text() + " + " + v.b.Text() }

// eventAmountValue is a ValueSource that reads the triggering event's amount
// (e.g. damage dealt). Used for "gain that much life" / "deal that much damage" triggers.
type eventAmountValue struct{}

// EventAmountValue creates a ValueSource that reads g.EventAmount().
func EventAmountValue() ValueSource { return eventAmountValue{} }
func (v eventAmountValue) Resolve(g GameReader, _, _ uuid.UUID, _ []uuid.UUID) int {
	return g.EventAmount()
}
func (v eventAmountValue) Text() string { return "that much" }

// playerLifeValue reads the controller's current life total.
type playerLifeValue struct{}

// PlayerLifeValue creates a ValueSource returning the controller's life total.
func PlayerLifeValue() ValueSource { return playerLifeValue{} }
func (v playerLifeValue) Resolve(g GameReader, _, controller uuid.UUID, _ []uuid.UUID) int {
	p := g.GetPlayer(controller)
	if p == nil {
		return 0
	}
	return p.Life()
}
func (v playerLifeValue) Text() string { return "your life total" }

// halfRoundUpValue returns (inner + 1) / 2.
type halfRoundUpValue struct {
	inner ValueSource
}

// HalfRoundUp creates a ValueSource returning ceil(inner / 2).
func HalfRoundUp(inner ValueSource) ValueSource { return halfRoundUpValue{inner: inner} }
func (v halfRoundUpValue) Resolve(g GameReader, sourceID, controller uuid.UUID, targets []uuid.UUID) int {
	n := v.inner.Resolve(g, sourceID, controller, targets)
	return (n + 1) / 2
}
func (v halfRoundUpValue) Text() string { return "half " + v.inner.Text() + " rounded up" }

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
func (v countBattlefieldValue) Resolve(g GameReader, sourceID, controller uuid.UUID, targets []uuid.UUID) int {
	if v.who == nil {
		return g.CountBattlefield(v.filter)
	}
	total := 0
	for _, pid := range v.who.Select(g, sourceID, controller, targets) {
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

// untappedLandsAtTurnStartValue resolves to the number of untapped lands the
// selected player controlled at the start of the current turn (snapshot taken
// before the untap step).
type untappedLandsAtTurnStartValue struct {
	who PlayerSelector
}

// UntappedLandsAtTurnStart creates a ValueSource that reads the snapshot of
// untapped lands the selected player controlled at the very start of the
// current turn — before the untap step ran. Used by Power Surge.
func UntappedLandsAtTurnStart(who PlayerSelector) ValueSource {
	return untappedLandsAtTurnStartValue{who: who}
}

func (v untappedLandsAtTurnStartValue) Resolve(g GameReader, sourceID, controller uuid.UUID, targets []uuid.UUID) int {
	total := 0
	for _, pid := range v.who.Select(g, sourceID, controller, targets) {
		total += g.UntappedLandsAtTurnStart(pid)
	}
	return total
}
func (v untappedLandsAtTurnStartValue) Text() string {
	return "the number of untapped lands " + v.who.Text() + " controlled at the beginning of this turn"
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
func (v countZoneValue) Resolve(g GameReader, sourceID, controller uuid.UUID, targets []uuid.UUID) int {
	total := 0
	for _, pid := range v.who.Select(g, sourceID, controller, targets) {
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

// topOfLibraryManaValue resolves to the mana value of the top card of the
// selected player's library. The library is not modified; this matches the
// engine's existing convention that "look at" and "reveal" are read-only at
// the rules layer (see RevealTopN). Returns 0 if the library is empty.
type topOfLibraryManaValue struct {
	who PlayerSelector
}

// TopOfLibraryManaValue returns a ValueSource equal to the mana value of the
// top card of the selected player's library. Used for spells like Riddle of
// Lightning ("reveal the top card of your library; deals damage equal to that
// card's mana value to that permanent or player").
func TopOfLibraryManaValue(who PlayerSelector) ValueSource {
	return topOfLibraryManaValue{who: who}
}
func (v topOfLibraryManaValue) Resolve(g GameReader, sourceID, controller uuid.UUID, targets []uuid.UUID) int {
	ids := v.who.Select(g, sourceID, controller, targets)
	if len(ids) == 0 {
		return 0
	}
	p := g.GetPlayer(ids[0])
	if p == nil {
		return 0
	}
	lib := p.Library()
	if len(lib) == 0 {
		return 0
	}
	return lib[0].ManaCost().CMC()
}
func (v topOfLibraryManaValue) Text() string {
	return "the mana value of the top card of " + v.who.Text() + "'s library"
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

// selectDefendingPlayer returns the defending player of the source's combat
// group — i.e. the player being attacked (or the controller of the attacked
// planeswalker). Falls back to the non-active player if the source isn't in
// a combat group, which keeps it safe to use on triggers that fire just
// outside combat resolution.
type selectDefendingPlayer struct{}

// SelectDefendingPlayer creates a PlayerSelector that returns the defending
// player of the source's combat group. Use this for "defending player ..."
// triggers on attacking creatures (Mindstab Thrull, Necrite, etc.).
func SelectDefendingPlayer() PlayerSelector { return selectDefendingPlayer{} }
func (s selectDefendingPlayer) Select(g GameReader, sourceID, _ uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	if group := g.CombatGroupFor(sourceID); group != nil && group.DefenderID != uuid.Nil {
		if p := g.GetPlayer(group.DefenderID); p != nil {
			return []uuid.UUID{p.PlayerID()}
		}
		// Planeswalker defender: return its controller.
		if perm := g.FindPermanent(group.DefenderID); perm != nil {
			return []uuid.UUID{perm.Controller}
		}
	}
	if nap := g.NonActivePlayerObj(); nap != nil {
		return []uuid.UUID{nap.PlayerID()}
	}
	return nil
}
func (s selectDefendingPlayer) Text() string { return "defending player" }

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

// SelectTargetPlayer creates a PlayerSelector that reads targets[0] as a player ID.
// Use for targeted spells that target a player.
func SelectTargetPlayer() PlayerSelector { return selectEventController{} }

// selectTargetPermanentController resolves targets[0] as a permanent and
// returns its controller. Used for triggers where the target is a permanent
// and the effect needs to affect that permanent's controller.
type selectTargetPermanentController struct{}

func SelectTargetPermanentController() PlayerSelector { return selectTargetPermanentController{} }
func (s selectTargetPermanentController) Select(g GameReader, _, _ uuid.UUID, targets []uuid.UUID) []uuid.UUID {
	if len(targets) == 0 {
		return nil
	}
	if game, ok := g.(*Game); ok {
		if view := game.LookupObject(targets[0]); view != nil {
			return []uuid.UUID{view.ViewController()}
		}
	}
	if perm := g.FindPermanent(targets[0]); perm != nil {
		return []uuid.UUID{perm.Controller}
	}
	return nil
}
func (s selectTargetPermanentController) Text() string { return "that permanent's controller" }
