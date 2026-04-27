package ai

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

// ── Item 9: Modal Spell Move Generation ─────────────────────────────────────

func TestGeneratePriorityMoves_ModalSpellGeneratesPerModeMove(t *testing.T) {
	g, pa, _ := makeGame()

	// Create a modal spell with 3 modes
	charm := mage.NewInstant("Charm", "{W}",
		mage.NewSpellAbility(mage.GainLife(3)),
	)
	charm.SetModes([]string{"Gain 3 life", "Prevent 3 damage", "Destroy target enchantment"})
	charm.SetOwner(pa.PlayerID())
	pa.AddToHand(charm)

	addLands(g, pa, "Plains", 1)
	g.SetStep(core.PrecombatMain)

	moves := GeneratePriorityMoves(g, pa, 0, true)

	// Count moves for "Charm" — should be one per mode (3 modes)
	charmMoves := 0
	modesSeen := map[int]bool{}
	for _, m := range moves {
		if m.CardName == "Charm" {
			charmMoves++
			modesSeen[m.ModeIndex] = true
		}
	}
	if charmMoves != 3 {
		t.Errorf("expected 3 moves for 3-mode Charm, got %d", charmMoves)
	}
	for i := 0; i < 3; i++ {
		if !modesSeen[i] {
			t.Errorf("missing move for mode %d", i)
		}
	}
}

func TestGeneratePriorityMoves_NonModalSpellNoModeIndex(t *testing.T) {
	g, pa, _ := makeGame()

	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	addLands(g, pa, "Mountain", 1)
	g.SetStep(core.PrecombatMain)

	moves := GeneratePriorityMoves(g, pa, 0, true)
	for _, m := range moves {
		if m.CardName == "Lightning Bolt" && m.ModeIndex != 0 {
			t.Errorf("non-modal spell should have ModeIndex 0, got %d", m.ModeIndex)
		}
	}
}

// ── Item 9: AIPlayer.ChooseMode heuristic ───────────────────────────────────

func TestAIChooseMode_DealDamageMode(t *testing.T) {
	ai := NewAIPlayer("Bot")
	ai.SetLife(20)
	got := ai.ChooseMode([]string{"Draw a card", "Deal 3 damage to any target"}, "Test Charm")
	if got != 1 {
		t.Errorf("ChooseMode with damage mode = %d, want 1 (deal damage)", got)
	}
}

func TestAIChooseMode_DrawCardsWhenLowHand(t *testing.T) {
	ai := NewAIPlayer("Bot")
	ai.SetLife(20)
	// With empty hand (< 3 cards), draw card mode should be preferred
	got := ai.ChooseMode([]string{"Draw a card", "Gain 1 life"}, "Test Charm")
	if got != 0 {
		t.Errorf("ChooseMode with empty hand should prefer draw, got mode %d", got)
	}
}

func TestAIChooseMode_GainLifeWhenLow(t *testing.T) {
	ai := NewAIPlayer("Bot")
	ai.SetLife(5)
	got := ai.ChooseMode([]string{"Gain 3 life", "Draw a card"}, "Test Charm")
	if got != 0 {
		t.Errorf("ChooseMode at low life should prefer gain life, got mode %d", got)
	}
}

func TestAIChooseMode_DestroyMode(t *testing.T) {
	ai := NewAIPlayer("Bot")
	ai.SetLife(20)
	got := ai.ChooseMode([]string{"Unknown mode", "Destroy target creature"}, "Test Charm")
	if got != 1 {
		t.Errorf("ChooseMode should prefer destroy mode, got mode %d", got)
	}
}

func TestAIChooseMode_FallbackToZero(t *testing.T) {
	ai := NewAIPlayer("Bot")
	ai.SetLife(20)
	got := ai.ChooseMode([]string{"Unknown A", "Unknown B"}, "Mystery Card")
	if got != 0 {
		t.Errorf("ChooseMode should fall back to mode 0, got %d", got)
	}
}

