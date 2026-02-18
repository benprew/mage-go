package mage

import (
	"testing"

	"github.com/google/uuid"
)

// TestGame provides a DSL for scripting and asserting game states.
type TestGame struct {
	*Game
	t       *testing.T
	playerA *TestPlayer
	playerB *TestPlayer
	stopAt  struct {
		turn int
		step PhaseStep
	}

	// Scripted cast/activate actions
	castActions     []castAction
	activateActions []activateAction
	counterActions  []counterAction
	actionSeq       int // global insertion order counter
}

type castAction struct {
	seq       int // global insertion order
	turn      int
	step      PhaseStep
	player    PlayerRef
	spell     string
	targets   []string
	xValue    int // X value for X-cost spells
	responses []responseAction // spells/abilities cast in response before resolution
}

type responseAction struct {
	player   PlayerRef
	spell    string   // for spell responses
	perm     string   // for activated ability responses
	targets  []string // explicit targets (empty = auto-target spell on stack)
	xValue   int
}

type activateAction struct {
	seq      int // global insertion order
	turn     int
	step     PhaseStep
	player   PlayerRef
	permName string
	targets  []string
}

type counterAction struct {
	turn   int
	step   PhaseStep
	player PlayerRef
	card   string
	ct     CounterType
	n      int
}

// NewTestGame creates a new test game with two test players.
func NewTestGame(t *testing.T) *TestGame {
	t.Helper()
	pA := NewTestPlayer("PlayerA")
	pB := NewTestPlayer("PlayerB")
	g := NewGame(pA, pB)
	return &TestGame{
		Game:    g,
		t:       t,
		playerA: pA,
		playerB: pB,
	}
}

func (tg *TestGame) getPlayer(ref PlayerRef) *TestPlayer {
	if ref == PlayerA {
		return tg.playerA
	}
	return tg.playerB
}

func (tg *TestGame) getPlayerID(ref PlayerRef) uuid.UUID {
	return tg.getPlayer(ref).PlayerID()
}

// AddCard adds a card to a zone for a player.
func (tg *TestGame) AddCard(zone Zone, p PlayerRef, name string, count ...int) {
	tg.t.Helper()
	n := 1
	if len(count) > 0 {
		n = count[0]
	}
	player := tg.getPlayer(p)
	playerID := player.PlayerID()

	for i := 0; i < n; i++ {
		card, err := CreateCard(name)
		if err != nil {
			tg.t.Fatalf("AddCard: %v", err)
		}
		card.SetOwner(playerID)

		switch zone {
		case ZoneBattlefield:
			perm := tg.Game.PutOnBattlefield(card, playerID)
			perm.SummonSick = false // test cards are not summoning sick
		case ZoneHand:
			player.AddToHand(card)
		case ZoneGraveyard:
			player.AddToGraveyard(card)
		case ZoneLibrary:
			player.library = append(player.library, card)
		}
	}
}

// SetLife sets a player's life total.
func (tg *TestGame) SetLife(p PlayerRef, life int) {
	tg.getPlayer(p).SetLife(life)
}

// AddCounters schedules adding counters during execution.
func (tg *TestGame) AddCounters(turn int, step PhaseStep, p PlayerRef, card string, ct CounterType, n int) {
	tg.counterActions = append(tg.counterActions, counterAction{
		turn: turn, step: step, player: p, card: card, ct: ct, n: n,
	})
}

// CastSpell scripts a spell cast at a specific turn/step.
func (tg *TestGame) CastSpell(turn int, step PhaseStep, p PlayerRef, spell string, targets ...string) {
	tg.actionSeq++
	tg.castActions = append(tg.castActions, castAction{
		seq: tg.actionSeq, turn: turn, step: step, player: p, spell: spell, targets: targets,
	})
}

// CastSpellWithX scripts an X-cost spell cast at a specific turn/step with a given X value.
func (tg *TestGame) CastSpellWithX(turn int, step PhaseStep, p PlayerRef, spell string, xValue int, targets ...string) {
	tg.actionSeq++
	tg.castActions = append(tg.castActions, castAction{
		seq: tg.actionSeq, turn: turn, step: step, player: p, spell: spell, targets: targets, xValue: xValue,
	})
}

// CastInResponseTo scripts a spell cast in response to the most recently scripted
// cast action. The response spell is cast while the original spell is still on the
// stack, then both resolve LIFO. If no explicit targets are given, the response
// automatically targets the original spell on the stack.
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

// ActivateInResponseTo scripts an activated ability in response to the most recently
// scripted cast action. If no explicit targets are given, it auto-targets the spell on stack.
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
func (tg *TestGame) ActivateAbility(turn int, step PhaseStep, p PlayerRef, permName string, targets ...string) {
	tg.actionSeq++
	tg.activateActions = append(tg.activateActions, activateAction{
		seq: tg.actionSeq, turn: turn, step: step, player: p, permName: permName, targets: targets,
	})
}

// Attack scripts attacks for a turn.
func (tg *TestGame) Attack(turn int, p PlayerRef, creatures ...string) {
	tg.getPlayer(p).SetAttackers(turn, creatures)
}

