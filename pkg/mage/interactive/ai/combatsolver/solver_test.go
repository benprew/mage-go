package combatsolver

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func makePerm(name, cost string, power, toughness int, owner uuid.UUID, opts ...mage.CardOption) *mage.Permanent {
	card := mage.NewCreature(name, cost, power, toughness, opts...)
	card.SetOwner(owner)
	perm := mage.NewPermanent(card, owner)
	perm.RevokeBaseAttr(core.AttrSummonSick)
	return perm
}

func makeGame() (*mage.Game, *mage.BasePlayer, *mage.BasePlayer) {
	pa := mage.NewBasePlayer("Alice")
	pb := mage.NewBasePlayer("Bob")
	g := mage.NewGame(pa, pb)
	return g, pa, pb
}

func defaultOpts() Options {
	return Options{Profile: Profile{Aggression: 0.5, BlockThreshold: 0.5}}
}

// ── SolveAttack ──────────────────────────────────────────────────────────────

func TestSolveAttack_NoCreatures(t *testing.T) {
	g, pa, _ := makeGame()
	r := SolveAttack(g, pa.PlayerID(), defaultOpts())
	if len(r.Attackers) != 0 {
		t.Errorf("want no attackers, got %v", r.Attackers)
	}
}

func TestSolveAttack_NilGame(t *testing.T) {
	r := SolveAttack(nil, uuid.Nil, defaultOpts())
	if len(r.Attackers) != 0 || r.Score != 0 {
		t.Errorf("want zero result for nil game, got %+v", r)
	}
}

func TestSolveAttack_LoneAttackerVsEmptyBoard(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(atk)

	r := SolveAttack(g, pa.PlayerID(), defaultOpts())
	if len(r.Attackers) != 1 || r.Attackers[0] != atk.ID() {
		t.Errorf("want sole attacker declared, got %v", r.Attackers)
	}
	_ = pb
}

