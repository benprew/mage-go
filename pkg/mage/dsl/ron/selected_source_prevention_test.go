package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSelectedSourcePreventionContinuity(t *testing.T) {
	for _, tc := range []struct{ set, name, sourceSet, source string }{
		{"LEB", "Circle of Protection: Black", "LEB", "Frozen Shade"},
		{"LEA", "Circle of Protection: Blue", "LEA", "Prodigal Sorcerer"},
		{"LEA", "Circle of Protection: Green", "LEA", "Llanowar Elves"},
		{"LEA", "Circle of Protection: Red", "LEA", "Orcish Artillery"},
		{"LEA", "Circle of Protection: White", "LEA", "Samite Healer"},
		{"ATQ", "Circle of Protection: Artifacts", "ATQ", "Grapeshot Catapult"},
		{"LEG", "Greater Realm of Preservation", "LEA", "Orcish Artillery"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			record := staticBaseRecord(t, tc.set, tc.name)
			actions := record.Fields["Rules"].Args[0].Args[1].Args[0].Args
			filter, err := compileChosenDamageSourceFilter(actions[0].Args[0])
			if err != nil {
				t.Fatal(err)
			}
			plan := &chosenDamageSourcePlan{filter: filter}
			build, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
			if err != nil {
				t.Fatal(err)
			}
			binding := mage.BindDamageSource("selected")
			body := mage.Sequence(mage.ChooseDamageSourceFromLiveObjects(mage.You, mage.DamageSourceHasType(core.TypeCreature), binding), build(instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}))
			if err := mage.ValidateCardDefinition("Fixture", mage.NewArtifact("Fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), body)))); err != nil {
				t.Fatal(err)
			}
			source := repositoryCard(t, tc.sourceSet, tc.source)
			castRONSpell(t, g, a, source)
			if err := mage.ApplyEffect(g, body, source.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			for _, game := range []*mage.Game{g, g.Clone()} {
				game.ResolveTopOfStack()
				recipient := game.GetPlayer(b.PlayerID())
				if game.FindPermanent(source.ID()) == nil {
					t.Fatal("RON spell did not enter")
				}
				if tc.source == "Prodigal Sorcerer" || tc.source == "Orcish Artillery" {
					if err := game.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil {
						t.Fatal("sick source activated")
					}
					game.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
					if err := game.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
						t.Fatal(err)
					}
					game.ResolveTopOfStack()
					if tc.source == "Orcish Artillery" && game.GetPlayer(a.PlayerID()).Life() != 17 {
						t.Fatal("shield changed independent self damage")
					}
				} else {
					game.DealDamageToPlayer(recipient, 2, source.ID())
				}
				if recipient.Life() != 20 {
					t.Fatal("lowered shield lost permanent-spell continuity")
				}
				game.DealDamageToPlayer(recipient, 1, source.ID())
				if recipient.Life() != 19 {
					t.Fatal("shield not consumed")
				}
			}
		})
	}
}
