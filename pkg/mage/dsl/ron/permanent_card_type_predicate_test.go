package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentCardTypePredicate(t *testing.T) {
	for _, tc := range []struct {
		syntax         string
		land, creature bool
	}{
		{`IsCardtype(Land)`, true, false},
		{`IsNonCardtype(Creature)`, true, false},
		{`And([IsCardtype(Land),IsNonCardtype(Creature)])`, true, false},
		{`Or([IsCardtype(Land),IsCardtype(Creature)])`, true, true},
		{`AnyPermanent`, true, true}, {`And([])`, true, true}, {`Or([])`, false, false},
	} {
		filter, err := compileFilter(nameScopeNode(t, tc.syntax))
		if err != nil {
			t.Fatal(err)
		}
		plan, ok := mage.InspectPermanentCardTypeFilter(filter)
		if !ok || plan.Matches(core.TypeLand) != tc.land || plan.Matches(core.TypeCreature) != tc.creature {
			t.Fatalf("lost typed predicate: %s", tc.syntax)
		}
	}
	for _, syntax := range []string{`IsCardtype`, `IsCardtype()`, `IsCardtype(Unknown)`, `IsCardtype(Land,Creature)`, `IsNonCardtype(You)`, `And(IsCardtype(Land))`, `Or([IsCardtype(Land),You])`} {
		if _, err := compileFilter(nameScopeNode(t, syntax)); err == nil {
			t.Fatalf("accepted %s", syntax)
		}
	}
}

func TestRONPermanentCardTypePredicateGameplay(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Stone Rain", "Ice Storm", "Sinkhole", "Shatter", "Armageddon", "Wrath of God"} {
			if set == "3ED" && (name == "Ice Storm" || name == "Sinkhole") {
				continue
			}
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				var card mage.Card = mage.NewLand("Victim")
				kind := core.TypeLand
				if name == "Shatter" {
					card = mage.NewArtifact("Victim", "")
					kind = core.TypeArtifact
				}
				if name == "Wrath of God" {
					card = mage.NewCreature("Victim", "", 2, 2)
					kind = core.TypeCreature
				}
				card.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(card, b.PlayerID())
				spell := repositoryCard(t, set, name)
				mass := name == "Armageddon" || name == "Wrath of God"
				if !mass {
					obj := castRONSpell(t, g, a, spell, victim.ID())
					target := mage.UnwrapTarget(obj.TargetSpecs[0]).(*mage.PermanentTarget)
					for _, filter := range target.Filters {
						plan, ok := mage.InspectPermanentCardTypeFilter(filter)
						if !ok || !plan.Matches(kind) || plan.Matches(core.TypeInstant) {
							t.Fatal("target lost its pure predicate")
						}
					}
				} else {
					castRONSpell(t, g, a, spell)
				}
				clone := g.Clone()
				clone.ResolveStack()
				if g.FindPermanent(victim.ID()) == nil {
					t.Fatal("clone changed original")
				}
				g.ResolveStack()
				if g.FindPermanent(victim.ID()) != nil || clone.FindPermanent(victim.ID()) != nil {
					t.Fatal("RON destruction failed")
				}
			})
		}
	}
}
