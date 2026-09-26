package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidSacrificeStatsActivation(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	power, toughness := BindNumber("paid power"), BindNumber("paid toughness")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeCreatureCostWithStats(power, toughness)),
		GainLifeFor(You, FromNumber(toughness)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Small", "", 1, 1), a.PlayerID())
	chosen := NewCreature("Large", "", 3, 5)
	g.PutOnBattlefield(chosen, a.PlayerID())
	g.MutablePermanent(chosen.ID()).AddCounter(core.P1P1, 2)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.choices != 1 || g.FindPermanent(chosen.ID()) != nil {
		t.Fatal("payment did not sacrifice the chosen creature once")
	}
	g.ResolveStack()
	if a.Life() != 27 {
		t.Fatalf("life = %d, want paid toughness 7", a.Life())
	}
}

func TestPaidSacrificeStatsCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("paid power"), BindNumber("paid toughness")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeCreatureCostWithStats(power, toughness)),
		GainLifeFor(You, FromNumber(power)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	creature := NewCreature("Creature", "", 4, 2)
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || clone.GetPlayer(b.PlayerID()).Life() != 24 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone or stack copy lost its payment value")
	}
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 24 {
		t.Fatal("payment value escaped its stack object")
	}
}

func TestPaidSacrificeStatsPendingActivations(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("paid power"), BindNumber("paid toughness")
	target := DeclarePlayerTarget("recipient")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeCreatureCostWithStats(power, toughness)),
		GainLifeFor(target, FromNumber(power)), WithTargets(target),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(NewCreature("First", "", 2, 1), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Second", "", 5, 1), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.choices != 2 || a.Life() != 25 || b.Life() != 22 {
		t.Fatal("pending activations shared a payment result")
	}
}

func TestPaidSacrificeStatsModalActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("paid power"), BindNumber("paid toughness")
	source := NewArtifact("Offering", "", WithAbility(ActivateModes(
		Costs(SacrificeCreatureCostWithStats(power, toughness)),
		ModeOf("gain", GainLifeFor(You, FromNumber(toughness))),
		ModeOf("lose", LoseLifeFor(You, FromNumber(power))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Creature", "", 2, 4), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("life = %d, want paid toughness 4", a.Life())
	}
}

func TestPaidSacrificeStatsFailedCommit(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("paid power"), BindNumber("paid toughness")
	creature := NewCreature("Creature", "", 4, 2)
	g.PutOnBattlefield(creature, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), Costs: []Cost{SacrificeCreatureCostWithStats(power, toughness)},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(creature.ID()) == nil {
		t.Fatal("returned creature produced a payment result")
	}
}

func TestPaidSacrificeStatsDefinition(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost Cost
		body ResolvingEffect
		want string
	}{
		{"valid", SacrificeCreatureCostWithStats(BindNumber("power"), BindNumber("toughness")), GainLifeFor(You, Fixed(1)), ""},
		{"nil power", SacrificeCreatureCostWithStats(nil, BindNumber("toughness")), GainLifeFor(You, Fixed(1)), "non-nil"},
		{"nil both", SacrificeCreatureCostWithStats(nil, nil), GainLifeFor(You, Fixed(1)), "non-nil"},
		{"duplicate", SacrificeCreatureCostWithStats(BindNumber("same"), BindNumber("same")), GainLifeFor(You, Fixed(1)), "duplicate"},
		{"read before payment", SacrificeCreatureCost(), GainLifeFor(You, FromNumber(BindNumber("missing"))), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(tc.cost), tc.body)))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestPaidSacrificeStatsSpellAndResolutionCost(t *testing.T) {
	power, toughness := BindNumber("power"), BindNumber("toughness")
	cost := SacrificeCreatureCostWithStats(power, toughness)
	card := NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, FromNumber(toughness)))), WithAdditionalCost(cost))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatalf("additional cost validation = %v", err)
	}
	card = NewInstant("Invalid", "", WithAbility(NewAction(ActionSpell, WithCost(cost), WithEffect(GainLifeFor(You, FromNumber(toughness))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatalf("spell action cost validation = %v", err)
	}
	card = NewArtifact("Invalid", "", WithAbility(Activate(Costs(), IfPaid(You, Costs(cost), GainLifeFor(You, FromNumber(toughness))))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "activated ability cost") {
		t.Fatalf("resolution payment validation = %v", err)
	}
}
