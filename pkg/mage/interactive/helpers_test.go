package interactive

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ── getEligibleAttackers ────────────────────────────────────────────────────

func TestGetEligibleAttackers_SkipsOpponent(t *testing.T) {
	g, pa, pb := makeGame()
	own := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	opp := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(own, opp)

	eligible := getEligibleAttackers(g, pa.PlayerID())
	if len(eligible) != 1 || eligible[0].ID() != own.ID() {
		t.Errorf("should only include own creatures, got %d", len(eligible))
	}
}

func TestGetEligibleAttackers_SkipsTapped(t *testing.T) {
	g, pa, _ := makeGame()
	untapped := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	tapped := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	tapped.Tapped = true
	g.AddToBattlefield(untapped, tapped)

	eligible := getEligibleAttackers(g, pa.PlayerID())
	if len(eligible) != 1 {
		t.Errorf("should skip tapped creature, got %d", len(eligible))
	}
}

func TestGetEligibleAttackers_SkipsSummonSick(t *testing.T) {
	g, pa, _ := makeGame()
	ready := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	sick := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	sick.GrantBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(ready, sick)

	eligible := getEligibleAttackers(g, pa.PlayerID())
	if len(eligible) != 1 {
		t.Errorf("should skip summoning-sick creature, got %d", len(eligible))
	}
}

func TestGetEligibleAttackers_Empty(t *testing.T) {
	g, pa, _ := makeGame()
	eligible := getEligibleAttackers(g, pa.PlayerID())
	if len(eligible) != 0 {
		t.Errorf("expected no eligible attackers, got %d", len(eligible))
	}
}

// ── getEligibleBlockers ─────────────────────────────────────────────────────

func TestGetEligibleBlockers_SkipsTapped(t *testing.T) {
	g, _, pb := makeGame()
	untapped := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	tapped := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	tapped.Tapped = true
	g.AddToBattlefield(untapped, tapped)

	eligible := getEligibleBlockers(g, pb.PlayerID())
	if len(eligible) != 1 {
		t.Errorf("should skip tapped creature, got %d", len(eligible))
	}
}

func TestGetEligibleBlockers_IncludesUntapped(t *testing.T) {
	g, _, pb := makeGame()
	c1 := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	c2 := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(c1, c2)

	eligible := getEligibleBlockers(g, pb.PlayerID())
	if len(eligible) != 2 {
		t.Errorf("should include both untapped creatures, got %d", len(eligible))
	}
}

// ── resolveTargetName ───────────────────────────────────────────────────────

func TestResolveTargetName_Permanent(t *testing.T) {
	g, pa, _ := makeGame()
	perm := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(perm)

	if got := resolveTargetName(g, perm.ID()); got != "Bear" {
		t.Errorf("resolveTargetName(permanent) = %q, want %q", got, "Bear")
	}
}

func TestResolveTargetName_Player(t *testing.T) {
	g, pa, _ := makeGame()
	if got := resolveTargetName(g, pa.PlayerID()); got != "Alice" {
		t.Errorf("resolveTargetName(player) = %q, want %q", got, "Alice")
	}
}

func TestResolveTargetName_Unknown(t *testing.T) {
	g, _, _ := makeGame()
	if got := resolveTargetName(g, uuid.New()); got != "unknown" {
		t.Errorf("resolveTargetName(unknown) = %q, want %q", got, "unknown")
	}
}

// ── targetSuffix ────────────────────────────────────────────────────────────

func TestTargetSuffix_Empty(t *testing.T) {
	g, _, _ := makeGame()
	if got := targetSuffix(g, nil); got != "" {
		t.Errorf("targetSuffix(nil) = %q, want empty", got)
	}
}

func TestTargetSuffix_Single(t *testing.T) {
	g, pa, _ := makeGame()
	got := targetSuffix(g, []uuid.UUID{pa.PlayerID()})
	if got != " targeting Alice" {
		t.Errorf("targetSuffix(single) = %q, want %q", got, " targeting Alice")
	}
}

func TestTargetSuffix_Multiple(t *testing.T) {
	g, pa, pb := makeGame()
	got := targetSuffix(g, []uuid.UUID{pa.PlayerID(), pb.PlayerID()})
	if got != " targeting Alice, Bob" {
		t.Errorf("targetSuffix(multiple) = %q, want %q", got, " targeting Alice, Bob")
	}
}

// ── findPlayerIndex ─────────────────────────────────────────────────────────

