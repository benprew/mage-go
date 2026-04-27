package gametest

import (
	"fmt"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// TestGame provides a DSL for scripting and asserting game states.
type TestGame struct {
	*mage.Game
	t       *testing.T
	playerA *TestPlayer
	playerB *TestPlayer
	Combat  *mage.Combat
	Effects *mage.EffectManager
	Stack   *mage.Stack
	Battlefield []*mage.Permanent
	Schedule *mage.TurnSchedule
	Players []mage.Player
	OnPriority mage.PriorityHandler
	Step core.PhaseStep
	Turn int
	ActivePlayer int
	LandsPlayedThisTurn int
	CleanupPriorityRounds int
	stopAt  struct {
		turn int
		step core.PhaseStep
	}

	castActions     []castAction
	activateActions []activateAction
	counterActions  []counterAction
	actionSeq       int
}

type castAction struct {
	seq       int
	turn      int
	step      core.PhaseStep
	player    PlayerRef
	spell     string
	targets   []string
	xValue    int
	responses []responseAction

	// fired becomes true once the handler has popped this action.
	fired bool
	// responsesReleased becomes true once this cast's spell has been
	// observed on top of the stack and its responses have been enqueued.
	responsesReleased bool
}

type responseAction struct {
	player  PlayerRef
	spell   string
	perm    string
	targets []string
	xValue  int

	// parentCastIdx is the index into tg.castActions of the parent cast.
	// A response is gated on the parent's spell being the current top of
	// the stack (by name match), meaning no later spell/ability has been
	// layered on top of it.
	parentCastIdx int
	// nextResponseIdx is the index of this response within its parent's
	// responses slice; the response is only eligible once all prior
	// responses in that slice have already fired, preserving scripted order.
	nextResponseIdx int
	fired           bool
}

type activateAction struct {
	seq      int
	turn     int
	step     core.PhaseStep
	player   PlayerRef
	permName string
	targets  []string
	xValue   int
	fired    bool
}

type counterAction struct {
	turn   int
	step   core.PhaseStep
	player PlayerRef
	card   string
	ct     core.CounterType
	n      int
}

// NewTestGame creates a new test game with two test players.
func NewTestGame(t *testing.T) *TestGame {
	t.Helper()
	pA := NewTestPlayer("PlayerA")
	pB := NewTestPlayer("PlayerB")
	g := mage.NewGame(pA, pB)
	tg := &TestGame{
		Game:    g,
		t:       t,
		playerA: pA,
		playerB: pB,
	}
	tg.syncFromGame()
	return tg
}

func (tg *TestGame) syncFromGame() {
	tg.Combat = tg.GetCombat()
	tg.Effects = tg.GetEffects()
	tg.Stack = tg.GetStack()
	tg.Battlefield = tg.AllBattlefield()
	tg.Schedule = tg.GetSchedule()
	tg.Players = tg.AllPlayers()
	tg.Step = tg.GetStep()
	tg.Turn = tg.CurrentTurn()
	tg.ActivePlayer = tg.ActivePlayerIndex()
	tg.LandsPlayedThisTurn = tg.GetLandsPlayedThisTurn()
	tg.CleanupPriorityRounds = tg.GetCleanupPriorityRounds()
}

func (tg *TestGame) syncToGame() {
	tg.SetStep(tg.Step)
	tg.SetTurn(tg.Turn)
	tg.SetActivePlayerIndex(tg.ActivePlayer)
	tg.SetLandsPlayedThisTurn(tg.LandsPlayedThisTurn)
	tg.SetOnPriority(tg.OnPriority)
}

func (tg *TestGame) RunStepWithPriority(step core.PhaseStep) {
	tg.Step = step
	tg.syncToGame()
	tg.Game.RunStepWithPriority(step)
	tg.syncFromGame()
}

func (tg *TestGame) PlayLand(playerID, cardID uuid.UUID) error {
	tg.syncToGame()
	err := tg.Game.PlayLand(playerID, cardID)
	tg.syncFromGame()
	return err
}

func (tg *TestGame) GetPlayer(ref PlayerRef) *TestPlayer {
	if ref == PlayerA {
		return tg.playerA
	}
	return tg.playerB
}

func (tg *TestGame) getPlayerID(ref PlayerRef) uuid.UUID {
	return tg.GetPlayer(ref).PlayerID()
}

// AddCard adds a card to a zone for a player.
func (tg *TestGame) AddCard(zone core.Zone, p PlayerRef, name string, count ...int) uuid.UUID {
	tg.t.Helper()
	n := 1
	if len(count) > 0 {
		n = count[0]
	}
	player := tg.GetPlayer(p)
	playerID := player.PlayerID()

	var lastID uuid.UUID
	for i := 0; i < n; i++ {
		card, err := mage.CreateCard(name)
		if err != nil {
			tg.t.Fatalf("AddCard: %v", err)
		}
		card.SetOwner(playerID)

		switch zone {
		case core.ZoneBattlefield:
			perm := tg.PutOnBattlefield(card, playerID)
			perm.RevokeBaseAttr(core.AttrSummonSick)
			lastID = perm.ID()
		case core.ZoneHand:
			player.AddToHand(card)
		case core.ZoneGraveyard:
			player.AddToGraveyard(card)
		case core.ZoneLibrary:
			player.AddToLibrary(card)
		case core.ZoneAnte:
			player.AddToAnte(card)
		}
	}
	return lastID
}

// SetLife sets a player's life total.
func (tg *TestGame) SetLife(p PlayerRef, life int) {
	tg.GetPlayer(p).SetLife(life)
}

// AddCounters schedules adding counters during execution.
func (tg *TestGame) AddCounters(turn int, step core.PhaseStep, p PlayerRef, card string, ct core.CounterType, n int) {
	tg.counterActions = append(tg.counterActions, counterAction{
		turn: turn, step: step, player: p, card: card, ct: ct, n: n,
	})
}

// CastSpell scripts a spell cast at a specific turn/step.
func (tg *TestGame) CastSpell(turn int, step core.PhaseStep, p PlayerRef, spell string, targets ...string) {
	tg.actionSeq++
	tg.castActions = append(tg.castActions, castAction{
		seq: tg.actionSeq, turn: turn, step: step, player: p, spell: spell, targets: targets,
	})
}

// CastSpellWithX scripts an X-cost spell cast at a specific turn/step with a given X value.
func (tg *TestGame) CastSpellWithX(turn int, step core.PhaseStep, p PlayerRef, spell string, xValue int, targets ...string) {
	tg.actionSeq++
	tg.castActions = append(tg.castActions, castAction{
		seq: tg.actionSeq, turn: turn, step: step, player: p, spell: spell, targets: targets, xValue: xValue,
	})
}

// CastInResponseTo scripts a spell cast in response to the most recently scripted cast action.
func (tg *TestGame) CastInResponseTo(p PlayerRef, spell string, targets ...string) {
	if len(tg.castActions) == 0 {
		return
	}
	last := &tg.castActions[len(tg.castActions)-1]
	last.responses = append(last.responses, responseAction{
		player:  p,
		spell:   spell,
		targets: targets,
	})
}

// CastInResponseToWithX is like CastInResponseTo but for X-cost spells.
func (tg *TestGame) CastInResponseToWithX(p PlayerRef, spell string, xValue int, targets ...string) {
	if len(tg.castActions) == 0 {
		return
	}
	last := &tg.castActions[len(tg.castActions)-1]
	last.responses = append(last.responses, responseAction{
		player:  p,
		spell:   spell,
		targets: targets,
		xValue:  xValue,
	})
}

// ActivateInResponseTo scripts an activated ability in response to the most recently scripted cast action.
func (tg *TestGame) ActivateInResponseTo(p PlayerRef, permName string, targets ...string) {
	if len(tg.castActions) == 0 {
		return
	}
	last := &tg.castActions[len(tg.castActions)-1]
	last.responses = append(last.responses, responseAction{
		player:  p,
		perm:    permName,
		targets: targets,
	})
}

// ActivateAbility scripts an ability activation at a specific turn/step.
func (tg *TestGame) ActivateAbility(turn int, step core.PhaseStep, p PlayerRef, permName string, targets ...string) {
	tg.actionSeq++
	tg.activateActions = append(tg.activateActions, activateAction{
		seq: tg.actionSeq, turn: turn, step: step, player: p, permName: permName, targets: targets,
	})
}

// ActivateAbilityWithX scripts an ability activation with a specific X value.
func (tg *TestGame) ActivateAbilityWithX(turn int, step core.PhaseStep, p PlayerRef, permName string, x int, targets ...string) {
	tg.actionSeq++
	tg.activateActions = append(tg.activateActions, activateAction{
		seq: tg.actionSeq, turn: turn, step: step, player: p, permName: permName, targets: targets, xValue: x,
	})
}

// Attack scripts attacks for a turn.
func (tg *TestGame) Attack(turn int, p PlayerRef, creatures ...string) {
	tg.GetPlayer(p).SetAttackers(turn, creatures)
}

// Block scripts blocks for a turn.
func (tg *TestGame) Block(turn int, p PlayerRef, blocker, attacker string) {
	tp := tg.GetPlayer(p)
	tp.blockActions[turn] = append(tp.blockActions[turn], blockPair{blocker, attacker})
}

// ChooseTarget scripts which target a player will choose when prompted by
// ChooseTargets (e.g. for triggered abilities that declare .AddTarget(...)).
// The name is matched against possible player names (e.g. "PlayerA",
// "PlayerB") or permanent names.
func (tg *TestGame) ChooseTarget(p PlayerRef, name string) {
	tp := tg.GetPlayer(p)
	tp.chooseTarget = append(tp.chooseTarget, name)
}

// ChoosePermanent scripts which permanent a player will choose when asked.
func (tg *TestGame) ChoosePermanent(p PlayerRef, name string) {
	tp := tg.GetPlayer(p)
	tp.choosePermanent = append(tp.choosePermanent, name)
}

// ChooseDiscard scripts which card(s) a player will discard when asked.
func (tg *TestGame) ChooseDiscard(p PlayerRef, names ...string) {
	tp := tg.GetPlayer(p)
	tp.chooseDiscard = append(tp.chooseDiscard, names)
}

// ChooseManaColor scripts what mana color a player will choose when asked.
func (tg *TestGame) ChooseManaColor(p PlayerRef, color core.Color) {
	tp := tg.GetPlayer(p)
	tp.chooseManaColor = append(tp.chooseManaColor, color)
}

// ChooseFromLibrary scripts which card a player will find when searching library.
func (tg *TestGame) ChooseFromLibrary(p PlayerRef, name string) {
	tp := tg.GetPlayer(p)
	tp.chooseFromLibrary = append(tp.chooseFromLibrary, name)
}

// ChooseScry queues a scry placement for the next scry the player performs
// (CR 701.18). `bottom` lists card names (in placement order) that go to the
// bottom of the library; `topOrder` lists the remaining card names in their
// new top-of-library order. Both must reference cards revealed by the scry.
func (tg *TestGame) ChooseScry(p PlayerRef, bottom []string, topOrder []string) {
	tg.GetPlayer(p).AddScryDecision(bottom, topOrder)
}

// FormBand scripts which creatures form an attacking band on the given turn.
func (tg *TestGame) FormBand(turn int, p PlayerRef, creatures ...string) {
	tg.GetPlayer(p).AddBandFormation(turn, creatures)
}

// ChooseMode scripts which mode a player will choose when a modal spell or ability asks.
func (tg *TestGame) ChooseMode(p PlayerRef, mode int) {
	tp := tg.GetPlayer(p)
	tp.chooseMode = append(tp.chooseMode, mode)
}

// ChooseBandingDistribution scripts how a player distributes incoming damage among banded creatures.
func (tg *TestGame) ChooseBandingDistribution(p PlayerRef, distribution map[string]int) {
	tp := tg.GetPlayer(p)
	tp.chooseBandingDistribution = append(tp.chooseBandingDistribution, distribution)
}

// ChooseNumber scripts the number a player will choose when asked to pick a number.
func (tg *TestGame) ChooseNumber(p PlayerRef, n int) {
	tp := tg.GetPlayer(p)
	tp.chooseNumber = append(tp.chooseNumber, n)
}

// AssignCombatDamage scripts the attacker's controller's choice of how to
// divide the attacker's combat damage among multiple blockers (CR 510.1c).
// The owning player is inferred from whichever player controls the named
// attacker at damage time. Distribution maps blocker names to assigned damage.
func (tg *TestGame) AssignCombatDamage(attackerName string, distribution map[string]int) {
	tg.playerA.SetCombatDamageAssignment(attackerName, distribution)
	tg.playerB.SetCombatDamageAssignment(attackerName, distribution)
}

// ChooseBlockerOrder scripts the damage-assignment order the attacker's
// controller picks for a multi-blocker block (CR 510.1c, also relevant for
// trample CR 702.19b). Blockers are listed in the order they receive damage.
func (tg *TestGame) ChooseBlockerOrder(attackerName string, blockerNames ...string) {
	tg.playerA.SetCombatBlockerOrder(attackerName, blockerNames)
	tg.playerB.SetCombatBlockerOrder(attackerName, blockerNames)
}

// AssertBanded checks whether two named permanents are in the same attacking band.
func (tg *TestGame) AssertBanded(p1 PlayerRef, name1 string, p2 PlayerRef, name2 string, want bool) {
	tg.t.Helper()
	pid1 := tg.getPlayerID(p1)
	pid2 := tg.getPlayerID(p2)
	perm1 := tg.FindPermanentByName(name1, pid1)
	perm2 := tg.FindPermanentByName(name2, pid2)
	if perm1 == nil {
		tg.t.Errorf("AssertBanded: %s not found for %v", name1, p1)
		return
	}
	if perm2 == nil {
		tg.t.Errorf("AssertBanded: %s not found for %v", name2, p2)
		return
	}
	got := tg.IsBandedWith(perm1.ID(), perm2.ID())
	if got != want {
		tg.t.Errorf("AssertBanded(%s, %s): got %v, want %v", name1, name2, got, want)
	}
}

// StopAt sets when the game should stop.
func (tg *TestGame) StopAt(turn int, step core.PhaseStep) {
	tg.stopAt.turn = turn
	tg.stopAt.step = step
}

// padLibraries ensures each player has enough library cards to not deck out
// during normal test execution. Tests that explicitly test deck-out should
// not call this (or should empty the library after setup).
func (tg *TestGame) padLibraries() {
	for _, p := range tg.AllPlayers() {
		if len(p.Library()) == 0 {
			for i := 0; i < 60; i++ {
				p.AddToLibrary(mage.NewLand("Plains"))
			}
		}
	}
}

// Execute runs the game with all scripted actions.
func (tg *TestGame) Execute() {
	tg.t.Helper()

	tg.padLibraries()
	tg.autoAddMana()
	tg.OnPriority = tg.scriptedPriorityHandler()
	tg.syncToGame()

	maxTurns := tg.stopAt.turn + 5
	for tg.Turn <= maxTurns {
		// Turn-skip support (CR 500.11) via the engine schedule.
		activeID := tg.ActivePlayerObj().PlayerID()
		if tg.Schedule != nil && tg.Schedule.ConsumeTurnSkip(activeID) {
			tg.advanceToNextTurn()
			continue
		}
		if tg.Schedule == nil {
			tg.Schedule = mage.NewTurnSchedule()
		}
		for _, p := range tg.AllPlayers() {
			p.ManaPool().ResetProducedThisTurn()
		}
		tg.Schedule.BuildNextTurn()
		for {
			step, ok := tg.Schedule.PopNextStep()
			if !ok {
				break
			}
			if tg.Turn == tg.stopAt.turn && step == tg.stopAt.step {
				tg.Step = step
				tg.SetStep(step)
				tg.ApplyEffects()
				// Re-queue the stop step so a subsequent Execute resumes here.
				tg.Schedule.Remaining = append([]core.PhaseStep{step}, tg.Schedule.Remaining...)
				tg.syncFromGame()
				return
			}

			tg.Step = step
			tg.executeCounterActions(tg.Turn, step)

			tg.RunStepWithPriority(step)
		}
		tg.advanceToNextTurn()
	}
}

func (tg *TestGame) advanceToNextTurn() {
	if extraPlayerID, ok := tg.PopExtraTurn(); ok {
		for i, p := range tg.AllPlayers() {
			if p.PlayerID() == extraPlayerID {
				tg.ActivePlayer = i
				break
			}
		}
	} else {
		tg.ActivePlayer = (tg.ActivePlayer + 1) % len(tg.AllPlayers())
	}
	tg.Turn++
	tg.syncToGame()
	tg.syncFromGame()
}

// autoPassHandler returns a PriorityHandler that always passes priority.
// Used by engine-internals tests that don't want scripted-action behavior.
func autoPassHandler() mage.PriorityHandler {
	return func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
}

// scriptedPriorityHandler returns a PriorityHandler that drives the scripted
// cast / activate / response actions through the engine's priority loop, and
// auto-plays lands for the active player during their main phases.
//
// Ordering (CR 117.1b, 509.2, etc. are respected automatically because the
// engine now controls when priority opens):
//  1. Responses whose parent cast is currently on top of the stack, and
//     whose player-to-act matches, fire first.
//  2. Otherwise, for the step's active player on an empty stack, a queued
//     CastSpell / ActivateAbility whose (turn, step, player) gate matches
//     fires. Among those, actions fire in the order they were scripted
//     (via actionSeq).
//  3. Finally, the active player auto-plays one land per priority round in
//     a main phase if they have one in hand and haven't exceeded the cap.
//  4. Otherwise, pass.
func (tg *TestGame) scriptedPriorityHandler() mage.PriorityHandler {
	return func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		playerRef := tg.playerRefForIndex(playerIdx)

		// 1. Response actions whose parent is live on the stack.
		if top := g.StackPeek(); top != nil {
			if r := tg.popResponseFor(top.SourceID, playerRef); r != nil {
				return tg.buildPriorityActionForResponse(*r, playerIdx)
			}
		}

		// Only try to fire scheduled casts / activations and lands when
		// the stack is empty. If the stack has anything on it, and no
		// queued response matched, pass to let it resolve. Tests that
		// need a scripted action to interact with a same-step trigger
		// or another scripted spell must express that using
		// CastInResponseTo / ActivateInResponseTo — which gives CR-
		// correct response ordering (CR 117.1b).
		if g.StackSize() > 0 {
			return mage.PriorityAction{Type: mage.PriorityPass}
		}

		// 2. Scheduled cast / activate for this (turn, step, player).
		if action, ok := tg.popScheduledForStep(g.CurrentTurn(), g.GetStep(), playerRef, playerIdx); ok {
			return action
		}

		// 3. Active player auto-plays a land in a main phase.
		if mainPhase && playerIdx == g.ActivePlayerIndex() && g.GetLandsPlayedThisTurn() < g.MaxLandPlays() {
			active := g.PlayerAt(playerIdx)
			for _, c := range active.Hand() {
				if c.HasType(core.TypeLand) {
					return mage.PriorityAction{
						Type:   mage.PriorityPlayLand,
						CardID: c.ID(),
					}
				}
			}
		}

		return mage.PriorityAction{Type: mage.PriorityPass}
	}
}

