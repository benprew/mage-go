package main

import (
	"fmt"
	"sort"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// mirrorMsg carries a priority action from the main goroutine to the game
// goroutine's priority handler.
type mirrorMsg struct {
	playerIdx int
	action    actionInfo
	step      string // canonical step name from XMage (e.g. "precombat_main")
}

// stepInfo is sent over stepCh to tell the game goroutine to enter a step.
// XMage drives all step transitions; Go just runs whatever XMage says.
type stepInfo struct {
	turn            int
	step            core.PhaseStep
	activePlayerIdx int
}

// mirrorResult is sent back from the game goroutine after processing a message.
type mirrorResult struct {
	state *cvState
	err   error
}

// mirrorGame manages a Go game instance that mirrors XMage's game.
type mirrorGame struct {
	game     *mage.Game
	players  [2]*crossValPlayer
	stepCh   chan stepInfo     // step transitions from main loop (XMage-driven)
	msgCh    chan mirrorMsg    // priority actions from main loop
	resultCh chan mirrorResult // state sent back after each priority action
	doneCh   chan struct{}     // closed when the game goroutine exits
	gameErr  error
	debug    bool
}

// snapshotState returns a short string describing the Go game's current
// turn/step. Safe to call from any goroutine — only reads.
func (mg *mirrorGame) snapshotState() string {
	if mg == nil || mg.game == nil {
		return "<nil>"
	}
	return fmt.Sprintf("T%d/%s/active=%d",
		mg.game.CurrentTurn(),
		canonicalStep(mg.game.GetStep()),
		mg.game.ActivePlayerIndex())
}

// crossValPlayer is a Player that receives combat decisions via channels,
// allowing the main goroutine to feed XMage's choices into the Go game.
type crossValPlayer struct {
	*mage.BasePlayer
	mg       *mirrorGame
	attackCh chan []string
	blockCh  chan []blockerPair
}

func newCrossValPlayer(name string) *crossValPlayer {
	return &crossValPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		attackCh:   make(chan []string, 1),
		blockCh:    make(chan []blockerPair, 1),
	}
}

// DeclareAttackers blocks until the main loop feeds attacker names from XMage.
// XMage always runs DECLARE_ATTACKERS (it isn't a skippable step), so we
// simply wait for attackCh.
func (p *crossValPlayer) DeclareAttackers(g *mage.Game) []uuid.UUID {
	dbg(p.mg.debug, "game: DeclareAttackers blocking on attackCh (%s, p=%s)",
		canonicalStep(g.GetStep()), p.Name())
	names, ok := <-p.attackCh
	if !ok {
		return nil
	}
	dbg(p.mg.debug, "game: DeclareAttackers got %d names", len(names))
	return p.resolveAttackerNames(names, g)
}

func (p *crossValPlayer) resolveAttackerNames(names []string, g *mage.Game) []uuid.UUID {
	var ids []uuid.UUID
	for _, name := range names {
		perm := g.FindPermanentByName(name, p.PlayerID())
		if perm != nil {
			ids = append(ids, perm.ID())
		}
	}
	return ids
}

// DeclareBlockers blocks until the main loop feeds blocker pairs from XMage.
// XMage skips DECLARE_BLOCKERS entirely when noAttackers, so we never reach
// here in that case — XMage simply emits no step_begin for it.
func (p *crossValPlayer) DeclareBlockers(g *mage.Game) []mage.BlockAssignment {
	dbg(p.mg.debug, "game: DeclareBlockers blocking on blockCh (%s, p=%s)",
		canonicalStep(g.GetStep()), p.Name())
	pairs, ok := <-p.blockCh
	if !ok {
		return nil
	}
	dbg(p.mg.debug, "game: DeclareBlockers got %d pairs", len(pairs))
	return p.resolveBlockerPairs(pairs, g)
}

func (p *crossValPlayer) resolveBlockerPairs(pairs []blockerPair, g *mage.Game) []mage.BlockAssignment {
	var result []mage.BlockAssignment
	for _, bp := range pairs {
		blocker := g.FindPermanentByName(bp.Blocker, p.PlayerID())
		var attacker *mage.Permanent
		for _, perm := range g.AllBattlefield() {
			if perm.Name() == bp.Attacker && g.IsAttackingInCombat(perm.ID()) {
				attacker = perm
				break
			}
		}
		if blocker != nil && attacker != nil {
			result = append(result, mage.BlockAssignment{
				BlockerID:  blocker.ID(),
				AttackerID: attacker.ID(),
			})
		}
	}
	return result
}

func (p *crossValPlayer) ChooseMayAbility(description string) bool {
	return true
}

