package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDestroyEachMatchingLands(t *testing.T) {
	for _, subtype := range []string{"", "Island", "Plains", "Forest"} {
		name := "all lands"
		filter := IsLand
		if subtype != "" {
			name = subtype
			filter = HasSubType(subtype)
		}
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			effect := DestroyEach(MatchingPermanents(filter))
			if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			var destroyed, survivors []*Permanent
			for _, owner := range []*BasePlayer{a, b} {
				for _, scenario := range []string{"ordinary", "indestructible", "regenerate", "phased", "Mountain", "creature"} {
					card := NewLand(scenario, WithSubTypes(subtype))
					if scenario == "Mountain" {
						card = NewLand("Mountain", WithSubTypes("Mountain"))
					}
					if scenario == "creature" {
						card = NewCreature("Creature", "", 2, 2)
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
					if scenario == "ordinary" || (scenario == "Mountain" && subtype == "") {
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
		})
	}
}