func TestFindPlayerIndex_First(t *testing.T) {
	g, pa, _ := makeGame()
	if got := findPlayerIndex(g, pa.PlayerID()); got != 0 {
		t.Errorf("findPlayerIndex(first) = %d, want 0", got)
	}
}

func TestFindPlayerIndex_Second(t *testing.T) {
	g, _, pb := makeGame()
	if got := findPlayerIndex(g, pb.PlayerID()); got != 1 {
		t.Errorf("findPlayerIndex(second) = %d, want 1", got)
	}
}

func TestFindPlayerIndex_Unknown(t *testing.T) {
	g, _, _ := makeGame()
	if got := findPlayerIndex(g, uuid.New()); got != 0 {
		t.Errorf("findPlayerIndex(unknown) = %d, want 0 (default)", got)
	}
}

// ── captureForUndo + restoreFromUndo ────────────────────────────────────────

func TestCaptureAndRestoreUndo(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	snap := captureForUndo(g, pa.PlayerID(), 0)
	if !snap.valid {
		t.Fatal("snapshot should be valid")
	}

	// Mutate the game state
	pa.SetHand(nil)
	g.SetLandsPlayedThisTurn(5)

	// Restore
	restoreFromUndo(g, pa.PlayerID(), snap)
	if len(pa.Hand()) != 1 {
		t.Errorf("hand should be restored to 1 card, got %d", len(pa.Hand()))
	}
	if g.GetLandsPlayedThisTurn() != 0 {
		t.Errorf("landsPlayed should be restored to 0, got %d", g.GetLandsPlayedThisTurn())
	}
}

func TestRestoreFromUndo_InvalidSnapshot(t *testing.T) {
	g, pa, _ := makeGame()
	pa.SetLife(15)
	// Invalid snapshot should be a no-op
	restoreFromUndo(g, pa.PlayerID(), undoSnapshot{valid: false})
	if pa.Life() != 15 {
		t.Error("invalid snapshot should not modify game state")
	}
}

func TestCaptureForUndo_NilPlayer(t *testing.T) {
	g, _, _ := makeGame()
	snap := captureForUndo(g, uuid.New(), 0)
	if snap.valid {
		t.Error("snapshot for unknown player should be invalid")
	}
}

func TestCaptureUndo_TappedStatePreserved(t *testing.T) {
	g, pa, _ := makeGame()
	perm := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(perm)

	snap := captureForUndo(g, pa.PlayerID(), 0)

	// Tap the permanent
	perm.Tapped = true

	// Restore
	restoreFromUndo(g, pa.PlayerID(), snap)
	if perm.Tapped {
		t.Error("tapped state should be restored to untapped")
	}
}

// ── attackerOptions / blockerOptions ────────────────────────────────────────

func TestAttackerOptions_Format(t *testing.T) {
	perm := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	opts := attackerOptions([]*mage.Permanent{perm})
	if len(opts) != 1 {
		t.Fatalf("expected 1 option, got %d", len(opts))
	}
	if opts[0].Label != "Bear 2/2" {
		t.Errorf("label = %q, want %q", opts[0].Label, "Bear 2/2")
	}
	if opts[0].Type != ActionSelectAttackers {
		t.Errorf("type = %v, want ActionSelectAttackers", opts[0].Type)
	}
}

func TestBlockerOptions_DoneOption(t *testing.T) {
	g, pa, _ := makeGame()
	perm := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	opts := blockerOptions(g, pa.PlayerID(), []*mage.Permanent{perm})
	if len(opts) != 2 {
		t.Fatalf("expected 2 options (1 creature + done), got %d", len(opts))
	}
	last := opts[len(opts)-1]
	if last.Type != ActionPass {
		t.Errorf("last option type = %v, want ActionPass", last.Type)
	}
	if last.Label != "Done (confirm blocks)" {
		t.Errorf("last option label = %q, want %q", last.Label, "Done (confirm blocks)")
	}
}

func TestBlockerOptions_EmptyStillHasDone(t *testing.T) {
	g, pa, _ := makeGame()
	opts := blockerOptions(g, pa.PlayerID(), nil)
	if len(opts) != 1 {
		t.Fatalf("expected 1 option (done only), got %d", len(opts))
	}
	if opts[0].Label != "Done (confirm blocks)" {
		t.Errorf("label = %q", opts[0].Label)
	}
}

// ── GetAvailableActions ─────────────────────────────────────────────────────

