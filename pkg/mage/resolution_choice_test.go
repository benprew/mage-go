package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type resolutionChooser struct {
	*BasePlayer
	choice, prompts int
}

func (p *resolutionChooser) ChooseMode([]string, string) int {
	p.prompts++
	return p.choice
}

func TestResolutionChoiceDispatch(t *testing.T) {
	for _, choice := range []int{-1, 0, 1, 9} {
		a := &resolutionChooser{BasePlayer: NewBasePlayer("A"), choice: choice}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		options := []ResolutionOption{{Label: "one", Body: GainLifeFor(You, Fixed(1))}, {Label: "two", Body: GainLifeFor(You, Fixed(2))}}
		effect := ChooseAtResolution(You, options...)
		options[0].Body = GainLifeFor(You, Fixed(20))
		card := NewInstant("Choice", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		if a.prompts != 0 {
			t.Fatal("choice occurred during declaration")
		}
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 21
		if choice == 1 {
			want = 22
		}
		if a.Life() != want || b.Life() != 20 || a.prompts != 1 {
			t.Fatal("wrong branch or chooser")
		}
	}
}

func TestResolutionChoiceAvailableOptions(t *testing.T) {
	a := &resolutionChooser{BasePlayer: NewBasePlayer("A"), choice: 0}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	condition := PermanentIsTapped(This)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	effect := ChooseAtResolution(You,
		ResolutionOption{Label: "conditional", Body: GainLifeFor(You, Fixed(7)), When: condition},
		ResolutionOption{Label: "always", Body: GainLifeFor(You, Fixed(1))})
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 || a.prompts != 0 {
		t.Fatal("unavailable branch was offered")
	}
	g.MutablePermanent(p.ID()).Tapped = true
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 28 || a.prompts != 1 {
		t.Fatal("condition was not checked at resolution")
	}
	if err := ApplyEffect(g, ChooseAtResolution(You, ResolutionOption{Label: "none", Body: GainLifeFor(You, Fixed(3)), When: IsYourTurn()}), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 20 {
		t.Fatal("unavailable option executed")
	}
}

func TestResolutionChoiceDeclarations(t *testing.T) {
	number := BindNumber("number")
	for _, effect := range []Effect{
		ChooseAtResolution(nil, ResolutionOption{Label: "one", Body: GainLifeFor(You, Fixed(1))}),
		ChooseAtResolution(You),
		ChooseAtResolution(You, ResolutionOption{Body: GainLifeFor(You, Fixed(1))}),
		ChooseAtResolution(You, ResolutionOption{Label: "nil"}),
		ChooseAtResolution(You, ResolutionOption{Label: "unbound", Body: Until(core.EndOfTurn, GrantKeywordModification(DeclarePermanentTarget("missing", AnyPermanent), core.Flying))}),
		ChooseAtResolution(You, ResolutionOption{Label: "producer", Body: SnapshotNumber(Fixed(1), number)}, ResolutionOption{Label: "other branch", Body: GainLifeFor(You, FromNumber(number))}),
		Sequence(ChooseAtResolution(You, ResolutionOption{Label: "producer", Body: SnapshotNumber(Fixed(1), number)}), GainLifeFor(You, FromNumber(number))),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid choice accepted")
		}
	}
}
