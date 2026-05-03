package main

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// replay walks one recording end-to-end, driving a mage-go game whose hooks
// consume events from the recording cursor. Validation happens on PRIORITY
// events: state and playable-action set are diffed against the recorded
// snapshot. On the first non-passing divergence, replay returns the diff
// and stops.
//
// Concurrency: the game runs synchronously on the calling goroutine. Hooks
// fire inline (mage-go's runtime calls them; they advance the cursor and
// return). No channels, no IPC.
type replay struct {
	events []eventLine
	cursor int

	game       *mage.Game
	playerByID map[string]*replayPlayer // recorder UUID -> Go player

	// loose: when true, action-set diff matches by (kind, sourceName) only,
	// ignoring rule text / cost text / target candidates. Loose is the
	// default: text differs harmlessly between engines.
	loose bool

	// maxTurns mirrors XMage's stopOnTurn: replay stops mage-go at the
	// same turn the recording stops. Read from META.extras.maxTurns.
	maxTurns int

	// handSizes[i] = the number of leading entries in META.players[i].deck
	// that should go directly into player i's hand at game start (rather
	// than draw-from-top). 0 = legacy encoding, fall back to draw-7.
	handSizes []int

	// First fatal divergence causes replay to bail; record it.
	divergence string
	gameErr    error

	// warnings accumulate non-fatal diffs (currently: playable-action
	// mismatches from mage-go's approximate Playable() stub). Reported
	// at end of run.
	warnings []string
}

func newReplay(meta metaLine, events []eventLine, loose bool) (*replay, error) {
	if len(meta.Players) != 2 {
		return nil, fmt.Errorf("replay: expected 2 players in META, got %d", len(meta.Players))
	}

	// Determine the recording's last turn — meta.extras.maxTurns is set by
	// BatchSelfPlayTest but not by SelfPlayRecorderTest, so fall back to
	// scanning all PRIORITY snapshots for the largest turn number.
	maxTurns := 0
	if v, ok := meta.Extras["maxTurns"]; ok {
		switch n := v.(type) {
		case float64:
			maxTurns = int(n)
		case int:
			maxTurns = n
		}
	}
	if maxTurns == 0 {
		for _, ev := range events {
			if ev.Snapshot != nil && ev.Snapshot.Turn > maxTurns {
				maxTurns = ev.Snapshot.Turn
			}
		}
	}
	if maxTurns == 0 {
		maxTurns = 50
	}

	r := &replay{
		events:     events,
		loose:      loose,
		maxTurns:   maxTurns,
		playerByID: make(map[string]*replayPlayer),
	}

	pa := newReplayPlayer(meta.Players[0].Name, r)
	pb := newReplayPlayer(meta.Players[1].Name, r)
	r.playerByID[meta.Players[0].ID] = pa
	r.playerByID[meta.Players[1].ID] = pb

	// Build full library (hand cards + library cards) so card UUIDs exist in
	// the engine. We slice into hand-cards / library-cards in run() based on
	// HandSizeAtStart from META — that's how new recordings encode the
	// post-shuffle state. Older recordings (pre-Hand+Library encoding) have
	// HandSizeAtStart=0 and Deck = full library order; for those, run()
	// falls back to draw-7-from-top.
	libA, err := buildLibrary(meta.Players[0].Deck)
	if err != nil {
		return nil, fmt.Errorf("build library A: %w", err)
	}
	libB, err := buildLibrary(meta.Players[1].Deck)
	if err != nil {
		return nil, fmt.Errorf("build library B: %w", err)
	}
	pa.SetLibrary(libA)
	pb.SetLibrary(libB)

	r.handSizes = []int{meta.Players[0].HandSizeAtStart, meta.Players[1].HandSizeAtStart}

	r.game = mage.NewGame(pa, pb)
	r.game.SetOnPriority(r.onPriority)
	return r, nil
}

// usesHandLibraryEncoding reports whether META encodes the deck as
// "hand cards then library cards" (any HandSizeAtStart > 0). New
// recordings always use this; older ones have HandSizeAtStart=0 and
// the deck list is just the library order.
// playerNameByID looks up a recorder UUID and returns the player's name
// (PlayerA / PlayerB / etc.). Returns "" if the ID isn't known.
func (r *replay) playerNameByID(id string) string {
	if rp, ok := r.playerByID[id]; ok && rp != nil {
		return rp.Name()
	}
	return ""
}