func TestGetAvailableActions_MainPhaseIncludesLands(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	pa.AddToHand(land)

	actions := GetAvailableActions(g, pa.PlayerID())
	foundLand := false
	for _, a := range actions {
		if a.Type == ActionPlayLand {
			foundLand = true
		}
	}
	if !foundLand {
		t.Error("main phase with land in hand should include PlayLand action")
	}
}

func TestGetAvailableActions_NoLandIfAlreadyPlayed(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	pa.AddToHand(land)
	g.SetLandsPlayedThisTurn(1)

	actions := GetAvailableActions(g, pa.PlayerID())
	for _, a := range actions {
		if a.Type == ActionPlayLand {
			t.Error("should not offer land play when already played one")
		}
	}
}

func TestGetAvailableActions_NoLandForNonActivePlayer(t *testing.T) {
	g, _, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	land := mage.NewLand("Forest")
	land.SetOwner(pb.PlayerID())
	pb.AddToHand(land)

	actions := GetAvailableActions(g, pb.PlayerID())
	for _, a := range actions {
		if a.Type == ActionPlayLand {
			t.Error("non-active player should not be offered land plays")
		}
	}
}

func TestGetAvailableActions_AlwaysIncludesPass(t *testing.T) {
	g, pa, _ := makeGame()
	actions := GetAvailableActions(g, pa.PlayerID())
	foundPass := false
	for _, a := range actions {
		if a.Type == ActionPass {
			foundPass = true
		}
	}
	if !foundPass {
		t.Error("available actions should always include Pass")
	}
}

func TestGetAvailableActions_NonMainOnlyInstants(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.DeclareAttackers)
	sorcery := mage.NewSorcery("Divination", "{2}{U}", mage.NewSpellAbility(mage.DrawCards(mage.Fixed(2))))
	sorcery.SetOwner(pa.PlayerID())
	pa.AddToHand(sorcery)

	actions := GetAvailableActions(g, pa.PlayerID())
	for _, a := range actions {
		if a.Type == ActionCastSpell {
			t.Error("non-main phase should not offer sorcery-speed spells")
		}
	}
}

// ── SnapshotGameState deeper coverage ───────────────────────────────────────

// ── OnDamageDealt callback ──────────────────────────────────────────────────

func TestOnDamageDealt_PlayerDamage(t *testing.T) {
	g, pa, pb := makeGame()
	perm := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(perm)

	var logged []string
	g.SetOnDamageDealt(func(sourceName, targetName string, amount int, isCombat bool) {
		logged = append(logged, fmt.Sprintf("%s deals %d damage to %s (combat=%v)", sourceName, amount, targetName, isCombat))
	})

	g.DealDamageToPlayer(pb, 2, perm.ID())

	if len(logged) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logged))
	}
	if !strings.Contains(logged[0], "Bear") {
		t.Errorf("log should mention source name Bear: %q", logged[0])
	}
	if !strings.Contains(logged[0], "Bob") {
		t.Errorf("log should mention target name Bob: %q", logged[0])
	}
	if !strings.Contains(logged[0], "2") {
		t.Errorf("log should mention damage amount 2: %q", logged[0])
	}
}

func TestOnDamageDealt_CreatureDamage(t *testing.T) {
	g, pa, pb := makeGame()
	attacker := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blocker := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(attacker, blocker)

	var logged []string
	g.SetOnDamageDealt(func(sourceName, targetName string, amount int, isCombat bool) {
		logged = append(logged, fmt.Sprintf("%s deals %d to %s", sourceName, amount, targetName))
	})

	g.DealDamageToPermanent(blocker, 2, attacker.ID())

	if len(logged) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logged))
	}
	if !strings.Contains(logged[0], "Bear") {
		t.Errorf("log should mention source Bear: %q", logged[0])
	}
	if !strings.Contains(logged[0], "Elf") {
		t.Errorf("log should mention target Elf: %q", logged[0])
	}
}

func TestSnapshotGameState_BattlefieldCreatures(t *testing.T) {
	g, pa, _ := makeGame()
	perm := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(perm)

	snap := SnapshotGameState(g, 0)
	if len(snap.You.Battlefield) != 1 {
		t.Fatalf("expected 1 battlefield permanent, got %d", len(snap.You.Battlefield))
	}
	ps := snap.You.Battlefield[0]
	if ps.Name != "Bear" {
		t.Errorf("permanent name = %q, want Bear", ps.Name)
	}
	if !ps.IsCreature {
		t.Error("permanent should be marked as creature")
	}
	if ps.Power != 2 || ps.Toughness != 2 {
		t.Errorf("P/T = %d/%d, want 2/2", ps.Power, ps.Toughness)
	}
}

