package ai

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

// ── AIPlayer.ChooseMode heuristic ───────────────────────────────────────────

func TestAIChooseMode_DealDamageMode(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	ai.SetLife(20)
	got := ai.ChooseMode([]string{"Draw a card", "Deal 3 damage to any target"}, "Test Charm")
	if got != 1 {
		t.Errorf("ChooseMode with damage mode = %d, want 1 (deal damage)", got)
	}
}

func TestAIChooseMode_DrawCardsWhenLowHand(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	ai.SetLife(20)
	got := ai.ChooseMode([]string{"Draw a card", "Gain 1 life"}, "Test Charm")
	if got != 0 {
		t.Errorf("ChooseMode with empty hand should prefer draw, got mode %d", got)
	}
}

func TestAIChooseMode_GainLifeWhenLow(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	ai.SetLife(5)
	got := ai.ChooseMode([]string{"Gain 3 life", "Draw a card"}, "Test Charm")
	if got != 0 {
		t.Errorf("ChooseMode at low life should prefer gain life, got mode %d", got)
	}
}

func TestAIChooseMode_DestroyMode(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	ai.SetLife(20)
	got := ai.ChooseMode([]string{"Unknown mode", "Destroy target creature"}, "Test Charm")
	if got != 1 {
		t.Errorf("ChooseMode should prefer destroy mode, got mode %d", got)
	}
}

func TestAIChooseMode_FallbackToZero(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	ai.SetLife(20)
	got := ai.ChooseMode([]string{"Unknown A", "Unknown B"}, "Mystery Card")
	if got != 0 {
		t.Errorf("ChooseMode should fall back to mode 0, got %d", got)
	}
}

func TestAIChooseModeWithEffects_UsesAIHints(t *testing.T) {
	pa := NewAIPlayer("Bot", &dummyStrategy{})
	pb := mage.NewBasePlayer("Opponent")
	g := mage.NewGame(pa, pb)

	modes := []mage.Mode{
		{
			Label:   "Unknown A",
			Effects: []mage.Effect{mage.FuncEffect("opaque", mage.EffectProperties{}, func(*mage.Game, uuid.UUID, uuid.UUID, []uuid.UUID) error { return nil })},
		},
		{
			Label: "Unknown B",
			Effects: []mage.Effect{mage.FuncEffect("engine", mage.EffectProperties{
				AIRoles:   []mage.AIRole{mage.AIRoleEngine},
				ValueBias: 2,
			}, func(*mage.Game, uuid.UUID, uuid.UUID, []uuid.UUID) error { return nil })},
		},
	}

	got := pa.ChooseModeWithEffects(modes, "Mystery Charm", g)
	if got != 1 {
		t.Fatalf("mode effect hints should choose mode 1, got %d", got)
	}
}

// ── SearchPlayer Choice Stubs (cross-package via mage helpers) ──────────────

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