// Block scripts blocks for a turn. The same blocker can block multiple
// attackers if it has the ability to do so (e.g. Two-Headed Giant).
func (tg *TestGame) Block(turn int, p PlayerRef, blocker, attacker string) {
	tp := tg.getPlayer(p)
	tp.blockActions[turn] = append(tp.blockActions[turn], blockPair{blocker, attacker})
}

// ChoosePermanent scripts which permanent a player will choose when asked to sacrifice/pick.
func (tg *TestGame) ChoosePermanent(p PlayerRef, name string) {
	tp := tg.getPlayer(p)
	tp.choosePermanent = append(tp.choosePermanent, name)
}

// ChooseDiscard scripts which card(s) a player will discard when asked.
func (tg *TestGame) ChooseDiscard(p PlayerRef, names ...string) {
	tp := tg.getPlayer(p)
	tp.chooseDiscard = append(tp.chooseDiscard, names)
}

// ChooseManaColor scripts what mana color a player will choose when asked.
func (tg *TestGame) ChooseManaColor(p PlayerRef, color Color) {
	tp := tg.getPlayer(p)
	tp.chooseManaColor = append(tp.chooseManaColor, color)
}

// ChooseFromLibrary scripts which card a player will find when searching library.
func (tg *TestGame) ChooseFromLibrary(p PlayerRef, name string) {
	tp := tg.getPlayer(p)
	tp.chooseFromLibrary = append(tp.chooseFromLibrary, name)
}

// StopAt sets when the game should stop.
func (tg *TestGame) StopAt(turn int, step PhaseStep) {
	tg.stopAt.turn = turn
	tg.stopAt.step = step
}

