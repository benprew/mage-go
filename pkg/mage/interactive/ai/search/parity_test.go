package search

// Parity tests: verify that the unified minimax in this package preserves the
// behavioral properties of the simple-minimax reference implementation at
// /Users/ben/src/simple-minimax/minimax. The first two tests are direct ports
// of TestTurn11AliceAttacksWithSpecters and TestTurn17UnholyStrengthTargetsOwnCreature
// from that package; the rest are property-style tests that exercise the
// algorithm without depending on the card registry.

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/google/uuid"

	_ "git.sr.ht/~cdcarter/mage-go/cards"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

// ── Helpers shared across parity tests ──────────────────────────────────────

func parityCard(t *testing.T, name string, owner uuid.UUID) mage.Card {
	t.Helper()
	c, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%q): %v", name, err)
	}
	c.SetOwner(owner)
	return c
}

func parityAddHand(t *testing.T, p mage.Player, names ...string) {
	t.Helper()
	for _, name := range names {
		p.AddToHand(parityCard(t, name, p.PlayerID()))
	}
}

func parityAddLibrary(t *testing.T, p mage.Player, n int, name string) {
	t.Helper()
	for range n {
		p.AddToLibrary(parityCard(t, name, p.PlayerID()))
	}
}

func parityAddBattlefield(t *testing.T, g *mage.Game, controller uuid.UUID, name string) *mage.Permanent {
	t.Helper()
	return g.PutOnBattlefield(parityCard(t, name, controller), controller)
}

func parityAddReady(t *testing.T, g *mage.Game, controller uuid.UUID, name string) *mage.Permanent {
	t.Helper()
	perm := parityAddBattlefield(t, g, controller, name)
	perm.RevokeBaseAttr(core.AttrSummonSick)
	return perm
}

// searchActiveRootForTest is the test analogue of simple-minimax's
// searchForTest: drives the raw searcher directly without going through the
// Strategy adapter. Used so that test scope mirrors simple-minimax exactly —
// no synthetic step changes, no Fallback intervention, just s.search.
func searchActiveRootForTest(g *mage.Game, rootPlayerID uuid.UUID, useTT bool) Result {
	root := g.Clone()
	rootPlayer := root.GetPlayer(rootPlayerID)
	var tt map[uint64]simpleTTEntry
	if useTT {
		tt = make(map[uint64]simpleTTEntry)
	}
	s := &searcher{rootPlayer: rootPlayer, tt: tt, zobrist: DefaultZobrist}
	chain, score := s.search(root, rootPlayerID, false, negInf, posInf)
	return Result{Chain: chain, Score: score, Nodes: s.nodes, TTHits: s.ttHits, TTStores: s.ttStores}
}

func ttLabel(useTT bool) string {
	if useTT {
		return "tt=true"
	}
	return "tt=false"
}

func firstAttackInChain(chain []ChainStep, player string) []uuid.UUID {
	for _, step := range chain {
		if step.Phase == core.DeclareAttackers && step.Player == player && step.Attackers != nil {
			return step.Attackers
		}
	}
	return nil
}

// ── Port: Turn 11 — Alice attacks with both Hypnotic Specters ───────────────

func TestParity_Turn11_AliceAttacksWithBothSpecters(t *testing.T) {
	for _, useTT := range []bool{false, true} {
		t.Run(ttLabel(useTT), func(t *testing.T) {
			g, aliceID := turn11SpecterAttackGame(t)
			res := searchActiveRootForTest(g, aliceID, useTT)

			first := firstAttackInChain(res.Chain, "Alice")
			if len(first) == 0 {
				t.Fatalf("Alice chose no attackers, want both Hypnotic Specters")
			}
			got := attackerSubsetName(g, first)
			if got != "[Hypnotic Specter, Hypnotic Specter]" {
				t.Fatalf("Alice attackers = %s, want both Hypnotic Specters", got)
			}
		})
	}
}