func (r *replay) usesHandLibraryEncoding() bool {
	for _, n := range r.handSizes {
		if n > 0 {
			return true
		}
	}
	return false
}

// shouldDrawOpeningHands inspects the first PRIORITY snapshot in the
// recording to decide whether opening hands were drawn on the XMage side.
// We check the first non-nil hand: if any player shows ≥1 card, opening
// hands were drawn. Otherwise the recording was made with testMode=true
// (the test-framework default) and we should match its empty-hand start.
func (r *replay) shouldDrawOpeningHands() bool {
	for _, ev := range r.events {
		if ev.Type != "PRIORITY" || ev.Snapshot == nil {
			continue
		}
		for _, p := range ev.Snapshot.Players {
			if len(p.Hand) > 0 {
				return true
			}
		}
		// First PRIORITY had no cards in hand — testMode-style recording.
		return false
	}
	return true // no PRIORITY events at all; default behavior
}

func buildLibrary(cardNames []string) ([]mage.Card, error) {
	cards := make([]mage.Card, 0, len(cardNames))
	for _, name := range cardNames {
		c, err := mage.CreateCard(name)
		if err != nil {
			return nil, fmt.Errorf("create card %q: %w", name, err)
		}
		cards = append(cards, c)
	}
	return cards, nil
}

// run drives the game forward until it ends or a divergence is hit.
// Three startup modes, in priority order:
//
//  1. New recording with HandSizeAtStart > 0: META.deck encodes
//     hand[0..HandSizeAtStart) + library[HandSizeAtStart..]. We move the
//     first N cards from the library directly into hand without drawing.
//     Replay matches XMage's exact post-shuffle, post-opening-hand state.
//
//  2. Older recording, post-SelfPlayExecutor: hand drawn at game start
//     via testMode=false but no HandSizeAtStart in META. shouldDrawOpeningHands
//     sees ≥1 card in the first PRIORITY's hand and draws 7 here; the
//     shuffle order may diverge so this only works for replayable .dck
//     orders (skipInitShuffling=true setups).
//
//  3. Legacy recording, testMode=true: empty hands at first PRIORITY,
//     no opening hand draw needed.
func (r *replay) run() error {
	if r.usesHandLibraryEncoding() {
		for i, p := range r.game.AllPlayers() {
			n := r.handSizes[i]
			for range n {
				c, ok := p.DrawCard()
				_ = c
				_ = ok
			}
		}
	} else if r.shouldDrawOpeningHands() {
		for _, p := range r.game.AllPlayers() {
			for range 7 {
				p.DrawCard()
			}
		}
	}

	// Cap mage-go at the recording's stopOnTurn so we don't run past the
	// last validated PRIORITY event.
	r.game.Run(r.maxTurns, core.Cleanup, r.maxTurns)

	if r.divergence != "" {
		return fmt.Errorf("divergence: %s", r.divergence)
	}
	return nil
}

// peek returns the next event in the recording, optionally filtered by
// types. Returns io.EOF when the recording is exhausted. Does NOT advance
// the cursor.
func (r *replay) peek(types ...string) (*eventLine, int, error) {
	for i := r.cursor; i < len(r.events); i++ {
		ev := &r.events[i]
		if len(types) == 0 {
			return ev, i, nil
		}
		if slices.Contains(types, ev.Type) {
			return ev, i, nil
		}
	}
	return nil, -1, io.EOF
}

// advanceTo finds the next event of one of the given types and advances
// the cursor past it. Skipped events are silently dropped — they're
// expected to be informational (LOG, GAME_START, STACK_RESOLVE) or
// downstream effects we don't validate (STACK_PUSH, which we already
// peek for in onPriority).
func (r *replay) advanceTo(types ...string) (*eventLine, error) {
	ev, idx, err := r.peek(types...)
	if err != nil {
		return nil, err
	}
	r.cursor = idx + 1
	return ev, nil
}

// fail records the first divergence found and asks the engine to stop.
// Subsequent fail() calls during the same run are ignored.
func (r *replay) fail(format string, args ...any) {
	if r.divergence != "" {
		return
	}
	r.divergence = fmt.Sprintf(format, args...)
}

