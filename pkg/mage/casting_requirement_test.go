package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCastingRequirementPaths(t *testing.T) {
	for _, path := range []string{"hand", "free", "alternate", "cast-copy"} {
		for _, blocked := range []bool{false, true} {
			t.Run(path+map[bool]string{false: "/allowed", true: "/blocked"}[blocked], func(t *testing.T) {
				g := newPriorityTestGame()
				g.SetStep(core.PrecombatMain)
				p := g.players[0]
				card := NewEnchantment("Restricted", "{U}", WithAbility(CastingRequirement(BattlefieldHasNone(IsLand))))
				card.SetOwner(p.PlayerID())
				p.AddToHand(card)
				p.ManaPool().Add(core.Blue, 1)
				if blocked {
					g.PutOnBattlefield(NewLand("Foreign land"), g.players[1].PlayerID())
				}
				if got := len(g.GetCastableSpells(p.PlayerID())); (got == 0) != blocked {
					t.Fatalf("castable spells = %d, blocked = %v", got, blocked)
				}
				var err error
				switch path {
				case "hand":
					err = g.CastSpellByID(p.PlayerID(), card.ID(), nil, 0)
				case "free":
					err = g.CastCardFromZoneWithoutPaying(p.PlayerID(), card.ID(), core.ZoneHand, nil, 0)
				case "alternate":
					err = g.CastCardFromZoneWithAlternateCost(p.PlayerID(), card.ID(), core.ZoneHand, card.ManaCost(), nil, 0)
				case "cast-copy":
					_, err = g.pushCastSpellObject(castStackObjectOptions{Card: card.Copy(), Controller: p.PlayerID(), IsCopy: true})
				}
				if (err != nil) != blocked {
					t.Fatalf("cast error = %v, blocked = %v", err, blocked)
				}
				if blocked && (len(p.Hand()) != 1 || p.ManaPool().TotalMana() != 1 || g.StackSize() != 0) {
					t.Fatal("rejected casting changed resources")
				}
			})
		}
	}
}

func TestCastingRequirementLifetime(t *testing.T) {
	g := newPriorityTestGame()
	g.SetStep(core.PrecombatMain)
	a, b := g.players[0], g.players[1]
	card := NewEnchantment("Restricted", "", WithAbility(CastingRequirement(BattlefieldHasNone(IsLand))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	land := g.PutOnBattlefield(NewLand("Late land"), b.PlayerID())
	clone := g.Clone()
	if clone.CopySpellOnStack(card.ID(), b.PlayerID(), false) == nil {
		t.Fatal("spell copy repeated casting requirement")
	}
	clone.ResolveStack()
	if clone.CountBattlefield(AnyPermanent) != 3 || g.FindPermanent(card.ID()) != nil {
		t.Fatal("resolution requirement or clone isolation")
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("resolution repeated casting requirement")
	}
	copied := card.Copy()
	a.AddToHand(copied)
	if err := g.CastSpellByID(a.PlayerID(), copied.ID(), nil, 0); err == nil {
		t.Fatal("card copy lost casting requirement")
	}
	g.PhaseOut(land.ID())
	if len(g.GetCastableSpells(a.PlayerID())) != 1 {
		t.Fatal("phased permanent prevented casting")
	}
	g.PhaseIn([]uuid.UUID{land.ID()})
	if len(g.GetCastableSpells(a.PlayerID())) != 0 {
		t.Fatal("phase-in did not restore restriction")
	}
}

func TestCastingRequirementDefinitions(t *testing.T) {
	for _, condition := range []StateCondition{nil, NotCondition(nil), PermanentIsTapped(This), PermanentIsTapped(DeclarePermanentTarget("unbound", IsCreature))} {
		card := NewEnchantment("Invalid", "", WithAbility(CastingRequirement(condition)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid casting requirement accepted")
		}
	}
	g := newPriorityTestGame()
	g.SetStep(core.DeclareAttackers)
	a := g.players[0]
	card := NewEnchantment("Timing", "", WithAbility(CastingRequirement(IsDuringSteps(core.DeclareAttackers))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("requirement granted casting permission")
	}
}