// Execute runs the game with all scripted actions.
func (tg *TestGame) Execute() {
	tg.t.Helper()

	// Add mana to pools for all scripted casts (auto-mana)
	tg.autoAddMana()

	// Hook into the game loop to execute scripted actions
	maxTurns := tg.stopAt.turn + 5
	for tg.Turn <= maxTurns {
		for _, step := range AllSteps() {
			if tg.Turn == tg.stopAt.turn && step == tg.stopAt.step {
				tg.Step = step
				tg.Effects.Apply(tg.Game)
				return
			}

			// Set step before executing actions so sorcery-speed checks work
			tg.Step = step

			// Execute scripted counter additions
			tg.executeCounterActions(tg.Turn, step)

			// Execute scripted casts and activations in the order they
			// were scripted (sequence number preserves insertion order).
			tg.executeOrderedActions(tg.Turn, step)

			// Auto-play lands from hand during PrecombatMain
			if step == PrecombatMain {
				tg.autoPlayLands()
			}

			tg.RunStep(step)
		}
		// Check for extra turns
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

// autoPlayLands plays all land cards from the active player's hand (respecting limits).
func (tg *TestGame) autoPlayLands() {
	active := tg.Game.ActivePlayerObj()
	for {
		if tg.LandsPlayedThisTurn >= tg.MaxLandPlays() {
			break
		}
		var landID uuid.UUID
		for _, c := range active.Hand() {
			if c.HasType(TypeLand) {
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

// ensureManaForCast tops up a player's pool so they can afford a scripted spell.
// This handles spells cast on later turns when the pool may have been cleared.
func (tg *TestGame) ensureManaForCast(ca castAction) {
	player := tg.getPlayer(ca.player)
	card, err := CreateCard(ca.spell)
	if err != nil {
		return
	}
	mc := card.ManaCost()
	pool := player.ManaPool()

	// Ensure enough of each colored mana
	colors := []struct {
		color Color
		need  int
	}{
		{White, mc.White}, {Blue, mc.Blue}, {Black, mc.Black},
		{Red, mc.Red}, {Green, mc.Green},
	}
	for _, c := range colors {
		have := pool.Count(c.color)
		if have < c.need {
			pool.Add(c.color, c.need-have)
		}
	}

	// Ensure enough for generic + X cost
	genericNeeded := mc.Generic
	if mc.HasX {
		genericNeeded += ca.xValue * mc.XCount
	}
	if genericNeeded > 0 {
		pool.Add(Colorless, genericNeeded)
	}
}

// autoAddMana adds enough mana to cast all scripted spells.
func (tg *TestGame) autoAddMana() {
	for _, ca := range tg.castActions {
		player := tg.getPlayer(ca.player)
		card, err := CreateCard(ca.spell)
		if err != nil {
			continue
		}
		mc := card.ManaCost()
		pool := player.ManaPool()
		pool.Add(White, mc.White)
		pool.Add(Blue, mc.Blue)
		pool.Add(Black, mc.Black)
		pool.Add(Red, mc.Red)
		pool.Add(Green, mc.Green)
		pool.Add(Colorless, mc.Generic)
		// Add mana for X costs
		if mc.HasX {
			pool.Add(Colorless, ca.xValue*mc.XCount)
		}

		// Add mana for response spells
		for _, r := range ca.responses {
			if r.spell != "" {
				rCard, err := CreateCard(r.spell)
				if err != nil {
					continue
				}
				rmc := rCard.ManaCost()
				rPlayer := tg.getPlayer(r.player)
				rPool := rPlayer.ManaPool()
				rPool.Add(White, rmc.White)
				rPool.Add(Blue, rmc.Blue)
				rPool.Add(Black, rmc.Black)
				rPool.Add(Red, rmc.Red)
				rPool.Add(Green, rmc.Green)
				rPool.Add(Colorless, rmc.Generic)
				if rmc.HasX {
					rPool.Add(Colorless, r.xValue*rmc.XCount)
				}
			}
			if r.perm != "" {
				// Add mana for activated ability responses
				rPlayer := tg.getPlayer(r.player)
				rPlayer.ManaPool().Add(White, 5)
				rPlayer.ManaPool().Add(Blue, 5)
				rPlayer.ManaPool().Add(Black, 5)
				rPlayer.ManaPool().Add(Red, 5)
				rPlayer.ManaPool().Add(Green, 5)
				rPlayer.ManaPool().Add(Colorless, 10)
			}
		}
	}

	for _, aa := range tg.activateActions {
		player := tg.getPlayer(aa.player)
		// Add colored and colorless mana so activated abilities with colored costs work
		player.ManaPool().Add(White, 5)
		player.ManaPool().Add(Blue, 5)
		player.ManaPool().Add(Black, 5)
		player.ManaPool().Add(Red, 5)
		player.ManaPool().Add(Green, 5)
		player.ManaPool().Add(Colorless, 10)
	}
}

// executeOrderedActions processes cast and activate actions for the given
// turn/step in the order they were scripted (by sequence number).
func (tg *TestGame) executeOrderedActions(turn int, step PhaseStep) {
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

	// Sort by sequence number to preserve scripting order
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

	// Just-in-time mana: ensure the player can afford the spell right now.
	// This handles spells cast on later turns whose mana may have been cleared.
	tg.ensureManaForCast(ca)

	// Validate targets against targeting restrictions
	card, _ := CreateCard(ca.spell)
	if card != nil {
		validTargets := tg.validateTargets(card, targets, playerID)
		if len(validTargets) == 0 && len(targets) > 0 {
			return // All targets illegal, spell fizzles
		}
		targets = validTargets
	}

	err := tg.Game.CastSpellByName(playerID, ca.spell, targets, ca.xValue)
	if err != nil {
		tg.t.Logf("CastSpell %s failed: %v", ca.spell, err)
	}

	// If there are responses, cast them before resolving the stack
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

		// Resolve explicit targets, or auto-target the top spell on the stack
		var targets []uuid.UUID
		if len(r.targets) > 0 {
			targets = tg.resolveTargets(r.targets, respPlayerID)
		} else if tg.Game.Stack.Peek() != nil {
			// Auto-target the top spell on the stack
			targets = []uuid.UUID{tg.Game.Stack.Peek().SourceID}
		}

		if r.perm != "" {
			// Activated ability response
			err := tg.Game.ActivateAbilityByText(respPlayerID, r.perm, targets)
			if err != nil {
				tg.t.Logf("ActivateInResponseTo %s failed: %v", r.perm, err)
			}
		} else {
			// Spell response
			err := tg.Game.CastSpellByName(respPlayerID, r.spell, targets, r.xValue)
			if err != nil {
				tg.t.Logf("CastInResponseTo %s failed: %v", r.spell, err)
			}
		}
	}
}

func (tg *TestGame) executeCounterActions(turn int, step PhaseStep) {
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
		// Check battlefield
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
		// Check players
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
		// Check graveyard
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
		// Check hand
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

func (tg *TestGame) validateTargets(sourceCard Card, targets []uuid.UUID, controllerID uuid.UUID) []uuid.UUID {
	var valid []uuid.UUID
	for _, tid := range targets {
		perm := tg.Game.FindPermanent(tid)
		if perm != nil {
			if perm.CanBeTargetedBy(sourceCard, controllerID, tg.Game) {
				valid = append(valid, tid)
			}
			continue
		}
		// Players are always valid targets
		valid = append(valid, tid)
	}
	return valid
}

// Assertions

// AssertLife checks that a player's life total matches the expected value.
func (tg *TestGame) AssertLife(p PlayerRef, want int) {
	tg.t.Helper()
	got := tg.getPlayer(p).Life()
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
	player := tg.getPlayer(p)
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
	player := tg.getPlayer(p)
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
func (tg *TestGame) AssertCounterCount(p PlayerRef, name string, ct CounterType, want int) {
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
func (tg *TestGame) AssertHasAbility(p PlayerRef, name string, kw Keyword, has bool) {
	tg.t.Helper()
	playerID := tg.getPlayerID(p)
	perm := tg.Game.FindPermanentByName(name, playerID)
	if perm == nil {
		tg.t.Errorf("AssertHasAbility(%v, %s): permanent not found", p, name)
		return
	}
	got := perm.HasAbility(kw)
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