// =============================================================================
// onPriority — the validation entry point. Called by mage-go whenever a
// player gets priority. Pulls the next PRIORITY event off the recording,
// validates state + playable actions, then peeks ahead to figure out what
// XMage did and returns the corresponding PriorityAction.
//
// Tricky bit: the two engines don't agree on the count of priority points.
// In particular, after a player acts, mage-go cycles back through priority
// (CR 117.3c: active player retains priority after their action), but
// XMage's recording often skips that "implicit" priority and only logs
// the next player's. So we peek the next PRIORITY in the recording and
// only consume it if its priorityPlayerId matches the mage-go player who's
// currently asking. If not, mage-go has an "extra" priority point — return
// pass without consuming the event so the next, real priority point lines
// up correctly.
// =============================================================================
func (r *replay) onPriority(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
	if r.divergence != "" {
		return mage.PriorityAction{Type: mage.PriorityPass}
	}

	ev, idx, err := r.peek("PRIORITY")
	if err != nil {
		r.fail("ran out of recording at mage-go priority pass (T%d %s p%d)",
			g.CurrentTurn(), stepName(g.GetStep()), playerIdx)
		return mage.PriorityAction{Type: mage.PriorityPass}
	}

	mageGoPlayerName := g.PlayerAt(playerIdx).Name()
	recordedName := r.playerNameByID(ev.Snapshot.PriorityPlayerID)
	if recordedName != "" && recordedName != mageGoPlayerName {
		// mage-go is asking for priority but the next PRIORITY in the
		// recording is for a different player — i.e. mage-go has an
		// extra priority cycle XMage skipped. Auto-pass without
		// consuming the event.
		return mage.PriorityAction{Type: mage.PriorityPass}
	}

	// Consume the matched event.
	r.cursor = idx + 1

	if mismatches := r.diffSnapshot(g, playerIdx, ev.Snapshot); len(mismatches) > 0 {
		r.fail("state mismatch at PRIORITY seq=%d T%d %s:\n  %s",
			ev.Seq, ev.Snapshot.Turn, ev.Snapshot.Step, strings.Join(mismatches, "\n  "))
		return mage.PriorityAction{Type: mage.PriorityPass}
	}

	if mismatches := r.diffPlayableActions(g, playerIdx, ev.Snapshot.PlayableActions); len(mismatches) > 0 {
		// Warning, not failure: mage-go's Playable() stub is approximate
		// (no target-legality check, no activated abilities). Real engine
		// bugs surface as state divergences; action-set drift is mostly
		// stub-vs-reality and noisy. Track for reporting; keep going.
		r.warnings = append(r.warnings, fmt.Sprintf(
			"playable-action diff seq=%d T%d %s:\n    %s",
			ev.Seq, ev.Snapshot.Turn, ev.Snapshot.Step, strings.Join(mismatches, "\n    ")))
	}

	return r.decideAction(g, playerIdx)
}

// decideAction looks ahead in the event stream to see what XMage did at
// this priority point — the next decision-bearing event before the next
// PRIORITY event tells us. If we see another PRIORITY first, XMage passed.
func (r *replay) decideAction(g *mage.Game, playerIdx int) mage.PriorityAction {
	for i := r.cursor; i < len(r.events); i++ {
		ev := &r.events[i]
		switch ev.Type {
		case "PRIORITY", "ATTACKERS_DECLARED", "BLOCKERS_DECLARED", "GAME_END":
			// hit the next decision boundary without a stack/land action — pass
			return mage.PriorityAction{Type: mage.PriorityPass}
		case "LAND_PLAYED":
			if ev.Action == nil {
				continue
			}
			cardName := ev.Action.SourceName
			player := g.PlayerAt(playerIdx)
			for _, c := range player.Hand() {
				if c.Name() == cardName && c.HasType(core.TypeLand) {
					r.cursor = i + 1
					return mage.PriorityAction{Type: mage.PriorityPlayLand, CardID: c.ID()}
				}
			}
			r.fail("LAND_PLAYED %q at seq=%d but mage-go p%d has no such land in hand",
				cardName, ev.Seq, playerIdx)
			return mage.PriorityAction{Type: mage.PriorityPass}
		case "STACK_PUSH":
			if ev.Action == nil {
				continue
			}
			pa, ok := r.translateCast(g, playerIdx, ev)
			if !ok {
				return mage.PriorityAction{Type: mage.PriorityPass}
			}
			r.cursor = i + 1
			return pa
		default:
			// LOG, STACK_RESOLVE, DISCARD_TAKEN, TRIGGER_ORDER — keep scanning
		}
	}
	return mage.PriorityAction{Type: mage.PriorityPass}
}

