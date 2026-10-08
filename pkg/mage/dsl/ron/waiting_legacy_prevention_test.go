package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingLegacyPreventionProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black}, {"LEA", "Circle of Protection: Blue", core.Blue},
		{"LEA", "Circle of Protection: Green", core.Green}, {"LEA", "Circle of Protection: Red", core.Red},
		{"LEA", "Circle of Protection: White", core.White}, {"ATQ", "Circle of Protection: Artifacts", core.White},
		{"LEG", "Greater Realm of Preservation", core.Black},
	} {
		for _, half := range []bool{false, true} {
			for _, state := range []string{"live", "depart", "return", "phase"} {
				t.Run(tc.name+"/"+state+map[bool]string{false: "/finite", true: "/half"}[half], func(t *testing.T) {
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
					source := g.PutOnBattlefield(mage.NewCreature("Legacy shield subject", "{R}", 1, 20), a.PlayerID())
					mods := []mage.ContinuousModification{mage.SetColors(mage.This, tc.color)}
					if tc.name == "Circle of Protection: Artifacts" {
						mods = append(mods, mage.AddCardType(mage.This, core.TypeArtifact))
					}
					if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mods...), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if half {
						g.AddHalfDamageFromSourcePreventionShield(a.PlayerID(), source.ID())
					} else {
						g.AddSourcePreventionShield(a.PlayerID(), source.ID(), 2)
					}
					saved := g.CaptureDamageSource(source.ID())
					switch state {
					case "depart", "return":
						g.RemoveFromBattlefield(source)
						if state == "return" {
							g.PutOnBattlefield(source.Card, a.PlayerID())
						}
					case "phase":
						g.PhaseOut(source.ID())
					}
					if refs := g.WaitingPreventionPermanentSources(); len(refs) != 1 || !refs[0].SameObject(saved) {
						t.Fatal("missing legacy subject")
					}
					binding := mage.BindDamageSource("chosen")
					bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
					body := mage.Sequence(plan.waitingPreventionChoice(bindings), shield(bindings), mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(2)))
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

						game.GetEffects().ClearReplacementsEndOfTurn()
						if err := mage.ApplyEffect(game, body, uuid.Nil, b.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
						if game.GetPlayer(b.PlayerID()).Life() != 19 {
							t.Fatal("empty choice retained previous binding")
						}
					}
					if _, err := Compile(record); err != nil {
						t.Fatal("complete source-choice record rejected", err)
					}
					if err := mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), plan.waitingPreventionChoice(instanceBindings{}))))); err == nil {
						t.Fatal("missing binding accepted")
					}
				})
			}
		}
	}
}

func TestRONWaitingLegacyPreventionInvalid(t *testing.T) {
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
	binding := mage.BindDamageSource("legacy subject")
	bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
	consumer := mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1))
	card := mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(consumer, plan.waitingPreventionChoice(bindings)))))
	if err := mage.ValidateCardDefinition("Invalid", card); err == nil {
		t.Fatal("consumer before producer accepted")
	}
}
