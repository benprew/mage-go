package mage

import (
	"fmt"

	"github.com/google/uuid"
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

	// Control flags
	stopped bool
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
		Players: []Player{playerA, playerB},
		Stack:   NewStack(),
		Combat:  NewCombat(),
		Effects: NewEffectManager(),
		Turn:    1,
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
	switch eff := e.(type) {
	case *boostAttachedEffect:
		eff.sourceID = id
	case *grantKeywordAttachedEffect:
		eff.sourceID = id
	case *preventAttackEffect:
		eff.sourceID = id
	case *boostAllCreaturesEffect:
		eff.sourceID_ = id
	case *boostAllCreaturesIncludingSelfEffect:
		eff.sourceID_ = id
	case *grantKeywordToAllEffect:
		eff.sourceID_ = id
	case *controlChangeEffect:
		eff.sourceID_ = id
	case *boostControlledCreaturesEffect:
		eff.sourceID_ = id
	}
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
	perm.Damage += amount
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
		if _, ok := pt.ability.(*WheneverLandEntersBattlefieldTriggered); ok {
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
	for _, eff := range obj.Effects {
		eff.Apply(g, obj.SourceID, obj.Controller, obj.Targets)
	}
	g.CurrentX = 0

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

			g.CheckStateBasedActions()
			return
		}

		// Instants and sorceries go to graveyard
		p := g.GetPlayer(owner)
		if p != nil {
			p.AddToGraveyard(obj.Card)
		}
	}

	g.CheckStateBasedActions()
}

// CastSpellByName finds a card in player's hand, puts it on the stack.
func (g *Game) CastSpellByName(playerID uuid.UUID, name string, targets []uuid.UUID, xValues ...int) error {
	p := g.GetPlayer(playerID)
	if p == nil {
		return fmt.Errorf("player not found")
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
		aa, ok := a.(ActivatedAbilityI)
		if !ok {
			continue
		}
		if !aa.CanActivate(playerID, g) {
			continue
		}

		// Check sorcery speed
		if aa.SorcerySpeed() && !g.Step.IsMainPhase() {
			return fmt.Errorf("can only activate at sorcery speed")
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

	return fmt.Errorf("no activatable ability found on %s", permName)
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
	case Cleanup:
		g.doCleanup()
	}

	// Check SBAs after each step
	g.CheckStateBasedActions()

	// Resolve stack
	g.ResolveStack()
}

func (g *Game) doUntap() {
	active := g.ActivePlayerObj()
	for _, p := range g.Battlefield {
		if p.Controller == active.PlayerID() {
			p.Tapped = false
			p.SummonSick = false
		}
	}
}

func (g *Game) doUpkeep() {
	active := g.ActivePlayerObj()
	g.FireEvent(GameEvent{
		Type:     EvtUpkeep,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
	g.ResolveStack()
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
	blockers := nonActive.DeclareBlockers(g)
	if blockers == nil {
		return
	}

	for blockerID, attackerID := range blockers {
		blocker := g.FindPermanent(blockerID)
		attacker := g.FindPermanent(attackerID)
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
		g.Combat.AddBlocker(blockerID, attackerID)
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredBlocker,
			SourceID: blockerID,
			TargetID: attackerID,
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
