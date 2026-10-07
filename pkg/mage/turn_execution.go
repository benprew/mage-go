package mage

import (
	"math/rand"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// UntapPermanent untaps the given permanent unless it has a stun counter
// (CR 122.1g — "If a permanent with a stun counter would become untapped,
// remove a stun counter from it instead. It doesn't untap.") If the
// permanent has at least one stun counter, exactly one is removed and the
// permanent stays tapped; no EvtBecameUntapped fires. Otherwise, if the
// permanent is currently tapped it untaps and EvtBecameUntapped fires.
// Returns true if the permanent actually untapped.
func (g *Game) UntapPermanent(p *Permanent) bool {
	if p == nil {
		return false
	}
	p = g.MutablePermanent(p.ID())
	if p == nil {
		return false
	}
	if !g.untapWithoutEvent(p) {
		return false
	}
	if g.effects.expireSourceTapped(p.ID()) {
		g.ApplyContinuousEffects()
	}
	g.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: p.ID()})
	return true
}

func (g *Game) untapWithoutEvent(p *Permanent) bool {
	if p.Counters[Stun] > 0 {
		p.RemoveCounter(Stun, 1)
		return false
	}
	if !p.Tapped {
		return false
	}
	p.Tapped = false
	return true
}

// untapPermanentsTogether completes each untap before it sends untap events.
func (g *Game) untapPermanentsTogether(permanents []*Permanent) {
	untapped := make([]*Permanent, 0, len(permanents))
	for _, permanent := range permanents {
		live := g.MutablePermanent(permanent.ID())
		if live != nil && g.untapWithoutEvent(live) {
			untapped = append(untapped, live)
		}
	}
	expired := false
	for _, permanent := range untapped {
		expired = g.effects.expireSourceTapped(permanent.ID()) || expired
	}
	if expired {
		g.ApplyContinuousEffects()
	}
	for _, permanent := range untapped {
		g.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: permanent.ID()})
	}
}

// SkipNextUntap prevents untapping during the next untap step of the current
// controller. Repeated applications expire during that same step.
func (g *Game) SkipNextUntap(permanentID uuid.UUID) {
	if g.FindPermanent(permanentID) == nil {
		return
	}
	g.turns.SkipNextUntap(permanentID)
}

func (g *Game) doUntap() {
	active := g.ActivePlayerObj()

	// CR 502.1: Phase in all phased-out permanents under the active player's control
	// (excluding indirectly phased attachments whose host is still phased out).
	var toPhaseIn []uuid.UUID
	for _, p := range g.zones.battlefield {
		if p.PhasedOut && p.ControllerID() == active.PlayerID() {
			if !p.IsAttached() || g.FindPermanent(p.AttachedTo) != nil {
				toPhaseIn = append(toPhaseIn, p.ID())
			}
		}
	}
	if len(toPhaseIn) > 0 {
		g.zones.PhaseIn(toPhaseIn)
		g.effects.Apply(g)
	}

	matchingUntap := g.matchingUntapRestrictions()
	capturedUntap := g.turns.consumeCapturedUntap(active.PlayerID())
	// Island Sanctuary: clear protection at the start of the player's turn
	g.effects.Rules.ClearSanctuary(active.PlayerID())

	// Snapshot the active player's untapped land count before the untap loop
	// runs (CR 502.1 happens at the very start of the turn). Power Surge and
	// similar effects read this at upkeep.
	count := 0
	for _, p := range g.zones.battlefield {
		if p.ControllerID() == active.PlayerID() && p.HasType(TypeLand) && !p.Tapped {
			count++
		}
	}
	g.trackers.Turn.SetUntappedLandsAtTurnStart(active.PlayerID(), count)

	blocked := matchingUntap
	for _, p := range g.zones.battlefield {
		if p.ControllerID() == active.PlayerID() && (g.turns.ConsumeSkipNextUntap(p.ID()) || capturedUntap[p.ID()]) {
			blocked[p.ID()] = true
		}
	}
	selected := g.chooseStepUntaps(active, blocked)
	g.untapPermanentsTogether(selected)
	for _, p := range g.zones.battlefield {
		if !p.PhasedOut && p.ControllerID() == active.PlayerID() {
			if mp := g.MutablePermanent(p.ID()); mp != nil {
				mp.RevokeBaseAttr(AttrSummonSick)
			}
		}
	}
	g.trackers.Turn.ResetLandsPlayed()
}