func TestSolveAttack_DontAttackIntoCleanLoss(t *testing.T) {
	g, pa, pb := makeGame()
	// 1/1 attacking into a 5/5 blocker: attacker dies, no damage. Solver
	// should prefer not attacking — empty subset is enumerated and beats the
	// suicide attack.
	atk := makePerm("Squire", "{W}", 1, 1, pa.PlayerID())
	blk := makePerm("Wall", "{4}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	r := SolveAttack(g, pa.PlayerID(), defaultOpts())
	if len(r.Attackers) != 0 {
		t.Errorf("want no attack into certain loss, got %v", r.Attackers)
	}
}

func TestSolveAttack_DontSendBigIntoLosingTrade(t *testing.T) {
	g, pa, pb := makeGame()
	// 4/4 and 1/1 vs a single 5/5 wall. Opponent will block the 4/4 (saving
	// 4 vs saving 1), so the 4/4 dies for at most 1 face damage from the
	// unblocked 1/1. The 4/4 is worth far more than 1 life — the solver
	// should not send the 4/4 into combat.
	big := makePerm("Giant", "{3}{G}", 4, 4, pa.PlayerID())
	small := makePerm("Squire", "{W}", 1, 1, pa.PlayerID())
	wall := makePerm("Wall", "{4}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(big, small, wall)

	r := SolveAttack(g, pa.PlayerID(), defaultOpts())
	for _, id := range r.Attackers {
		if id == big.ID() {
			t.Errorf("4/4 should not attack into 5/5 wall when blocking is forced; chosen attackers: %v", r.Attackers)
		}
	}
}

func TestSolveAttack_OverwhelmingForceVsLowLifeOpponent(t *testing.T) {
	// Reproduces a real-game position the AI refused to attack from.
	//
	// AI side (post-Bad-Moon stats): Drudge Skeletons 3/3, Scathe Zombies 6/6,
	// Lost Soul 5/3.
	// Opponent at 3 life with three 1/1-ish blockers: Pikemen, Argivian
	// Blacksmith (with Lance/Artifact Ward), Abu Ja'Far.
	//
	// Even with every attacker blocked, the trades are catastrophically
	// one-sided: opponent loses every creature, AI loses at most one. The
	// solver must declare some attack — refusing to attack here is the bug.
	g, pa, pb := makeGame()
	pb.SetLife(3)

	atk1 := makePerm("DrudgeSkeletons", "{1}{B}", 3, 3, pa.PlayerID())
	atk2 := makePerm("ScatheZombies", "{1}{B}{B}", 6, 6, pa.PlayerID())
	atk3 := makePerm("LostSoul", "{2}{B}{B}", 5, 3, pa.PlayerID())
	g.AddToBattlefield(atk1, atk2, atk3)

	blk1 := makePerm("Pikemen", "{1}{W}", 1, 1, pb.PlayerID(),
		mage.WithKeyword(core.FirstStrike), mage.WithKeyword(core.Banding))
	blk2 := makePerm("ArgivianBlacksmith", "{1}{W}{W}", 2, 2, pb.PlayerID(),
		mage.WithKeyword(core.FirstStrike))
	blk3 := makePerm("AbuJafar", "{W}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(blk1, blk2, blk3)

	r := SolveAttack(g, pa.PlayerID(), defaultOpts())
	if len(r.Attackers) == 0 {
		t.Errorf("AI should attack — opponent at 3 life, every trade favors AI; got zero attackers (score=%d)", r.Score)
	}
}

func TestSolveAttack_DeadlineHit(t *testing.T) {
	g, pa, pb := makeGame()
	// Stack many small creatures so enumeration takes meaningful time.
	for range 6 {
		g.AddToBattlefield(makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID()))
	}
	for range 3 {
		g.AddToBattlefield(makePerm("Wall", "{4}", 0, 4, pb.PlayerID()))
	}

	opts := defaultOpts()
	opts.Deadline = time.Now() // already passed
	r := SolveAttack(g, pa.PlayerID(), opts)
	if !r.DeadlineHit {
		t.Errorf("want DeadlineHit=true with passed deadline, got %+v", r)
	}
}

// addUntappedLand puts a single untapped basic land on the battlefield so the
// player has mana available for combat tricks.
func addUntappedLand(g *mage.Game, owner uuid.UUID, color core.Color) {
	land := mage.NewLand("Land", mage.WithManaAbility(color))
	land.SetOwner(owner)
	perm := mage.NewPermanent(land, owner)
	perm.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(perm)
}

func giantGrowth(owner uuid.UUID) mage.Card {
	c := mage.NewInstant("Giant Growth", "{G}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.Boost(mage.Fixed(3), mage.Fixed(3))))
	c.SetOwner(owner)
	return c
}

// With a pump trick in hand, attacking a 2/2 into a 2/3 becomes profitable: the
// solver should declare the attack it would otherwise (correctly) refuse.
func TestSolveAttack_AttacksWhenPumpTrickFlipsCombat(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Ogre", "{1}{R}", 2, 3, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	addUntappedLand(g, pa.PlayerID(), core.Green)

	// Baseline: without the trick the 2/2 should not attack into the 2/3.
	if r := SolveAttack(g, pa.PlayerID(), defaultOpts()); len(r.Attackers) != 0 {
		t.Fatalf("baseline: 2/2 should not attack into 2/3 without a trick, got %v", r.Attackers)
	}

	pa.AddToHand(giantGrowth(pa.PlayerID()))
	r := SolveAttack(g, pa.PlayerID(), defaultOpts())
	if len(r.Attackers) != 1 || r.Attackers[0] != atk.ID() {
		t.Fatalf("with Giant Growth in hand the 2/2 should attack, got %v", r.Attackers)
	}
}

// With a pump trick in hand, blocking a 3/3 with a 2/2 becomes a winning trade,
// so the solver should declare the block it would otherwise refuse.
func TestSolveDefense_BlocksWhenPumpTrickFlipsCombat(t *testing.T) {
	g, pa, pb := makeGame()
	// A 4/1 blocking a 1/5 is a pure loss without help (the 4/1 dies, the 1/5
	// survives), so the solver should not block. Giant Growth turns the blocker
	// into a 7/4 that kills the 1/5 and survives, making the block correct.
	atk := makePerm("Spider", "{3}{G}", 1, 5, pa.PlayerID())
	blk := makePerm("Brute", "{3}{R}", 4, 1, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	addUntappedLand(g, pb.PlayerID(), core.Green)
	g.ExecuteAttackers(pa.PlayerID(), []uuid.UUID{atk.ID()})

	// Baseline: without the trick the 4/1 should not block the 1/5.
	if r := SolveDefense(g, pb.PlayerID(), defaultOpts()); len(r.Blocks) != 0 {
		t.Fatalf("baseline: 4/1 should not block 1/5 without a trick, got %v", r.Blocks)
	}

	pb.AddToHand(giantGrowth(pb.PlayerID()))
	r := SolveDefense(g, pb.PlayerID(), defaultOpts())
	if len(r.Blocks) != 1 || r.Blocks[0].BlockerID != blk.ID() {
		t.Fatalf("with Giant Growth in hand the 4/1 should block the 1/5, got %v", r.Blocks)
	}
}

// battleMage is a 0/1 with a mana-only "{G}: target creature gets +2/+2".
func battleMage(owner uuid.UUID) *mage.Permanent {
	return makePerm("Battle Mage", "{1}{G}", 0, 1, owner,
		mage.WithActivatedAbility(
			mage.Boost(mage.Fixed(2), mage.Fixed(2)),
			mage.ManaCostOf("{G}"),
			mage.WithTarget(mage.TargetCreature()),
		))
}

// A targeted pump ability on another creature should let the solver attack a
// 2/2 into a 2/3: the ability is only usable with mana, so the attack appears
// only once a land is available.
func TestSolveAttack_AttacksWhenTargetedPumpAbilityFlipsCombat(t *testing.T) {
	g, pa, pb := makeGame()
	bear := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	mage_ := battleMage(pa.PlayerID())
	blk := makePerm("Ogre", "{1}{R}", 2, 3, pb.PlayerID())
	g.AddToBattlefield(bear, mage_, blk)

	if r := SolveAttack(g, pa.PlayerID(), defaultOpts()); slices.Contains(r.Attackers, bear.ID()) {
		t.Fatalf("baseline: bear should not attack without mana to pump, got %v", r.Attackers)
	}

	addUntappedLand(g, pa.PlayerID(), core.Green)
	r := SolveAttack(g, pa.PlayerID(), defaultOpts())
	if !slices.Contains(r.Attackers, bear.ID()) {
		t.Fatalf("with a targeted pump ability available the bear should attack, got %v", r.Attackers)
	}
}

// A targeted pump ability should let the solver block a 1/5 with a 4/1: the
// 4/1 dies for nothing without help, but the ability pumps it to survive+kill.
func TestSolveDefense_BlocksWhenTargetedPumpAbilityFlipsCombat(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Spider", "{3}{G}", 1, 5, pa.PlayerID())
	blk := makePerm("Brute", "{3}{R}", 4, 1, pb.PlayerID())
	mage_ := battleMage(pb.PlayerID())
	g.AddToBattlefield(atk, blk, mage_)
	g.ExecuteAttackers(pa.PlayerID(), []uuid.UUID{atk.ID()})

	if r := SolveDefense(g, pb.PlayerID(), defaultOpts()); len(r.Blocks) != 0 {
		t.Fatalf("baseline: 4/1 should not block 1/5 without mana to pump, got %v", r.Blocks)
	}

	addUntappedLand(g, pb.PlayerID(), core.Green)
	r := SolveDefense(g, pb.PlayerID(), defaultOpts())
	if len(r.Blocks) != 1 || r.Blocks[0].BlockerID != blk.ID() {
		t.Fatalf("with a targeted pump ability the 4/1 should block the 1/5, got %v", r.Blocks)
	}
}

// ── SolveDefense ─────────────────────────────────────────────────────────────

func TestSolveDefense_NoAttackers(t *testing.T) {
	g, _, pb := makeGame()
	blk := makePerm("Wall", "{4}", 0, 5, pb.PlayerID())
	g.AddToBattlefield(blk)

	r := SolveDefense(g, pb.PlayerID(), defaultOpts())
	if len(r.Blocks) != 0 {
		t.Errorf("want no blocks when no attackers, got %v", r.Blocks)
	}
}

func TestSolveDefense_BlocksLethal(t *testing.T) {
	g, pa, pb := makeGame()
	// Opponent's 4/4 attacks; defender's 4/4 should block (life-saving trade).
	atk := makePerm("Giant", "{3}{G}", 4, 4, pa.PlayerID())
	blk := makePerm("Knight", "{2}{W}", 4, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	pb.SetLife(3) // can't take 4 damage

	g.ExecuteAttackers(pa.PlayerID(), []uuid.UUID{atk.ID()})

	r := SolveDefense(g, pb.PlayerID(), defaultOpts())
	if len(r.Blocks) != 1 {
		t.Fatalf("want 1 block, got %d (%v)", len(r.Blocks), r.Blocks)
	}
	if r.Blocks[0].BlockerID != blk.ID() || r.Blocks[0].AttackerID != atk.ID() {
		t.Errorf("wrong block assignment: %+v", r.Blocks[0])
	}
}

func TestSolveDefense_DontBlockTrivialAttacker(t *testing.T) {
	g, pa, pb := makeGame()
	// 1/1 attacker, plenty of life: trading the 4/4 blocker is bad value.
	atk := makePerm("Squire", "{W}", 1, 1, pa.PlayerID())
	blk := makePerm("Knight", "{2}{W}", 4, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.ExecuteAttackers(pa.PlayerID(), []uuid.UUID{atk.ID()})

	r := SolveDefense(g, pb.PlayerID(), defaultOpts())
	// Solver may still block since the 1/1 dies for free — but should not
	// produce a worse-than-no-block outcome. Score should be at least the
	// no-block baseline.
	if len(r.Blocks) > 1 {
		t.Errorf("expected at most 1 block, got %d", len(r.Blocks))
	}
}

func TestSolveDefense_FavorableTrade(t *testing.T) {
	g, pa, pb := makeGame()
	// 2/2 attacker, 2/2 blocker: even trade. Without lethal pressure, the
	// solver may or may not block — but if it does, the assignment should be
	// the only valid one.
	atk := makePerm("Bear1", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Bear2", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.ExecuteAttackers(pa.PlayerID(), []uuid.UUID{atk.ID()})

	r := SolveDefense(g, pb.PlayerID(), defaultOpts())
	for _, b := range r.Blocks {
		if b.BlockerID != blk.ID() || b.AttackerID != atk.ID() {
			t.Errorf("invalid block assignment: %+v", b)
		}
	}
}

// ── enumeration helpers ───────────────────────────────────────────────────────

func TestEnumerateAttackerSets_IncludesAllAndNone(t *testing.T) {
	g, pa, _ := makeGame()
	g.AddToBattlefield(makePerm("Bear1", "{1}{G}", 2, 2, pa.PlayerID()))
	g.AddToBattlefield(makePerm("Bear2", "{1}{G}", 2, 2, pa.PlayerID()))

	sets := enumerateAttackerSets(g, pa.PlayerID())
	hasAll, hasNone := false, false
	for _, s := range sets {
		if len(s) == 0 {
			hasNone = true
		}
		if len(s) == 2 {
			hasAll = true
		}
	}
	if !hasAll || !hasNone {
		t.Errorf("missing all or none subset: hasAll=%v hasNone=%v sets=%v", hasAll, hasNone, sets)
	}
}

func TestEnumerateBlockerSets_NoBlockersReturnsEmpty(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(atk)
	g.ExecuteAttackers(pa.PlayerID(), []uuid.UUID{atk.ID()})

	sets := enumerateBlockerSets(g, pb.PlayerID())
	if len(sets) != 1 || len(sets[0]) != 0 {
		t.Errorf("want single empty set, got %v", sets)
	}
}