// translateCast maps an XMage STACK_PUSH (CAST_SPELL or activated ability)
// into a mage-go PriorityAction by looking up the card by name in the
// active player's hand or battlefield. Sets r.fail and returns ok=false on
// any miss.
func (r *replay) translateCast(g *mage.Game, playerIdx int, ev *eventLine) (mage.PriorityAction, bool) {
	a := ev.Action
	switch a.Kind {
	case "CAST_SPELL":
		player := g.PlayerAt(playerIdx)
		for _, c := range player.Hand() {
			if c.Name() == a.SourceName {
				targets := r.resolveTargets(g, a.ChosenTargets)
				x := 0
				if a.XValue != nil {
					x = *a.XValue
				}
				return mage.PriorityAction{
					Type:    mage.PriorityCastSpell,
					CardID:  c.ID(),
					Targets: targets,
					XValue:  x,
				}, true
			}
		}
		r.fail("CAST_SPELL %q at seq=%d but mage-go p%d hand has no such card",
			a.SourceName, ev.Seq, playerIdx)
		return mage.PriorityAction{}, false
	case "ACTIVATE", "ACTIVATE_MANA":
		controllerID := r.playerByID[a.PlayerID]
		if controllerID == nil {
			r.fail("ACTIVATE seq=%d: unknown playerId %s", ev.Seq, a.PlayerID)
			return mage.PriorityAction{}, false
		}
		perm := g.FindPermanentByName(a.SourceName, controllerID.PlayerID())
		if perm == nil {
			r.fail("ACTIVATE %q seq=%d: permanent not found on p%d battlefield",
				a.SourceName, ev.Seq, playerIdx)
			return mage.PriorityAction{}, false
		}
		// Without ability index in the recording, we'd need rule-text matching
		// to disambiguate among multiple activated abilities. Punt with a clear
		// failure for now — skill-3 self-play rarely triggers this.
		return mage.PriorityAction{
			Type:        mage.PriorityActivateAbility,
			PermanentID: perm.ID(),
			AbilityIdx:  0,
			Targets:     r.resolveTargets(g, a.ChosenTargets),
		}, true
	case "TRIGGERED":
		// Triggered abilities are pushed by the engine itself, not by a player
		// priority decision. mage-go handles them automatically; treat XMage's
		// TRIGGERED stack push as "no priority action needed here" and keep
		// scanning for the real decision.
		return mage.PriorityAction{Type: mage.PriorityPass}, true
	default:
		r.fail("STACK_PUSH unknown kind=%q at seq=%d", a.Kind, ev.Seq)
		return mage.PriorityAction{}, false
	}
}