// playerRefForIndex maps a player index into the PlayerRef enum.
func (tg *TestGame) playerRefForIndex(idx int) PlayerRef {
	if idx < len(tg.AllPlayers()) && tg.PlayerAt(idx).PlayerID() == tg.playerA.PlayerID() {
		return PlayerA
	}
	return PlayerB
}

func (tg *TestGame) ensureManaForCast(ca castAction) {
	player := tg.GetPlayer(ca.player)
	card, err := mage.CreateCard(ca.spell)
	if err != nil {
		return
	}
	mc := card.ManaCost()
	pool := player.ManaPool()

	// Let the engine auto-tap untapped mana sources for the cost; only
	// pre-fill the pool with what those sources cannot cover. This keeps
	// mana-ability side effects (source becomes tapped, mana credited via
	// the engine path) visible to tests that assert on them.
	supply := tg.untappedManaSupply(player.PlayerID())
	totalSupply := 0
	for _, v := range supply {
		totalSupply += v
	}
	colors := []struct {
		color core.Color
		need  int
	}{
		{core.White, mc.White}, {core.Blue, mc.Blue}, {core.Black, mc.Black},
		{core.Red, mc.Red}, {core.Green, mc.Green},
	}
	for _, c := range colors {
		have := pool.Count(c.color)
		shortfall := c.need - have
		if shortfall <= 0 {
			continue
		}
		avail := supply[c.color]
		if avail >= shortfall {
			supply[c.color] -= shortfall
			totalSupply -= shortfall
			continue
		}
		supply[c.color] = 0
		totalSupply -= avail
		pool.Add(c.color, shortfall-avail)
	}

	genericNeeded := mc.Generic
	if mc.HasX {
		genericNeeded += ca.xValue * mc.XCount
	}
	if genericNeeded > 0 {
		deficit := genericNeeded - totalSupply
		if deficit > 0 {
			pool.Add(core.Colorless, deficit)
		}
	}
}

