package mage

import (
	"strings"
	"testing"
)

func TestSacrificeSubtypeResultChosenLand(t *testing.T) {
	for _, tc := range []struct {
		name     string
		island   bool
		wantLife int
	}{
		{"island", true, 17},
		{"other land", false, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			chosen := BindPermanent("chosen land")
			matched := BindNumber("was Island")
			body := Sequence(ChoosePermanent(You, ControlledPermanents(IsLand), chosen),
				Sacrifice(From(chosen), SacrificeSubtypeInto("Island", matched)),
				IfPositive(FromNumber(matched), DealDamageFrom(This, You, Fixed(3))))
			source := NewCreature("Watcher", "", 2, 2)
			g.PutOnBattlefield(source, a.PlayerID())
			land := NewLand("Land")
			if tc.island {
				land = NewLand("Island", WithSubTypes("Island"))
			}
			g.PutOnBattlefield(land, a.PlayerID())
			opponentLand := NewLand("Opponent Island", WithSubTypes("Island"))
			g.PutOnBattlefield(opponentLand, b.PlayerID())
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != tc.wantLife || g.FindPermanent(land.ID()) != nil || g.FindPermanent(opponentLand.ID()) == nil {
				t.Fatalf("life = %d, land = %v", a.Life(), g.FindPermanent(land.ID()))
			}
		})
	}
}

func TestSacrificeSubtypeResultPendingCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	matched := BindNumber("was Island")
	source := NewLand("Island", WithSubTypes("Island"), WithAbility(Activate(Costs(), Sequence(
		Sacrifice(This, SacrificeSubtypeInto("Island", matched)),
		IfPositive(FromNumber(matched), GainLifeFor(You, Fixed(3)))))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 20 || a.Life() != 20 {
		t.Fatal("clone or copied ability reused a sacrifice result")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("pending abilities shared a sacrifice result")
	}
}

func TestSacrificeSubtypeResultFailureClearsPriorValue(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Island", WithSubTypes("Island"))
	p := g.PutOnBattlefield(land, a.PlayerID())
	matched := BindNumber("was Island")
	body := Sacrifice(This, SacrificeSubtypeInto("Island", matched))
	ctx := &EffectContext{Game: g, SourceID: land.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	for _, state := range []string{"phased", "returned", "other controller", "absent"} {
		g.resolution.numberBindings = map[*NumberBinding]int{matched: 1}
		switch state {
		case "phased":
			p.PhasedOut = true
		case "returned":
			p.PhasedOut = false
			g.RemoveFromBattlefield(p)
			p = g.PutOnBattlefield(land, a.PlayerID())
		case "other controller":
			ctx.sourceIncarnation = p.incarnationID
			ctx.Controller = b.PlayerID()
		case "absent":
			g.RemoveFromBattlefield(p)
		}
		if err := body.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if g.resolution.numberBindings[matched] != 0 {
			t.Fatalf("%s kept a stale subtype result", state)
		}
	}
}

func TestSacrificeSubtypeResultDeclaration(t *testing.T) {
	matched := BindNumber("matched")
	info := InspectDefinition(Sacrifice(This, SacrificeSubtypeInto("Island", matched)))
	if info.Role != "resolving-effect" || len(info.Children) != 2 || info.Children[1].Role != "number-binding" {
		t.Fatalf("inspection = %+v", info)
	}
	for _, tc := range []struct {
		name string
		body ResolvingEffect
		want string
	}{
		{"valid", Sequence(Sacrifice(This, SacrificeSubtypeInto("Island", matched)), GainLifeFor(You, FromNumber(matched))), ""},
		{"empty subtype", Sacrifice(This, SacrificeSubtypeInto("", matched)), "subtype"},
		{"nil binding", Sacrifice(This, SacrificeSubtypeInto("Island", nil)), "binding"},
		{"duplicate output", Sacrifice(This, SacrificeSubtypeInto("Island", matched), SacrificeAmountInto(matched)), "duplicate"},
		{"read before output", Sequence(GainLifeFor(You, FromNumber(matched)), Sacrifice(This, SacrificeSubtypeInto("Island", matched))), "producer"},
		{"missing producer", GainLifeFor(You, FromNumber(matched)), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewLand("Test", WithAbility(Activate(Costs(), tc.body)))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}