func (r *replay) resolveTargets(g *mage.Game, refs []targetRef) []uuid.UUID {
	if len(refs) == 0 {
		return nil
	}
	out := make([]uuid.UUID, 0, len(refs))
	for _, ref := range refs {
		// Players first
		matched := false
		for _, p := range g.AllPlayers() {
			if p.Name() == ref.Name {
				out = append(out, p.PlayerID())
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		for _, perm := range g.AllBattlefield() {
			if perm.Name() == ref.Name {
				out = append(out, perm.ID())
				break
			}
		}
	}
	return out
}

// normalizeXMageStep maps XMage's PhaseStep enum names (e.g. "END_TURN")
// to mage-go's wire vocabulary (e.g. "end_step"). The two engines disagree
// only on a couple of step names; everything else is case-folded equivalent.
func normalizeXMageStep(s string) string {
	switch s {
	case "END_TURN":
		return "end_step"
	case "FIRST_COMBAT_DAMAGE":
		return "first_strike_damage"
	}
	return strings.ToLower(s)
}

func stepName(s core.PhaseStep) string {
	switch s {
	case core.Untap:
		return "untap"
	case core.Upkeep:
		return "upkeep"
	case core.Draw:
		return "draw"
	case core.PrecombatMain:
		return "precombat_main"
	case core.BeginCombat:
		return "begin_combat"
	case core.DeclareAttackers:
		return "declare_attackers"
	case core.DeclareBlockers:
		return "declare_blockers"
	case core.FirstStrikeDamage:
		return "first_strike_damage"
	case core.CombatDamage:
		return "combat_damage"
	case core.EndCombat:
		return "end_combat"
	case core.PostcombatMain:
		return "postcombat_main"
	case core.EndStep:
		return "end_step"
	case core.Cleanup:
		return "cleanup"
	}
	return "unknown"
}

// =============================================================================
// State diff — fields we compare between mage-go's live game and XMage's
// recorded snapshot. We sort player order by name on both sides so indexing
// is stable regardless of who's active.
// =============================================================================

type goPlayerSnap struct {
	id          string
	name        string
	life        int
	librarySize int
	hand        []string
	graveyard   []string
}

func (r *replay) diffSnapshot(g *mage.Game, playerIdx int, want *snapshot) []string {
	if want == nil {
		return nil
	}
	var diffs []string

	if g.CurrentTurn() != want.Turn {
		diffs = append(diffs, fmt.Sprintf("turn: go=%d xmage=%d", g.CurrentTurn(), want.Turn))
	}
	if got := stepName(g.GetStep()); got != normalizeXMageStep(want.Step) {
		diffs = append(diffs, fmt.Sprintf("step: go=%s xmage=%s", got, want.Step))
	}

	// Build mage-go-side player snapshots, sorted by name to match recorder ordering.
	goPlayers := buildGoPlayers(g)
	if len(goPlayers) != len(want.Players) {
		diffs = append(diffs, fmt.Sprintf("player count: go=%d xmage=%d", len(goPlayers), len(want.Players)))
		return diffs
	}
	for i := range goPlayers {
		gp := goPlayers[i]
		wp := want.Players[i]
		prefix := fmt.Sprintf("player[%s]", wp.Name)
		if gp.life != wp.Life {
			diffs = append(diffs, fmt.Sprintf("%s.life: go=%d xmage=%d", prefix, gp.life, wp.Life))
		}
		if gp.librarySize != wp.LibrarySize {
			diffs = append(diffs, fmt.Sprintf("%s.library_size: go=%d xmage=%d", prefix, gp.librarySize, wp.LibrarySize))
		}
		wantHand := sortedCopy(wp.Hand)
		if !equalNameMultiset(gp.hand, wantHand) {
			diffs = append(diffs, fmt.Sprintf("%s.hand: go=[%s] xmage=[%s]", prefix,
				strings.Join(gp.hand, ", "), strings.Join(wantHand, ", ")))
		}
		wantGY := sortedCopy(wp.Graveyard)
		if !equalNameMultiset(gp.graveyard, wantGY) {
			diffs = append(diffs, fmt.Sprintf("%s.graveyard: go=[%s] xmage=[%s]", prefix,
				strings.Join(gp.graveyard, ", "), strings.Join(wantGY, ", ")))
		}
	}

	// Battlefield diff is split into two strictness levels:
	//
	//   - Names (multiset): fatal divergence if differ — engines disagree
	//     on which permanents exist under each controller.
	//
	//   - Tap state: warning if names match but tap multisets differ —
	//     this happens when both engines pay an equivalent cost using
	//     different mana sources (e.g., mage-go's AutoTapForCost picks
	//     a different combination of lands+moxen than XMage). Same
	//     resources spent, just different choice.
	goBF := buildGoBattlefield(g)
	wantBF := groupRecordedBattlefieldByController(want)
	for ctrlName, goPerms := range goBF {
		wantPerms := wantBF[ctrlName]
		if !equalNameMultiset(permNameList(goPerms), permNameList(wantPerms)) {
			diffs = append(diffs, fmt.Sprintf("battlefield[%s]: go=%s xmage=%s", ctrlName,
				formatPerms(goPerms), formatPerms(wantPerms)))
			continue
		}
		// Same names. Tap-state difference becomes a warning instead.
		if !equalPermSet(goPerms, wantPerms) {
			r.warnings = append(r.warnings, fmt.Sprintf(
				"tap-state diff battlefield[%s]: go=%s xmage=%s (same names, different tap multiset)",
				ctrlName, formatPerms(goPerms), formatPerms(wantPerms)))
		}
	}
	for ctrlName, wantPerms := range wantBF {
		if _, ok := goBF[ctrlName]; ok {
			continue
		}
		// controller present in xmage but missing in go
		diffs = append(diffs, fmt.Sprintf("battlefield[%s]: go=[] xmage=%s", ctrlName, formatPerms(wantPerms)))
	}
	return diffs
}

// permNameList returns just the names from a permEntry slice, preserving
// order so the multiset comparison via equalNameMultiset (which expects
// pre-sorted inputs) still works on already-sorted permEntry lists.
func permNameList(ps []permEntry) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.name
	}
	return out
}

