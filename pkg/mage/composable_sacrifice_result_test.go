package mage

import (
	"strings"
	"testing"
)

func TestSacrificeAmountResult(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("sacrificed")
	card := NewCreature("Victim", "", 2, 2)
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	effect := Sequence(Sacrifice(This, SacrificeAmountInto(result)), GainLifeFor(You, FromNumber(result)))
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil || a.Life() != 21 {
		t.Fatal("successful sacrifice result was not observed")
	}
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 {
		t.Fatal("an absent permanent reported a sacrifice")
	}
	g.PutOnBattlefield(card, b.PlayerID())
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 || g.FindPermanent(card.ID()) == nil {
		t.Fatal("another player's permanent was sacrificed")
	}
}

func TestSacrificeAmountResultPhasingReturnAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("sacrificed")
	card := NewCreature("Victim", "", 2, 2)
	p := g.PutOnBattlefield(card, a.PlayerID())
	effect := Sequence(Sacrifice(This, SacrificeAmountInto(result)), GainLifeFor(You, FromNumber(result)))
	p.PhasedOut = true
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || g.FindPermanentIncludingPhased(card.ID()) == nil {
		t.Fatal("phased permanent reported a sacrifice")
	}
	p.PhasedOut = false
	clone := g.Clone()
	if err := ApplyEffect(clone, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 || g.FindPermanent(card.ID()) == nil {
		t.Fatal("clone shared sacrifice state")
	}
	old := p.incarnationID
	g.DoSacrifice(p)
	g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: old}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || g.FindPermanent(card.ID()) == nil {
		t.Fatal("returned permanent reported a sacrifice")
	}
}

func TestSacrificeAmountResultClearsAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("sacrificed")
	ability := Activate(Costs(), Sequence(
		Sacrifice(This, SacrificeAmountInto(result)),
		GainLifeFor(You, FromNumber(result)),
		Sacrifice(This, SacrificeAmountInto(result)),
		GainLifeFor(You, FromNumber(result)),
	))
	card := NewCreature("Victim", "", 2, 2, WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 20 || a.Life() != 20 {
		t.Fatal("copy or clone result escaped its resolution")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("a copied ability reused a prior sacrifice result")
	}
}

func TestSacrificeAmountResultDeclaration(t *testing.T) {
	result := BindNumber("sacrificed")
	for _, tc := range []struct {
		name string
		body ResolvingEffect
		want string
	}{
		{"valid", Sequence(Sacrifice(This, SacrificeAmountInto(result)), GainLifeFor(You, FromNumber(result))), ""},
		{"missing producer", GainLifeFor(You, FromNumber(result)), "producer"},
		{"before producer", Sequence(GainLifeFor(You, FromNumber(result)), Sacrifice(This, SacrificeAmountInto(result))), "producer"},
		{"nil result", Sacrifice(This, SacrificeAmountInto(nil)), "non-nil"},
		{"duplicate result", Sacrifice(This, SacrificeAmountInto(result), SacrificeAmountInto(result)), "duplicate"},
		{"nil option", Sacrifice(This, nil), "non-nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCreature("Synthetic", "", 1, 1, WithAbility(Activate(Costs(), tc.body)))
			err := ValidateCardDefinition(c.Name(), c)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}