// untappedManaSupply returns the per-color production capacity available to
// playerID from their untapped mana sources (one tap each).
func (tg *TestGame) untappedManaSupply(playerID uuid.UUID) map[core.Color]int {
	out := map[core.Color]int{}
	for _, perm := range tg.AllBattlefield() {
		if perm.Controller != playerID || perm.Tapped {
			continue
		}
		if !perm.CanTapForEffect(tg.Game) {
			continue
		}
		for _, a := range perm.RuntimeAbilities {
			inner := mage.UnwrapAbility(a)
			ma, ok := inner.(*mage.ManaAbility)
			if !ok {
				continue
			}
			out[ma.PrimaryColor()] += ma.ProducedAmount()
			break
		}
	}
	return out
}

func (tg *TestGame) ensureManaForResponse(r responseAction) {
	player := tg.GetPlayer(r.player)
	card, err := mage.CreateCard(r.spell)
	if err != nil {
		return
	}
	mc := card.ManaCost()
	pool := player.ManaPool()

	colors := []struct {
		color core.Color
		need  int
	}{
		{core.White, mc.White}, {core.Blue, mc.Blue}, {core.Black, mc.Black},
		{core.Red, mc.Red}, {core.Green, mc.Green},
	}
	for _, c := range colors {
		have := pool.Count(c.color)
		if have < c.need {
			pool.Add(c.color, c.need-have)
		}
	}

	genericNeeded := mc.Generic
	if mc.HasX {
		genericNeeded += r.xValue * mc.XCount
	}
	if genericNeeded > 0 {
		pool.Add(core.Colorless, genericNeeded)
	}
}