// ── Item 10: SearchPlayer Choice Stubs (cross-package via ai helpers) ───────

func TestSearchPlayer_ChooseMayAbility_AcceptsDrawDamageDestroy(t *testing.T) {
	bp := mage.NewBasePlayer("Search")
	sp := mage.NewSearchPlayer(bp)

	tests := []struct {
		desc string
		want bool
	}{
		{"draw a card", true},
		{"deal 2 damage", true},
		{"destroy target creature", true},
		{"tap target creature", false},
		{"sacrifice a creature", false},
	}
	for _, tt := range tests {
		got := sp.ChooseMayAbility(tt.desc)
		if got != tt.want {
			t.Errorf("ChooseMayAbility(%q) = %v, want %v", tt.desc, got, tt.want)
		}
	}
}

func TestSearchPlayer_ChooseNumber_DamageMax(t *testing.T) {
	bp := mage.NewBasePlayer("Search")
	sp := mage.NewSearchPlayer(bp)

	got := sp.ChooseNumber(1, 5, "damage")
	if got != 5 {
		t.Errorf("ChooseNumber damage context = %d, want 5 (max)", got)
	}
}

func TestSearchPlayer_ChooseNumber_DiscardMin(t *testing.T) {
	bp := mage.NewBasePlayer("Search")
	sp := mage.NewSearchPlayer(bp)

	got := sp.ChooseNumber(1, 5, "discard")
	if got != 1 {
		t.Errorf("ChooseNumber discard context = %d, want 1 (min)", got)
	}
}

func TestSearchPlayer_ChooseMode_Heuristic(t *testing.T) {
	bp := mage.NewBasePlayer("Search")
	bp.SetLife(5)
	sp := mage.NewSearchPlayer(bp)

	got := sp.ChooseMode([]string{"Gain 3 life", "Draw a card"}, "Test")
	if got != 0 {
		t.Errorf("ChooseMode at low life = %d, want 0 (gain life)", got)
	}
}

func TestSearchPlayer_ChoosePermanent_SacrificePicks_Lowest(t *testing.T) {
	bp := mage.NewBasePlayer("Search")
	sp := mage.NewSearchPlayer(bp)

	small := makePerm("Elf", "{G}", 1, 1, sp.PlayerID())
	big := makePerm("Giant", "{4}{G}", 5, 5, sp.PlayerID())

	got := sp.ChoosePermanent([]*mage.Permanent{big, small}, "sacrifice", nil)
	if got == nil || got.Name() != "Elf" {
		name := ""
		if got != nil {
			name = got.Name()
		}
		t.Errorf("ChoosePermanent sacrifice should pick lowest-value: got %s, want Elf", name)
	}
}

func TestSearchPlayer_ChoosePermanent_DestroyPicks_Highest(t *testing.T) {
	bp := mage.NewBasePlayer("Search")
	sp := mage.NewSearchPlayer(bp)

	small := makePerm("Elf", "{G}", 1, 1, sp.PlayerID())
	big := makePerm("Giant", "{4}{G}", 5, 5, sp.PlayerID())

	got := sp.ChoosePermanent([]*mage.Permanent{small, big}, "destroy", nil)
	if got == nil || got.Name() != "Giant" {
		name := ""
		if got != nil {
			name = got.Name()
		}
		t.Errorf("ChoosePermanent destroy should pick highest-value: got %s, want Giant", name)
	}
}

// ── Item 11: Smarter Spell Resolution in Search ─────────────────────────────

func TestApplySpellCast_LifeGain(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)
	pa.SetLife(17)

	lifeSpell := mage.NewSorcery("Healing Touch", "{W}",
		mage.NewSpellAbility(mage.GainLife(3)),
	)
	lifeSpell.SetOwner(pa.PlayerID())
	pa.AddToHand(lifeSpell)
	addLands(g, pa, "Plains", 1)

	clone := g.Clone()
	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   lifeSpell.ID(),
		CardName: "Healing Touch",
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	clonePA := clone.GetPlayer(pa.PlayerID())
	if clonePA.Life() != 20 {
		t.Errorf("life after gain = %d, want 20", clonePA.Life())
	}
}