func TestSnapshotGameState_HandCards(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewInstant("Lightning Bolt", "{R}", mage.NewSpellAbility(mage.DealDamage(mage.Fixed(3))))
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	snap := SnapshotGameState(g, 0)
	if len(snap.You.Hand) != 1 {
		t.Fatalf("expected 1 hand card, got %d", len(snap.You.Hand))
	}
	cs := snap.You.Hand[0]
	if cs.Name != "Lightning Bolt" {
		t.Errorf("card name = %q, want Lightning Bolt", cs.Name)
	}
}

func TestSnapshotGameState_Graveyard(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	card.SetOwner(pa.PlayerID())
	pa.AddToGraveyard(card)

	snap := SnapshotGameState(g, 0)
	if snap.You.GraveyardCount != 1 {
		t.Errorf("graveyard count = %d, want 1", snap.You.GraveyardCount)
	}
	if len(snap.You.Graveyard) != 1 {
		t.Fatalf("expected 1 graveyard card, got %d", len(snap.You.Graveyard))
	}
	if snap.You.Graveyard[0].Name != "Bear" {
		t.Errorf("graveyard card name = %q, want Bear", snap.You.Graveyard[0].Name)
	}
}

func TestSnapshotGameState_TurnAndStep(t *testing.T) {
	g, _, _ := makeGame()
	g.SetTurn(5)
	snap := SnapshotGameState(g, 0)
	if snap.Turn != 5 {
		t.Errorf("turn = %d, want 5", snap.Turn)
	}
	if snap.Step == "" {
		t.Error("step should not be empty")
	}
}

func TestSnapshotGameState_OpponentHandHidden(t *testing.T) {
	g, _, pb := makeGame()
	card := mage.NewCreature("Secret", "{1}", 1, 1)
	card.SetOwner(pb.PlayerID())
	pb.AddToHand(card)

	snap := SnapshotGameState(g, 0)
	// Opponent hand should not be shown (showHand=false)
	if len(snap.Opponent.Hand) != 0 {
		t.Error("opponent hand should be hidden")
	}
	if snap.Opponent.HandCount != 1 {
		t.Errorf("opponent hand count = %d, want 1", snap.Opponent.HandCount)
	}
}

func TestSnapshotGameState_PermanentCounters(t *testing.T) {
	g, pa, _ := makeGame()
	perm := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	perm.AddCounter(core.P1P1, 3)
	g.AddToBattlefield(perm)

	snap := SnapshotGameState(g, 0)
	ps := snap.You.Battlefield[0]
	if ps.Counters == nil {
		t.Fatal("counters should not be nil")
	}
	if ps.Counters["+1/+1"] != 3 {
		t.Errorf("counter count = %d, want 3", ps.Counters["+1/+1"])
	}
}

func TestSnapshotGameState_PermanentKeywords(t *testing.T) {
	g, pa, _ := makeGame()
	perm := makePerm("Bird", "{1}{W}", 1, 1, pa.PlayerID(), mage.WithKeyword(core.Flying))
	g.AddToBattlefield(perm)

	snap := SnapshotGameState(g, 0)
	ps := snap.You.Battlefield[0]
	found := false
	for _, kw := range ps.Keywords {
		if strings.EqualFold(kw, "Flying") {
			found = true
		}
	}
	if !found {
		t.Errorf("keywords = %v, expected Flying", ps.Keywords)
	}
}

func TestSnapshotGameState_ManaPool(t *testing.T) {
	g, pa, _ := makeGame()
	pa.ManaPool().Add(core.Red, 3)

	snap := SnapshotGameState(g, 0)
	if snap.You.ManaPool.Red != 3 {
		t.Errorf("mana pool red = %d, want 3", snap.You.ManaPool.Red)
	}
}

func TestSnapshotGameState_LibraryCount(t *testing.T) {
	g, pa, _ := makeGame()
	c1 := mage.NewCreature("A", "{1}", 1, 1)
	c2 := mage.NewCreature("B", "{1}", 1, 1)
	pa.AddToLibrary(c1)
	pa.AddToLibrary(c2)

	snap := SnapshotGameState(g, 0)
	if snap.You.LibraryCount != 2 {
		t.Errorf("library count = %d, want 2", snap.You.LibraryCount)
	}
}

