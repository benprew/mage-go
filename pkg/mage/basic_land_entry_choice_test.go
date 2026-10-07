package mage

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type basicLandEntryPlayer struct {
	*BasePlayer
	answer string
	calls  int
}

func (p *basicLandEntryPlayer) ChooseString(options []string, reason string) string {
	p.calls++
	if !slices.Equal(options, []string{"Plains", "Island", "Swamp", "Mountain", "Forest"}) || reason != "choose a basic land type" {
		panic("incorrect basic land choice")
	}
	options[0] = "Desert"
	return p.answer
}

func TestChooseBasicLandTypeAsEnters(t *testing.T) {
	for _, landType := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest", "Desert", ""} {
		t.Run(landType, func(t *testing.T) {
			a := &basicLandEntryPlayer{BasePlayer: NewBasePlayer("A"), answer: landType}
			b := &basicLandEntryPlayer{BasePlayer: NewBasePlayer("B"), answer: "Forest"}
			g := NewGame(a, b)
			want := landType
			if want == "Desert" || want == "" {
				want = "Plains"
			}
			checked := 0
			probe := &opponentEntryProbe{check: func(g *Game, p *Permanent) {
				checked++
				for _, live := range g.AllBattlefield() {
					if live.ID() == p.ID() {
						t.Fatal("choice after battlefield registration")
					}
				}
				if p.ChosenBasicLandType.String() != want {
					t.Fatal("choice must precede battlefield registration")
				}
			}}
			card := NewArtifact("Entry", "", WithAbility(ChooseBasicLandTypeAsEnters()), WithAbility(probe))
			card.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			if checked != 1 || a.calls != 1 || b.calls != 0 || g.StackSize() != 0 {
				t.Fatal("wrong entry chooser or timing")
			}
			g.PhaseIn(g.PhaseOut(p.ID()))
			g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			if a.calls != 1 || b.calls != 0 || g.FindPermanent(p.ID()).ChosenBasicLandType.String() != want {
				t.Fatal("phase or control repeated choice")
			}
			clone := g.Clone()
			clone.MutablePermanent(p.ID()).ChosenBasicLandType = core.NoBasicLandType
			if g.FindPermanent(p.ID()).ChosenBasicLandType.String() != want {
				t.Fatal("clone shares choice")
			}
			want = "Forest"
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			b.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, b.PlayerID())
			if returned.ChosenBasicLandType != core.ForestLandType || b.calls != 1 || checked != 2 {
				t.Fatal("new entry kept its old choice")
			}
		})
	}
	if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(ChooseBasicLandTypeAsEnters()))); err == nil {
		t.Fatal("nonpermanent entry choice accepted")
	}
}