func turn11SpecterAttackGame(t *testing.T) (*mage.Game, uuid.UUID) {
	t.Helper()
	alice := mage.NewBasePlayer("Alice")
	bob := mage.NewBasePlayer("Bob")
	alice.SetLife(14)
	bob.SetLife(16)

	g := mage.NewGame(alice, bob)
	g.SetActivePlayerIndex(0)
	g.SetTurn(11)
	g.SetStep(core.DeclareAttackers)

	parityAddHand(t, alice, "Mountain", "Swamp", "Mountain", "Hill Giant")
	parityAddLibrary(t, alice, 28, "Mountain")
	parityAddLibrary(t, bob, 30, "Forest")

	parityAddReady(t, g, alice.PlayerID(), "Mountain")
	parityAddReady(t, g, alice.PlayerID(), "Mountain")
	parityAddReady(t, g, alice.PlayerID(), "Swamp")
	parityAddReady(t, g, alice.PlayerID(), "Swamp")
	parityAddReady(t, g, alice.PlayerID(), "Hypnotic Specter")
	parityAddReady(t, g, alice.PlayerID(), "Mountain")
	parityAddReady(t, g, alice.PlayerID(), "Hypnotic Specter")

	parityAddReady(t, g, bob.PlayerID(), "Forest")
	luredElf := parityAddReady(t, g, bob.PlayerID(), "Llanowar Elves")
	luredElf.Tapped = true
	parityAddReady(t, g, bob.PlayerID(), "Plains").Tapped = true
	lure := parityAddReady(t, g, bob.PlayerID(), "Lure")
	g.Attach(lure.ID(), luredElf.ID())
	parityAddReady(t, g, bob.PlayerID(), "Forest")
	parityAddReady(t, g, bob.PlayerID(), "Grizzly Bears").Tapped = true
	parityAddReady(t, g, bob.PlayerID(), "Savannah Lions").Tapped = true
	parityAddReady(t, g, bob.PlayerID(), "Llanowar Elves")
	parityAddReady(t, g, bob.PlayerID(), "Plains")
	holyStrength := parityAddReady(t, g, bob.PlayerID(), "Holy Strength")
	g.Attach(holyStrength.ID(), luredElf.ID())

	return g, alice.PlayerID()
}

// ── Port: Turn 17 — Unholy Strength scores higher on own creature ───────────

// Faithful port of simple-minimax's TestTurn17UnholyStrengthTargetsOwnCreature.
// The strict assertion is on per-target scoring (Bears < Orcs): if Alice cast
// Unholy Strength on her own Ironclaw Orcs the resulting position scores
// higher than casting it on Bob's Grizzly Bears. Simple-minimax baseline:
// Orcs=8.10, Bears=0.90. The top-level move choice isn't asserted because
// other lines (e.g. Disintegrate at PostcombatMain) outscore Unholy Strength
// and validly win — what matters is that Unholy Strength → own creature is
// preferred over Unholy Strength → opponent creature in the conditional eval.
func TestParity_Turn17_UnholyStrengthScoresHigherOnOwnCreature(t *testing.T) {
	for _, useTT := range []bool{false, true} {
		t.Run(ttLabel(useTT), func(t *testing.T) {
			g, aliceID := turn17UnholyStrengthGame(t)
			outcomes := unholyStrengthTargetOutcomes(t, g, aliceID, useTT)
			orcs, okO := bestOutcomeForTarget(outcomes, "Ironclaw Orcs", "Alice")
			bears, okB := bestOutcomeForTarget(outcomes, "Grizzly Bears", "Bob")
			if !okO {
				t.Fatalf("no per-target outcome for Ironclaw Orcs (Alice)")
			}
			if !okB {
				t.Fatalf("no per-target outcome for Grizzly Bears (Bob)")
			}
			if bears.score >= orcs.score {
				t.Fatalf("Unholy Strength → Bears scored %.2f, want < Orcs %.2f", bears.score, orcs.score)
			}
		})
	}
}

type unholyOutcome struct {
	target     string
	controller string
	score      float64
}

func unholyStrengthTargetOutcomes(t *testing.T, g *mage.Game, aliceID uuid.UUID, useTT bool) []unholyOutcome {
	t.Helper()
	alice := g.GetPlayer(aliceID)
	var card mage.Card
	for _, c := range alice.Hand() {
		if c.Name() == "Unholy Strength" {
			card = c
			break
		}
	}
	if card == nil {
		t.Fatalf("Unholy Strength not in Alice's hand")
	}
	targets := card.CastTargets()
	if len(targets) == 0 {
		t.Fatalf("Unholy Strength has no cast targets")
	}
	var out []unholyOutcome
	for _, targetID := range targets[0].Possible(aliceID, card, g) {
		m := &Move{
			Type:     interactive.ActionCastSpell,
			CardID:   card.ID(),
			CardName: card.Name(),
			Targets:  []uuid.UUID{targetID},
		}
		clone := g.Clone()
		applyMove(clone, aliceID, m)
		res := searchActiveRootForTest(clone, aliceID, useTT)
		target := "none"
		controller := "none"
		if p := g.FindPermanent(targetID); p != nil {
			target = p.Name()
			if player := g.GetPlayer(p.Controller); player != nil {
				controller = player.Name()
			}
		}
		out = append(out, unholyOutcome{
			target:     target,
			controller: controller,
			score:      res.Score,
		})
	}
	return out
}

