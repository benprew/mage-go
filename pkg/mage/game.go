package mage

import (
	. "github.com/mage/mage/pkg/mage/core"
	"errors"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
)

// Sentinel errors for common failure conditions.
var (
	ErrPlayerNotFound    = errors.New("player not found")
	ErrSourceNotFound    = errors.New("source not found on battlefield")
	ErrPermanentNotFound = errors.New("permanent not found")
	ErrCardNotInHand     = errors.New("card not found in hand")
	ErrSourceTapped      = errors.New("source is already tapped")
	ErrNoCreature        = errors.New("no creature to sacrifice")
	ErrSorcerySpeed      = errors.New("can only activate at sorcery speed")
)

// ExiledCard tracks a card in exile along with metadata about why it was exiled.
type ExiledCard struct {
	Card     Card
	ExiledBy uuid.UUID              // ID of the permanent/spell that caused the exile
	Counters map[CounterType]int    // noted counter state (e.g. Tawnos's Coffin)
	Owner    uuid.UUID              // original controller when exiled
}

// Game is the central game state and engine.
type Game struct {
	Players     []Player
	Battlefield []*Permanent
	Exile       []ExiledCard // exile zone with metadata
	Stack       *Stack
	Combat      *Combat
	Effects     *EffectManager

	Turn         int
	Step         PhaseStep
	ActivePlayer int // index into Players

	// Event handling
	pendingTriggers []*pendingTrigger

	// Extra turns
	ExtraTurns []uuid.UUID // player IDs who get extra turns

	// X value for the currently resolving spell
	CurrentX int

	// Chosen mode for the currently resolving modal spell (0-indexed)
	CurrentMode int

	// Card currently being resolved (set during ResolveStackObject)
	ResolvingCard Card

	// Interactive play tracking
	LandsPlayedThisTurn int

	// Damage tracking: maps target permanent ID -> set of source permanent IDs that dealt damage this turn
	DamageDealtBy map[uuid.UUID]map[uuid.UUID]bool

	// Player damage tracking: maps player ID -> total damage taken this turn
	DamageTakenThisTurn map[uuid.UUID]int

	// Artifact damage tracking: maps player ID -> artifact damage taken this turn
	ArtifactDamageTakenThisTurn map[uuid.UUID]int

	// Artifact mana restriction: players who have activated artifact-only mana sources
	ArtifactManaOnly map[uuid.UUID]bool

	// Creature mana restriction: players who have creature-only mana (Metamorphosis)
	CreatureManaOnly map[uuid.UUID]bool

	// Creatures that attacked this turn (survives combat reset for end-of-turn checks)
	AttackedThisTurn map[uuid.UUID]bool

	// Creature deaths this turn (total count across all players)
	CreatureDeathsThisTurn int

	// Targets of the spell currently being resolved (for ETB copy effects)
	ResolvingTargets []uuid.UUID

	// Delayed triggers
	delayedTriggers []*DelayedTrigger

	// Coin flip results (for test determinism; popped in order)
	CoinFlipResults []bool

	// Priority handler — called when a player receives priority.
	// If nil, the engine drains the stack atomically (legacy behavior).
	OnPriority PriorityHandler

	// AfterPriorityAction is called after a non-pass priority action is executed.
	// Used by the interactive layer for logging and display.
	AfterPriorityAction func(g *Game, playerIdx int, action PriorityAction)

	// BeforeStackResolve is called before the top of the stack is resolved
	// during a priority round. Used by the interactive layer for logging.
	BeforeStackResolve func(g *Game)

	// Control flags
	stopped bool
}

// DelayedTrigger represents a one-shot triggered ability that fires when
// a specific event occurs (e.g., "destroy this creature at end of turn").
type DelayedTrigger struct {
	EventType    EventType
	TargetID     uuid.UUID
	Effects      []Effect
	SourceID     uuid.UUID
	Controller   uuid.UUID
	MatchEventID  uuid.UUID // if set, only fire when evt.SourceID matches
	MatchPlayerID uuid.UUID // if set, only fire when evt.PlayerID matches
}

type pendingTrigger struct {
	ability    TriggeredAbility
	event      *GameEvent
	sourceID   uuid.UUID
	controller uuid.UUID
}

// NewGame creates a new 2-player game.
func NewGame(playerA, playerB Player) *Game {
	return &Game{
		Players:                    []Player{playerA, playerB},
		Stack:                      NewStack(),
		Combat:                     NewCombat(),
		Effects:                    NewEffectManager(),
		Turn:                       1,
		DamageDealtBy:              make(map[uuid.UUID]map[uuid.UUID]bool),
		DamageTakenThisTurn:        make(map[uuid.UUID]int),
		ArtifactDamageTakenThisTurn: make(map[uuid.UUID]int),
		AttackedThisTurn:           make(map[uuid.UUID]bool),
		ArtifactManaOnly:           make(map[uuid.UUID]bool),
		CreatureManaOnly:           make(map[uuid.UUID]bool),
	}
}

// GetPlayer returns the player with the given ID.
func (g *Game) GetPlayer(id uuid.UUID) Player {
	for _, p := range g.Players {
		if p.PlayerID() == id {
			return p
		}
	}
	return nil
}

// GetOpponent returns the other player.
func (g *Game) GetOpponent(id uuid.UUID) Player {
	for _, p := range g.Players {
		if p.PlayerID() != id {
			return p
		}
	}
	return nil
}

// ActivePlayerObj returns the currently active player.
func (g *Game) ActivePlayerObj() Player {
	return g.Players[g.ActivePlayer]
}

// NonActivePlayerObj returns the non-active player.
func (g *Game) NonActivePlayerObj() Player {
	return g.Players[(g.ActivePlayer+1)%2]
}