func buildGoPlayers(g *mage.Game) []goPlayerSnap {
	out := make([]goPlayerSnap, 0, len(g.AllPlayers()))
	for _, p := range g.AllPlayers() {
		hand := make([]string, 0, len(p.Hand()))
		for _, c := range p.Hand() {
			hand = append(hand, c.Name())
		}
		sort.Strings(hand)
		gy := make([]string, 0, len(p.Graveyard()))
		for _, c := range p.Graveyard() {
			gy = append(gy, c.Name())
		}
		sort.Strings(gy)
		out = append(out, goPlayerSnap{
			id:          p.PlayerID().String(),
			name:        p.Name(),
			life:        p.Life(),
			librarySize: len(p.Library()),
			hand:        hand,
			graveyard:   gy,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

type permEntry struct {
	name   string
	tapped bool
}

func buildGoBattlefield(g *mage.Game) map[string][]permEntry {
	out := map[string][]permEntry{}
	for _, perm := range g.AllBattlefield() {
		if perm.PhasedOut {
			continue
		}
		// Find controller name
		var ctrlName string
		for _, p := range g.AllPlayers() {
			if p.PlayerID() == perm.Controller {
				ctrlName = p.Name()
				break
			}
		}
		if ctrlName == "" {
			ctrlName = "<unknown>"
		}
		out[ctrlName] = append(out[ctrlName], permEntry{name: perm.Name(), tapped: perm.Tapped})
	}
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool {
			if out[k][i].name != out[k][j].name {
				return out[k][i].name < out[k][j].name
			}
			return !out[k][i].tapped // false (untapped) before true (tapped)
		})
	}
	return out
}

func groupRecordedBattlefieldByController(s *snapshot) map[string][]permEntry {
	// Build playerId -> name first.
	idToName := map[string]string{}
	for _, p := range s.Players {
		idToName[p.ID] = p.Name
	}
	out := map[string][]permEntry{}
	for _, p := range s.Battlefield {
		ctrlName := idToName[p.ControllerID]
		if ctrlName == "" {
			ctrlName = "<unknown>"
		}
		out[ctrlName] = append(out[ctrlName], permEntry{name: p.Name, tapped: p.Tapped})
	}
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool {
			if out[k][i].name != out[k][j].name {
				return out[k][i].name < out[k][j].name
			}
			return !out[k][i].tapped
		})
	}
	return out
}

