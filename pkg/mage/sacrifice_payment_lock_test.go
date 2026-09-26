package mage

import (
	"testing"

	"github.com/google/uuid"
)

type lastSacrificeChoicePlayer struct {
	*BasePlayer
	choices int
}

type invalidSacrificeChoicePlayer struct {
	*BasePlayer
	choice *Permanent
}

func (p *invalidSacrificeChoicePlayer) ChoosePermanent(_ []*Permanent, _ string, _ GameReader) *Permanent {
	return p.choice
}

func (p *lastSacrificeChoicePlayer) ChoosePermanent(candidates []*Permanent, _ string, _ GameReader) *Permanent {
	p.choices++
	return candidates[len(candidates)-1]
}

func TestSacrificePaymentKeepsValidatedChoice(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "", 1, 1)
	second := NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{SacrificeCreatureCost()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) != nil || a.choices != 1 {
		t.Fatal("payment did not keep the player's single choice")
	}
}

func TestSacrificePaymentRejectsChangedIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 1, 1)
	second := NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), SourceID: uuid.Nil, Costs: []Cost{SacrificeCreatureCost()},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, a.PlayerID())
	if err := tx.Commit(); err == nil {
		t.Fatal("payment used a returned permanent")
	}
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil {
		t.Fatal("failed payment sacrificed a different permanent")
	}
}

func TestSacrificePaymentLocksAllChoicesBeforeCommit(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first := NewCreature("First", "", 1, 1)
	second := NewCreature("Second", "", 2, 2)
	third := NewCreature("Third", "", 3, 3)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.PutOnBattlefield(third, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), Costs: []Cost{SacrificeNMatchingCost(2, IsCreature, "sacrifice two")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.choices != 2 {
		t.Fatalf("made %d choices during validation", a.choices)
	}
	g.PhaseOut(third.ID())
	if err := tx.Commit(); err == nil {
		t.Fatal("payment used a phased-out choice")
	}
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil {
		t.Fatal("invalid payment partly sacrificed its choices")
	}
	g.PhaseIn([]uuid.UUID{third.ID()})
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.choices != 2 || g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) != nil || g.FindPermanent(third.ID()) != nil {
		t.Fatal("commit changed or repeated the validated choices")
	}
}

func TestSacrificePaymentValidationFailureDoesNotSacrifice(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	creature := NewCreature("Only", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	_, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), Costs: []Cost{SacrificeCreatureCost(), SacrificeCreatureCost()},
	})
	if err == nil || g.FindPermanent(creature.ID()) == nil {
		t.Fatal("failed total payment changed the live battlefield")
	}
}

func TestSacrificePaymentActivationKeepsPlayerChoice(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeCreatureCost()), GainLifeFor(You, Fixed(1)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "", 1, 1)
	second := NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	if err := ValidateCardDefinition("Offering", source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.choices != 1 || g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) != nil {
		t.Fatal("activation paid with a different permanent")
	}
}

func TestSacrificePaymentRejectsChoiceOutsideCandidates(t *testing.T) {
	a := &invalidSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	creature := NewCreature("Creature", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	a.choice = g.FindPermanent(source.ID())
	_, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{SacrificeCreatureCost()},
	})
	if err == nil || g.FindPermanent(source.ID()) == nil || g.FindPermanent(creature.ID()) == nil {
		t.Fatal("invalid payment choice changed the live battlefield")
	}
}
