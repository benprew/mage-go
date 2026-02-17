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
}

type castAction struct {
	turn    int
	step    PhaseStep
	player  PlayerRef
	spell   string
	targets []string
	xValue  int // X value for X-cost spells
}

type activateAction struct {
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
			player.Library_ = append(player.Library_, card)
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
	tg.castActions = append(tg.castActions, castAction{
		turn: turn, step: step, player: p, spell: spell, targets: targets,
	})
}

// CastSpellWithX scripts an X-cost spell cast at a specific turn/step with a given X value.
func (tg *TestGame) CastSpellWithX(turn int, step PhaseStep, p PlayerRef, spell string, xValue int, targets ...string) {
	tg.castActions = append(tg.castActions, castAction{
		turn: turn, step: step, player: p, spell: spell, targets: targets, xValue: xValue,
	})
}

// ActivateAbility scripts an ability activation at a specific turn/step.
func (tg *TestGame) ActivateAbility(turn int, step PhaseStep, p PlayerRef, permName string, targets ...string) {
	tg.activateActions = append(tg.activateActions, activateAction{
		turn: turn, step: step, player: p, permName: permName, targets: targets,
	})
}

// Attack scripts attacks for a turn.
func (tg *TestGame) Attack(turn int, p PlayerRef, creatures ...string) {
	tg.getPlayer(p).SetAttackers(turn, creatures)
}

// Block scripts blocks for a turn.
func (tg *TestGame) Block(turn int, p PlayerRef, blocker, attacker string) {
	tp := tg.getPlayer(p)
	if tp.blockActions[turn] == nil {
		tp.blockActions[turn] = make(map[string]string)
	}
	tp.blockActions[turn][blocker] = attacker
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

			// Execute scripted casts before the step runs
			tg.executeCastActions(tg.Turn, step)

			// Execute scripted ability activations
			tg.executeActivateActions(tg.Turn, step)

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

func (tg *TestGame) executeCastActions(turn int, step PhaseStep) {
	for _, ca := range tg.castActions {
		if ca.turn != turn || ca.step != step {
			continue
		}
		playerID := tg.getPlayerID(ca.player)
		targets := tg.resolveTargets(ca.targets, playerID)

		// Validate targets against targeting restrictions
		card, _ := CreateCard(ca.spell)
		if card != nil {
			validTargets := tg.validateTargets(card, targets, playerID)
			if len(validTargets) == 0 && len(targets) > 0 {
				// All targets illegal, spell fizzles
				continue
			}
			targets = validTargets
		}

		err := tg.Game.CastSpellByName(playerID, ca.spell, targets, ca.xValue)
		if err != nil {
			tg.t.Logf("CastSpell %s failed: %v", ca.spell, err)
		}
		tg.Game.ResolveStack()
	}
}

func (tg *TestGame) executeActivateActions(turn int, step PhaseStep) {
	for _, aa := range tg.activateActions {
		if aa.turn != turn || aa.step != step {
			continue
		}
		playerID := tg.getPlayerID(aa.player)
		targets := tg.resolveTargets(aa.targets, playerID)
		err := tg.Game.ActivateAbilityByText(playerID, aa.permName, targets)
		if err != nil {
			tg.t.Logf("ActivateAbility %s failed: %v", aa.permName, err)
		}
		tg.Game.ResolveStack()
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
