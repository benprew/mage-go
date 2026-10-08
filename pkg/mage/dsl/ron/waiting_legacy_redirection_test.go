package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingLegacyRedirectionProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black}, {"LEA", "Circle of Protection: Blue", core.Blue},
		{"LEA", "Circle of Protection: Green", core.Green}, {"LEA", "Circle of Protection: Red", core.Red},
		{"LEA", "Circle of Protection: White", core.White}, {"ATQ", "Circle of Protection: Artifacts", core.White},
		{"LEG", "Greater Realm of Preservation", core.Black},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := staticBaseRecord(t, tc.set, tc.name)
			actions := record.Fields["Rules"].Args[0].Args[1].Args[0].Args
			plan, err := compileChosenDamageSourceChoice(actions[0])
			if err != nil {
				t.Fatal(err)
			}
			shield, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			source := g.PutOnBattlefield(mage.NewCreature("Redirect destination", "{R}", 1, 20), a.PlayerID())
			mods := []mage.ContinuousModification{mage.SetColors(mage.This, tc.color)}
			if tc.name == "Circle of Protection: Artifacts" {
				mods = append(mods, mage.AddCardType(mage.This, core.TypeArtifact))
			}
			if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mods...), source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.SetArtifactDamageRedirect(a.PlayerID(), source.ID())
			saved := g.CaptureDamageSource(source.ID())
			if refs := g.WaitingRedirectionPermanentSources(); len(refs) != 1 || !refs[0].SameObject(saved) {
				t.Fatal("missing redirect destination")
			}

			binding := mage.BindDamageSource("chosen")
			bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
			body := mage.Sequence(plan.waitingRedirectionChoice(bindings), shield(bindings), mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(2)))
			if err := mage.ValidateCardDefinition("Consumer", mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(), body)))); err != nil {
				t.Fatal(err)
			}
			for _, game := range []*mage.Game{g, g.Clone()} {
				if err := mage.ApplyEffect(game, body, uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("choice prevention failed")
				}
				game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, saved)
				if game.GetPlayer(b.PlayerID()).Life() != 19 {
					t.Fatal("shield did not consume")
				}

				game.PhaseOut(source.ID())
				if err := mage.ApplyEffect(game, body, uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if game.GetPlayer(b.PlayerID()).Life() != 19 {
					t.Fatal("empty choice retained previous binding")
				}
			}
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "unsupported action (ChooseADamageSource)") {
				t.Fatal("incomplete source enumeration accepted", err)
			}
			if err := mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), plan.waitingRedirectionChoice(instanceBindings{}))))); err == nil {
				t.Fatal("missing binding accepted")
			}
		})
	}
}

func TestRONWaitingLegacyRedirectionInvalid(t *testing.T) {
	for _, declaration := range []string{"ChooseADamageSource", "ChooseADamageSource()", "ChooseADamageSource(AnyDamageSource())", "ChooseADamageSource(IsColor(Colorless))", "ChooseADamageSource(IsColor(Red,Blue))", "ChooseADamageSource(And([]))", "ChooseADamageSource(ThisPermanent)"} {
		nodes, err := Parse(strings.NewReader(declaration), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileChosenDamageSourceChoice(nodes[0]); err == nil {
			t.Fatal("invalid choice accepted", declaration)
		}
	}
	nodes, err := Parse(strings.NewReader("ChooseADamageSource(AnyDamageSource)"), "choice.ron")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := compileChosenDamageSourceChoice(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	binding := mage.BindDamageSource("redirect destination")
	bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
	consumer := mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1))
	card := mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(consumer, plan.waitingRedirectionChoice(bindings)))))
	if err := mage.ValidateCardDefinition("Invalid", card); err == nil {
		t.Fatal("consumer before producer accepted")
	}
}