func bestOutcomeForTarget(outcomes []unholyOutcome, target, controller string) (unholyOutcome, bool) {
	var best unholyOutcome
	ok := false
	for _, o := range outcomes {
		if o.target != target || o.controller != controller {
			continue
		}
		if !ok || o.score > best.score {
			best = o
			ok = true
		}
	}
	return best, ok
}

func turn17UnholyStrengthGame(t *testing.T) (*mage.Game, uuid.UUID) {
	t.Helper()
	alice := mage.NewBasePlayer("Alice")
	bob := mage.NewBasePlayer("Bob")
	alice.SetLife(22)
	bob.SetLife(18)

	g := mage.NewGame(alice, bob)
	g.SetActivePlayerIndex(0)
	g.SetTurn(17)
	g.SetStep(core.PrecombatMain)

	parityAddHand(t, alice, "Unholy Strength", "Black Knight", "Disintegrate", "Disintegrate", "Black Knight", "Hypnotic Specter")
	parityAddLibrary(t, alice, 26, "Mountain")
	parityAddLibrary(t, bob, 28, "Forest")

	parityAddBattlefield(t, g, alice.PlayerID(), "Mountain")
	parityAddBattlefield(t, g, alice.PlayerID(), "Swamp")
	parityAddBattlefield(t, g, alice.PlayerID(), "Ironclaw Orcs")
	parityAddBattlefield(t, g, alice.PlayerID(), "Ironclaw Orcs")

	parityAddBattlefield(t, g, bob.PlayerID(), "Forest")
	parityAddBattlefield(t, g, bob.PlayerID(), "Llanowar Elves")
	parityAddBattlefield(t, g, bob.PlayerID(), "Forest")
	parityAddBattlefield(t, g, bob.PlayerID(), "Llanowar Elves")
	parityAddBattlefield(t, g, bob.PlayerID(), "Plains").Tapped = true
	parityAddBattlefield(t, g, bob.PlayerID(), "Grizzly Bears")
	parityAddBattlefield(t, g, bob.PlayerID(), "Savannah Lions")
	parityAddBattlefield(t, g, bob.PlayerID(), "Forest")

	return g, alice.PlayerID()
}

// ── Property: moveLess tiebreaker is deterministic ──────────────────────────