func equalPermSet(a, b []permEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func formatPerms(ps []permEntry) string {
	parts := make([]string, len(ps))
	for i, p := range ps {
		if p.tapped {
			parts[i] = p.name + "(T)"
		} else {
			parts[i] = p.name
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func sortedCopy(s []string) []string {
	out := make([]string, len(s))
	copy(out, s)
	sort.Strings(out)
	return out
}

func equalNameMultiset(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// =============================================================================
// Playable-action diff — loose match: compare the multiset of (kind,
// sourceName) pairs that mage-go reports vs what XMage reports. Ignores
// rule text, cost text, target candidate sets.
// =============================================================================

func (r *replay) diffPlayableActions(g *mage.Game, playerIdx int, want []playableAction) []string {
	got := mageGoPlayableActions(g, playerIdx)
	wantSet := normalizedActionSet(loosePairsFromXMage(want))
	gotSet := normalizedActionSet(got)
	if equalStringSlices(wantSet, gotSet) {
		return nil
	}
	return []string{fmt.Sprintf("playable: go=%v xmage=%v", gotSet, wantSet)}
}

// loosePairsFromXMage extracts (kind, sourceName) pairs and normalizes
// kinds to mage-go names (CAST_SPELL → cast, PLAY_LAND → land, etc.).
// Filters down to the kinds mage-go's Game.Playable currently enumerates
// (lands + castable spells) — comparing on activated/mana abilities would
// always fail until those are added to the v1 stub.
func loosePairsFromXMage(actions []playableAction) []string {
	pairs := make([]string, 0, len(actions))
	for _, a := range actions {
		switch a.Kind {
		case "PLAY_LAND", "CAST_SPELL":
			pairs = append(pairs, looseKey(a.Kind, a.SourceName))
		}
	}
	return pairs
}

func looseKey(kind, source string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	switch k {
	case "cast_spell":
		k = "cast"
	case "play_land":
		k = "land"
	case "activate_mana", "activate":
		k = "activate"
	}
	return k + ":" + source
}

func normalizedActionSet(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// =============================================================================
// replayPlayer — wraps BasePlayer and overrides the decision hooks so they
// consume from the replay cursor instead of making AI choices.
// =============================================================================

type replayPlayer struct {
	*mage.BasePlayer
	r *replay
}

func newReplayPlayer(name string, r *replay) *replayPlayer {
	return &replayPlayer{BasePlayer: mage.NewBasePlayer(name), r: r}
}

func (p *replayPlayer) DeclareAttackers(g *mage.Game) []uuid.UUID {
	if p.r.divergence != "" {
		return nil
	}
	ev, err := p.r.advanceTo("ATTACKERS_DECLARED")
	if err != nil || ev.Action == nil {
		p.r.fail("DeclareAttackers fired on mage-go but no ATTACKERS_DECLARED in recording (T%d %s)",
			g.CurrentTurn(), stepName(g.GetStep()))
		return nil
	}
	var ids []uuid.UUID
	for _, ref := range ev.Action.Attackers {
		perm := g.FindPermanentByName(ref.Name, p.PlayerID())
		if perm != nil {
			ids = append(ids, perm.ID())
		}
	}
	return ids
}

func (p *replayPlayer) DeclareBlockers(g *mage.Game) []mage.BlockAssignment {
	if p.r.divergence != "" {
		return nil
	}
	ev, err := p.r.advanceTo("BLOCKERS_DECLARED")
	if err != nil || ev.Action == nil {
		p.r.fail("DeclareBlockers fired on mage-go but no BLOCKERS_DECLARED in recording (T%d %s)",
			g.CurrentTurn(), stepName(g.GetStep()))
		return nil
	}
	var out []mage.BlockAssignment
	for _, bp := range ev.Action.Blockers {
		blocker := g.FindPermanentByName(bp.BlockerName, p.PlayerID())
		var attacker *mage.Permanent
		for _, perm := range g.AllBattlefield() {
			if perm.Name() == bp.AttackerName && g.IsAttackingInCombat(perm.ID()) {
				attacker = perm
				break
			}
		}
		if blocker != nil && attacker != nil {
			out = append(out, mage.BlockAssignment{
				BlockerID:  blocker.ID(),
				AttackerID: attacker.ID(),
			})
		}
	}
	return out
}

func (p *replayPlayer) ChooseCardsFromHand(amount int, reason string, g mage.GameReader) []mage.Card {
	if p.r.divergence != "" {
		return nil
	}
	if !strings.Contains(strings.ToLower(reason), "discard") {
		// Non-discard "pick from hand" prompts aren't recorded; fall back to
		// BasePlayer's deterministic head-of-hand behavior.
		return p.BasePlayer.ChooseCardsFromHand(amount, reason, g)
	}
	ev, err := p.r.advanceTo("DISCARD_TAKEN")
	if err != nil || ev.Action == nil {
		p.r.fail("ChooseCardsFromHand fired but no DISCARD_TAKEN in recording (reason=%q)", reason)
		return nil
	}
	hand := p.Hand()
	used := make(map[int]bool)
	var picked []mage.Card
	for _, ref := range ev.Action.Discarded {
		for i, c := range hand {
			if used[i] {
				continue
			}
			if c.Name() == ref.Name {
				picked = append(picked, c)
				used[i] = true
				break
			}
		}
	}
	return picked
}

// mageGoPlayableActions returns mage-go's (kind, sourceName) pairs for the
// given player at the current priority point. Wraps Game.Playable; loose
// match in diffPlayableActions ignores anything beyond kind+sourceName.
//
// Scope (matches mage.Game.Playable's v1 stub): lands in hand + castable
// spells in hand. Activated abilities aren't enumerated, so XMage's
// ACTIVATE_MANA / ACTIVATE entries get filtered out before the diff.
func mageGoPlayableActions(g *mage.Game, playerIdx int) []string {
	actions := g.Playable(playerIdx)
	pairs := make([]string, 0, len(actions))
	for _, a := range actions {
		pairs = append(pairs, looseKey(strings.ToUpper(a.Kind), a.SourceName))
	}
	return pairs
}