func newMirrorGame(deckA, deckB []string) (*mirrorGame, error) {
	playerA := newCrossValPlayer("Alice")
	playerB := newCrossValPlayer("Bob")

	libA, err := buildLibrary(deckA)
	if err != nil {
		return nil, fmt.Errorf("build library A: %w", err)
	}
	libB, err := buildLibrary(deckB)
	if err != nil {
		return nil, fmt.Errorf("build library B: %w", err)
	}
	playerA.SetLibrary(libA)
	playerB.SetLibrary(libB)

	game := mage.NewGame(playerA, playerB)

	mg := &mirrorGame{
		game:     game,
		players:  [2]*crossValPlayer{playerA, playerB},
		stepCh:   make(chan stepInfo, 1),
		msgCh:    make(chan mirrorMsg),
		resultCh: make(chan mirrorResult),
		doneCh:   make(chan struct{}),
	}
	playerA.mg = mg
	playerB.mg = mg
	return mg, nil
}

// start runs the Go game loop in a goroutine. The game is purely reactive:
// it advances only when XMage emits a step_begin (delivered via stepCh) and
// makes priority decisions only when XMage emits a priority action (via
// msgCh). XMage decides when to skip a step (e.g. declare_blockers when no
// attackers); Go simply doesn't see those steps.
func (mg *mirrorGame) start() {
	mg.game.SetOnPriority(func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		dbg(mg.debug, "game: OnPriority blocking on msgCh (%s p%d main=%v)",
			canonicalStep(g.GetStep()), playerIdx, mainPhase)
		msg, ok := <-mg.msgCh
		if !ok {
			dbg(mg.debug, "game: OnPriority msgCh closed; passing")
			return mage.PriorityAction{Type: mage.PriorityPass}
		}
		dbg(mg.debug, "game: OnPriority got msg (kind=%s xmage-step=%s p%d)",
			msg.action.Kind, msg.step, msg.playerIdx)

		// Divergence detection: if the player XMage thinks has priority is
		// not the player Go is asking about, the engines have desynced (the
		// classic cause is a silent cast_spell/play_land/activate_ability
		// failure on the Go side that didn't break runPriorityRound's inner
		// loop, so Go advanced to the next player while XMage is still
		// giving the active player priority again after their action).
		if msg.playerIdx != playerIdx {
			err := fmt.Errorf(
				"priority-player divergence: xmage gave priority to p%d but go expected p%d (step=%s xmage-step=%s action=%s card=%s); typically caused by a prior silent action failure in mage-go",
				msg.playerIdx, playerIdx, canonicalStep(g.GetStep()), msg.step, msg.action.Kind, msg.action.CardName)
			mg.resultCh <- mirrorResult{state: extractGoState(g), err: err}
			return mage.PriorityAction{Type: mage.PriorityPass}
		}

		pa := mg.translateAction(msg, g)
		if pa.Type == mage.PriorityPass && msg.action.Kind != "pass" {
			err := translateFailureError(msg, playerIdx, g)
			mg.resultCh <- mirrorResult{state: extractGoState(g), err: err}
			return mage.PriorityAction{Type: mage.PriorityPass}
		}

		dbg(mg.debug, "game: OnPriority -> resultCh (action=%v)", pa.Type)
		mg.resultCh <- mirrorResult{state: extractGoState(g)}

		return pa
	})

	go func() {
		defer close(mg.doneCh)
		defer func() {
			if r := recover(); r != nil {
				mg.gameErr = fmt.Errorf("game panic: %v", r)
			}
		}()
		mg.runGameLoop()
	}()
}

func (mg *mirrorGame) runGameLoop() {
	g := mg.game

	// Draw opening hands (7 cards each). XMage does the same in its game
	// startup before any step_begin fires.
	for _, p := range g.AllPlayers() {
		for range 7 {
			p.DrawCard()
		}
	}

	for {
		s, ok := <-mg.stepCh
		if !ok {
			return
		}
		g.SetTurn(s.turn)
		g.SetActivePlayerIndex(s.activePlayerIdx)
		dbg(mg.debug, "game: --> step %s (T%d active=%d)",
			canonicalStep(s.step), s.turn, s.activePlayerIdx)
		g.RunStepWithPriority(s.step)
		dbg(mg.debug, "game: <-- step %s done", canonicalStep(s.step))
		if g.IsGameOver() {
			return
		}
	}
}

func (mg *mirrorGame) stop() {
	close(mg.stepCh)
	close(mg.msgCh)
}

// translateFailureError returns a specific error explaining why translateAction
// could not produce a real PriorityAction for the given xmage action — i.e. the
// card or permanent xmage referenced doesn't exist in the Go game's
// hand/battlefield. Distinguishing these cases makes divergence reports
// actionable rather than generic.
func translateFailureError(msg mirrorMsg, playerIdx int, g *mage.Game) error {
	step := canonicalStep(g.GetStep())
	switch msg.action.Kind {
	case "cast_spell":
		return fmt.Errorf("cannot mirror cast_spell %q: card not in p%d's hand (step=%s xmage-step=%s); xmage cast it but go's hand has no card with that name",
			msg.action.CardName, playerIdx, step, msg.step)
	case "play_land":
		return fmt.Errorf("cannot mirror play_land %q: card not in p%d's hand or is not a land (step=%s xmage-step=%s)",
			msg.action.CardName, playerIdx, step, msg.step)
	case "activate_ability":
		return fmt.Errorf("cannot mirror activate_ability on %q: permanent not on battlefield under p%d's control (step=%s xmage-step=%s ability_idx=%d)",
			msg.action.PermanentName, playerIdx, step, msg.step, msg.action.AbilityIndex)
	default:
		return fmt.Errorf("cannot mirror %s on %q (step=%s xmage-step=%s p%d): unknown action kind",
			msg.action.Kind, msg.action.CardName, step, msg.step, playerIdx)
	}
}