func (tg *TestGame) autoAddMana() {
	for _, ca := range tg.castActions {
		player := tg.GetPlayer(ca.player)
		card, err := mage.CreateCard(ca.spell)
		if err != nil {
			continue
		}
		mc := card.ManaCost()
		pool := player.ManaPool()
		pool.Add(core.White, mc.White)
		pool.Add(core.Blue, mc.Blue)
		pool.Add(core.Black, mc.Black)
		pool.Add(core.Red, mc.Red)
		pool.Add(core.Green, mc.Green)
		pool.Add(core.Colorless, mc.Generic)
		if mc.HasX {
			pool.Add(core.Colorless, ca.xValue*mc.XCount)
		}

		for _, r := range ca.responses {
			if r.spell != "" {
				rCard, err := mage.CreateCard(r.spell)
				if err != nil {
					continue
				}
				rmc := rCard.ManaCost()
				rPlayer := tg.GetPlayer(r.player)
				rPool := rPlayer.ManaPool()
				rPool.Add(core.White, rmc.White)
				rPool.Add(core.Blue, rmc.Blue)
				rPool.Add(core.Black, rmc.Black)
				rPool.Add(core.Red, rmc.Red)
				rPool.Add(core.Green, rmc.Green)
				rPool.Add(core.Colorless, rmc.Generic)
				if rmc.HasX {
					rPool.Add(core.Colorless, r.xValue*rmc.XCount)
				}
			}
			if r.perm != "" {
				rPlayer := tg.GetPlayer(r.player)
				rPlayer.ManaPool().Add(core.White, 5)
				rPlayer.ManaPool().Add(core.Blue, 5)
				rPlayer.ManaPool().Add(core.Black, 5)
				rPlayer.ManaPool().Add(core.Red, 5)
				rPlayer.ManaPool().Add(core.Green, 5)
				rPlayer.ManaPool().Add(core.Colorless, 10)
			}
		}
	}

	// Activation mana is added just-in-time in ensureManaForActivate.
}