// TODO this should "tell" turn to do upkeep actions and give turn a list of actions to do
// this should iterate over permanents & cards in the game and ask if they have upkeep actions to do
// What the fuck is this method doing here? Why isn't it part of Turn or similar?
func (g *Game) doUpkeepActions() {
	active := g.ActivePlayerObj()

	// Expire "until your next upkeep" effects for the active player.
	g.effects.RemoveUntilYourNextTurn(g, active.PlayerID())
	// Upkeep triggers must see characteristics after duration expiry.
	g.ApplyContinuousEffects()

	g.FireEvent(GameEvent{
		Type:     EvtUpkeep,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
}

func (g *Game) doDrawActions() {
	active := g.ActivePlayerObj()

	// Step conditions must use the state before the normal draw.
	g.FireEvent(GameEvent{
		Type:     EvtDrawStep,
		PlayerID: active.PlayerID(),
	})
}

func (g *Game) doDrawNormalDraw() {
	active := g.ActivePlayerObj()

	// Run through replacement pipeline (skip draw, Aladdin's Lamp, etc.)
	action := NewDrawCardAction(uuid.Nil, active.PlayerID(), true)
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return // draw was replaced (skip draw, Aladdin's Lamp, etc.)
	}

	g.drawCardRaw(active)
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
	newLib := make([]Card, 1, 1+len(lib)-count+len(rest))
	newLib[0] = chosen
	newLib = append(newLib, lib[count:]...)
	newLib = append(newLib, rest...)
	p.SetLibrary(newLib)
	g.drawCardRaw(p)
}

// 1. Rule 508.1: First, the active player declares attackers.
// 2. Rule 508.2: Second, the active player gets priority.
func (g *Game) doDeclareAttackers() {
	active := g.ActivePlayerObj()
	attackerIDs := active.DeclareAttackers(g)
	defender := g.NonActivePlayerObj()

	// Auto-add creatures with MustAttack keyword (from Nettling Imp, etc.)
	declared := make(map[uuid.UUID]bool)
	declaredCount := 0
	for _, id := range attackerIDs {
		declared[id] = true
	}
	for _, p := range g.zones.battlefield {
		if p.ControllerID() == active.PlayerID() && p.HasAttr(AttrMustAttack) && !declared[p.ID()] {
			if p.CanDeclareAsAttacker(g) {
				attackerIDs = append(attackerIDs, p.ID())
			}
		}
	}

	for _, id := range attackerIDs {
		if g.combat.IsAttacking(id) {
			continue
		}
		atk := g.FindPermanent(id)
		if atk == nil {
			continue
		}
		if !atk.CanDeclareAsAttacker(g) {
			continue
		}
		// Island Sanctuary: only flying or islandwalk creatures can attack
		if g.effects.Rules.IsSanctuaryActive(defender.PlayerID()) {
			if !atk.HasKeyword(Flying) && !atk.HasKeyword(Islandwalk) {
				continue
			}
		}

		// Attack costs (CR 508.1e): declined or unpayable means not declared.
		if !g.PayAttackCosts(atk, active.PlayerID(), true) {
			continue
		}
		atk = g.attackerAfterPayment(atk, active.PlayerID())
		if atk == nil {
			continue
		}

		// Tap attacker (unless vigilance)
		if !atk.HasKeyword(Vigilance) {
			g.TapPermanent(atk)
		}

		g.combat.AddAttackerIncarnation(id, atk.incarnationID, defender.PlayerID())
		declaredCount++
		g.trackers.Turn.RecordAttacked(id)
		g.trackers.Turn.RecordAttackedIncarnation(id, atk.incarnationID)
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredAttacker,
			SourceID: id,
			PlayerID: active.PlayerID(),
			TargetID: defender.PlayerID(),
		})
	}

	// Form attacking bands if the player has scripted them.
	if bf, ok := active.(BandFormer); ok {
		for _, band := range bf.GetBandFormations(g.turns.Turn(), g) {
			if g.isValidBand(band) {
				g.combat.AddBand(band)
			}
		}
	}

	// CR 506.5 — snapshot "attacks alone" once all attackers have been
	// declared this step.
	g.combat.SnapshotAttackedAlone()

	// CR 506.4 / 603.6e: once-per-combat "whenever one or more creatures
	// attack" trigger. Fired only if at least one creature was declared as
	// an attacker. evt.Amount carries the attacker count.
	if declaredCount > 0 {
		g.FireEvent(GameEvent{
			Type:     EvtAttackersDeclared,
			PlayerID: active.PlayerID(),
			Amount:   declaredCount,
		})
	}

	g.ResolveStack()
}