// The tiebreaker makes alpha-beta's choice independent of exploration order.
// Ordering: non-pass always beats pass; among non-pass, lower Type first;
// then CardID, PermanentID, XValue, ModeIndex, Targets. Pass-loses-to-action
// is the one deviation from simple-minimax — without it, equal-score lethal
// in this main phase vs. the next collapses to Pass, and Strategy's current-
// step filter masks the lethal entirely.
func TestParity_MoveLess_Deterministic(t *testing.T) {
	cardA := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	cardB := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	cases := []struct {
		name string
		a, b Move
		want bool
	}{
		{"pass loses to land", Move{Type: interactive.ActionPass}, Move{Type: interactive.ActionPlayLand}, false},
		{"land beats pass", Move{Type: interactive.ActionPlayLand}, Move{Type: interactive.ActionPass}, true},
		{"pass loses to cast", Move{Type: interactive.ActionPass}, Move{Type: interactive.ActionCastSpell}, false},
		{"cast beats pass", Move{Type: interactive.ActionCastSpell}, Move{Type: interactive.ActionPass}, true},
		{"land<cast", Move{Type: interactive.ActionPlayLand}, Move{Type: interactive.ActionCastSpell}, true},
		{"cast<activate", Move{Type: interactive.ActionCastSpell}, Move{Type: interactive.ActionActivateAbility}, true},
		{"same type, lower cardID first",
			Move{Type: interactive.ActionCastSpell, CardID: cardA},
			Move{Type: interactive.ActionCastSpell, CardID: cardB},
			true,
		},
		{"same type, equal cards, lower XValue first",
			Move{Type: interactive.ActionCastSpell, CardID: cardA, XValue: 1},
			Move{Type: interactive.ActionCastSpell, CardID: cardA, XValue: 2},
			true,
		},
		{"same type, equal cards, lower ModeIndex first",
			Move{Type: interactive.ActionCastSpell, CardID: cardA, ModeIndex: 0},
			Move{Type: interactive.ActionCastSpell, CardID: cardA, ModeIndex: 1},
			true,
		},
		{"identical moves are not less",
			Move{Type: interactive.ActionCastSpell, CardID: cardA},
			Move{Type: interactive.ActionCastSpell, CardID: cardA},
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, b := c.a, c.b
			if got := moveLess(&a, &b); got != c.want {
				t.Fatalf("moveLess(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

// ── Property: legalPriorityMoves restricts non-root to abilities + pass ─────

// Simple-minimax simplifies the opponent's choices: when isRoot=false, only
// activated abilities and pass are enumerated. No land plays, no spell casts.
// This is the "opponent plays no cards from hand" simplification, and it
// avoids needing to know the opponent's hand at all.
func TestParity_LegalPriorityMoves_NonRootDropsHandPlays(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)

	land := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
	land.SetOwner(pa.PlayerID())
	pa.AddToHand(land)

	bear := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	bear.SetOwner(pa.PlayerID())
	pa.AddToHand(bear)
	addLands(g, pa, "Forest", 2)

	asRoot := legalPriorityMoves(g, pa, true)
	asNonRoot := legalPriorityMoves(g, pa, false)

	rootHasLand := false
	rootHasCast := false
	for _, m := range asRoot {
		if m.Type == interactive.ActionPlayLand {
			rootHasLand = true
		}
		if m.Type == interactive.ActionCastSpell {
			rootHasCast = true
		}
	}
	if !rootHasLand || !rootHasCast {
		t.Fatalf("root moves missing land/cast: land=%v cast=%v", rootHasLand, rootHasCast)
	}

	for _, m := range asNonRoot {
		if m.Type == interactive.ActionPlayLand {
			t.Errorf("non-root should not emit ActionPlayLand")
		}
		if m.Type == interactive.ActionCastSpell {
			t.Errorf("non-root should not emit ActionCastSpell")
		}
	}

	if len(asNonRoot) == 0 || asNonRoot[len(asNonRoot)-1].Type != interactive.ActionPass {
		t.Errorf("non-root moves must always end with pass")
	}
}

// ── Property: terminalEval is bounded and signed by who's alive ─────────────

// Simple-minimax returns ±terminalScore (modulo a constant offset) based on
// IsAlive — the absolute magnitude is what makes search prefer terminal lines
// over heuristic-evaluated ones. Verify the sign and magnitude.
func TestParity_TerminalEval_SignedByWinner(t *testing.T) {
	g, pa, pb := makeGame()

	pa.SetLife(20)
	pb.SetLife(-1)
	if got := terminalEval(g, pa.PlayerID()); got <= 1e5 {
		t.Fatalf("alive-vs-dead = %v, want large positive", got)
	}
	if got := terminalEval(g, pb.PlayerID()); got >= -1e5 {
		t.Fatalf("dead-vs-alive = %v, want large negative", got)
	}

	pa.SetLife(20)
	pb.SetLife(20)
	if got := terminalEval(g, pa.PlayerID()); got != 0 {
		t.Errorf("both alive eval = %v, want 0", got)
	}
}

// ── Property: blockerSubsets always includes empty + every legal singleton ──

// Simple-minimax's minimum guarantee for blockerSubsets is: include the
// no-blocks option and every legal singleton blocker→attacker pairing. The
// greedy 1:1 set may or may not be present depending on legality. Mage-go
// adds gang blocks on top but must preserve this baseline.
func TestParity_BlockerSubsets_BaselineCoverage(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{3}{R}", 3, 3, pa.PlayerID())
	blk1 := makePerm("Wall", "{1}{W}", 0, 4, pb.PlayerID())
	blk2 := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk1, blk2)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	sets := blockerSubsets(g, pb.PlayerID())

	foundNil := false
	foundWallBlock := false
	foundBearBlock := false
	for _, s := range sets {
		if len(s) == 0 {
			foundNil = true
			continue
		}
		if len(s) == 1 && s[0].BlockerID == blk1.ID() && s[0].AttackerID == atk.ID() {
			foundWallBlock = true
		}
		if len(s) == 1 && s[0].BlockerID == blk2.ID() && s[0].AttackerID == atk.ID() {
			foundBearBlock = true
		}
	}
	if !foundNil {
		t.Error("blockerSubsets must include the no-blocks option")
	}
	if !foundWallBlock {
		t.Error("blockerSubsets must include singleton: Wall blocks Giant")
	}
	if !foundBearBlock {
		t.Error("blockerSubsets must include singleton: Bear blocks Giant")
	}
}

// ── Property: searchAs of active player is symmetric with searchAs of opponent ─

// In simple-minimax, the eval is zero-sum: evaluate(g, A) == -evaluate(g, B)
// for the deterministic terms (board, mana, life pressure). The incoming-
// damage penalty is symmetric (cancels out), so the full evaluate should be
// strictly anti-symmetric on positions where IsGameOver is false.
func TestParity_Evaluate_AntiSymmetric(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(17)
	pb.SetLife(11)

	c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	c2 := makePerm("Ogre", "{2}{R}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(c1, c2)
	addLands(g, pa, "Forest", 3)
	addLands(g, pb, "Mountain", 2)

	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	sA := evaluate(g, pa.PlayerID())
	sB := evaluate(g, pb.PlayerID())

	if math.Abs(sA+sB) > 1e-9 {
		t.Fatalf("evaluate not anti-symmetric: A=%v B=%v sum=%v", sA, sB, sA+sB)
	}
}

// ── Property: TT-on vs TT-off return the same Score for canonical positions ─

// Simple-minimax's TT must never change the search result — only its
// performance. If TT-on and TT-off disagree, the TT key is missing a state
// dimension or its bounds (alpha/beta) are stored wrong. Chain-move identity
// is not asserted: bounded TT entries can change which equal-score sibling
// becomes "first" once alpha tightens. Score must agree exactly.
func TestParity_TTOnOff_ConsistentScore_Turn17(t *testing.T) {
	g, aliceID := turn17UnholyStrengthGame(t)
	off := searchActiveRootForTest(g, aliceID, false)
	on := searchActiveRootForTest(g, aliceID, true)

	if off.Score != on.Score {
		t.Fatalf("TT mismatch: off=%.6f on=%.6f", off.Score, on.Score)
	}
	if on.TTStores == 0 {
		t.Errorf("TT enabled but no stores recorded — TT path not exercised")
	}
}

// ── Property: DeclareBlockers short-circuit doesn't change the answer ───────

// mage-go's searchAsWithOptions routes a defender-asking DeclareBlockers query
// directly to searchActiveRoot, skipping searchInstantResponse. Simple-minimax
// always routes non-active root through searchInstantResponse, which then
// recurses into searchActiveRoot (negating once). The two routes must produce
// identical chains. Build a position where it matters: defender chooses
// blockers, no instant-speed responses.
func TestParity_DeclareBlockers_RouteEquivalence(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{3}{R}", 3, 3, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())
	g.SetStep(core.DeclareBlockers)

	// Route A: the production short-circuit — directly into searchActiveRoot.
	short := searchAsWithOptions(g, pb.PlayerID(), nil, false, DefaultZobrist)

	// Route B: simple-minimax's path — searchInstantResponse → searchActiveRoot.
	// We mimic it directly: defender has only Pass at this priority window
	// (no instants/activated abilities in this position), so a one-iteration
	// instant-response collapses to running searchActiveRoot on a clone and
	// negating.
	clone := g.Clone()
	deep := searchActiveRoot(clone, clone.ActivePlayerObj().PlayerID(), nil, false, DefaultZobrist)

	// Pull the chosen block out of both routes.
	pullBlocks := func(r Result) []mage.BlockAssignment {
		for _, c := range r.Chain {
			if c.Blocks != nil {
				return c.Blocks
			}
		}
		return nil
	}
	a, b := pullBlocks(short), pullBlocks(deep)
	if len(a) != len(b) {
		t.Fatalf("route divergence: short=%v deep=%v", a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("route divergence at i=%d: short=%v deep=%v", i, a[i], b[i])
		}
	}

	// And the score from defender's perspective: short returns from defender's
	// view; deep returns from active (attacker) view, so negate.
	if math.Abs(short.Score-(-deep.Score)) > 1e-9 {
		t.Fatalf("route score mismatch: short=%.6f want=-deep=%.6f", short.Score, -deep.Score)
	}
}

// ── Smoke: a debug-print helper used when failures need investigating ───────

// dumpChain is unused by passing tests; failing-test debugging may call it
// from the test body. Keep it linked so we don't get an unused-function lint.
func dumpChain(g *mage.Game, chain []ChainStep) string {
	if len(chain) == 0 {
		return "(empty)"
	}
	parts := make([]string, len(chain))
	for i, c := range chain {
		parts[i] = fmt.Sprintf("%d. %s", i+1, c.String())
	}
	_ = g
	return strings.Join(parts, " | ")
}

var _ = dumpChain