// popScheduledForStep looks for the earliest-seq queued cast/activate whose
// (turn, step, player) matches and pops it, returning a PriorityAction.
func (tg *TestGame) popScheduledForStep(turn int, step core.PhaseStep, ref PlayerRef, playerIdx int) (mage.PriorityAction, bool) {
	bestSeq := -1
	bestCastIdx := -1
	bestActIdx := -1
	for i := range tg.castActions {
		ca := &tg.castActions[i]
		if ca.fired || ca.turn != turn || ca.step != step || ca.player != ref {
			continue
		}
		if bestSeq == -1 || ca.seq < bestSeq {
			bestSeq = ca.seq
			bestCastIdx = i
			bestActIdx = -1
		}
	}
	for i := range tg.activateActions {
		aa := &tg.activateActions[i]
		if aa.fired || aa.turn != turn || aa.step != step || aa.player != ref {
			continue
		}
		if bestSeq == -1 || aa.seq < bestSeq {
			bestSeq = aa.seq
			bestActIdx = i
			bestCastIdx = -1
		}
	}
	if bestCastIdx >= 0 {
		return tg.buildPriorityActionForCast(bestCastIdx)
	}
	if bestActIdx >= 0 {
		return tg.buildPriorityActionForActivate(bestActIdx)
	}
	return mage.PriorityAction{}, false
}

// buildPriorityActionForCast prepares a scripted cast for the priority loop.
// It resolves targets, ensures mana, finds the card by name in the player's
// hand, and returns a PriorityCastSpell action. The cast is marked fired
// immediately so we don't re-fire on every priority round; if the engine
// rejects it, the action is simply lost (consistent with the prior harness
// behavior, which logged and moved on).
func (tg *TestGame) buildPriorityActionForCast(idx int) (mage.PriorityAction, bool) {
	ca := &tg.castActions[idx]
	ca.fired = true
	playerID := tg.getPlayerID(ca.player)
	targets := tg.resolveTargets(ca.targets, playerID)

	tg.ensureManaForCast(*ca)

	card, _ := mage.CreateCard(ca.spell)
	if card != nil {
		validTargets := tg.validateTargets(card, targets, playerID)
		if len(validTargets) == 0 && len(targets) > 0 {
			return mage.PriorityAction{}, false
		}
		targets = validTargets
	}

	// Locate the real card in the player's hand.
	p := tg.Game.GetPlayer(playerID)
	var cardID uuid.UUID
	if p != nil {
		for _, c := range p.Hand() {
			if c.Name() == ca.spell {
				cardID = c.ID()
				break
			}
		}
	}
	if cardID == uuid.Nil {
		tg.t.Logf("CastSpell %s: card not in %v hand", ca.spell, ca.player)
		return mage.PriorityAction{}, false
	}

	return mage.PriorityAction{
		Type:    mage.PriorityCastSpell,
		CardID:  cardID,
		Targets: targets,
		XValue:  ca.xValue,
	}, true
}

func (tg *TestGame) buildPriorityActionForActivate(idx int) (mage.PriorityAction, bool) {
	aa := &tg.activateActions[idx]
	aa.fired = true
	playerID := tg.getPlayerID(aa.player)
	targets := tg.resolveTargets(aa.targets, playerID)
	tg.ensureManaForActivate(*aa)
	tg.SetXValue(aa.xValue)
	perm, abilityIdx, err := tg.findActivatableAbilityByName(playerID, aa.permName, targets)
	if err != nil {
		tg.t.Logf("ActivateAbility %s failed: %v", aa.permName, err)
		return mage.PriorityAction{}, false
	}
	return mage.PriorityAction{
		Type:        mage.PriorityActivateAbility,
		PermanentID: perm.ID(),
		AbilityIdx:  abilityIdx,
		Targets:     targets,
		XValue:      aa.xValue,
	}, true
}

// popResponseFor looks up a queued response whose parent cast is currently
// the top object on the stack (by card ID) and whose responder matches the
// player currently being asked for priority. Responses fire in the scripted
// order within their parent (nextResponseIdx guard).
func (tg *TestGame) popResponseFor(topSourceID uuid.UUID, ref PlayerRef) *responseAction {
	for ci := range tg.castActions {
		ca := &tg.castActions[ci]
		if !ca.fired || len(ca.responses) == 0 {
			continue
		}
		// Parent cast's spell must still be the top of the stack.
		// CastSpellByID sets SourceID to the spell's card ID.
		playerID := tg.getPlayerID(ca.player)
		p := tg.Game.GetPlayer(playerID)
		_ = p
		// Find the card ID of this cast's spell by scanning the stack:
		// we compare by name because the underlying card ID isn't
		// preserved across the cast.
		obj := tg.StackPeek()
		if obj == nil {
			continue
		}
		// Match stack top to this cast by spell name.
		if !stackTopMatchesSpell(tg.Game, obj, ca.spell) {
			continue
		}
		// Find first unfired response whose player matches.
		for ri := range ca.responses {
			r := &ca.responses[ri]
			if r.fired {
				continue
			}
			// Maintain scripted order within this cast's responses.
			if r.player != ref {
				return nil
			}
			r.fired = true
			return r
		}
	}
	return nil
}

// stackTopMatchesSpell returns true if the top stack object represents a
// spell with the given name. The Stack stores StackObjects with Card pointers
// for spells; abilities have IsAbility=true and a nil Card.
func stackTopMatchesSpell(g *mage.Game, obj *mage.StackObject, name string) bool {
	if obj == nil {
		return false
	}
	if obj.IsAbility {
		return false
	}
	if obj.Card != nil && obj.Card.Name() == name {
		return true
	}
	return false
}