func (mg *mirrorGame) translateAction(msg mirrorMsg, g *mage.Game) mage.PriorityAction {
	playerID := mg.players[msg.playerIdx].PlayerID()
	a := msg.action

	switch a.Kind {
	case "play_land":
		for _, c := range g.PlayerAt(msg.playerIdx).Hand() {
			if c.Name() == a.CardName && c.HasType(core.TypeLand) {
				return mage.PriorityAction{
					Type:   mage.PriorityPlayLand,
					CardID: c.ID(),
				}
			}
		}

	case "cast_spell":
		for _, c := range g.PlayerAt(msg.playerIdx).Hand() {
			if c.Name() == a.CardName {
				targets := resolveTargetNames(a.Targets, g, playerID)
				return mage.PriorityAction{
					Type:    mage.PriorityCastSpell,
					CardID:  c.ID(),
					Targets: targets,
				}
			}
		}

	case "activate_ability":
		perm := g.FindPermanentByName(a.PermanentName, playerID)
		if perm != nil {
			targets := resolveTargetNames(a.Targets, g, playerID)
			return mage.PriorityAction{
				Type:        mage.PriorityActivateAbility,
				PermanentID: perm.ID(),
				AbilityIdx:  a.AbilityIndex,
				Targets:     targets,
			}
		}
	}

	return mage.PriorityAction{Type: mage.PriorityPass}
}

func resolveTargetNames(names []string, g *mage.Game, controllerID uuid.UUID) []uuid.UUID {
	if len(names) == 0 {
		return nil
	}
	var ids []uuid.UUID
	for _, name := range names {
		for _, p := range g.AllPlayers() {
			if p.Name() == name {
				ids = append(ids, p.PlayerID())
				goto next
			}
		}
		for _, perm := range g.AllBattlefield() {
			if perm.Name() == name {
				ids = append(ids, perm.ID())
				goto next
			}
		}
	next:
	}
	return ids
}

func extractGoState(g *mage.Game) *cvState {
	type indexedPlayer struct {
		player mage.Player
		idx    int
	}
	var sorted []indexedPlayer
	for i, p := range g.AllPlayers() {
		sorted = append(sorted, indexedPlayer{p, i})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].player.Name() < sorted[j].player.Name()
	})

	activeIdx := 0
	for i, sp := range sorted {
		if sp.idx == g.ActivePlayerIndex() {
			activeIdx = i
			break
		}
	}

	var players [2]cvPlayer
	for i, sp := range sorted {
		players[i] = buildCvPlayer(sp.player, g)
	}

	var stack []string
	for _, so := range g.StackObjects() {
		if so.Card != nil {
			stack = append(stack, so.Card.Name())
		} else {
			stack = append(stack, "Ability")
		}
	}

	return &cvState{
		Turn:            g.CurrentTurn(),
		Step:            canonicalStep(g.GetStep()),
		ActivePlayerIdx: activeIdx,
		Players:         players,
		Stack:           stack,
	}
}

func buildCvPlayer(p mage.Player, g *mage.Game) cvPlayer {
	hand := make([]string, 0, len(p.Hand()))
	for _, c := range p.Hand() {
		hand = append(hand, c.Name())
	}
	sort.Strings(hand)

	var battlefield []cvPermanent
	for _, perm := range g.AllBattlefield() {
		if perm.Controller != p.PlayerID() {
			continue
		}
		if perm.PhasedOut {
			continue
		}
		battlefield = append(battlefield, cvPermanent{
			Name:       perm.Name(),
			Tapped:     perm.Tapped,
			Power:      perm.CurrentPower(g),
			Toughness:  perm.CurrentToughness(g),
			SummonSick: perm.HasAttr(core.AttrSummonSick),
		})
	}
	sort.Slice(battlefield, func(i, j int) bool {
		return battlefield[i].Name < battlefield[j].Name
	})

	gy := make([]string, 0, len(p.Graveyard()))
	for _, c := range p.Graveyard() {
		gy = append(gy, c.Name())
	}
	sort.Strings(gy)

	return cvPlayer{
		Name:        p.Name(),
		Life:        p.Life(),
		Hand:        hand,
		Battlefield: battlefield,
		Graveyard:   gy,
		LibrarySize: len(p.Library()),
	}
}

func buildLibrary(cardNames []string) ([]mage.Card, error) {
	var cards []mage.Card
	for _, name := range cardNames {
		card, err := mage.CreateCard(name)
		if err != nil {
			return nil, fmt.Errorf("create card %q: %w", name, err)
		}
		cards = append(cards, card)
	}
	return cards, nil
}
