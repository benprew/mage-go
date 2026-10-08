package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONWaitingPreventionSpell(t *testing.T) {
	for _, tc := range []struct{ set, name, sourceSet, source string }{
		{"LEB", "Circle of Protection: Black", "LEB", "Frozen Shade"},
		{"LEA", "Circle of Protection: Blue", "LEA", "Prodigal Sorcerer"},
		{"LEA", "Circle of Protection: Green", "LEA", "Llanowar Elves"},
		{"LEA", "Circle of Protection: Red", "LEA", "Orcish Artillery"},
		{"LEA", "Circle of Protection: White", "LEA", "Samite Healer"},
		{"ATQ", "Circle of Protection: Artifacts", "ATQ", "Grapeshot Catapult"},
		{"LEG", "Greater Realm of Preservation", "LEA", "Orcish Artillery"},
	} {
		for _, state := range []string{"stack", "resolved", "recast", "copy", "cleanup"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				record := staticBaseRecord(t, tc.set, tc.name)
				actions := record.Fields["Rules"].Args[0].Args[1].Args[0].Args
				plan, err := compileChosenDamageSourceChoice(actions[0])
				if err != nil {
					t.Fatal(err)
				}
				build, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
				if err != nil {
					t.Fatal(err)
				}
				first, second := mage.BindDamageSource("first"), mage.BindDamageSource("second")
				bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: first}}
				source := repositoryCard(t, tc.sourceSet, tc.source)
				castRONSpell(t, g, a, source)
				saved := g.CaptureSpellDamageSource(g.StackPeek().ID)
				if len(g.WaitingPreventionSpellSources()) != 0 {
					t.Fatal("announcement created shield")
				}
				body := mage.Sequence(mage.ChooseDamageSourceFromLiveObjects(mage.You, plan.filter, first), build(bindings))
				if err := mage.ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				switch state {
				case "resolved":
					g.ResolveTopOfStack()
				case "recast":
					g.CounterStackObject(saved.StackID())
					a.RemoveFromGraveyard(source.ID())
					castRONSpell(t, g, a, source)
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy")
					}
				case "cleanup":
					g.GetEffects().ClearReplacementsEndOfTurn()
				}
				for _, game := range []*mage.Game{g, g.Clone()} {
					refs := game.WaitingPreventionSpellSources()
					if state == "cleanup" {
						if len(refs) != 0 {
							t.Fatal("expired shield")
						}
						continue
					}
					if len(refs) != 1 || !refs[0].SameObject(saved) {
						t.Fatal("exact waiting spell identity")
					}
					nextBindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: second}}
					loweredChoice := plan.waitingPreventionSpellChoice(nextBindings)
					effect := mage.Sequence(loweredChoice, build(nextBindings), mage.DealDamageFrom(mage.FromDamageSource(second), mage.You, mage.Fixed(2)))
					card := mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(), effect)))
					if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					if err := mage.ApplyEffect(game, effect, uuid.Nil, b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if game.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("waiting choice or shield failed")
					}
					game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, saved)
					if game.GetPlayer(b.PlayerID()).Life() != 20 || len(game.WaitingPreventionSpellSources()) != 0 {
						t.Fatal("independent shield consumption")
					}
					game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, saved)
					if game.GetPlayer(b.PlayerID()).Life() != 19 {
						t.Fatal("consumed shields still prevented damage")
					}
				}
				if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "unsupported action (ChooseADamageSource)") {
					t.Fatal("incomplete enumeration published", err)
				}
			})
		}
	}
}

func TestRONWaitingPreventionSpellInvalid(t *testing.T) {
	for _, expression := range []string{`ChooseADamageSource`, `ChooseADamageSource()`, `ChooseADamageSource(IsColor(Red),You)`, `ChooseADamageSource(IsColor(Colorless))`, `ChooseADamageSource(Or([]))`, `ChooseADamageSource(IsColor(Red()))`} {
		nodes, err := Parse(strings.NewReader(expression), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileChosenDamageSourceChoice(nodes[0]); err == nil {
			t.Fatal("invalid declaration", expression)
		}
	}
	node, err := Parse(strings.NewReader(`ChooseADamageSource(IsColor(Red))`), "choice.ron")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := compileChosenDamageSourceChoice(node[0])
	if err != nil {
		t.Fatal(err)
	}
	effect := plan.waitingPreventionSpellChoice(instanceBindings{})
	card := mage.NewArtifact("Missing", "", mage.WithAbility(mage.Activate(mage.Costs(), effect)))
	if mage.ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("missing instance binding accepted")
	}
}