func TestApplySpellCast_BuffEffect(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)

	creature := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(creature)

	buff := mage.NewInstant("Giant Growth", "{G}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.Boost(mage.Fixed(3), mage.Fixed(3))),
	)
	buff.SetOwner(pa.PlayerID())
	pa.AddToHand(buff)
	addLands(g, pa, "Forest", 1)

	clone := g.Clone()
	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   buff.ID(),
		CardName: "Giant Growth",
		Targets:  []uuid.UUID{creature.ID()},
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	perm := clone.FindPermanent(creature.ID())
	if perm == nil {
		t.Fatal("creature not found in clone")
	}
	gotP := perm.CurrentPower(clone)
	gotT := perm.CurrentToughness(clone)
	if gotP != 5 || gotT != 5 {
		t.Errorf("creature P/T after buff = %d/%d, want 5/5", gotP, gotT)
	}
}

func TestApplySpellCast_MultiEffect_DrawAndDamage(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)
	pa.SetLife(20)
	pb.SetLife(20)

	// Use targeted damage + active player draw (untargeted draw goes to caster).
	multiSpell := mage.NewSorcery("Arcane Blast", "{1}{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(),
			mage.DealDamage(mage.Fixed(3)),
			mage.DrawCardsActivePlayer(mage.Fixed(2)),
		),
	)
	multiSpell.SetOwner(pa.PlayerID())
	pa.AddToHand(multiSpell)

	for i := 0; i < 5; i++ {
		c := mage.NewCreature("Filler", "{G}", 1, 1)
		c.SetOwner(pa.PlayerID())
		pa.AddToLibrary(c)
	}

	addLands(g, pa, "Mountain", 2)

	clone := g.Clone()
	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   multiSpell.ID(),
		CardName: "Arcane Blast",
		Targets:  []uuid.UUID{pb.PlayerID()},
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	clonePB := clone.GetPlayer(pb.PlayerID())
	if clonePB.Life() != 17 {
		t.Errorf("opponent life after damage = %d, want 17", clonePB.Life())
	}

	clonePA := clone.GetPlayer(pa.PlayerID())
	if len(clonePA.Hand()) < 2 {
		t.Errorf("should have drawn 2 cards, hand size = %d", len(clonePA.Hand()))
	}
}

func TestApplySpellCast_BounceEffect(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)

	target := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(target)

	bounce := mage.NewInstant("Unsummon", "{U}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.ReturnToHandTarget()),
	)
	bounce.SetOwner(pa.PlayerID())
	pa.AddToHand(bounce)
	addLands(g, pa, "Island", 1)

	clone := g.Clone()
	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   bounce.ID(),
		CardName: "Unsummon",
		Targets:  []uuid.UUID{target.ID()},
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	perm := clone.FindPermanent(target.ID())
	if perm != nil {
		t.Error("bounced permanent should be removed from battlefield")
	}
}

func TestApplySpellCast_XSpellDamage(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)
	pb.SetLife(20)

	fireball := mage.NewSorcery("Fireball", "{X}{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue())),
	)
	fireball.SetOwner(pa.PlayerID())
	pa.AddToHand(fireball)
	addLands(g, pa, "Mountain", 5)

	clone := g.Clone()
	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   fireball.ID(),
		CardName: "Fireball",
		Targets:  []uuid.UUID{pb.PlayerID()},
		XValue:   4,
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	clonePB := clone.GetPlayer(pb.PlayerID())
	if clonePB.Life() != 16 {
		t.Errorf("opponent life after X=4 Fireball = %d, want 16", clonePB.Life())
	}
}