// AttackedDuringLastTurn reports whether the given creature attacked during playerID's most recent turn.
func (g *Game) AttackedDuringLastTurn(playerID, permID uuid.UUID) bool {
	if m := g.trackers.Turn.AttackedLastTurn(playerID); m != nil {
		return m[permID]
	}
	return false
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
		if perm == nil || !g.combat.IsAttacking(id) {
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
	blockedBefore := g.blockedAttackersBeforeDeclaration()
	pairsBefore := g.blockerPairsBeforeDeclaration()
	assignments := nonActive.DeclareBlockers(g)
	g.commitBlockAssignments(nonActive.PlayerID(), assignments)

	// CR 506.5 — snapshot "blocks alone" once all blockers have been
	// declared this step.
	g.combat.SnapshotBlockedAlone()
	g.fireDeclaredBlockerPairs(pairsBefore)
	g.fireNewBlockerPairs(pairsBefore)
	g.fireNewBlockerGroups(pairsBefore)
	g.fireNewlyBlockedAttackers(blockedBefore)
	g.fireUnblockedAttackers()

	// Fire EvtBlockersDecl once after all blockers are assigned
	g.FireEvent(GameEvent{
		Type:     EvtBlockersDecl,
		PlayerID: nonActive.PlayerID(),
	})
}

// doCleanupActions performs cleanup housekeeping and places any triggers on the stack.
// It returns true if state-based actions or triggers require another cleanup step.
func (g *Game) doCleanupActions() bool {
	active := g.ActivePlayerObj()
	g.FireEvent(GameEvent{
		Type:     EvtCleanup,
		PlayerID: active.PlayerID(),
	})
	// Hand size discard: active player discards down to max hand size (CR 514.1)
	p := active
	maxHS := g.effects.Rules.MaxHandSize(p.PlayerID())
	for maxHS >= 0 && len(p.Hand()) > maxHS {
		chosen := p.ChooseCardsFromHand(1, "discard to hand size", g)
		if len(chosen) == 0 {
			break
		}
		g.PlayerDiscard(p, chosen[0].ID())
	}
	// CR 514.2 includes phased-out permanents in damage removal.
	for _, p := range g.zones.battlefield {
		if p.Damage == 0 && !p.DeathtouchDamage {
			continue
		}
		p = g.MutablePermanentIncludingPhased(p.ID())
		if p == nil {
			continue
		}
		p.Damage = 0
		p.DeathtouchDamage = false
	}
	// Clear mana pools
	for _, p := range g.players {
		p.ManaPool().Clear()
	}
	// Remove end-of-turn effects and clear turn-scoped state
	g.effects.RemoveEndOfTurn()
	g.effects.Apply(g)
	g.effects.ClearReplacementsEndOfTurn()
	g.zones.ClearExileInstead()
	g.damage.expireEndOfTurnEffects()
	g.effects.Rules.ClearEndOfTurn()
	// Clear non-persistent delayed triggers (they only last "this turn")
	g.triggers.ClearEndOfTurn()
	// CR 514.3a keeps turn history available during cleanup priority.
	performed := g.checkStateBasedActions()
	if performed || !g.stack.IsEmpty() {
		return true
	}
	// Pending cleanup abilities can still need departure information.
	g.zones.ClearLKI()
	g.trackers.Turn.ResetForNewTurn(active.PlayerID())
	g.damage.ClearEndOfTurn()
	// Clear last-drawn-card tracking for all players
	for _, p := range g.players {
		p.ClearLastDrawnCard()
	}
	for _, p := range g.zones.battlefield {
		p = g.MutablePermanentIncludingPhased(p.ID())
		// Clear activation tracking (Charge counters used for per-turn counts)
		p.Counters[Charge] = 0
		// Cloned games can share abilities before this reset.
		p.RuntimeAbilities = clonePaymentAbilities(p.RuntimeAbilities)
		for _, a := range p.RuntimeAbilities {
			if aa, ok := UnwrapAbility(a).(*SimpleActivatedAbility); ok {
				aa.ResetActivation()
			}
		}
	}
	return false
}

// Run executes the game until the stop condition.
func (g *Game) Run(stopTurn int, stopStep PhaseStep, maxTurns int) {
	for g.turns.Turn() <= maxTurns {
		// Turn-skip (CR 500.11): if the active player's next turn is
		// marked as skipped, consume the skip and advance past this turn
		// without running any steps.
		activeID := g.ActivePlayerObj().PlayerID()
		if g.turns.ConsumeTurnSkip(activeID) {
			// Fall through to the extra-turn / next-player logic below.
		} else if g.RunTurn(stopTurn, stopStep) {
			return
		}
		// Check for extra turns
		if extraPlayerID, ok := g.turns.PopExtraTurn(); ok {
			// Find the player index
			for i, p := range g.players {
				if p.PlayerID() == extraPlayerID {
					g.turns.SetActivePlayerIndex(i)
					break
				}
			}
		} else {
			// Next turn: swap active player
			g.turns.SetActivePlayerIndex((g.turns.ActivePlayerIndex() + 1) % len(g.players))
		}
		g.turns.IncrementTurn()
	}
}