func (tg *TestGame) buildPriorityActionForResponse(r responseAction, playerIdx int) mage.PriorityAction {
	respPlayerID := tg.PlayerAt(playerIdx).PlayerID()

	var targets []uuid.UUID
	if len(r.targets) > 0 {
		targets = tg.resolveTargets(r.targets, respPlayerID)
	} else if tg.StackPeek() != nil {
		targets = []uuid.UUID{tg.StackPeek().SourceID}
	}

	if r.perm != "" {
		// Ensure responder has enough mana to activate; CR 500.5's
		// per-step pool empty means pre-turn top-up doesn't survive.
		respPlayer := tg.GetPlayer(r.player)
		pool := respPlayer.ManaPool()
		pool.Add(core.White, 5)
		pool.Add(core.Blue, 5)
		pool.Add(core.Black, 5)
		pool.Add(core.Red, 5)
		pool.Add(core.Green, 5)
		pool.Add(core.Colorless, 10)
		perm, abilityIdx, err := tg.findActivatableAbilityByName(respPlayerID, r.perm, targets)
		if err != nil {
			tg.t.Logf("ActivateInResponseTo %s failed: %v", r.perm, err)
			return mage.PriorityAction{Type: mage.PriorityPass}
		}
		return mage.PriorityAction{
			Type:        mage.PriorityActivateAbility,
			PermanentID: perm.ID(),
			AbilityIdx:  abilityIdx,
			Targets:     targets,
			XValue:      r.xValue,
		}
	}

	tg.ensureManaForResponse(r)

	// Locate real card in responder's hand.
	p := tg.Game.GetPlayer(respPlayerID)
	var cardID uuid.UUID
	if p != nil {
		for _, c := range p.Hand() {
			if c.Name() == r.spell {
				cardID = c.ID()
				break
			}
		}
	}
	if cardID == uuid.Nil {
		tg.t.Logf("CastInResponseTo %s: card not in hand", r.spell)
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
	return mage.PriorityAction{
		Type:    mage.PriorityCastSpell,
		CardID:  cardID,
		Targets: targets,
		XValue:  r.xValue,
	}
}

func (tg *TestGame) ensureManaForActivate(aa activateAction) {
	player := tg.GetPlayer(aa.player)
	pool := player.ManaPool()
	pool.Add(core.White, 5)
	pool.Add(core.Blue, 5)
	pool.Add(core.Black, 5)
	pool.Add(core.Red, 5)
	pool.Add(core.Green, 5)
	pool.Add(core.Colorless, 10)
}

// findActivatableAbilityByName finds the first activatable ability on a named permanent.
// Prefers non-mana activated abilities, falls back to mana abilities.
func (tg *TestGame) findActivatableAbilityByName(playerID uuid.UUID, permName string, targets []uuid.UUID) (*mage.Permanent, int, error) {
	perm := tg.FindPermanentByName(permName, playerID)
	if perm == nil {
		for _, p := range tg.AllBattlefield() {
			if p.Name() == permName {
				perm = p
				break
			}
		}
	}
	if perm == nil {
		return nil, -1, fmt.Errorf("permanent %s not found", permName)
	}

	// First pass: activated abilities (non-mana)
	for i, a := range perm.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		if _, isMana := inner.(*mage.ManaAbility); isMana {
			continue
		}
		aa, ok := inner.(mage.ActivatedAbility)
		if !ok {
			continue
		}
		if !aa.CanActivate(playerID, tg.Game) {
			continue
		}
		if saa, isSAA := inner.(*mage.SimpleActivatedAbility); isSAA && saa.OpponentOnlyMayUse {
			if perm.Controller == playerID {
				continue
			}
		}

		if len(aa.Targets()) > 0 && len(targets) > 0 {
			validTargets := true
			for j, t := range aa.Targets() {
				if j >= len(targets) {
					break
				}
				if tf, ok := t.(interface{ Filter() mage.PermanentFilter }); ok {
					targetPerm := tg.FindPermanent(targets[j])
					if targetPerm != nil {
						if !tf.Filter().Match(targetPerm, tg.Game) || !targetPerm.CanBeTargetedBy(perm.Card, playerID, tg.Game) {
							validTargets = false
							break
						}
					}
				} else {
					possible := t.Possible(playerID, perm.Card, tg.Game)
					found := false
					for _, pid := range possible {
						if pid == targets[j] {
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

		return perm, i, nil
	}

	// Second pass: mana abilities
	for i, a := range perm.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		if _, ok := inner.(*mage.ManaAbility); ok {
			if !perm.Tapped && perm.CanTapForEffect(tg.Game) {
				return perm, i, nil
			}
		}
	}

	return nil, -1, fmt.Errorf("no activatable ability found on %s", permName)
}

func (tg *TestGame) executeCounterActions(turn int, step core.PhaseStep) {
	for _, ca := range tg.counterActions {
		if ca.turn != turn || ca.step != step {
			continue
		}
		playerID := tg.getPlayerID(ca.player)
		perm := tg.FindPermanentByName(ca.card, playerID)
		if perm != nil {
			perm.AddCounter(ca.ct, ca.n)
		}
	}
}

func (tg *TestGame) resolveTargets(names []string, controllerID uuid.UUID) []uuid.UUID {
	var targets []uuid.UUID
	for _, name := range names {
		found := false
		for _, p := range tg.AllBattlefield() {
			if p.PhasedOut {
				continue
			}
			if p.Name() == name {
				targets = append(targets, p.ID())
				found = true
				break
			}
		}
		if found {
			continue
		}
		for _, p := range tg.AllPlayers() {
			if p.Name() == name {
				targets = append(targets, p.PlayerID())
				found = true
				break
			}
		}
		if found {
			continue
		}
		for _, pl := range tg.AllPlayers() {
			if pl.PlayerID() == controllerID {
				for _, c := range pl.Graveyard() {
					if c.Name() == name {
						targets = append(targets, c.ID())
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}
		if found {
			continue
		}
		for _, obj := range tg.StackObjects() {
			if obj.IsAbility || obj.Card == nil {
				continue
			}
			if obj.Card.Name() == name {
				targets = append(targets, obj.SourceID)
				found = true
				break
			}
		}
		if found {
			continue
		}
		for _, pl := range tg.AllPlayers() {
			for _, c := range pl.Hand() {
				if c.Name() == name {
					targets = append(targets, c.ID())
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	return targets
}

func (tg *TestGame) validateTargets(sourceCard mage.Card, targets []uuid.UUID, controllerID uuid.UUID) []uuid.UUID {
	var valid []uuid.UUID
	for _, tid := range targets {
		perm := tg.FindPermanent(tid)
		if perm != nil {
			if perm.CanBeTargetedBy(sourceCard, controllerID, tg.Game) {
				valid = append(valid, tid)
			}
			continue
		}
		valid = append(valid, tid)
	}
	return valid
}

// PlayToEnd runs the game until IsGameOver() returns true or maxTurns is reached.
// If no maxTurns argument is given, defaults to 50.
func (tg *TestGame) PlayToEnd(maxTurns ...int) {
	tg.t.Helper()

	limit := 50
	if len(maxTurns) > 0 {
		limit = maxTurns[0]
	}

	tg.padLibraries()
	tg.autoAddMana()
	tg.OnPriority = tg.scriptedPriorityHandler()
	tg.syncToGame()

	for tg.Turn <= limit {
		for _, step := range core.AllSteps() {
			tg.Step = step
			tg.executeCounterActions(tg.Turn, step)

			tg.RunStepWithPriority(step)

			if tg.IsGameOver() {
				return
			}
		}
		tg.advanceToNextTurn()
	}
	tg.t.Fatalf("PlayToEnd: game did not end within %d turns", limit)
}

// Assertions

// AssertWinner checks that the winner matches the expected player.
func (tg *TestGame) AssertWinner(p PlayerRef) {
	tg.t.Helper()
	want := tg.GetPlayer(p).Name()
	got := tg.Winner()
	if got != want {
		tg.t.Errorf("AssertWinner: got %q, want %q", got, want)
	}
}

// AssertGameOver checks whether the game has ended.
func (tg *TestGame) AssertGameOver(want bool) {
	tg.t.Helper()
	got := tg.IsGameOver()
	if got != want {
		tg.t.Errorf("AssertGameOver: got %v, want %v", got, want)
	}
}

// AssertTotalTurns checks the current turn number.
func (tg *TestGame) AssertTotalTurns(want int) {
	tg.t.Helper()
	got := tg.CurrentTurn()
	if got != want {
		tg.t.Errorf("AssertTotalTurns: got %d, want %d", got, want)
	}
}

// AssertLife checks that a player's life total matches the expected value.
func (tg *TestGame) AssertLife(p PlayerRef, want int) {
	tg.t.Helper()
	got := tg.GetPlayer(p).Life()
	if got != want {
		tg.t.Errorf("AssertLife(%v): got %d, want %d", p, got, want)
	}
}

// AssertPoisonCounters checks that a player has the expected number of poison counters.
func (tg *TestGame) AssertPoisonCounters(p PlayerRef, want int) {
	tg.t.Helper()
	got := tg.GetPlayer(p).PoisonCounters()
	if got != want {
		tg.t.Errorf("AssertPoisonCounters(%v): got %d, want %d", p, got, want)
	}
}

// AssertPermanentCount checks the number of permanents with a given name.
func (tg *TestGame) AssertPermanentCount(p PlayerRef, name string, want int) {
	tg.t.Helper()
	playerID := tg.getPlayerID(p)
	got := 0
	for _, perm := range tg.AllBattlefield() {
		if perm.PhasedOut {
			continue
		}
		if perm.Name() == name && perm.Controller == playerID {
			got++
		}
	}
	if got != want {
		tg.t.Errorf("AssertPermanentCount(%v, %s): got %d, want %d", p, name, got, want)
	}
}

// AssertGraveyardCount checks the number of cards with a given name in graveyard.
func (tg *TestGame) AssertGraveyardCount(p PlayerRef, name string, want int) {
	tg.t.Helper()
	player := tg.GetPlayer(p)
	got := 0
	for _, c := range player.Graveyard() {
		if c.Name() == name {
			got++
		}
	}
	if got != want {
		tg.t.Errorf("AssertGraveyardCount(%v, %s): got %d, want %d", p, name, got, want)
	}
}

// AssertHandCount checks the number of cards with a given name in hand.
func (tg *TestGame) AssertHandCount(p PlayerRef, name string, want int) {
	tg.t.Helper()
	player := tg.GetPlayer(p)
	got := 0
	for _, c := range player.Hand() {
		if c.Name() == name {
			got++
		}
	}
	if got != want {
		tg.t.Errorf("AssertHandCount(%v, %s): got %d, want %d", p, name, got, want)
	}
}

// AssertPowerToughness checks a permanent's current power and toughness.
func (tg *TestGame) AssertPowerToughness(p PlayerRef, name string, power, tough int) {
	tg.t.Helper()
	playerID := tg.getPlayerID(p)
	perm := tg.FindPermanentByName(name, playerID)
	if perm == nil {
		tg.t.Errorf("AssertPowerToughness(%v, %s): permanent not found", p, name)
		return
	}
	gotP := perm.CurrentPower(tg.Game)
	gotT := perm.CurrentToughness(tg.Game)
	if gotP != power || gotT != tough {
		tg.t.Errorf("AssertPowerToughness(%v, %s): got %d/%d, want %d/%d", p, name, gotP, gotT, power, tough)
	}
}

// AssertCounterCount checks the number of counters of a type on a permanent.
func (tg *TestGame) AssertCounterCount(p PlayerRef, name string, ct core.CounterType, want int) {
	tg.t.Helper()
	playerID := tg.getPlayerID(p)
	perm := tg.FindPermanentByName(name, playerID)
	if perm == nil {
		tg.t.Errorf("AssertCounterCount(%v, %s): permanent not found", p, name)
		return
	}
	got := int(perm.Counters[ct])
	if got != want {
		tg.t.Errorf("AssertCounterCount(%v, %s, %s): got %d, want %d", p, name, ct, got, want)
	}
}

// AssertTapped checks whether a permanent is tapped.
func (tg *TestGame) AssertTapped(p PlayerRef, name string, tapped bool) {
	tg.t.Helper()
	playerID := tg.getPlayerID(p)
	perm := tg.FindPermanentByName(name, playerID)
	if perm == nil {
		tg.t.Errorf("AssertTapped(%v, %s): permanent not found", p, name)
		return
	}
	if perm.Tapped != tapped {
		tg.t.Errorf("AssertTapped(%v, %s): got %v, want %v", p, name, perm.Tapped, tapped)
	}
}

// AssertHasAbility checks if a permanent has a keyword ability.
func (tg *TestGame) AssertHasAbility(p PlayerRef, name string, kw core.Keyword, has bool) {
	tg.t.Helper()
	playerID := tg.getPlayerID(p)
	perm := tg.FindPermanentByName(name, playerID)
	if perm == nil {
		tg.t.Errorf("AssertHasAbility(%v, %s): permanent not found", p, name)
		return
	}
	got := perm.HasKeyword(kw)
	if got != has {
		tg.t.Errorf("AssertHasAbility(%v, %s, %s): got %v, want %v", p, name, kw, got, has)
	}
}

// AssertAttachedTo checks if an attachment is attached to a host.
func (tg *TestGame) AssertAttachedTo(p PlayerRef, attachment, host string) {
	tg.t.Helper()
	playerID := tg.getPlayerID(p)
	att := tg.FindPermanentByName(attachment, playerID)
	if att == nil {
		for _, perm := range tg.AllBattlefield() {
			if perm.Name() == attachment {
				att = perm
				break
			}
		}
	}
	if att == nil {
		tg.t.Errorf("AssertAttachedTo(%v, %s, %s): attachment not found", p, attachment, host)
		return
	}
	hostPerm := tg.FindPermanentByName(host, playerID)
	if hostPerm == nil {
		tg.t.Errorf("AssertAttachedTo(%v, %s, %s): host not found", p, attachment, host)
		return
	}
	if att.AttachedTo != hostPerm.ID() {
		tg.t.Errorf("AssertAttachedTo(%v, %s, %s): not attached", p, attachment, host)
	}
}

// AssertExileCount checks the number of cards with a given name in exile.
func (tg *TestGame) AssertExileCount(name string, want int) {
	tg.t.Helper()
	got := 0
	for _, ec := range tg.GetExile() {
		if ec.Card.Name() == name {
			got++
		}
	}
	if got != want {
		tg.t.Errorf("AssertExileCount(%s): got %d, want %d", name, got, want)
	}
}

// AssertLibraryCount checks the number of cards with a given name in a player's library.
func (tg *TestGame) AssertLibraryCount(p PlayerRef, name string, want int) {
	tg.t.Helper()
	player := tg.GetPlayer(p)
	got := 0
	for _, c := range player.Library() {
		if c.Name() == name {
			got++
		}
	}
	if got != want {
		tg.t.Errorf("AssertLibraryCount(%v, %s): got %d, want %d", p, name, got, want)
	}
}

// AssertLibraryTop checks that the top N cards of a player's library match
// the expected names in order (index 0 = top of library).
func (tg *TestGame) AssertLibraryTop(p PlayerRef, names ...string) {
	tg.t.Helper()
	player := tg.GetPlayer(p)
	lib := player.Library()
	if len(lib) < len(names) {
		tg.t.Errorf("AssertLibraryTop(%v): library has %d cards, want at least %d", p, len(lib), len(names))
		return
	}
	for i, want := range names {
		got := lib[i].Name()
		if got != want {
			tg.t.Errorf("AssertLibraryTop(%v): position %d: got %q, want %q", p, i, got, want)
		}
	}
}

// AssertGraveyardOrder checks that a player's graveyard contains exactly the
// named cards in the given order (index 0 = bottom of graveyard, last = top).
func (tg *TestGame) AssertGraveyardOrder(p PlayerRef, names ...string) {
	tg.t.Helper()
	player := tg.GetPlayer(p)
	gy := player.Graveyard()
	if len(gy) != len(names) {
		actual := make([]string, len(gy))
		for i, c := range gy {
			actual[i] = c.Name()
		}
		tg.t.Errorf("AssertGraveyardOrder(%v): got %d cards %v, want %d cards %v", p, len(gy), actual, len(names), names)
		return
	}
	for i, want := range names {
		got := gy[i].Name()
		if got != want {
			tg.t.Errorf("AssertGraveyardOrder(%v): position %d: got %q, want %q", p, i, got, want)
		}
	}
}

// AssertHandSize checks the total number of cards in a player's hand.
func (tg *TestGame) AssertHandSize(p PlayerRef, want int) {
	tg.t.Helper()
	got := len(tg.GetPlayer(p).Hand())
	if got != want {
		tg.t.Errorf("AssertHandSize(%v): got %d, want %d", p, got, want)
	}
}

// AssertManaProduced checks how much mana of the given color has been
// added to the player's pool during the current turn. Unlike Count on the
// live pool, this survives per-step pool emptying (CR 500.5).
func (tg *TestGame) AssertManaProduced(p PlayerRef, color core.Color, want int) {
	tg.t.Helper()
	got := tg.GetPlayer(p).ManaPool().CountProducedThisTurn(color)
	if got != want {
		tg.t.Errorf("AssertManaProduced(%v, %v): got %d, want %d", p, color, got, want)
	}
}

// AssertManaProducedAtLeast is the >= variant of AssertManaProduced.
func (tg *TestGame) AssertManaProducedAtLeast(p PlayerRef, color core.Color, min int) {
	tg.t.Helper()
	got := tg.GetPlayer(p).ManaPool().CountProducedThisTurn(color)
	if got < min {
		tg.t.Errorf("AssertManaProducedAtLeast(%v, %v): got %d, want >= %d", p, color, got, min)
	}
}

// AssertHasColor checks whether a permanent has a given color.
func (tg *TestGame) AssertHasColor(p PlayerRef, name string, color core.Color, has bool) {
	tg.t.Helper()
	playerID := tg.getPlayerID(p)
	perm := tg.FindPermanentByName(name, playerID)
	if perm == nil {
		tg.t.Errorf("AssertHasColor(%v, %s): permanent not found", p, name)
		return
	}
	got := false
	for _, c := range perm.Colors() {
		if c == color {
			got = true
			break
		}
	}
	if got != has {
		tg.t.Errorf("AssertHasColor(%v, %s, %v): got %v, want %v", p, name, color, got, has)
	}
}

// AssertAnteCount checks that a player's ante zone contains the expected number
// of cards with the given name.
func (tg *TestGame) AssertAnteCount(p PlayerRef, name string, want int) {
	tg.t.Helper()
	player := tg.GetPlayer(p)
	got := 0
	for _, c := range player.Ante() {
		if c.Name() == name {
			got++
		}
	}
	if got != want {
		tg.t.Errorf("AssertAnteCount(%v, %s): got %d, want %d", p, name, got, want)
	}
}
