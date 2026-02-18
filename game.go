package mage

import (
	"errors"
	"fmt"

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

// Game is the central game state and engine.
type Game struct {
	Players     []Player
	Battlefield []*Permanent
	Exile       []Card // exile zone
	Stack       *Stack
	Combat      *Combat
	Effects     *EffectManager

	Turn     int
	Step     PhaseStep
	ActivePlayer int // index into Players

	// Event handling
	pendingTriggers []*pendingTrigger

	// Damage prevention
	PreventCombatDamage bool

	// Extra turns
	ExtraTurns []uuid.UUID // player IDs who get extra turns

	// X value for the currently resolving spell
	CurrentX int

	// Card currently being resolved (set during ResolveStackObject)
	ResolvingCard Card

	// Interactive play tracking
	LandsPlayedThisTurn int

	// Damage tracking: maps target permanent ID -> set of source permanent IDs that dealt damage this turn
	DamageDealtBy map[uuid.UUID]map[uuid.UUID]bool


	// Delayed triggers
	delayedTriggers []*DelayedTrigger

	// Control flags
	stopped bool
}

// DelayedTrigger represents a one-shot triggered ability that fires when
// a specific event occurs (e.g., "destroy this creature at end of turn").
type DelayedTrigger struct {
	EventType  EventType
	TargetID   uuid.UUID
	Effects    []Effect
	SourceID   uuid.UUID
	Controller uuid.UUID
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
		Players:      []Player{playerA, playerB},
		Stack:        NewStack(),
		Combat:       NewCombat(),
		Effects:      NewEffectManager(),
		Turn:         1,
		DamageDealtBy: make(map[uuid.UUID]map[uuid.UUID]bool),
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
func (g *Game) FindPermanent(id uuid.UUID) *Permanent {
	for _, p := range g.Battlefield {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// FindPermanentByName finds a permanent by name on the battlefield (first match).
func (g *Game) FindPermanentByName(name string, controller uuid.UUID) *Permanent {
	for _, p := range g.Battlefield {
		if p.Name() == name && p.Controller == controller {
			return p
		}
	}
	return nil
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
	return nil
}

// PutOnBattlefield puts a card onto the battlefield under the given controller.
func (g *Game) PutOnBattlefield(card Card, controller uuid.UUID) *Permanent {
	perm := NewPermanent(card, controller)

	// Set ability sources and controllers
	for _, a := range perm.RuntimeAbilities {
		a.SetSource(perm.ID())
		a.SetController(controller)
	}

	// EntersTapped keyword check
	if perm.HasAbility(EntersTapped) {
		perm.Tapped = true
	}

	// Add X counters if configured (replacement effect, not a trigger)
	for _, a := range perm.RuntimeAbilities {
		if xc, ok := a.(*EntersWithXCountersAbility); ok && g.CurrentX > 0 {
			perm.AddCounter(xc.CounterType, g.CurrentX)
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

// RemoveFromBattlefield removes a permanent and handles cleanup.
func (g *Game) RemoveFromBattlefield(perm *Permanent) {
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

	g.FireEvent(GameEvent{
		Type:     EvtLeavesBattlefield,
		SourceID: perm.ID(),
		PlayerID: perm.Controller,
	})
}

// DestroyPermanent destroys a permanent (sends to graveyard).
func (g *Game) DestroyPermanent(perm *Permanent) {
	if perm.HasAbility(Indestructible) {
		return
	}
	// Regeneration replaces destruction: tap, remove damage, remove from combat
	if g.Effects.ConsumeRegenerationShield(perm.ID()) {
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
		diedEvt := GameEvent{
			Type:     EvtCreatureDied,
			SourceID: permID,
			PlayerID: controller,
		}
		g.FireEvent(diedEvt)
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
	})

	if isCreature {
		g.FireEvent(GameEvent{
			Type:     EvtCreatureDied,
			SourceID: permID,
			PlayerID: controller,
		})
	}
}

// ExilePermanent removes a permanent from the battlefield to exile.
func (g *Game) ExilePermanent(perm *Permanent) {
	card := perm.Card
	g.RemoveFromBattlefield(perm)
	g.Exile = append(g.Exile, card)
}

// CounterSpellOnStack removes a spell from the stack by its source ID.
func (g *Game) CounterSpellOnStack(spellID uuid.UUID) {
	g.Stack.RemoveBySourceID(spellID)
}

// DealDamageToPlayer deals damage to a player.
func (g *Game) DealDamageToPlayer(p Player, amount int, sourceID uuid.UUID) {
	if amount <= 0 {
		return
	}
	p.LoseLife(amount)
	g.FireEvent(GameEvent{
		Type:     EvtDamageDealt,
		SourceID: sourceID,
		TargetID: p.PlayerID(),
		Amount:   amount,
		Flag:     false, // not combat damage
	})
	// Lifelink
	src := g.FindPermanent(sourceID)
	if src != nil && src.HasAbility(Lifelink) {
		srcPlayer := g.GetPlayer(src.Controller)
		if srcPlayer != nil {
			srcPlayer.GainLife(amount)
		}
	}
}

// DealDamageToPermanent deals damage to a permanent.
func (g *Game) DealDamageToPermanent(perm *Permanent, amount int, sourceID uuid.UUID) {
	if amount <= 0 {
		return
	}
	// Apply damage prevention shield
	if prevented := g.Effects.PreventDamage(perm.ID(), amount); prevented > 0 {
		amount -= prevented
	}
	if amount <= 0 {
		return
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
	// Deathtouch
	src := g.FindPermanent(sourceID)
	if src != nil && src.HasAbility(Deathtouch) && amount > 0 {
		// Mark for destruction in SBAs
		perm.Damage = perm.CurrentToughness(g)
	}
	// Lifelink
	if src != nil && src.HasAbility(Lifelink) {
		srcPlayer := g.GetPlayer(src.Controller)
		if srcPlayer != nil {
			srcPlayer.GainLife(amount)
		}
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
		for _, e := range pt.ability.Effects() {
			obj.Effects = append(obj.Effects, e)
		}
		// For triggers that need to pass the event's player as a target
		// (e.g., "deal damage to that land's controller"), store the event
		// PlayerID as a target on the stack object.
		if gt, ok := pt.ability.(*GenericTriggered); ok && gt.eventType == EvtEntersBattlefield {
			if pt.event != nil && pt.event.PlayerID != uuid.Nil {
				obj.Targets = []uuid.UUID{pt.event.PlayerID}
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

// ResolveStackObject resolves a single stack object.
func (g *Game) ResolveStackObject(obj *StackObject) {
	g.CurrentX = obj.XValue
	g.ResolvingCard = obj.Card
	for _, eff := range obj.Effects {
		eff.Apply(g, obj.SourceID, obj.Controller, obj.Targets)
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

			// Handle aura attachment
			if obj.Card.HasType(TypeEnchantment) && len(obj.Targets) > 0 {
				g.Attach(perm.ID(), obj.Targets[0])
			}

			g.CurrentX = 0
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
	g.ResolvingCard = nil

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

	// Determine X value
	xValue := 0
	if len(xValues) > 0 {
		xValue = xValues[0]
	}

	// Pay mana cost (auto-pay from pool)
	mc := card.ManaCost()
	// For X costs, add X to the generic cost for payment purposes
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

		// Pay costs
		for _, c := range aa.Costs() {
			if err := c.Pay(perm.ID(), playerID, g); err != nil {
				return err
			}
		}

		obj := &StackObject{
			ID:         uuid.New(),
			Controller: playerID,
			SourceID:   perm.ID(),
			IsAbility:  true,
			Targets:    targets,
		}
		for _, e := range aa.Effects() {
			obj.Effects = append(obj.Effects, e)
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
		if perm.Tapped {
			continue // already tapped
		}
		perm.Tapped = true
		p := g.GetPlayer(playerID)
		if p != nil {
			p.ManaPool().Add(ma.Color, 1)
			// Check for mana bonus effects (e.g. Gauntlet of Might)
			g.applyManaBonuses(perm, ma.Color, p)
		}
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
				if mb.Filter(tappedPerm, g) {
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

		// Check for player death (life <= 0)
		// (Not destroying anything, just noting)

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
		g.doUntap()
	case Upkeep:
		g.doUpkeep()
	case Draw:
		g.doDraw()
	case DeclareAttackers:
		g.doDeclareAttackers()
	case DeclareBlockers:
		g.doDeclareBlockers()
	case FirstStrikeDamage:
		if !g.Combat.HasFirstStrikers(g) {
			return // skip if no first strikers
		}
		g.doCombatDamage(true)
	case CombatDamage:
		g.doCombatDamage(false)
	case EndCombat:
		g.Combat.Reset()
	case EndStep:
		g.doEndStep()
	case Cleanup:
		g.doCleanup()
	}

	// Check SBAs after each step
	g.CheckStateBasedActions()

	// Resolve stack
	g.ResolveStack()
}

func (g *Game) doEndStep() {
	active := g.ActivePlayerObj()
	g.FireEvent(GameEvent{
		Type:     EvtEndStep,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
	g.ResolveStack()
}

func (g *Game) doUntap() {
	active := g.ActivePlayerObj()
	g.Effects.ClearRegenerationShields(active.PlayerID(), g)
	for _, p := range g.Battlefield {
		if p.Controller == active.PlayerID() {
			if !p.HasAbility(DoesNotUntapKW) {
				p.Tapped = false
			}
			p.SummonSick = false
		}
	}
	g.LandsPlayedThisTurn = 0
}

func (g *Game) doUpkeep() {
	active := g.ActivePlayerObj()

	// Check for graveyard returns (e.g. Nether Shadow)
	g.checkGraveyardReturns(active)

	g.FireEvent(GameEvent{
		Type:     EvtUpkeep,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
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

func (g *Game) doDraw() {
	if g.Turn == 1 && g.ActivePlayer == 0 {
		return // first player doesn't draw on turn 1
	}
	g.ActivePlayerObj().DrawCard()
}

func (g *Game) doDeclareAttackers() {
	active := g.ActivePlayerObj()
	attackerIDs := active.DeclareAttackers(g)
	defender := g.NonActivePlayerObj()

	for _, id := range attackerIDs {
		atk := g.FindPermanent(id)
		if atk == nil {
			continue
		}
		// Can't attack if tapped, summoning sick (without haste), or prevented
		if atk.Tapped {
			continue
		}
		if atk.SummonSick && !atk.HasAbility(Haste) {
			continue
		}
		if !g.Effects.CanAttack(id) {
			continue
		}
		// Defender: creature with defender can't attack
		if !CanAttackCheck(atk, g) {
			continue
		}

		// Tap attacker (unless vigilance)
		if !atk.HasAbility(Vigilance) {
			atk.Tapped = true
		}

		g.Combat.AddAttacker(id, defender.PlayerID())
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredAttacker,
			SourceID: id,
			PlayerID: active.PlayerID(),
		})
	}
}

func (g *Game) doDeclareBlockers() {
	nonActive := g.NonActivePlayerObj()
	assignments := nonActive.DeclareBlockers(g)
	if assignments == nil {
		return
	}

	for _, ba := range assignments {
		blocker := g.FindPermanent(ba.BlockerID)
		attacker := g.FindPermanent(ba.AttackerID)
		if blocker == nil || attacker == nil {
			continue
		}
		if blocker.Tapped {
			continue
		}
		if !CanBlock(blocker, attacker, g) {
			continue
		}
		// Landwalk: if attacker has landwalk and defender controls matching land, can't be blocked
		if HasLandwalkEvasion(attacker, nonActive.PlayerID(), g) {
			continue
		}
		g.Combat.AddBlocker(ba.BlockerID, ba.AttackerID)
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredBlocker,
			SourceID: ba.BlockerID,
			TargetID: ba.AttackerID,
			PlayerID: nonActive.PlayerID(),
		})
	}
}

func (g *Game) doCombatDamage(isFirstStrikeStep bool) {
	if g.PreventCombatDamage {
		return
	}
	for _, group := range g.Combat.Groups {
		atk := g.FindPermanent(group.AttackerID)
		if atk == nil {
			continue
		}

		if len(group.BlockerIDs) == 0 {
			// Unblocked — damage to defending player
			if g.Combat.DealsDamageInStep(atk, isFirstStrikeStep) {
				defender := g.GetPlayer(group.DefenderID)
				if defender != nil {
					dmg := atk.CurrentPower(g)
					g.DealDamageToPlayer(defender, dmg, atk.ID())
				}
			}
		} else {
			// Blocked — damage to/from blockers
			if g.Combat.DealsDamageInStep(atk, isFirstStrikeStep) {
				// Attacker deals damage to first blocker
				remainingDmg := atk.CurrentPower(g)
				for _, bid := range group.BlockerIDs {
					blk := g.FindPermanent(bid)
					if blk == nil {
						continue
					}
					needed := blk.CurrentToughness(g) - blk.Damage
					if needed <= 0 {
						continue
					}
					dealt := min(remainingDmg, needed)
					g.DealDamageToPermanent(blk, dealt, atk.ID())
					remainingDmg -= dealt
					if remainingDmg <= 0 {
						break
					}
				}
				// Trample: remaining damage goes to defending player
				if remainingDmg > 0 && atk.HasAbility(Trample) {
					defender := g.GetPlayer(group.DefenderID)
					if defender != nil {
						g.DealDamageToPlayer(defender, remainingDmg, atk.ID())
					}
				}
			}

			// Each blocker deals damage to the attacker
			for _, bid := range group.BlockerIDs {
				blk := g.FindPermanent(bid)
				if blk == nil {
					continue
				}
				if g.Combat.DealsDamageInStep(blk, isFirstStrikeStep) {
					g.DealDamageToPermanent(atk, blk.CurrentPower(g), blk.ID())
				}
			}
		}
	}

	g.CheckStateBasedActions()
}

func (g *Game) doCleanup() {
	// Clear damage from all creatures
	for _, p := range g.Battlefield {
		p.Damage = 0
	}
	// Clear mana pools
	for _, p := range g.Players {
		p.ManaPool().Clear()
	}
	// Remove end-of-turn effects
	g.Effects.RemoveEndOfTurn()
	// Reset combat damage prevention
	g.PreventCombatDamage = false
	// Clear damage tracking
	g.DamageDealtBy = make(map[uuid.UUID]map[uuid.UUID]bool)
	// Clear damage prevention shields
	g.Effects.ClearPreventionShields()
	for _, p := range g.Battlefield {
		// Clear activation tracking (Charge counters used for per-turn counts)
		delete(p.Counters, Charge)
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

// PlayLand moves a land from a player's hand to the battlefield.
func (g *Game) PlayLand(playerID, cardID uuid.UUID) error {
	if !g.Step.IsMainPhase() {
		return fmt.Errorf("can only play lands during a main phase")
	}
	if g.ActivePlayerObj().PlayerID() != playerID {
		return fmt.Errorf("only the active player can play a land")
	}
	if g.LandsPlayedThisTurn >= 1 {
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

	g.PutOnBattlefield(card, playerID)
	g.LandsPlayedThisTurn++
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
			if perm.HasType(TypeCreature) && perm.SummonSick && !perm.HasAbility(Haste) {
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
		if perm.HasType(TypeCreature) && perm.SummonSick && !perm.HasAbility(Haste) {
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
		if perm.Controller != playerID {
			continue
		}
		for i, a := range perm.RuntimeAbilities {
			aa, ok := a.(ActivatedAbility)
			if !ok {
				continue
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

	obj := &StackObject{
		ID:         uuid.New(),
		Controller: playerID,
		SourceID:   perm.ID(),
		IsAbility:  true,
		Targets:    targets,
	}
	for _, e := range aa.Effects() {
		obj.Effects = append(obj.Effects, e)
	}

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
