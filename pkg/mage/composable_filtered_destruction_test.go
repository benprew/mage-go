package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDestroyEachMatchingBlackCreatures(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	effect := DestroyEach(MatchingPermanents(And(IsCreature, HasColorFilter(core.Black))))
	if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	var destroyed, survivors []*Permanent
	for _, owner := range []*BasePlayer{a, b} {
		for _, scenario := range []string{"black", "multicolor", "white", "colorless", "artifact", "indestructible", "regenerate", "phased"} {
			card := NewCreature(scenario, "{B}", 2, 2)
			switch scenario {
			case "multicolor":
				card = NewCreature(scenario, "{W}{B}", 2, 2)
			case "white":
				card = NewCreature(scenario, "{W}", 2, 2)
			case "colorless":
				card = NewCreature(scenario, "{2}", 2, 2)
			case "artifact":
				card = NewArtifact(scenario, "{B}")
			}
			card.SetOwner(owner.PlayerID())
			p := g.PutOnBattlefield(card, owner.PlayerID())
			switch scenario {
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
			case "regenerate":
				g.AddRegenerationShield(p.ID())
			case "phased":
				p.PhasedOut = true
			}
			if scenario == "black" || scenario == "multicolor" {
				destroyed = append(destroyed, p)
			} else {
				survivors = append(survivors, p)
			}
		}
	}
	clone := g.Clone()
	if err := effect.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	for _, p := range destroyed {
		if clone.FindPermanent(p.ID()) != nil || g.FindPermanent(p.ID()) == nil {
			t.Fatalf("wrong destruction or clone isolation for %s", p.Name())
		}
	}
	for _, p := range survivors {
		got := clone.FindPermanentIncludingPhased(p.ID())
		if got == nil {
			t.Fatalf("destroyed %s", p.Name())
		}
		if p.Name() == "regenerate" && (!got.Tapped || p.Tapped) {
			t.Fatal("regeneration did not tap only the cloned permanent")
		}
	}
}
