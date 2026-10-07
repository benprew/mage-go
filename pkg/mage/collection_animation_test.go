package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCollectionAnimationCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Forest", WithSubTypes("Forest"))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	other := g.PutOnBattlefield(NewLand("Other Forest", WithSubTypes("Forest")), b.PlayerID())
	phased := g.PutOnBattlefield(NewLand("Phased Forest", WithSubTypes("Forest")), a.PlayerID())
	g.PhaseOut(phased.ID())
	set := PermanentsControlledBy(You, And(HasSubType("Forest"), Not(IsCreature)))
	if err := ApplyEffect(g, Until(EndOfTurn,
		SetEachBasePowerToughness(set, Fixed(2), Fixed(3)), AddEachCardType(set, TypeCreature)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).HasType(TypeLand) || !g.FindPermanent(p.ID()).HasType(TypeCreature) || g.FindPermanent(p.ID()).CurrentPower(g) != 2 || g.FindPermanent(p.ID()).CurrentToughness(g) != 3 {
		t.Fatal("combined type and base values")
	}
	late := g.PutOnBattlefield(NewLand("Late Forest", WithSubTypes("Forest")), a.PlayerID())
	g.PhaseIn([]uuid.UUID{phased.ID()})
	for _, excluded := range []*Permanent{other, phased, late} {
		if g.FindPermanent(excluded.ID()).HasType(TypeCreature) {
			t.Fatal("capture added a later or excluded member")
		}
	}
	clone := g.Clone()
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.PhaseIn(g.PhaseOut(p.ID()))
	if !g.FindPermanent(p.ID()).HasType(TypeCreature) {
		t.Fatal("captured membership lost after control or phasing")
	}
	if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(1), Fixed(2)), SwitchPowerToughness(This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if p = g.FindPermanent(p.ID()); p.CurrentPower(g) != 5 || p.CurrentToughness(g) != 3 {
		t.Fatal("base, bonus and switch layers")
	}
	g.RemoveFromBattlefield(p)
	p = g.PutOnBattlefield(card, a.PlayerID())
	if p.HasType(TypeCreature) {
		t.Fatal("returned incarnation inherited animation")
	}
	if !clone.FindPermanent(p.ID()).HasType(TypeCreature) || clone.FindPermanent(p.ID()).CurrentPower(clone) != 2 {
		t.Fatal("clone changed with original")
	}
	clone.RunStepWithPriority(Cleanup)
	if clone.FindPermanent(p.ID()).HasType(TypeCreature) {
		t.Fatal("animation survived cleanup")
	}
}

func TestCollectionAnimationInvalid(t *testing.T) {
	for name, modification := range map[string]ContinuousModification{
		"nil type set": AddEachCardType(nil, TypeCreature),
		"invalid type": AddEachCardType(MatchingPermanents(AnyPermanent), TypeInstant),
		"nil base set": SetEachBasePowerToughness(nil, Fixed(2), Fixed(3)),
		"nil value":    SetEachBasePowerToughness(MatchingPermanents(AnyPermanent), nil, Fixed(3)),
	} {
		t.Run(name, func(t *testing.T) {
			card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), Until(EndOfTurn, modification))))
			if err := ValidateCardDefinition(card.Name(), card); err == nil {
				t.Fatal("invalid collection animation accepted")
			}
		})
	}
}