// ── SnapshotGameState exile zone ────────────────────────────────────────────

func TestSnapshotGameState_ExileFaceUpVisibleToBoth(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	card.SetOwner(pa.PlayerID())
	g.ExileCard(card, uuid.Nil)

	for _, idx := range []int{0, 1} {
		snap := SnapshotGameState(g, idx)
		owner := snap.You
		if idx == 1 {
			owner = snap.Opponent
		}
		if len(owner.Exile) != 1 {
			t.Fatalf("viewer=%d: owner exile len = %d, want 1", idx, len(owner.Exile))
		}
		if owner.Exile[0].Name != "Bear" {
			t.Errorf("viewer=%d: exile name = %q, want Bear", idx, owner.Exile[0].Name)
		}
		if owner.Exile[0].FaceDown {
			t.Errorf("viewer=%d: face-up card reported FaceDown", idx)
		}
	}
}

func TestSnapshotGameState_ExileFaceDownOwnerSeesIdentity(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewCreature("Secret", "{1}{B}", 3, 3)
	card.SetOwner(pa.PlayerID())
	// Exiled face down by pb (Gonti-style); only pb may inspect identity.
	g.ExileCardFaceDown(card, uuid.Nil, pb.PlayerID())

	// From pa's perspective (the owner): pa is NOT in RevealedTo, so
	// pa cannot see the card identity even though pa owns the card.
	snapA := SnapshotGameState(g, 0)
	if len(snapA.You.Exile) != 1 {
		t.Fatalf("owner exile len = %d, want 1", len(snapA.You.Exile))
	}
	if snapA.You.Exile[0].Name != "" {
		t.Errorf("owner saw face-down identity = %q, want redacted", snapA.You.Exile[0].Name)
	}
	if !snapA.You.Exile[0].FaceDown {
		t.Errorf("owner snapshot did not mark FaceDown")
	}
	if snapA.You.Exile[0].ID == uuid.Nil {
		t.Errorf("owner snapshot dropped tracking ID")
	}

	// From pb's perspective: pa is the opponent (still owns the card),
	// and pb is in RevealedTo, so pb sees Secret in pa's exile.
	snapB := SnapshotGameState(g, 1)
	if len(snapB.Opponent.Exile) != 1 {
		t.Fatalf("pb view of pa exile len = %d, want 1", len(snapB.Opponent.Exile))
	}
	if snapB.Opponent.Exile[0].Name != "Secret" {
		t.Errorf("revealedTo viewer name = %q, want Secret", snapB.Opponent.Exile[0].Name)
	}
	if !snapB.Opponent.Exile[0].FaceDown {
		t.Errorf("FaceDown bit lost when viewer is in RevealedTo")
	}
}

func TestSnapshotGameState_ExileGroupedByOwner(t *testing.T) {
	g, pa, pb := makeGame()
	cardA := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	cardA.SetOwner(pa.PlayerID())
	cardB := mage.NewCreature("Elf", "{G}", 1, 1)
	cardB.SetOwner(pb.PlayerID())
	g.ExileCard(cardA, uuid.Nil)
	g.ExileCard(cardB, uuid.Nil)

	snap := SnapshotGameState(g, 0)
	if len(snap.You.Exile) != 1 || snap.You.Exile[0].Name != "Bear" {
		t.Errorf("You.Exile = %+v, want one Bear", snap.You.Exile)
	}
	if len(snap.Opponent.Exile) != 1 || snap.Opponent.Exile[0].Name != "Elf" {
		t.Errorf("Opponent.Exile = %+v, want one Elf", snap.Opponent.Exile)
	}
}

// ── PromptType.String ───────────────────────────────────────────────────────

func TestPromptType_String(t *testing.T) {
	tests := []struct {
		pt   PromptType
		want string
	}{
		{PromptMainPhaseAction, "Main Phase"},
		{PromptPriority, "Priority"},
		{PromptDeclareAttackers, "Declare Attackers"},
		{PromptDeclareBlockers, "Declare Blockers"},
		{PromptChooseTargets, "Choose Targets"},
		{PromptNone, ""},
	}
	for _, tt := range tests {
		if got := tt.pt.String(); got != tt.want {
			t.Errorf("PromptType(%d).String() = %q, want %q", tt.pt, got, tt.want)
		}
	}
}
