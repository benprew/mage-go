package gametest

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// TestGame provides a DSL for scripting and asserting game states.
type TestGame struct {
	*mage.Game
	t       *testing.T
	playerA *TestPlayer
	playerB *TestPlayer
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
}

type responseAction struct {
	player  PlayerRef
	spell   string
	perm    string
	targets []string
	xValue  int
}

type activateAction struct {
	seq      int
	turn     int
	step     core.PhaseStep
	player   PlayerRef
	permName string
	targets  []string
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
	return &TestGame{
		Game:    g,
		t:       t,
		playerA: pA,
		playerB: pB,
	}
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
			perm := tg.Game.PutOnBattlefield(card, playerID)
			perm.SummonSick = false
			lastID = perm.ID()
		case core.ZoneHand:
			player.AddToHand(card)
		case core.ZoneGraveyard:
			player.AddToGraveyard(card)
		case core.ZoneLibrary:
			player.AddToLibrary(card)
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

// Attack scripts attacks for a turn.
func (tg *TestGame) Attack(turn int, p PlayerRef, creatures ...string) {
	tg.GetPlayer(p).SetAttackers(turn, creatures)
}

// Block scripts blocks for a turn.
func (tg *TestGame) Block(turn int, p PlayerRef, blocker, attacker string) {
	tp := tg.GetPlayer(p)
	tp.blockActions[turn] = append(tp.blockActions[turn], blockPair{blocker, attacker})
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

// AssertBanded checks whether two named permanents are in the same attacking band.
func (tg *TestGame) AssertBanded(p1 PlayerRef, name1 string, p2 PlayerRef, name2 string, want bool) {
	tg.t.Helper()
	pid1 := tg.getPlayerID(p1)
	pid2 := tg.getPlayerID(p2)
	perm1 := tg.Game.FindPermanentByName(name1, pid1)
	perm2 := tg.Game.FindPermanentByName(name2, pid2)
	if perm1 == nil {
		tg.t.Errorf("AssertBanded: %s not found for %v", name1, p1)
		return
	}
	if perm2 == nil {
		tg.t.Errorf("AssertBanded: %s not found for %v", name2, p2)
		return
	}
	got := tg.Game.Combat.IsBandedWith(perm1.ID(), perm2.ID())
	if got != want {
		tg.t.Errorf("AssertBanded(%s, %s): got %v, want %v", name1, name2, got, want)
	}
}

// StopAt sets when the game should stop.
func (tg *TestGame) StopAt(turn int, step core.PhaseStep) {
	tg.stopAt.turn = turn
	tg.stopAt.step = step
}

// Execute runs the game with all scripted actions.
func (tg *TestGame) Execute() {
	tg.t.Helper()

	tg.autoAddMana()

	maxTurns := tg.stopAt.turn + 5
	for tg.Turn <= maxTurns {
		for _, step := range core.AllSteps() {
			if tg.Turn == tg.stopAt.turn && step == tg.stopAt.step {
				tg.Step = step
				tg.Effects.Apply(tg.Game)
				return
			}

			tg.Step = step
			tg.executeCounterActions(tg.Turn, step)
			tg.executeOrderedActions(tg.Turn, step)

			if step == core.PrecombatMain {
				tg.autoPlayLands()
			}

			tg.RunStep(step)
		}
		if len(tg.ExtraTurns) > 0 {
			extraPlayerID := tg.ExtraTurns[0]
			tg.ExtraTurns = tg.ExtraTurns[1:]
			for i, p := range tg.Players {
				if p.PlayerID() == extraPlayerID {
					tg.ActivePlayer = i
					break
				}
			}
		} else {
			tg.ActivePlayer = (tg.ActivePlayer + 1) % len(tg.Players)
		}
		tg.Turn++
	}
}

func (tg *TestGame) autoPlayLands() {
	active := tg.Game.ActivePlayerObj()
	for {
		if tg.LandsPlayedThisTurn >= tg.MaxLandPlays() {
			break
		}
		var landID uuid.UUID
		for _, c := range active.Hand() {
			if c.HasType(core.TypeLand) {
				landID = c.ID()
				break
			}
		}
		if landID == uuid.Nil {
			break
		}
		err := tg.Game.PlayLand(active.PlayerID(), landID)
		if err != nil {
			break
		}
	}
}

func (tg *TestGame) ensureManaForCast(ca castAction) {
	player := tg.GetPlayer(ca.player)
	card, err := mage.CreateCard(ca.spell)
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
		genericNeeded += ca.xValue * mc.XCount
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

	for _, aa := range tg.activateActions {
		player := tg.GetPlayer(aa.player)
		player.ManaPool().Add(core.White, 5)
		player.ManaPool().Add(core.Blue, 5)
		player.ManaPool().Add(core.Black, 5)
		player.ManaPool().Add(core.Red, 5)
		player.ManaPool().Add(core.Green, 5)
		player.ManaPool().Add(core.Colorless, 10)
	}
}

func (tg *TestGame) executeOrderedActions(turn int, step core.PhaseStep) {
	type ordered struct {
		seq    int
		isCast bool
		idx    int
	}
	var actions []ordered

	for i, ca := range tg.castActions {
		if ca.turn == turn && ca.step == step {
			actions = append(actions, ordered{seq: ca.seq, isCast: true, idx: i})
		}
	}
	for i, aa := range tg.activateActions {
		if aa.turn == turn && aa.step == step {
			actions = append(actions, ordered{seq: aa.seq, isCast: false, idx: i})
		}
	}

	for i := 1; i < len(actions); i++ {
		for j := i; j > 0 && actions[j].seq < actions[j-1].seq; j-- {
			actions[j], actions[j-1] = actions[j-1], actions[j]
		}
	}

	for _, a := range actions {
		if a.isCast {
			tg.executeSingleCast(tg.castActions[a.idx])
		} else {
			tg.executeSingleActivate(tg.activateActions[a.idx])
		}
	}
}

func (tg *TestGame) executeSingleCast(ca castAction) {
	playerID := tg.getPlayerID(ca.player)
	targets := tg.resolveTargets(ca.targets, playerID)

	tg.ensureManaForCast(ca)

	card, _ := mage.CreateCard(ca.spell)
	if card != nil {
		validTargets := tg.validateTargets(card, targets, playerID)
		if len(validTargets) == 0 && len(targets) > 0 {
			return
		}
		targets = validTargets
	}

	err := tg.Game.CastSpellByName(playerID, ca.spell, targets, ca.xValue)
	if err != nil {
		tg.t.Logf("CastSpell %s failed: %v", ca.spell, err)
	}

	if len(ca.responses) > 0 {
		tg.executeResponses(ca.responses)
	}

	tg.Game.ResolveStack()
}

func (tg *TestGame) executeSingleActivate(aa activateAction) {
	playerID := tg.getPlayerID(aa.player)
	targets := tg.resolveTargets(aa.targets, playerID)
	err := tg.Game.ActivateAbilityByText(playerID, aa.permName, targets)
	if err != nil {
		tg.t.Logf("ActivateAbility %s failed: %v", aa.permName, err)
	}
	tg.Game.ResolveStack()
}

func (tg *TestGame) executeResponses(responses []responseAction) {
	for _, r := range responses {
		respPlayerID := tg.getPlayerID(r.player)

		var targets []uuid.UUID
		if len(r.targets) > 0 {
			targets = tg.resolveTargets(r.targets, respPlayerID)
		} else if tg.Game.Stack.Peek() != nil {
			targets = []uuid.UUID{tg.Game.Stack.Peek().SourceID}
		}

		if r.perm != "" {
			err := tg.Game.ActivateAbilityByText(respPlayerID, r.perm, targets)
			if err != nil {
				tg.t.Logf("ActivateInResponseTo %s failed: %v", r.perm, err)
			}
		} else {
			err := tg.Game.CastSpellByName(respPlayerID, r.spell, targets, r.xValue)
			if err != nil {
				tg.t.Logf("CastInResponseTo %s failed: %v", r.spell, err)
			}
		}
	}
}

func (tg *TestGame) executeCounterActions(turn int, step core.PhaseStep) {
	for _, ca := range tg.counterActions {
		if ca.turn != turn || ca.step != step {
			continue
		}
		playerID := tg.getPlayerID(ca.player)
		perm := tg.Game.FindPermanentByName(ca.card, playerID)
		if perm != nil {
			perm.AddCounter(ca.ct, ca.n)
		}
	}
}

func (tg *TestGame) resolveTargets(names []string, controllerID uuid.UUID) []uuid.UUID {
	var targets []uuid.UUID
	for _, name := range names {
		found := false
		for _, p := range tg.Battlefield {
			if p.Name() == name {
				targets = append(targets, p.ID())
				found = true
				break
			}
		}
		if found {
			continue
		}
		for _, p := range tg.Players {
			if p.Name() == name {
				targets = append(targets, p.PlayerID())
				found = true
				break
			}
		}
		if found {
			continue
		}
		for _, pl := range tg.Players {
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
		for _, pl := range tg.Players {
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
		perm := tg.Game.FindPermanent(tid)
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

// Assertions

// AssertLife checks that a player's life total matches the expected value.
func (tg *TestGame) AssertLife(p PlayerRef, want int) {
	tg.t.Helper()
	got := tg.GetPlayer(p).Life()
	if got != want {
		tg.t.Errorf("AssertLife(%v): got %d, want %d", p, got, want)
	}
}

// AssertPermanentCount checks the number of permanents with a given name.
func (tg *TestGame) AssertPermanentCount(p PlayerRef, name string, want int) {
	tg.t.Helper()
	playerID := tg.getPlayerID(p)
	got := 0
	for _, perm := range tg.Battlefield {
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
	perm := tg.Game.FindPermanentByName(name, playerID)
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
	perm := tg.Game.FindPermanentByName(name, playerID)
	if perm == nil {
		tg.t.Errorf("AssertCounterCount(%v, %s): permanent not found", p, name)
		return
	}
	got := perm.Counters[ct]
	if got != want {
		tg.t.Errorf("AssertCounterCount(%v, %s, %s): got %d, want %d", p, name, ct, got, want)
	}
}

// AssertTapped checks whether a permanent is tapped.
func (tg *TestGame) AssertTapped(p PlayerRef, name string, tapped bool) {
	tg.t.Helper()
	playerID := tg.getPlayerID(p)
	perm := tg.Game.FindPermanentByName(name, playerID)
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
	perm := tg.Game.FindPermanentByName(name, playerID)
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
	att := tg.Game.FindPermanentByName(attachment, playerID)
	if att == nil {
		for _, perm := range tg.Battlefield {
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
	hostPerm := tg.Game.FindPermanentByName(host, playerID)
	if hostPerm == nil {
		tg.t.Errorf("AssertAttachedTo(%v, %s, %s): host not found", p, attachment, host)
		return
	}
	if att.AttachedTo != hostPerm.ID() {
		tg.t.Errorf("AssertAttachedTo(%v, %s, %s): not attached", p, attachment, host)
	}
}