// FindPermanent finds a permanent by ID on the battlefield.
// Phased-out permanents are invisible.
func (g *Game) FindPermanent(id uuid.UUID) *Permanent {
	for _, p := range g.Battlefield {
		if p.PhasedOut {
			continue
		}
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// FindPermanentIncludingPhased finds a permanent by ID even if phased out.
func (g *Game) FindPermanentIncludingPhased(id uuid.UUID) *Permanent {
	for _, p := range g.Battlefield {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// FindPermanentByName finds a permanent by name on the battlefield (first match).
// Phased-out permanents are invisible.
func (g *Game) FindPermanentByName(name string, controller uuid.UUID) *Permanent {
	for _, p := range g.Battlefield {
		if p.PhasedOut {
			continue
		}
		if p.Name() == name && p.Controller == controller {
			return p
		}
	}
	return nil
}

// AnyBattlefield returns true if any permanent on the battlefield matches f.
// Phased-out permanents are invisible.
func (g *Game) AnyBattlefield(f PermanentFilter) bool {
	for _, p := range g.Battlefield {
		if p.PhasedOut {
			continue
		}
		if f.Match(p, g) {
			return true
		}
	}
	return false
}

// FilterBattlefield returns all permanents on the battlefield matching f.
// Phased-out permanents are invisible.
func (g *Game) FilterBattlefield(f PermanentFilter) []*Permanent {
	var result []*Permanent
	for _, p := range g.Battlefield {
		if p.PhasedOut {
			continue
		}
		if f.Match(p, g) {
			result = append(result, p)
		}
	}
	return result
}

// CountBattlefield returns the number of permanents on the battlefield matching f.
// Phased-out permanents are invisible.
func (g *Game) CountBattlefield(f PermanentFilter) int {
	n := 0
	for _, p := range g.Battlefield {
		if p.PhasedOut {
			continue
		}
		if f.Match(p, g) {
			n++
		}
	}
	return n
}

// FindCardAnywhere finds a card by ID anywhere in the game.
func (g *Game) FindCardAnywhere(id uuid.UUID) Card {
	for _, p := range g.Battlefield {
		if p.ID() == id {
			return p.Card
		}
	}
	// Search the stack (spells that have been cast but not yet resolved)
	for _, obj := range g.Stack.Objects() {
		if obj.Card != nil && obj.Card.ID() == id {
			return obj.Card
		}
	}
	for _, pl := range g.Players {
		for _, c := range pl.Hand() {
			if c.ID() == id {
				return c
			}
		}
		for _, c := range pl.Graveyard() {
			if c.ID() == id {
				return c
			}
		}
	}
	for _, ec := range g.Exile {
		if ec.Card.ID() == id {
			return ec.Card
		}
	}
	return nil
}

// FindCardForDamageSource finds the Card associated with a damage source ID.
// This looks at permanents, graveyard, and the currently resolving card.
func (g *Game) FindCardForDamageSource(sourceID uuid.UUID) Card {
	perm := g.FindPermanent(sourceID)
	if perm != nil {
		return perm.Card
	}
	if g.ResolvingCard != nil && g.ResolvingCard.ID() == sourceID {
		return g.ResolvingCard
	}
	return g.FindCardAnywhere(sourceID)
}

// TryPayCostFromLands attempts to pay a mana cost by tapping untapped lands
// controlled by the player. Returns true if the cost was fully paid.
// FlipCoin simulates a coin flip. Returns true for "win" (heads).
// If CoinFlipResults is non-empty, pops from the front (for test determinism).
func (g *Game) FlipCoin(playerID uuid.UUID) bool {
	if len(g.CoinFlipResults) > 0 {
		result := g.CoinFlipResults[0]
		g.CoinFlipResults = g.CoinFlipResults[1:]
		return result
	}
	return rand.Intn(2) == 0
}

func (g *Game) TryPayCostFromLands(playerID uuid.UUID, manaCostStr string) bool {
	cost := ParseManaCost(manaCostStr)

	// Collect untapped lands controlled by the player
	var lands []*Permanent
	for _, p := range g.Battlefield {
		if p.Controller == playerID && !p.Tapped && p.HasType(TypeLand) {
			lands = append(lands, p)
		}
	}

	used := make(map[uuid.UUID]bool)

	// Pay colored costs first
	colorCosts := []struct {
		amount  int
		subtype string
	}{
		{cost.White, "Plains"},
		{cost.Blue, "Island"},
		{cost.Black, "Swamp"},
		{cost.Red, "Mountain"},
		{cost.Green, "Forest"},
	}

	for _, cc := range colorCosts {
		for i := 0; i < cc.amount; i++ {
			found := false
			for _, land := range lands {
				if !used[land.ID()] && land.HasSubType(cc.subtype) {
					used[land.ID()] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Pay generic cost with any remaining untapped land
	for i := 0; i < cost.Generic; i++ {
		found := false
		for _, land := range lands {
			if !used[land.ID()] {
				used[land.ID()] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Actually tap the selected lands
	for _, land := range lands {
		if used[land.ID()] {
			land.Tapped = true
		}
	}

	return true
}

// PutOnBattlefield puts a card onto the battlefield under the given controller.
func (g *Game) PutOnBattlefield(card Card, controller uuid.UUID) *Permanent {
	perm := NewPermanent(card, controller)
	perm.TurnControlGained = g.Turn

	// Set ability sources and controllers
	for _, a := range perm.RuntimeAbilities {
		a.SetSource(perm.ID())
		a.SetController(controller)
	}

	// EntersTapped keyword check — consumed on entry, attr cleared immediately after.
	if perm.HasKeyword(EntersTapped) {
		perm.Tapped = true
		perm.RevokeBaseAttr(EntersTapped)
	}

	// Add X counters if configured (replacement effect, not a trigger)
	for _, a := range perm.RuntimeAbilities {
		if xc, ok := a.(*EntersWithXCountersAbility); ok && g.CurrentX > 0 {
			perm.AddCounter(xc.CounterType, g.CurrentX)
			break
		}
	}

	// Copy creature on ETB (Vesuvan Doppelganger): copy target creature's P/T and keywords
	for _, a := range perm.RuntimeAbilities {
		if _, ok := a.(*CopyCreatureOnETBAbility); ok && len(g.ResolvingTargets) > 0 {
			target := g.FindPermanent(g.ResolvingTargets[0])
			if target != nil {
				g.Effects.AddCopyEffect(perm.ID(), target)
			}
			break
		}
	}

	g.Battlefield = append(g.Battlefield, perm)

	// Register continuous effects from static abilities
	for _, a := range perm.RuntimeAbilities {
		if sa, ok := a.(*StaticAbilityHolder); ok {
			for _, e := range sa.Effects {
				// Set the source ID on the continuous effect
				g.setEffectSource(e, perm.ID())
				g.Effects.Add(e)
			}
		}
	}

	// Run unconditional ETB effects (e.g. Primal Clay mode choice)
	for _, a := range perm.RuntimeAbilities {
		if etb, ok := a.(*ETBEffectAbility); ok {
			_ = etb.Effect.Apply(g, perm.ID(), controller, nil)
		}
	}

	// Run ETB-with-targets effects (e.g. Oubliette exile on entry)
	for _, a := range perm.RuntimeAbilities {
		if etb, ok := a.(*ETBWithTargetsAbility); ok && len(g.ResolvingTargets) > 0 {
			_ = etb.Effect.Apply(g, perm.ID(), controller, g.ResolvingTargets)
			break
		}
	}

	g.Effects.Apply(g)

	g.FireEvent(GameEvent{
		Type:     EvtEntersBattlefield,
		SourceID: perm.ID(),
		PlayerID: controller,
	})

	return perm
}

// setEffectSource sets the source ID on a continuous effect.
func (g *Game) setEffectSource(e ContinuousEffect, id uuid.UUID) {
	e.SetSourceID(id)
}

// TurnFaceUp flips a face-down permanent face up, restoring its original characteristics.
func (g *Game) TurnFaceUp(perm *Permanent) {
	if !perm.FaceDown {
		return
	}
	perm.FaceDown = false
	perm.BasePTOverride = nil
	// Restore original abilities from the card
	perm.RuntimeAbilities = nil
	for _, a := range perm.Card.Abilities() {
		cp := a
		perm.RuntimeAbilities = append(perm.RuntimeAbilities, cp)
	}
	// Set ability sources
	for _, a := range perm.RuntimeAbilities {
		a.SetSource(perm.ID())
		a.SetController(perm.Controller)
	}
	// Register continuous effects from static abilities
	for _, a := range perm.RuntimeAbilities {
		if sa, ok := a.(*StaticAbilityHolder); ok {
			for _, e := range sa.Effects {
				g.setEffectSource(e, perm.ID())
				g.Effects.Add(e)
			}
		}
	}
	g.Effects.Apply(g)
}

// RemoveFromBattlefield removes a permanent and handles cleanup.
func (g *Game) RemoveFromBattlefield(perm *Permanent) {
	// Capture abilities before removal (for "leaves battlefield" triggers on self)
	selfAbilities := make([]Ability, len(perm.RuntimeAbilities))
	copy(selfAbilities, perm.RuntimeAbilities)
	permID := perm.ID()
	controller := perm.Controller

	// Remove continuous effects sourced from this permanent
	g.Effects.Remove(perm.ID())

	// Detach anything attached to this permanent
	for _, attID := range perm.Attachments {
		att := g.FindPermanent(attID)
		if att != nil {
			att.AttachedTo = uuid.Nil
		}
	}

	// If this was attached to something, remove it from that thing's attachments
	if perm.IsAttached() {
		host := g.FindPermanent(perm.AttachedTo)
		if host != nil {
			filtered := host.Attachments[:0]
			for _, id := range host.Attachments {
				if id != perm.ID() {
					filtered = append(filtered, id)
				}
			}
			host.Attachments = filtered
		}
	}

	// Remove from battlefield
	for i, p := range g.Battlefield {
		if p.ID() == perm.ID() {
			g.Battlefield = append(g.Battlefield[:i], g.Battlefield[i+1:]...)
			break
		}
	}

	g.Effects.Apply(g)

	evt := GameEvent{
		Type:     EvtLeavesBattlefield,
		SourceID: permID,
		PlayerID: controller,
	}
	g.FireEvent(evt)
	// Also check the removed permanent's own triggers (since it's no longer on battlefield)
	g.checkAbilitiesForEvent(selfAbilities, &evt, permID, controller)
}

// DestroyPermanent destroys a permanent (sends to graveyard).
func (g *Game) DestroyPermanent(perm *Permanent) {
	if perm.HasKeyword(Indestructible) {
		return
	}
	// Regeneration replaces destruction: tap, remove damage, remove from combat
	if !perm.HasKeyword(CantRegenerate) && g.Effects.Damage.ConsumeRegenerationShield(perm.ID()) {
		perm.Tapped = true
		perm.Damage = 0
		// Remove from combat if attacking/blocking
		g.Combat.RemoveFromCombat(perm.ID())
		return
	}
	controller := perm.Controller
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}

	isCreature := perm.HasType(TypeCreature)
	permID := perm.ID()
	card := perm.Card

	// Capture abilities before removal (for "leaves battlefield" / "dies" triggers on self)
	selfAbilities := make([]Ability, len(perm.RuntimeAbilities))
	copy(selfAbilities, perm.RuntimeAbilities)

	// Handle attached auras - they go to graveyard
	attachments := make([]uuid.UUID, len(perm.Attachments))
	copy(attachments, perm.Attachments)

	g.RemoveFromBattlefield(perm)

	p := g.GetPlayer(owner)
	if p != nil {
		p.AddToGraveyard(card)
	}

	// Check the destroyed permanent's own abilities for self-referencing triggers
	graveyardEvt := GameEvent{
		Type:     EvtPutIntoGraveyardFromBattlefield,
		SourceID: permID,
		PlayerID: controller,
	}
	g.FireEvent(graveyardEvt)
	g.checkAbilitiesForEvent(selfAbilities, &graveyardEvt, permID, controller)

	if isCreature {
		g.CreatureDeathsThisTurn++
		diedEvt := GameEvent{
			Type:     EvtCreatureDied,
			SourceID: permID,
			PlayerID: controller,
		}
		g.FireEvent(diedEvt)
		g.checkAbilitiesForEvent(selfAbilities, &diedEvt, permID, controller)
	}

	// Handle attached auras going to graveyard
	for _, attID := range attachments {
		att := g.FindPermanent(attID)
		if att == nil {
			continue
		}
		if att.HasSubType("Aura") {
			g.DestroyPermanent(att)
		}
		// Equipment stays on the battlefield (already detached)
	}
}

// checkAbilitiesForEvent checks a set of abilities (from a removed permanent) for triggers.
func (g *Game) checkAbilitiesForEvent(abilities []Ability, evt *GameEvent, sourceID, controller uuid.UUID) {
	for _, a := range abilities {
		ta, ok := a.(TriggeredAbility)
		if !ok {
			continue
		}
		if !ta.CheckEventType(evt.Type) {
			continue
		}
		if ta.CheckTrigger(evt, g) {
			g.pendingTriggers = append(g.pendingTriggers, &pendingTrigger{
				ability:    ta,
				event:      evt,
				sourceID:   sourceID,
				controller: controller,
			})
		}
	}
}

// PutPermanentIntoGraveyard puts a permanent into its owner's graveyard without
// destroying it. This bypasses indestructible and regeneration. Used by SBAs
// (e.g., 0-toughness creatures) and other rules that move permanents to the
// graveyard without destruction.
func (g *Game) PutPermanentIntoGraveyard(perm *Permanent) {
	controller := perm.Controller
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}

	isCreature := perm.HasType(TypeCreature)
	permID := perm.ID()
	card := perm.Card

	// Capture abilities before removal (for "leaves battlefield" / "dies" triggers on self)
	selfAbilities := make([]Ability, len(perm.RuntimeAbilities))
	copy(selfAbilities, perm.RuntimeAbilities)

	g.RemoveFromBattlefield(perm)

	p := g.GetPlayer(owner)
	if p != nil {
		p.AddToGraveyard(card)
	}

	graveyardEvt := GameEvent{
		Type:     EvtPutIntoGraveyardFromBattlefield,
		SourceID: permID,
		PlayerID: controller,
	}
	g.FireEvent(graveyardEvt)
	g.checkAbilitiesForEvent(selfAbilities, &graveyardEvt, permID, controller)

	if isCreature {
		g.CreatureDeathsThisTurn++
		diedEvt := GameEvent{
			Type:     EvtCreatureDied,
			SourceID: permID,
			PlayerID: controller,
		}
		g.FireEvent(diedEvt)
		g.checkAbilitiesForEvent(selfAbilities, &diedEvt, permID, controller)
	}
}

// Sacrifice sacrifices a permanent (like destroy but doesn't check indestructible).
func (g *Game) Sacrifice(perm *Permanent) {
	controller := perm.Controller
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}

	isCreature := perm.HasType(TypeCreature)
	permID := perm.ID()
	card := perm.Card

	g.RemoveFromBattlefield(perm)

	p := g.GetPlayer(owner)
	if p != nil {
		p.AddToGraveyard(card)
	}

	g.FireEvent(GameEvent{
		Type:     EvtPutIntoGraveyardFromBattlefield,
		SourceID: permID,
		PlayerID: controller,
		Flag:     true, // Flag=true means this was a sacrifice (not destroy)
	})

	if isCreature {
		g.CreatureDeathsThisTurn++
		g.FireEvent(GameEvent{
			Type:     EvtCreatureDied,
			SourceID: permID,
			PlayerID: controller,
		})
	}
}

// PlayerGainLife handles life gain with replacement effects (Lich).
func (g *Game) PlayerGainLife(p Player, amount int) {
	if amount <= 0 {
		return
	}
	if g.Effects.Rules.IsLichActive(g, p.PlayerID()) {
		// Lich replacement: draw cards instead of gaining life
		for i := 0; i < amount; i++ {
			p.DrawCard()
		}
		return
	}
	p.GainLife(amount)
}

// sacrificePermanents sacrifices N nontoken permanents a player controls (for Lich).
func (g *Game) sacrificePermanents(playerID uuid.UUID, count int) {
	sacrificed := 0
	for sacrificed < count {
		var target *Permanent
		for _, p := range g.Battlefield {
			if p.Controller == playerID {
				target = p
				break
			}
		}
		if target == nil {
			break
		}
		g.Sacrifice(target)
		sacrificed++
	}
}

// ExilePermanent removes a permanent from the battlefield to exile.
func (g *Game) ExilePermanent(perm *Permanent) {
	g.ExilePermanentBy(perm, uuid.Nil)
}

// ExilePermanentBy removes a permanent from the battlefield to exile,
// recording the source that caused the exile.
func (g *Game) ExilePermanentBy(perm *Permanent, exiledBy uuid.UUID) {
	card := perm.Card
	g.RemoveFromBattlefield(perm)
	g.Exile = append(g.Exile, ExiledCard{Card: card, ExiledBy: exiledBy})
}

// FindExiledCard finds an exiled card by its ID.
func (g *Game) FindExiledCard(cardID uuid.UUID) *ExiledCard {
	for i := range g.Exile {
		if g.Exile[i].Card.ID() == cardID {
			return &g.Exile[i]
		}
	}
	return nil
}

// RemoveFromExile removes a card from exile by ID and returns it.
func (g *Game) RemoveFromExile(cardID uuid.UUID) (Card, bool) {
	for i, ec := range g.Exile {
		if ec.Card.ID() == cardID {
			g.Exile = append(g.Exile[:i], g.Exile[i+1:]...)
			return ec.Card, true
		}
	}
	return nil, false
}

// RemoveExiledCardBySource removes all exiled cards with the given ExiledBy ID
// and returns them. Used by Tawnos's Coffin and similar cards.
func (g *Game) RemoveExiledCardBySource(exiledBy uuid.UUID) []ExiledCard {
	var found []ExiledCard
	remaining := g.Exile[:0]
	for _, ec := range g.Exile {
		if ec.ExiledBy == exiledBy {
			found = append(found, ec)
		} else {
			remaining = append(remaining, ec)
		}
	}
	g.Exile = remaining
	return found
}

// CounterSpellOnStack removes a spell from the stack by its source ID.
// The countered spell's card goes to its owner's graveyard.
func (g *Game) CounterSpellOnStack(spellID uuid.UUID) {
	obj := g.Stack.RemoveBySourceID(spellID)
	if obj != nil && obj.Card != nil {
		owner := g.GetPlayer(obj.Card.Owner())
		if owner != nil {
			owner.AddToGraveyard(obj.Card)
		}
	}
}

// DealDamageToPlayer deals damage to a player.
func (g *Game) DealDamageToPlayer(p Player, amount int, sourceID uuid.UUID) {
	if amount <= 0 {
		return
	}
	// Check color prevention (Circle of Protection)
	sourceCard := g.FindCardForDamageSource(sourceID)
	if g.Effects.Damage.CheckColorPrevention(p.PlayerID(), sourceCard) {
		return // all damage from this source prevented
	}
	// Check type prevention (Circle of Protection: Artifacts)
	if g.Effects.Damage.CheckTypePrevention(p.PlayerID(), sourceCard) {
		return // all damage from this source prevented
	}
	// Apply damage prevention shield (also used for player)
	if prevented := g.Effects.Damage.PreventDamage(p.PlayerID(), amount); prevented > 0 {
		amount -= prevented
		// If there's a "reverse damage" effect, gain life equal to prevented
		if g.Effects.Damage.HasReverseDamageShield(p.PlayerID()) {
			p.GainLife(prevented)
			g.Effects.Damage.ClearReverseDamageShield(p.PlayerID())
		}
	}
	if amount <= 0 {
		return
	}
	// Martyrs of Korlis: redirect artifact damage to creature
	if sourceCard != nil && sourceCard.HasType(TypeArtifact) {
		if redirectID := g.Effects.Damage.GetArtifactDamageRedirect(p.PlayerID()); redirectID != uuid.Nil {
			redirectPerm := g.FindPermanent(redirectID)
			if redirectPerm != nil {
				g.DealDamageToPermanent(redirectPerm, amount, sourceID)
				return
			}
		}
	}
	// Personal Incarnation: redirect all damage to the creature instead
	if redirectID := g.Effects.Damage.GetPlayerDamageRedirect(p.PlayerID()); redirectID != uuid.Nil {
		redirectPerm := g.FindPermanent(redirectID)
		if redirectPerm != nil {
			g.DealDamageToPermanent(redirectPerm, amount, sourceID)
			return
		}
	}
	// Minimum life floor (Ali from Cairo): cap damage so life doesn't go below 1
	if g.Effects.Rules.IsMinimumLifeActive(p.PlayerID()) {
		maxDamage := p.Life() - 1
		if maxDamage < 0 {
			maxDamage = 0
		}
		if amount > maxDamage {
			amount = maxDamage
		}
		if amount <= 0 {
			return
		}
	}
	// Lich replacement: instead of losing life, sacrifice permanents
	if g.Effects.Rules.IsLichActive(g, p.PlayerID()) {
		g.sacrificePermanents(p.PlayerID(), amount)
	} else {
		p.LoseLife(amount)
	}
	g.DamageTakenThisTurn[p.PlayerID()] += amount
	// Track artifact damage separately (for Reverse Polarity)
	if sourceCard != nil && sourceCard.HasType(TypeArtifact) {
		g.ArtifactDamageTakenThisTurn[p.PlayerID()] += amount
	}
	g.FireEvent(GameEvent{
		Type:     EvtDamageDealt,
		SourceID: sourceID,
		TargetID: p.PlayerID(),
		Amount:   amount,
		Flag:     false, // not combat damage
	})
	// Lifelink
	src := g.FindPermanent(sourceID)
	if src != nil && src.HasKeyword(Lifelink) {
		srcPlayer := g.GetPlayer(src.Controller)
		if srcPlayer != nil {
			srcPlayer.GainLife(amount)
		}
	}
	// Face-down: flip the source if it dealt damage to a player
	if src != nil && src.FaceDown {
		g.TurnFaceUp(src)
	}
	// Eye for an Eye: reflect damage to source's controller
	if reflectEntry, ok := g.Effects.Damage.GetDamageReflection(p.PlayerID()); ok {
		if reflectEntry.chosenSource == uuid.Nil || reflectEntry.chosenSource == sourceID {
			g.Effects.Damage.ClearDamageReflection(p.PlayerID())
			// Find the controller of the original damage source
			sourceCard := g.FindCardForDamageSource(sourceID)
			if sourceCard != nil {
				sourceOwner := sourceCard.Owner()
				if sourceOwner != uuid.Nil {
					ownerPlayer := g.GetPlayer(sourceOwner)
					if ownerPlayer != nil {
						g.DealDamageToPlayer(ownerPlayer, amount, reflectEntry.eyeSourceID)
					}
				}
			}
		}
	}
}

// DealDamageToPermanent deals damage to a permanent.
func (g *Game) DealDamageToPermanent(perm *Permanent, amount int, sourceID uuid.UUID) {
	if amount <= 0 {
		return
	}

	// if any damage prevention rule blocks this interaction, no damage done.
	source := g.FindPermanent(sourceID)
	if g.Effects.Damage.CheckDamagePreventionRules(source, perm, g) {
		return
	}

	// Apply damage prevention shield
	if prevented := g.Effects.Damage.PreventDamage(perm.ID(), amount); prevented > 0 {
		amount -= prevented
	}
	if amount <= 0 {
		return
	}
	// Jade Monolith: redirect creature damage to a player instead (one-shot)
	if redirectPlayerID := g.Effects.Damage.GetCreatureDamageRedirect(perm.ID()); redirectPlayerID != uuid.Nil {
		g.Effects.Damage.ClearCreatureDamageRedirect(perm.ID())
		targetPlayer := g.GetPlayer(redirectPlayerID)
		if targetPlayer != nil {
			g.DealDamageToPlayer(targetPlayer, amount, sourceID)
			return
		}
	}
	perm.Damage += amount
	// Track which sources dealt damage to this permanent
	if g.DamageDealtBy[perm.ID()] == nil {
		g.DamageDealtBy[perm.ID()] = make(map[uuid.UUID]bool)
	}
	g.DamageDealtBy[perm.ID()][sourceID] = true
	g.FireEvent(GameEvent{
		Type:     EvtDamageDealt,
		SourceID: sourceID,
		TargetID: perm.ID(),
		Amount:   amount,
	})
	// Deathtouch / BasiliskTouch
	src := g.FindPermanent(sourceID)
	if src != nil && amount > 0 {
		if src.HasKeyword(Deathtouch) {
			perm.Damage = perm.CurrentToughness(g)
		} else if src.HasKeyword(BasiliskTouch) && !perm.HasSubType("Wall") {
			perm.Damage = perm.CurrentToughness(g)
		}
	}
	// Lifelink
	if src != nil && src.HasKeyword(Lifelink) {
		srcPlayer := g.GetPlayer(src.Controller)
		if srcPlayer != nil {
			srcPlayer.GainLife(amount)
		}
	}
	// Face-down: flip the target if it was dealt damage
	if perm.FaceDown {
		g.TurnFaceUp(perm)
	}
	// Face-down: flip the source if it dealt damage
	if src != nil && src.FaceDown {
		g.TurnFaceUp(src)
	}
}

// Attach attaches source to target (for auras and equipment).
func (g *Game) Attach(sourceID, targetID uuid.UUID) {
	src := g.FindPermanent(sourceID)
	target := g.FindPermanent(targetID)
	if src == nil || target == nil {
		return
	}

	// Detach from current host if any
	if src.IsAttached() {
		oldHost := g.FindPermanent(src.AttachedTo)
		if oldHost != nil {
			filtered := oldHost.Attachments[:0]
			for _, id := range oldHost.Attachments {
				if id != sourceID {
					filtered = append(filtered, id)
				}
			}
			oldHost.Attachments = filtered
		}
	}

	src.AttachedTo = targetID
	target.Attachments = append(target.Attachments, sourceID)

	g.Effects.Apply(g)

	g.FireEvent(GameEvent{
		Type:     EvtAttach,
		SourceID: sourceID,
		TargetID: targetID,
	})
}

// RegisterDelayedTrigger registers a one-shot delayed trigger that will fire
// when the specified event type occurs.
func (g *Game) RegisterDelayedTrigger(dt *DelayedTrigger) {
	g.delayedTriggers = append(g.delayedTriggers, dt)
}

// FireEvent dispatches an event and checks triggered abilities.
func (g *Game) FireEvent(evt GameEvent) {
	for _, perm := range g.Battlefield {
		for _, a := range perm.RuntimeAbilities {
			ta, ok := a.(TriggeredAbility)
			if !ok {
				continue
			}
			if !ta.CheckEventType(evt.Type) {
				continue
			}
			if ta.CheckTrigger(&evt, g) {
				g.pendingTriggers = append(g.pendingTriggers, &pendingTrigger{
					ability:    ta,
					event:      &evt,
					sourceID:   perm.ID(),
					controller: perm.Controller,
				})
			}
		}
	}

	// Check delayed triggers (one-shot, removed after matching)
	remaining := g.delayedTriggers[:0]
	for _, dt := range g.delayedTriggers {
		if dt.EventType == evt.Type {
			if dt.MatchEventID != uuid.Nil && evt.SourceID != dt.MatchEventID {
				remaining = append(remaining, dt)
				continue
			}
			if dt.MatchPlayerID != uuid.Nil && evt.PlayerID != dt.MatchPlayerID {
				remaining = append(remaining, dt)
				continue
			}
			obj := &StackObject{
				ID:         uuid.New(),
				Controller: dt.Controller,
				SourceID:   dt.SourceID,
				IsAbility:  true,
				Effects:    dt.Effects,
				Targets:    []uuid.UUID{dt.TargetID},
			}
			g.Stack.Push(obj)
		} else {
			remaining = append(remaining, dt)
		}
	}
	g.delayedTriggers = remaining
}

// PutTriggersOnStack puts all pending triggers onto the stack.
func (g *Game) PutTriggersOnStack() {
	for _, pt := range g.pendingTriggers {
		obj := &StackObject{
			ID:         uuid.New(),
			Controller: pt.controller,
			SourceID:   pt.sourceID,
			IsAbility:  true,
		}
		obj.Effects = append(obj.Effects, pt.ability.Effects()...)
		// For triggers that need to pass the event's player as a target
		// (e.g., "deal damage to that land's controller", "that player draws"),
		// store the event PlayerID as a target on the stack object.
		if pt.event != nil {
			if gt, ok := pt.ability.(*GenericTriggered); ok {
				switch gt.eventType {
				case EvtEntersBattlefield, EvtDrawStep:
					if pt.event.PlayerID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.PlayerID}
					}
				case EvtDamageDealt:
					if pt.event.TargetID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.TargetID}
					}
				case EvtTapped:
					// Pass the tapped permanent's ID so effects can identify it
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
					}
				case EvtDeclaredBlocker:
					// Pass the blocker's ID so effects can identify it
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
					}
				}
			}
		}
		g.Stack.Push(obj)
	}
	g.pendingTriggers = nil
}

// ResolveStack resolves all objects on the stack (simplified: no priority passing).
func (g *Game) ResolveStack() {
	for !g.Stack.IsEmpty() {
		obj := g.Stack.Pop()
		g.ResolveStackObject(obj)
		// Check for new triggers after each resolution
		g.PutTriggersOnStack()
	}
}

// isTargetStillLegal checks whether a target is still legal at resolution time.
func (g *Game) isTargetStillLegal(targetID uuid.UUID, sourceCard Card, controller uuid.UUID) bool {
	// Players are always legal targets (targeting doesn't check life/loss status)
	if g.GetPlayer(targetID) != nil {
		return true
	}
	// Permanents must still be on the battlefield and targetable
	if perm := g.FindPermanent(targetID); perm != nil {
		return perm.CanBeTargetedBy(sourceCard, controller, g)
	}
	// Cards in hand are legal if still in hand
	for _, p := range g.Players {
		for _, c := range p.Hand() {
			if c.ID() == targetID {
				return true
			}
		}
	}
	// Graveyard cards are legal if still in graveyard
	for _, p := range g.Players {
		for _, c := range p.Graveyard() {
			if c.ID() == targetID {
				return true
			}
		}
	}
	// Stack spells are legal if still on the stack
	if g.Stack.FindBySourceID(targetID) != nil {
		return true
	}
	// Target no longer exists in any known zone
	return false
}

// ResolveStackObject resolves a single stack object.
func (g *Game) ResolveStackObject(obj *StackObject) {
	// Check for fizzle: if the spell/ability has targets but all are now illegal,
	// it fails to resolve (MTG rule 608.2b)
	if len(obj.Targets) > 0 {
		hasRealTargets := false
		anyLegal := false
		for _, t := range obj.Targets {
			if t == uuid.Nil {
				continue // skip nil targets (used as data slots, not real targets)
			}
			hasRealTargets = true
			if g.isTargetStillLegal(t, obj.Card, obj.Controller) {
				anyLegal = true
				break
			}
		}
		if hasRealTargets && !anyLegal {
			// Spell fizzles — put card in graveyard without resolving effects
			if obj.Card != nil && !obj.IsAbility {
				owner := obj.Card.Owner()
				if owner == uuid.Nil {
					owner = obj.Controller
				}
				p := g.GetPlayer(owner)
				if p != nil {
					p.AddToGraveyard(obj.Card)
				}
			}
			g.CheckStateBasedActions()
			return
		}
	}

	g.CurrentX = obj.XValue
	g.CurrentMode = obj.ModeChoice
	g.ResolvingCard = obj.Card
	g.ResolvingTargets = obj.Targets
	for _, eff := range obj.Effects {
		_ = eff.Apply(g, obj.SourceID, obj.Controller, obj.Targets)
	}

	// If this was a spell (not an ability), put the card in the graveyard
	if obj.Card != nil && !obj.IsAbility {
		owner := obj.Card.Owner()
		if owner == uuid.Nil {
			owner = obj.Controller
		}

		// Permanents go to the battlefield instead
		if obj.Card.HasType(TypeCreature) || obj.Card.HasType(TypeArtifact) || obj.Card.HasType(TypeEnchantment) {
			perm := g.PutOnBattlefield(obj.Card, obj.Controller)

			// Handle aura attachment (only for Aura subtype, not all enchantments)
			if obj.Card.HasType(TypeEnchantment) && len(obj.Targets) > 0 {
				for _, st := range obj.Card.SubTypes() {
					if st == "Aura" {
						g.Attach(perm.ID(), obj.Targets[0])
						break
					}
				}
			}

			g.CurrentX = 0
			g.CurrentMode = 0
			g.ResolvingTargets = nil
			g.CheckStateBasedActions()
			return
		}

		// Instants and sorceries go to graveyard
		p := g.GetPlayer(owner)
		if p != nil {
			p.AddToGraveyard(obj.Card)
		}
	}

	g.CurrentX = 0
	g.CurrentMode = 0
	g.ResolvingCard = nil
	g.ResolvingTargets = nil

	g.CheckStateBasedActions()
}

// CastSpellByName finds a card in player's hand, puts it on the stack.
func (g *Game) CastSpellByName(playerID uuid.UUID, name string, targets []uuid.UUID, xValues ...int) error {
	p := g.GetPlayer(playerID)
	if p == nil {
		return ErrPlayerNotFound
	}

	// Find card in hand
	var card Card
	for _, c := range p.Hand() {
		if c.Name() == name {
			card = c
			break
		}
	}
	if card == nil {
		return fmt.Errorf("card %s not found in hand", name)
	}

	// Check expansion block (City in a Bottle)
	if exp := card.Expansion(); exp != "" && g.Effects.Rules.IsExpansionBlocked(exp) {
		return fmt.Errorf("can't cast %s: expansion %s is blocked", name, exp)
	}
	// Check artifact mana restriction (Mishra's Workshop)
	if g.ArtifactManaOnly[playerID] && !card.HasType(TypeArtifact) {
		return fmt.Errorf("mana restriction: can only cast artifact spells")
	}
	// Check creature mana restriction (Metamorphosis)
	if g.CreatureManaOnly[playerID] && !card.HasType(TypeCreature) {
		return fmt.Errorf("mana restriction: can only cast creature spells")
	}

	// Determine X value
	xValue := 0
	if len(xValues) > 0 {
		xValue = xValues[0]
	}

	mc := card.ManaCost()

	// Apply spell cost increases (e.g. Gloom)
	for _, col := range mc.Colors() {
		increase := g.Effects.Rules.SpellCostIncrease(col)
		if increase > 0 {
			mc.Generic += increase
			break // only apply once per spell
		}
	}

	// Apply spell cost reductions
	for _, col := range mc.Colors() {
		reduction := g.Effects.Rules.SpellCostReduction(col)
		if reduction > 0 {
			mc.Generic -= reduction
			if mc.Generic < 0 {
				mc.Generic = 0
			}
			break // only apply once per spell
		}
	}

	// Channel: pay life for generic/X costs instead of mana
	if g.Effects.Rules.IsChannelActive(playerID) && (mc.Generic > 0 || (mc.HasX && xValue > 0)) {
		// Pay colored portion from pool
		colorMC := mc
		colorMC.Generic = 0
		colorMC.HasX = false
		if !colorMC.IsZero() {
			if !p.ManaPool().CanPay(colorMC) {
				return fmt.Errorf("cannot pay mana cost %s for %s", colorMC, name)
			}
			if err := p.ManaPool().Pay(colorMC); err != nil {
				return err
			}
		}
		// Pay generic + X from life
		lifeCost := mc.Generic
		if mc.HasX {
			lifeCost += xValue * mc.XCount
		}
		p.LoseLife(lifeCost)
	} else {
		// Pay mana cost (auto-pay from pool)
		payMC := mc
		if mc.HasX {
			payMC.Generic += xValue * mc.XCount
		}
		if !payMC.IsZero() {
			if !p.ManaPool().CanPay(payMC) {
				return fmt.Errorf("cannot pay mana cost %s for %s", payMC, name)
			}
			if err := p.ManaPool().Pay(payMC); err != nil {
				return err
			}
		}
	}

	// Pay additional costs (sacrifice, discard, etc.)
	if bc, ok := card.(*BaseCard); ok {
		for _, cost := range bc.AdditionalCosts() {
			if !cost.CanPay(card.ID(), playerID, g) {
				return fmt.Errorf("cannot pay additional cost for %s: %s", name, cost.Text())
			}
			if err := cost.Pay(card.ID(), playerID, g); err != nil {
				return err
			}
		}
	}

	// If an additional cost set g.CurrentX (e.g. sacrifice-capture-CMC), use it
	if g.CurrentX != 0 && xValue == 0 {
		xValue = g.CurrentX
		g.CurrentX = 0
	}

	// Remove from hand
	p.RemoveFromHand(card.ID())

	// Build effects from spell abilities
	var effects []Effect
	for _, a := range card.Abilities() {
		if sa, ok := a.(*SpellAbility); ok {
			effects = append(effects, sa.Effects()...)
		}
	}

	obj := &StackObject{
		ID:         uuid.New(),
		Card:       card,
		Controller: playerID,
		SourceID:   card.ID(),
		Effects:    effects,
		Targets:    targets,
		XValue:     xValue,
	}

	if modes := card.Modes(); len(modes) > 0 {
		obj.ModeChoice = p.ChooseMode(modes, card.Name())
	}

	g.Stack.Push(obj)

	g.FireEvent(GameEvent{
		Type:     EvtSpellCast,
		SourceID: card.ID(),
		PlayerID: playerID,
	})

	return nil
}

// ActivateAbilityByText finds and activates an activated ability on a permanent.
func (g *Game) ActivateAbilityByText(playerID uuid.UUID, permName string, targets []uuid.UUID) error {
	perm := g.FindPermanentByName(permName, playerID)
	if perm == nil {
		// Try finding any permanent with this name (for AnyPlayerMayUse abilities)
		for _, p := range g.Battlefield {
			if p.Name() == permName {
				perm = p
				break
			}
		}
	}
	if perm == nil {
		return fmt.Errorf("permanent %s not found", permName)
	}

	for _, a := range perm.RuntimeAbilities {
		inner := UnwrapAbility(a)
		aa, ok := inner.(ActivatedAbility)
		if !ok {
			continue
		}
		if !aa.CanActivate(playerID, g) {
			continue
		}

		// Check sorcery speed
		if aa.SorcerySpeed() && !g.Step.IsMainPhase() {
			return ErrSorcerySpeed
		}

		// Validate targets against the ability's target filters
		if len(aa.Targets()) > 0 && len(targets) > 0 {
			validTargets := true
			for i, t := range aa.Targets() {
				if i >= len(targets) {
					break
				}
				if tf, ok := t.(interface{ Filter() PermanentFilter }); ok {
					targetPerm := g.FindPermanent(targets[i])
					if targetPerm != nil && !tf.Filter().Match(targetPerm, g) {
						validTargets = false
						break
					}
				} else {
					// Fallback: check if the target is in the Possible() list
					possible := t.Possible(playerID, perm.Card, g)
					found := false
					for _, pid := range possible {
						if pid == targets[i] {
							found = true
							break
						}
					}
					if !found {
						validTargets = false
						break
					}
				}
			}
			if !validTargets {
				continue
			}
		}

		// Pay costs
		for _, c := range aa.Costs() {
			if err := c.Pay(perm.ID(), playerID, g); err != nil {
				return err
			}
		}

		// Mark once-per-turn abilities as used
		if saa, ok := aa.(*SimpleActivatedAbility); ok {
			saa.MarkActivated()
		}

		obj := &StackObject{
			ID:         uuid.New(),
			Controller: playerID,
			SourceID:   perm.ID(),
			IsAbility:  true,
			Targets:    targets,
			XValue:     g.CurrentX,
		}
		obj.Effects = append(obj.Effects, aa.Effects()...)

		// Modal abilities: choose mode at activation time
		if modes := perm.Card.Modes(); len(modes) > 0 {
			p := g.GetPlayer(playerID)
			if p != nil {
				obj.ModeChoice = p.ChooseMode(modes, perm.Card.Name())
			}
		}

		g.Stack.Push(obj)
		return nil
	}

	// Try mana abilities (these don't use the stack)
	for _, a := range perm.RuntimeAbilities {
		inner := UnwrapAbility(a)
		ma, ok := inner.(*ManaAbility)
		if !ok {
			continue
		}
		if perm.Tapped || !perm.CanTapForEffect(g) {
			continue // already tapped or summoning sick without haste
		}
		perm.Tapped = true
		p := g.GetPlayer(playerID)
		if p != nil {
			color := ma.Color
			if ma.AnyColor {
				color = p.ChooseManaColor("add mana")
			}
			p.ManaPool().Add(color, 1)
			// Check for mana bonus effects (e.g. Gauntlet of Might)
			g.applyManaBonuses(perm, color, p)
		}
		g.FireEvent(GameEvent{
			Type:     EvtTapped,
			SourceID: perm.ID(),
			PlayerID: playerID,
		})
		g.FireEvent(GameEvent{
			Type:     EvtAbilityActivated,
			SourceID: perm.ID(),
			PlayerID: playerID,
		})
		return nil
	}

	return fmt.Errorf("no activatable ability found on %s", permName)
}

// applyManaBonuses checks for mana bonus effects when a permanent is tapped for mana.
func (g *Game) applyManaBonuses(tappedPerm *Permanent, producedColor Color, p Player) {
	for _, perm := range g.Battlefield {
		for _, a := range perm.RuntimeAbilities {
			inner := UnwrapAbility(a)
			if mb, ok := inner.(*ManaBonusAbility); ok {
				if mb.AttachedOnly {
					// Only triggers for the permanent this aura is attached to
					if perm.AttachedTo == tappedPerm.ID() {
						p.ManaPool().Add(mb.BonusMana, 1)
					}
				} else if mb.Filter.Match(tappedPerm, g) {
					p.ManaPool().Add(mb.BonusMana, 1)
				}
			}
		}
	}
}

// CheckStateBasedActions checks and processes state-based actions.
func (g *Game) CheckStateBasedActions() {
	for {
		actions := false

		// Check for creatures with lethal damage
		var toDestroy []*Permanent
		for _, p := range g.Battlefield {
			if p.HasType(TypeCreature) && p.LethalDamage(g) {
				toDestroy = append(toDestroy, p)
				actions = true
			}
		}
		for _, p := range toDestroy {
			g.DestroyPermanent(p)
		}

		// Check for creatures with 0 or less toughness (not destruction — bypasses indestructible)
		var zeroToughness []*Permanent
		for _, p := range g.Battlefield {
			if p.HasType(TypeCreature) && p.CurrentToughness(g) <= 0 {
				zeroToughness = append(zeroToughness, p)
				actions = true
			}
		}
		for _, p := range zeroToughness {
			g.PutPermanentIntoGraveyard(p)
		}

		// MTG rule 704.5q: +1/+1 and -1/-1 counter annihilation
		for _, p := range g.Battlefield {
			plus := p.Counters[P1P1]
			minus := p.Counters[M1M1]
			if plus > 0 && minus > 0 {
				remove := plus
				if minus < remove {
					remove = minus
				}
				p.Counters[P1P1] -= remove
				p.Counters[M1M1] -= remove
				if p.Counters[P1P1] == 0 {
					delete(p.Counters, P1P1)
				}
				if p.Counters[M1M1] == 0 {
					delete(p.Counters, M1M1)
				}
				actions = true
			}
		}

		// Check for auras attached to nothing or illegal targets
		var aurasToDrop []*Permanent
		for _, p := range g.Battlefield {
			if p.HasSubType("Aura") && p.IsAttached() {
				host := g.FindPermanent(p.AttachedTo)
				if host == nil {
					aurasToDrop = append(aurasToDrop, p)
					actions = true
				} else if host.HasProtectionFrom(p.Card) {
					aurasToDrop = append(aurasToDrop, p)
					actions = true
				}
			}
		}
		for _, a := range aurasToDrop {
			g.DestroyPermanent(a)
		}

		// Equipment attached to a non-creature becomes unattached
		for _, p := range g.Battlefield {
			if p.HasSubType("Equipment") && p.IsAttached() {
				host := g.FindPermanent(p.AttachedTo)
				if host != nil && !host.HasType(TypeCreature) {
					p.AttachedTo = uuid.Nil
					actions = true
				}
			}
		}

		// Sacrifice creatures that require a land type the controller doesn't have
		var toSacrifice []*Permanent
		for _, p := range g.Battlefield {
			var landSubtype string
			for _, a := range p.RuntimeAbilities {
				if sa, ok := a.(*SacrificeUnlessLandAbility); ok {
					landSubtype = sa.LandSubtype
					break
				}
			}
			if landSubtype == "" {
				continue
			}
			hasLand := false
			for _, other := range g.Battlefield {
				if other.Controller == p.Controller && other.HasSubType(landSubtype) {
					hasLand = true
					break
				}
			}
			if !hasLand {
				toSacrifice = append(toSacrifice, p)
				actions = true
			}
		}
		for _, p := range toSacrifice {
			g.Sacrifice(p)
		}

		// MTG rule 704.5j: Legend rule — if a player controls two or more legendary
		// permanents with the same name, they choose one and sacrifice the rest.
		legendCounts := make(map[uuid.UUID]map[string][]*Permanent) // controller -> name -> perms
		for _, p := range g.Battlefield {
			if p.Card.HasSuperType(SuperLegendary) {
				if legendCounts[p.Controller] == nil {
					legendCounts[p.Controller] = make(map[string][]*Permanent)
				}
				legendCounts[p.Controller][p.Name()] = append(legendCounts[p.Controller][p.Name()], p)
			}
		}
		for ctrlID, byName := range legendCounts {
			for _, perms := range byName {
				if len(perms) > 1 {
					player := g.GetPlayer(ctrlID)
					keep := player.ChoosePermanent(perms, "legend rule: keep one", g)
					for _, p := range perms {
						if p != keep {
							g.Sacrifice(p)
						}
					}
					actions = true
				}
			}
		}

		// MTG rule 704.5k: World rule — if two or more permanents have the World
		// supertype, all except the most recent one are put into their owners' graveyards.
		var worldPerms []*Permanent
		for _, p := range g.Battlefield {
			if p.Card.HasSuperType(SuperWorld) {
				worldPerms = append(worldPerms, p)
			}
		}
		if len(worldPerms) > 1 {
			// Keep the most recently entered one (last in Battlefield slice)
			keep := worldPerms[len(worldPerms)-1]
			for _, p := range worldPerms {
				if p != keep {
					g.PutPermanentIntoGraveyard(p)
				}
			}
			actions = true
		}

		// MTG rule 704.5b: player who attempted to draw from empty library loses
		for _, p := range g.Players {
			if p.DrewFromEmpty() {
				p.ClearDrewFromEmpty()
				p.SetLost()
			}
		}

		if !actions {
			break
		}
	}

	// Put any pending triggers on the stack
	g.PutTriggersOnStack()
}

// RunStep executes a single step of the turn.
func (g *Game) RunStep(step PhaseStep) {
	g.Step = step

	// Reapply continuous effects at start of each step
	g.Effects.Apply(g)

	switch step {
	case Untap:
		g.DoUntap()
	case Upkeep:
		g.DoUpkeep()
	case Draw:
		g.DoDraw()
	case BeginCombat:
		g.DoBeginCombat()
	case DeclareAttackers:
		g.doDeclareAttackers()
	case DeclareBlockers:
		g.doDeclareBlockers()
	case FirstStrikeDamage:
		if !g.Combat.HasFirstStrikers(g) {
			return // skip if no first strikers
		}
		g.Combat.ResolveDamage(g, true)
	case CombatDamage:
		g.Combat.ResolveDamage(g, false)
	case EndCombat:
		g.Effects.RemoveEndOfCombat()
		g.Effects.Apply(g)
		g.Combat.Reset()
	case EndStep:
		g.DoEndStep()
	case Cleanup:
		g.DoCleanup()
	}

	// Check SBAs after each step
	g.CheckStateBasedActions()

	// Resolve stack
	g.ResolveStack()
}

func (g *Game) doBeginCombatActions() {
	active := g.ActivePlayerObj()
	g.FireEvent(GameEvent{
		Type:     EvtBeginCombat,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
}

func (g *Game) DoBeginCombat() {
	g.doBeginCombatActions()
	g.ResolveStack()
}

func (g *Game) doEndStepActions() {
	active := g.ActivePlayerObj()
	g.FireEvent(GameEvent{
		Type:     EvtEndStep,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
}

func (g *Game) DoEndStep() {
	g.doEndStepActions()
	g.ResolveStack()
}

func (g *Game) DoUntap() {
	active := g.ActivePlayerObj()
	g.Effects.Damage.ClearRegenerationShields(active.PlayerID(), g)
	// Island Sanctuary: clear protection at the start of the player's turn
	g.Effects.Rules.ClearSanctuary(active.PlayerID())

	landUntapLimit := g.Effects.Rules.LandUntapMax
	landsUntapped := 0
	artifactUntapLimit := g.Effects.Rules.ArtifactUntapMax
	artifactsUntapped := 0

	for _, p := range g.Battlefield {
		if p.Controller == active.PlayerID() {
			if p.HasAttr(AttrDoesNotUntap) {
				// Does not untap — skip
			} else if p.Tapped && p.HasAttr(AttrMayNotUntap) {
				// Player may choose not to untap
				if !active.ChooseMayAbility("untap " + p.Name()) {
					continue
				}
				p.Tapped = false
			} else if p.HasType(TypeLand) && landUntapLimit >= 0 {
				// Land with untap limit in effect
				if p.Tapped && landsUntapped < landUntapLimit {
					p.Tapped = false
					landsUntapped++
				}
			} else if p.HasType(TypeArtifact) && !p.HasType(TypeLand) && artifactUntapLimit >= 0 {
				// Artifact (non-land) with untap limit in effect (Damping Field)
				if p.Tapped && artifactsUntapped < artifactUntapLimit {
					p.Tapped = false
					artifactsUntapped++
				}
			} else {
				p.Tapped = false
			}
			p.RevokeBaseAttr(AttrSummonSick)
		}
	}
	g.LandsPlayedThisTurn = 0
}

func (g *Game) doUpkeepActions() {
	active := g.ActivePlayerObj()

	// Check for graveyard returns (e.g. Nether Shadow)
	g.checkGraveyardReturns(active)

	g.FireEvent(GameEvent{
		Type:     EvtUpkeep,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
}

func (g *Game) DoUpkeep() {
	g.doUpkeepActions()
	g.ResolveStack()
}

// checkGraveyardReturns checks for cards in the graveyard that can return to the
// battlefield at the beginning of their controller's upkeep (e.g. Nether Shadow).
func (g *Game) checkGraveyardReturns(p Player) {
	graveyard := p.Graveyard()
	var toReturn []uuid.UUID

	for i, card := range graveyard {
		var minCreatures int
		for _, a := range card.Abilities() {
			if gra, ok := a.(*GraveyardReturnAbility); ok {
				minCreatures = gra.MinCreaturesAbove
				break
			}
		}
		if minCreatures <= 0 {
			continue
		}
		// Count creature cards above this one (higher indices = more recently added)
		creaturesAbove := 0
		for j := i + 1; j < len(graveyard); j++ {
			if graveyard[j].HasType(TypeCreature) {
				creaturesAbove++
			}
		}
		if creaturesAbove >= minCreatures {
			toReturn = append(toReturn, card.ID())
		}
	}

	for _, id := range toReturn {
		card, ok := p.RemoveFromGraveyard(id)
		if ok {
			g.PutOnBattlefield(card, p.PlayerID())
		}
	}
}

func (g *Game) doDrawActions() {
	active := g.ActivePlayerObj()

	// Fire draw step event before the normal draw so triggers can queue
	g.FireEvent(GameEvent{
		Type:     EvtDrawStep,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
}

func (g *Game) doDrawNormalDraw() {
	active := g.ActivePlayerObj()

	// First player doesn't draw on turn 1
	if g.Turn == 1 && g.ActivePlayer == 0 {
		return
	}

	// Island Sanctuary: skip the normal draw if flagged
	if g.Effects.Rules.ShouldSkipDraw(active.PlayerID()) {
		return
	}

	// Aladdin's Lamp: replace draw with library peek + choice
	if count, ok := g.Effects.Damage.GetDrawReplacement(active.PlayerID()); ok {
		g.Effects.Damage.ClearDrawReplacement(active.PlayerID())
		g.applyDrawReplacement(active, count)
		return
	}

	active.DrawCard()
}

// applyDrawReplacement handles Aladdin's Lamp draw replacement.
// Look at top X cards, choose one, put rest on bottom randomly, draw the chosen card.
func (g *Game) applyDrawReplacement(p Player, count int) {
	lib := p.Library()
	if count > len(lib) {
		count = len(lib)
	}
	if count == 0 {
		return
	}
	candidates := make([]Card, count)
	copy(candidates, lib[:count])
	chosen := p.ChooseCardFromLibrary(candidates, "choose card from Aladdin's Lamp", g)
	if chosen == nil {
		chosen = candidates[0]
	}
	var rest []Card
	for _, c := range candidates {
		if c.ID() != chosen.ID() {
			rest = append(rest, c)
		}
	}
	rand.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
	newLib := []Card{chosen}
	newLib = append(newLib, lib[count:]...)
	newLib = append(newLib, rest...)
	p.SetLibrary(newLib)
	p.DrawCard()
}

func (g *Game) DoDraw() {
	g.doDrawActions()
	g.ResolveStack()
	g.doDrawNormalDraw()
}

func (g *Game) doDeclareAttackers() {
	active := g.ActivePlayerObj()
	attackerIDs := active.DeclareAttackers(g)
	defender := g.NonActivePlayerObj()

	// Auto-add creatures with MustAttack keyword (from Nettling Imp, etc.)
	declared := make(map[uuid.UUID]bool)
	for _, id := range attackerIDs {
		declared[id] = true
	}
	for _, p := range g.Battlefield {
		if p.Controller == active.PlayerID() && p.HasAttr(AttrMustAttack) && !declared[p.ID()] {
			if p.CanDeclareAsAttacker(g) {
				attackerIDs = append(attackerIDs, p.ID())
			}
		}
	}

	for _, id := range attackerIDs {
		atk := g.FindPermanent(id)
		if atk == nil {
			continue
		}
		if !atk.CanDeclareAsAttacker(g) {
			continue
		}
		// Island Sanctuary: only flying or islandwalk creatures can attack
		if g.Effects.Rules.IsSanctuaryActive(defender.PlayerID()) {
			if !atk.HasKeyword(Flying) && !atk.HasKeyword(Islandwalk) {
				continue
			}
		}

		// Tap attacker (unless vigilance)
		if !atk.HasKeyword(Vigilance) {
			atk.Tapped = true
		}

		g.Combat.AddAttacker(id, defender.PlayerID())
		g.AttackedThisTurn[id] = true
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredAttacker,
			SourceID: id,
			PlayerID: active.PlayerID(),
		})
	}

	// Form attacking bands if the player has scripted them.
	if bf, ok := active.(BandFormer); ok {
		for _, band := range bf.GetBandFormations(g.Turn, g) {
			if g.isValidBand(band) {
				g.Combat.AddBand(band)
			}
		}
	}
}

// isValidBand checks that a slice of attacker IDs meets banding requirements:
// all must be currently attacking, at least one must have banding, and at most
// one may lack banding.
func (g *Game) isValidBand(memberIDs []uuid.UUID) bool {
	if len(memberIDs) < 2 {
		return false
	}
	bandingCount := 0
	nonBandingCount := 0
	for _, id := range memberIDs {
		perm := g.FindPermanent(id)
		if perm == nil || !g.Combat.IsAttacking(id) {
			return false
		}
		if perm.HasKeyword(Banding) {
			bandingCount++
		} else {
			nonBandingCount++
		}
	}
	return bandingCount >= 1 && nonBandingCount <= 1
}

func (g *Game) doDeclareBlockers() {
	nonActive := g.NonActivePlayerObj()
	assignments := nonActive.DeclareBlockers(g)
	if assignments == nil {
		// No blockers, but still fire the event so "attacks and isn't blocked" triggers work
		g.FireEvent(GameEvent{
			Type:     EvtBlockersDecl,
			PlayerID: nonActive.PlayerID(),
		})
		return
	}

	// Check for Lure: if any attacker has MustBeBlocked, redirect all blocks to it
	var luredAttackerID uuid.UUID
	for _, group := range g.Combat.Groups {
		atk := g.FindPermanent(group.AttackerID)
		if atk != nil && atk.HasKeyword(MustBeBlocked) {
			luredAttackerID = group.AttackerID
			break
		}
	}

	blockerCount := make(map[uuid.UUID]int) // how many attackers each blocker is assigned to
	for _, ba := range assignments {
		blocker := g.FindPermanent(ba.BlockerID)
		attackerID := ba.AttackerID
		// If there's a Lure creature, redirect all blocks to it
		if luredAttackerID != uuid.Nil {
			attackerID = luredAttackerID
		}
		attacker := g.FindPermanent(attackerID)
		if blocker == nil || attacker == nil {
			continue
		}
		if !blocker.CanDeclareAsBlocker(g) {
			continue
		}
		if !CanBlock(blocker, attacker, g) {
			continue
		}
		// Landwalk: if attacker has landwalk and defender controls matching land, can't be blocked
		if HasLandwalkEvasion(attacker, nonActive.PlayerID(), g) {
			continue
		}
		// Check multi-block limit: normally a creature can only block one attacker
		maxBlocks := 1
		if blocker.HasKeyword(CanBlockAny) {
			maxBlocks = 999
		} else if blocker.HasKeyword(CanBlockAdditional) {
			maxBlocks = 2
		}
		if blockerCount[ba.BlockerID] >= maxBlocks {
			continue
		}
		blockerCount[ba.BlockerID]++
		g.Combat.AddBlocker(ba.BlockerID, attackerID)
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredBlocker,
			SourceID: ba.BlockerID,
			TargetID: attackerID,
			PlayerID: nonActive.PlayerID(),
		})
	}

	// Fire EvtBlockersDecl once after all blockers are assigned
	g.FireEvent(GameEvent{
		Type:     EvtBlockersDecl,
		PlayerID: nonActive.PlayerID(),
	})
}

// doCleanupActions performs cleanup housekeeping and places any triggers on the stack.
// Returns true if triggers were placed on the stack (requiring priority + another cleanup).
func (g *Game) doCleanupActions() bool {
	// Hand size discard: each player discards down to max hand size
	for _, p := range g.Players {
		maxHS := g.Effects.Rules.MaxHandSize(p.PlayerID())
		for len(p.Hand()) > maxHS {
			chosen := p.ChooseCardsFromHand(1, "discard to hand size", g)
			if len(chosen) == 0 {
				break
			}
			p.RemoveFromHand(chosen[0].ID())
			p.AddToGraveyard(chosen[0])
		}
	}
	// Clear damage from all creatures
	for _, p := range g.Battlefield {
		p.Damage = 0
	}
	// Clear mana pools
	for _, p := range g.Players {
		p.ManaPool().Clear()
	}
	// Remove end-of-turn effects and clear turn-scoped state
	g.Effects.RemoveEndOfTurn()
	g.Effects.Damage.ClearEndOfTurn()
	g.Effects.Rules.ClearEndOfTurn()
	// Clear damage tracking
	g.DamageDealtBy = make(map[uuid.UUID]map[uuid.UUID]bool)
	g.DamageTakenThisTurn = make(map[uuid.UUID]int)
	g.ArtifactDamageTakenThisTurn = make(map[uuid.UUID]int)
	g.AttackedThisTurn = make(map[uuid.UUID]bool)
	g.CreatureDeathsThisTurn = 0
	// Clear mana restrictions
	g.ArtifactManaOnly = make(map[uuid.UUID]bool)
	g.CreatureManaOnly = make(map[uuid.UUID]bool)
	// Clear last-drawn-card tracking for all players
	for _, p := range g.Players {
		p.ClearLastDrawnCard()
	}
	for _, p := range g.Battlefield {
		// Clear activation tracking (Charge counters used for per-turn counts)
		delete(p.Counters, Charge)
		// Reset once-per-turn activated abilities
		for _, a := range p.RuntimeAbilities {
			if aa, ok := UnwrapAbility(a).(*SimpleActivatedAbility); ok {
				aa.ResetActivation()
			}
		}
	}
	// MTG 514.3a: if triggers fire during cleanup, put them on stack
	g.PutTriggersOnStack()
	return !g.Stack.IsEmpty()
}

func (g *Game) DoCleanup() {
	if g.doCleanupActions() {
		// Triggers fired — resolve, check SBAs, then do another cleanup step.
		g.ResolveStack()
		g.CheckStateBasedActions()
		g.DoCleanup()
	}
}

// RunTurn executes a complete turn for the active player.
// stopAt is checked: if we reach the specified turn+step, we stop.
func (g *Game) RunTurn(stopTurn int, stopStep PhaseStep) bool {
	for _, step := range AllSteps() {
		if g.Turn == stopTurn && step == stopStep {
			g.Step = step
			return true // signal to stop
		}
		g.RunStep(step)
		if g.stopped {
			return true
		}
	}
	return false
}

// Run executes the game until the stop condition.
func (g *Game) Run(stopTurn int, stopStep PhaseStep, maxTurns int) {
	for g.Turn <= maxTurns {
		if g.RunTurn(stopTurn, stopStep) {
			return
		}
		// Check for extra turns
		if len(g.ExtraTurns) > 0 {
			extraPlayerID := g.ExtraTurns[0]
			g.ExtraTurns = g.ExtraTurns[1:]
			// Find the player index
			for i, p := range g.Players {
				if p.PlayerID() == extraPlayerID {
					g.ActivePlayer = i
					break
				}
			}
		} else {
			// Next turn: swap active player
			g.ActivePlayer = (g.ActivePlayer + 1) % len(g.Players)
		}
		g.Turn++
	}
}

// MaxLandPlays returns the maximum number of lands that can be played this turn.
func (g *Game) MaxLandPlays() int {
	limit := 1
	if g.Effects.Rules.UnlimitedLandPlays {
		limit = 999
	}
	return limit
}

// playLandCore moves a land from a player's hand to the battlefield and fires
// landfall triggers, but does NOT resolve the stack. Callers are responsible
// for draining the stack (via ResolveStack or RunPriorityRound).
func (g *Game) playLandCore(playerID, cardID uuid.UUID) error {
	if !g.Step.IsMainPhase() {
		return fmt.Errorf("can only play lands during a main phase")
	}
	if g.ActivePlayerObj().PlayerID() != playerID {
		return fmt.Errorf("only the active player can play a land")
	}
	if g.LandsPlayedThisTurn >= g.MaxLandPlays() {
		return fmt.Errorf("already played a land this turn")
	}

	p := g.GetPlayer(playerID)
	if p == nil {
		return ErrPlayerNotFound
	}

	card, ok := p.RemoveFromHand(cardID)
	if !ok {
		return ErrCardNotInHand
	}
	if !card.HasType(TypeLand) {
		p.AddToHand(card)
		return fmt.Errorf("card is not a land")
	}
	// Check expansion block (City in a Bottle)
	if exp := card.Expansion(); exp != "" && g.Effects.Rules.IsExpansionBlocked(exp) {
		p.AddToHand(card)
		return fmt.Errorf("can't play %s: expansion %s is blocked", card.Name(), exp)
	}

	g.PutOnBattlefield(card, playerID)
	g.LandsPlayedThisTurn++

	g.FireEvent(GameEvent{
		Type:     EvtLandPlayed,
		SourceID: card.ID(),
		PlayerID: playerID,
		Amount:   g.LandsPlayedThisTurn, // which land number this was
	})
	g.PutTriggersOnStack()

	return nil
}

// PlayLand moves a land from a player's hand to the battlefield.
func (g *Game) PlayLand(playerID, cardID uuid.UUID) error {
	if err := g.playLandCore(playerID, cardID); err != nil {
		return err
	}
	g.ResolveStack()
	return nil
}

// TapForMana taps a permanent for mana using its mana ability.
func (g *Game) TapForMana(playerID, permanentID uuid.UUID) error {
	perm := g.FindPermanent(permanentID)
	if perm == nil {
		return ErrPermanentNotFound
	}
	if perm.Controller != playerID {
		return fmt.Errorf("you don't control that permanent")
	}
	if perm.Tapped {
		return fmt.Errorf("permanent is already tapped")
	}

	// Find a mana ability
	for _, a := range perm.RuntimeAbilities {
		if ma, ok := a.(*ManaAbility); ok {
			// Creatures with mana abilities need to not be summoning sick
			if !perm.CanTapForEffect(g) {
				return fmt.Errorf("creature has summoning sickness")
			}
			perm.Tapped = true
			p := g.GetPlayer(playerID)
			if p != nil {
				p.ManaPool().Add(ma.Color, 1)
			}
			return nil
		}
	}
	return fmt.Errorf("permanent has no mana ability")
}

// ManaSourceInfo describes a mana source available for tapping.
type ManaSourceInfo struct {
	PermanentID uuid.UUID
	Name        string
	Color       Color
}

// GetUntappedManaSources returns all untapped permanents with mana abilities for a player.
func (g *Game) GetUntappedManaSources(playerID uuid.UUID) []ManaSourceInfo {
	var sources []ManaSourceInfo
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID || perm.Tapped {
			continue
		}
		// Skip summoning-sick creatures without haste
		if !perm.CanTapForEffect(g) {
			continue
		}
		for _, a := range perm.RuntimeAbilities {
			if ma, ok := a.(*ManaAbility); ok {
				sources = append(sources, ManaSourceInfo{
					PermanentID: perm.ID(),
					Name:        perm.Name(),
					Color:       ma.Color,
				})
				break // one entry per permanent even if it has multiple mana abilities
			}
		}
	}
	return sources
}

// AutoTapForCost taps untapped lands/mana sources to pay a mana cost.
func (g *Game) AutoTapForCost(playerID uuid.UUID, mc ManaCost) error {
	sources := g.GetUntappedManaSources(playerID)

	// Collect how much of each color we need
	needed := map[Color]int{
		White: mc.White,
		Blue:  mc.Blue,
		Black: mc.Black,
		Red:   mc.Red,
		Green: mc.Green,
	}
	genericNeeded := mc.Generic

	var toTap []uuid.UUID

	// First pass: tap sources for exact color requirements
	for color, count := range needed {
		for i := 0; i < count; i++ {
			found := false
			for j, src := range sources {
				if src.Color == color && src.PermanentID != uuid.Nil {
					toTap = append(toTap, src.PermanentID)
					sources[j].PermanentID = uuid.Nil // mark as used
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("insufficient %s mana", color)
			}
		}
	}

	// Second pass: tap remaining sources for generic mana
	for i := 0; i < genericNeeded; i++ {
		found := false
		for j, src := range sources {
			if src.PermanentID != uuid.Nil {
				toTap = append(toTap, src.PermanentID)
				sources[j].PermanentID = uuid.Nil
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("insufficient mana for generic cost")
		}
	}

	// Actually tap all selected sources
	for _, id := range toTap {
		if err := g.TapForMana(playerID, id); err != nil {
			return err
		}
	}

	return nil
}

// CanAfford returns true if a player has enough untapped mana sources to pay a cost.
func (g *Game) CanAfford(playerID uuid.UUID, mc ManaCost) bool {
	sources := g.GetUntappedManaSources(playerID)

	avail := map[Color]int{}
	for _, src := range sources {
		avail[src.Color]++
	}

	remaining := 0
	for _, color := range []Color{White, Blue, Black, Red, Green} {
		need := 0
		switch color {
		case White:
			need = mc.White
		case Blue:
			need = mc.Blue
		case Black:
			need = mc.Black
		case Red:
			need = mc.Red
		case Green:
			need = mc.Green
		}
		if avail[color] < need {
			return false
		}
		remaining += avail[color] - need
	}
	remaining += avail[Colorless]
	return remaining >= mc.Generic
}

// ActivatableInfo describes an activated ability on a permanent that can currently be used.
type ActivatableInfo struct {
	PermanentID   uuid.UUID
	PermanentName string
	AbilityIndex  int
	Description   string
}

// GetCastableSpells returns cards in a player's hand they can currently cast.
func (g *Game) GetCastableSpells(playerID uuid.UUID) []Card {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil
	}
	isMainPhase := g.Step.IsMainPhase()
	isActive := g.ActivePlayerObj().PlayerID() == playerID

	var castable []Card
	for _, card := range p.Hand() {
		if card.HasType(TypeLand) {
			continue
		}
		// Sorceries can only be cast at sorcery speed (main phase, active player, empty stack)
		if card.HasType(TypeSorcery) {
			if !isMainPhase || !isActive || !g.Stack.IsEmpty() {
				continue
			}
		}
		// Creatures/artifacts/enchantments are sorcery speed
		if card.HasType(TypeCreature) || card.HasType(TypeArtifact) || card.HasType(TypeEnchantment) {
			if !isMainPhase || !isActive || !g.Stack.IsEmpty() {
				continue
			}
		}
		// Check mana (ignore X costs for now - those are always "castable" if base cost met)
		mc := card.ManaCost()
		checkMC := mc
		if mc.HasX {
			// For X spells, check if we can pay the non-X portion
			checkMC = ManaCost{
				Generic: mc.Generic,
				White:   mc.White,
				Blue:    mc.Blue,
				Black:   mc.Black,
				Red:     mc.Red,
				Green:   mc.Green,
			}
		}
		if !g.CanAfford(playerID, checkMC) {
			continue
		}
		castable = append(castable, card)
	}
	return castable
}

// GetActivatableAbilities returns activated abilities the player can currently use.
func (g *Game) GetActivatableAbilities(playerID uuid.UUID) []ActivatableInfo {
	var result []ActivatableInfo
	for _, perm := range g.Battlefield {
		isOwner := perm.Controller == playerID
		for i, a := range perm.RuntimeAbilities {
			aa, ok := a.(ActivatedAbility)
			if !ok {
				continue
			}
			// Check if this ability can be used by non-controllers
			if !isOwner {
				saa, isSAA := UnwrapAbility(a).(*SimpleActivatedAbility)
				if !isSAA || !saa.AnyPlayerMayUse {
					continue
				}
			}
			// Skip mana abilities - those are handled separately
			if _, isMana := a.(*ManaAbility); isMana {
				continue
			}
			if !aa.CanActivate(playerID, g) {
				continue
			}
			if aa.SorcerySpeed() && !g.Step.IsMainPhase() {
				continue
			}
			desc := ""
			for _, e := range aa.Effects() {
				if desc != "" {
					desc += ", "
				}
				desc += e.Text()
			}
			result = append(result, ActivatableInfo{
				PermanentID:   perm.ID(),
				PermanentName: perm.Name(),
				AbilityIndex:  i,
				Description:   desc,
			})
		}
	}
	return result
}

// CastSpellByID casts a spell from a player's hand by card ID.
func (g *Game) CastSpellByID(playerID, cardID uuid.UUID, targets []uuid.UUID, xValue int) error {
	p := g.GetPlayer(playerID)
	if p == nil {
		return ErrPlayerNotFound
	}

	// Find card in hand
	var card Card
	for _, c := range p.Hand() {
		if c.ID() == cardID {
			card = c
			break
		}
	}
	if card == nil {
		return ErrCardNotInHand
	}

	// Check expansion block (City in a Bottle)
	if exp := card.Expansion(); exp != "" && g.Effects.Rules.IsExpansionBlocked(exp) {
		return fmt.Errorf("can't cast %s: expansion %s is blocked", card.Name(), exp)
	}

	// Compute payment mana cost
	mc := card.ManaCost()
	payMC := mc
	if mc.HasX {
		payMC.Generic += xValue * mc.XCount
	}

	// Auto-tap lands to pay the cost
	if !payMC.IsZero() {
		if err := g.AutoTapForCost(playerID, payMC); err != nil {
			return fmt.Errorf("cannot pay for %s: %v", card.Name(), err)
		}
		// Now pay from the mana pool
		if err := p.ManaPool().Pay(payMC); err != nil {
			return err
		}
	}

	// Remove from hand
	p.RemoveFromHand(card.ID())

	// Build effects from spell abilities
	var effects []Effect
	for _, a := range card.Abilities() {
		if sa, ok := a.(*SpellAbility); ok {
			effects = append(effects, sa.Effects()...)
		}
	}

	obj := &StackObject{
		ID:         uuid.New(),
		Card:       card,
		Controller: playerID,
		SourceID:   card.ID(),
		Effects:    effects,
		Targets:    targets,
		XValue:     xValue,
	}

	if modes := card.Modes(); len(modes) > 0 {
		obj.ModeChoice = p.ChooseMode(modes, card.Name())
	}

	g.Stack.Push(obj)

	g.FireEvent(GameEvent{
		Type:     EvtSpellCast,
		SourceID: card.ID(),
		PlayerID: playerID,
	})

	return nil
}

// ActivateAbilityByIndex activates an ability on a permanent by index.
func (g *Game) ActivateAbilityByIndex(playerID, permanentID uuid.UUID, abilityIndex int, targets []uuid.UUID) error {
	perm := g.FindPermanent(permanentID)
	if perm == nil {
		return ErrPermanentNotFound
	}
	if abilityIndex < 0 || abilityIndex >= len(perm.RuntimeAbilities) {
		return fmt.Errorf("invalid ability index")
	}

	a := perm.RuntimeAbilities[abilityIndex]
	aa, ok := a.(ActivatedAbility)
	if !ok {
		return fmt.Errorf("not an activated ability")
	}
	if !aa.CanActivate(playerID, g) {
		return fmt.Errorf("cannot activate ability")
	}
	if aa.SorcerySpeed() && !g.Step.IsMainPhase() {
		return ErrSorcerySpeed
	}

	// Pay costs
	for _, c := range aa.Costs() {
		if err := c.Pay(perm.ID(), playerID, g); err != nil {
			return err
		}
	}

	// Mark once-per-turn abilities as used
	if saa, ok := aa.(*SimpleActivatedAbility); ok {
		saa.MarkActivated()
	}

	obj := &StackObject{
		ID:         uuid.New(),
		Controller: playerID,
		SourceID:   perm.ID(),
		IsAbility:  true,
		Targets:    targets,
	}
	obj.Effects = append(obj.Effects, aa.Effects()...)

	g.Stack.Push(obj)
	return nil
}

// ResolveTopOfStack resolves just the top item on the stack.
func (g *Game) ResolveTopOfStack() {
	if g.Stack.IsEmpty() {
		return
	}
	obj := g.Stack.Pop()
	g.ResolveStackObject(obj)
	g.PutTriggersOnStack()
}

// IsGameOver returns true if any player has 0 or less life.
func (g *Game) IsGameOver() bool {
	for _, p := range g.Players {
		if !p.IsAlive() {
			return true
		}
	}
	return false
}

// Winner returns the name of the winning player, or "" if no winner yet.
func (g *Game) Winner() string {
	for _, p := range g.Players {
		if !p.IsAlive() {
			return g.GetOpponent(p.PlayerID()).Name()
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
