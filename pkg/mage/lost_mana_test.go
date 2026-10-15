package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestLostManaTransfer(t *testing.T) {
	for _, self := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		donor := b
		if self {
			donor = a
		}
		donor.ManaPool().Add(Blue, 2)
		donor.ManaPool().AddRestricted(Colorless, 3, ArtifactSpellsOnly{})
		original := donor.ManaPool().SnapshotPool()
		result := BindLostMana("lost")
		ctx := &EffectContext{Game: g, Controller: donor.PlayerID()}
		if err := LoseUnspentManaInto(You, result).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if donor.ManaPool().TotalMana() != 0 {
			t.Fatal("loss did not empty pool")
		}
		clone := g.Clone()
		ctx.Controller = a.PlayerID()
		if err := AddLostManaFor(You, result).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(a.ManaPool().SnapshotPool(), original) {
			t.Fatal("transfer changed mana")
		}
		if donor.ManaPool().TotalProducedThisTurn() != 5 {
			t.Fatal("loss changed production history")
		}
		want := 0
		if self {
			want = 5
		}
		if a.ManaPool().TotalProducedThisTurn() != want {
			t.Fatal("transfer produced new mana")
		}
		if a.ManaPool().CanPay(ManaCost{Generic: 5}, nil) || !a.ManaPool().CanPay(ManaCost{Generic: 5}, &SpellPaymentContext{IsArtifact: true}) {
			t.Fatal("transfer lost restriction")
		}
		if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
			t.Fatal("clone shared pool")
		}
		if err := AddLostManaFor(You, result).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(clone.GetPlayer(a.PlayerID()).ManaPool().SnapshotPool(), original) {
			t.Fatal("clone lost saved mana")
		}
		if err := LoseUnspentManaInto(You, result).Apply(&EffectContext{Game: g, Controller: uuid.New()}); err != nil {
			t.Fatal(err)
		}
		if len(g.resolution.lostManaBindings[result]) != 0 {
			t.Fatal("absent donor kept stale result")
		}
	}
}

func TestLostManaDeclarations(t *testing.T) {
	result := BindLostMana("lost")
	for _, effect := range []Effect{
		LoseUnspentManaInto(nil, result), LoseUnspentManaInto(You, nil),
		LoseUnspentManaInto(You, BindLostMana("")), AddLostManaFor(You, result),
		Sequence(LoseUnspentManaInto(You, result), AddLostManaFor(nil, result)),
		Sequence(LoseUnspentManaInto(You, result), AddLostManaFor(FromPlayer(BindPlayer("missing")), result)),
	} {
		c := NewInstant("Bad", "", Spell(effect))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("accepted invalid declaration", effect.Text())
		}
	}
	c := NewInstant("Good", "", Spell(Sequence(LoseUnspentManaInto(You, result), AddLostManaFor(You, result))))
	if err := ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
}

func TestLostManaFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result, other := BindLostMana("outer"), BindLostMana("inner")
	a.ManaPool().Add(Green, 2)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := LoseUnspentManaInto(You, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(Red, 1)
	if err := LoseUnspentManaInto(You, other).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.resolution.lostManaBindings[result]) != 2 || len(g.resolution.lostManaBindings[other]) != 1 {
		t.Fatal("producers shared results")
	}
	delayed := captureDelayedBody(ctx, AddLostManaFor(You, result))
	clone := g.Clone()
	clone.resolution.lostManaBindings[result][0].Color = Blue
	if g.resolution.lostManaBindings[result][0].Color != Green {
		t.Fatal("clone shared result slice")
	}
	g.resolution.Clear()
	if len(g.resolution.lostManaBindings) != 0 {
		t.Fatal("clear kept results")
	}
	if err := delayed.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Green) != 2 || len(g.resolution.lostManaBindings) != 0 {
		t.Fatal("delayed capture or restoration failed")
	}
	if err := ApplyEffect(g, Sequence(LoseUnspentManaInto(You, result), AddLostManaFor(You, result)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Green) != 2 || len(g.resolution.lostManaBindings) != 0 {
		t.Fatal("effect frame leaked")
	}
	if err := LoseUnspentManaInto(You, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := LoseUnspentManaInto(You, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.resolution.lostManaBindings[result]) != 0 {
		t.Fatal("empty loss retained old result")
	}
}

func TestLostManaTransferIsNotProduction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewLand("Source", WithManaAbility(Green)), a.PlayerID())
	if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, AnyPermanent, Blue), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(Red, 2)
	result := BindLostMana("lost")
	if err := ApplyEffect(g, Sequence(LoseUnspentManaInto(You, result), AddLostManaFor(You, result)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Red) != 2 || a.ManaPool().Count(Blue) != 0 || a.ManaPool().TotalProducedThisTurn() != 2 || p.Tapped || g.StackSize() != 0 {
		t.Fatal("transfer ran production or activation services")
	}
}

func TestLostManaLoopAndPaymentIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	result := BindLostMana("lost")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID()}
	a.ManaPool().Add(Red, 2)
	if err := LoseUnspentManaInto(You, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(Green, 1)
	loop := ForEach(MatchingPermanents(AnyPermanent), BindPermanent("member"), LoseUnspentManaInto(You, result))
	if err := loop.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.resolution.lostManaBindings[result]) != 2 || g.resolution.lostManaBindings[result][0].Color != Red {
		t.Fatal("loop replaced outer result")
	}
	trial := g.Clone()
	if err := LoseUnspentManaInto(You, result).Apply(&EffectContext{Game: trial, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(g.resolution.lostManaBindings[result]) != 2 {
		t.Fatal("trial changed live result")
	}
	source := g.PutOnBattlefield(NewArtifact("Costly", "", WithAbility(Activate(Costs(ManaCostOf("{9}")), Sequence(LoseUnspentManaInto(You, result), AddLostManaFor(You, result))))), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("unaffordable activation succeeded")
	}
	if len(g.resolution.lostManaBindings[result]) != 2 || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
		t.Fatal("failed activation changed state")
	}
}
