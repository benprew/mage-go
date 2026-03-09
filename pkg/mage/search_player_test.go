package mage

import (
	"testing"
)

// ── Item 10: SearchPlayer Improved Stubs ────────────────────────────────────

func TestSearchPlayer_ChooseMayAbility_DrawAccepted(t *testing.T) {
	bp := NewBasePlayer("SP")
	sp := NewSearchPlayer(bp)
	if !sp.ChooseMayAbility("draw a card") {
		t.Error("should accept may ability with 'draw'")
	}
}

func TestSearchPlayer_ChooseMayAbility_DamageAccepted(t *testing.T) {
	bp := NewBasePlayer("SP")
	sp := NewSearchPlayer(bp)
	if !sp.ChooseMayAbility("deal 2 damage to target creature") {
		t.Error("should accept may ability with 'damage'")
	}
}

func TestSearchPlayer_ChooseMayAbility_DestroyAccepted(t *testing.T) {
	bp := NewBasePlayer("SP")
	sp := NewSearchPlayer(bp)
	if !sp.ChooseMayAbility("destroy target artifact") {
		t.Error("should accept may ability with 'destroy'")
	}
}

func TestSearchPlayer_ChooseMayAbility_UnknownDeclined(t *testing.T) {
	bp := NewBasePlayer("SP")
	sp := NewSearchPlayer(bp)
	if sp.ChooseMayAbility("tap target creature") {
		t.Error("should decline unknown may ability")
	}
}

func TestSearchPlayer_ChooseNumber_DamagePicksMax(t *testing.T) {
	bp := NewBasePlayer("SP")
	sp := NewSearchPlayer(bp)
	got := sp.ChooseNumber(0, 5, "how much damage")
	if got != 5 {
		t.Errorf("damage context: got %d, want 5 (max)", got)
	}
}

func TestSearchPlayer_ChooseNumber_DiscardPicksMin(t *testing.T) {
	bp := NewBasePlayer("SP")
	sp := NewSearchPlayer(bp)
	got := sp.ChooseNumber(1, 5, "cards to discard")
	if got != 1 {
		t.Errorf("discard context: got %d, want 1 (min)", got)
	}
}

func TestSearchPlayer_ChooseNumber_DefaultPicksMax(t *testing.T) {
	bp := NewBasePlayer("SP")
	sp := NewSearchPlayer(bp)
	got := sp.ChooseNumber(1, 5, "choose a number")
	if got != 5 {
		t.Errorf("default context: got %d, want 5 (max)", got)
	}
}

func TestSearchPlayer_ChooseMode_GainLifeLowLife(t *testing.T) {
	bp := NewBasePlayer("SP")
	bp.SetLife(5)
	sp := NewSearchPlayer(bp)
	got := sp.ChooseMode([]string{"Gain 3 life", "Draw a card"}, "Test")
	if got != 0 {
		t.Errorf("low life: got mode %d, want 0 (gain life)", got)
	}
}

func TestSearchPlayer_ChooseMode_DamagePreferred(t *testing.T) {
	bp := NewBasePlayer("SP")
	bp.SetLife(20)
	sp := NewSearchPlayer(bp)
	got := sp.ChooseMode([]string{"Unknown", "Deal 3 damage"}, "Test")
	if got != 1 {
		t.Errorf("got mode %d, want 1 (deal damage)", got)
	}
}

func TestSearchPlayer_ChoosePermanent_SacrificeCheapest(t *testing.T) {
	bp := NewBasePlayer("SP")
	sp := NewSearchPlayer(bp)

	small := NewPermanent(NewCreature("Elf", "{G}", 1, 1), sp.PlayerID())
	big := NewPermanent(NewCreature("Dragon", "{5}{R}", 6, 6), sp.PlayerID())

	got := sp.ChoosePermanent([]*Permanent{big, small}, "sacrifice a creature", nil)
	if got == nil || got.Name() != "Elf" {
		name := ""
		if got != nil {
			name = got.Name()
		}
		t.Errorf("sacrifice should pick cheapest: got %s, want Elf", name)
	}
}

func TestSearchPlayer_ChoosePermanent_DestroyBiggest(t *testing.T) {
	bp := NewBasePlayer("SP")
	sp := NewSearchPlayer(bp)

	small := NewPermanent(NewCreature("Elf", "{G}", 1, 1), sp.PlayerID())
	big := NewPermanent(NewCreature("Dragon", "{5}{R}", 6, 6), sp.PlayerID())

	got := sp.ChoosePermanent([]*Permanent{small, big}, "destroy target creature", nil)
	if got == nil || got.Name() != "Dragon" {
		name := ""
		if got != nil {
			name = got.Name()
		}
		t.Errorf("destroy should pick biggest: got %s, want Dragon", name)
	}
}
