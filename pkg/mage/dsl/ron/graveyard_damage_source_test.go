package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONGraveyardDamageSourceProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black},
		{"LEA", "Circle of Protection: Blue", core.Blue},
		{"LEA", "Circle of Protection: Green", core.Green},
		{"LEA", "Circle of Protection: Red", core.Red},
		{"LEA", "Circle of Protection: White", core.White},
		{"ATQ", "Circle of Protection: Artifacts", core.White},
		{"LEG", "Greater Realm of Preservation", core.Black},
	} {
		for _, state := range []string{"live", "departed", "returned", "copy", "clone", "target/live", "target/departed", "target/returned", "target/copy", "target/clone"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				record := staticBaseRecord(t, tc.set, tc.name)
				rule := record.Fields["Rules"].Args[0]
				actions := rule.Args[1].Args[0].Args
				plan, err := compileChosenDamageSourceChoice(actions[0])
				if err != nil {
					t.Fatal(err)
				}
				shield, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
				if err != nil {
					t.Fatal(err)
				}
				costs, _, err := compileCosts(rule.Args[0])
				if err != nil {
					t.Fatal(err)
				}
				_, a, b := ronGame()
				chooser := &ronSourceChoicePlayer{BasePlayer: b}
				g := mage.NewGame(a, chooser)
				paid := mage.BindCards("paid graveyard sources")
				keeper := mage.NewArtifact("Reference keeper", "", mage.WithAbility(mage.Activate(mage.Costs(mage.DiscardCostWithCards(1, paid)), mage.GainLifeFor(mage.You, mage.Fixed(1)))))
				g.PutOnBattlefield(keeper, a.PlayerID())
				card := mage.NewArtifact("Graveyard subject", map[core.Color]string{core.White: "{W}", core.Blue: "{U}", core.Black: "{B}", core.Red: "{R}", core.Green: "{G}"}[tc.color])
				card.SetOwner(a.PlayerID())
				if after, ok := strings.CutPrefix(state, "target/"); ok {
					state = after
					a.AddToGraveyard(card)
					g.PushStack(&mage.StackObject{ID: uuid.New(), IsAbility: true, Targets: []uuid.UUID{card.ID()}})
				} else {
					a.AddToHand(card)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				}
				saved := g.StackGraveyardSources()[0]
				binding := mage.BindDamageSource("chosen graveyard source")
				bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
				body := mage.Sequence(plan.stackGraveyardChoice(bindings), shield(bindings))
				payment := make([]mage.Cost, len(costs))
				for i, cost := range costs {
					payment[i] = cost()
				}
				consumer := mage.NewEnchantment("RON consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), body)))
				if err := mage.ValidateCardDefinition(consumer.Name(), consumer); err != nil {
					t.Fatal(err)
				}
				g.PutOnBattlefield(consumer, b.PlayerID())
				if err := g.ActivateAbilityByIndex(b.PlayerID(), consumer.ID(), 0, nil); err == nil || chooser.calls != 0 {
					t.Fatal("unpaid activation chose a source")
				}
				b.ManaPool().Add(core.White, 2)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), consumer.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if chooser.calls != 0 {
					t.Fatal("choice ran at announcement")
				}
				switch state {
				case "departed", "returned":
					a.RemoveFromGraveyard(card.ID())
					if state == "returned" {
						a.AddToGraveyard(card)
					}
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "clone":
					clone := g.Clone()
					clone.ResolveTopOfStack()
					clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 2, saved)
					if clone.GetPlayer(b.PlayerID()).Life() != 20 || chooser.calls != 0 {
						t.Fatalf("clone life=%d, original choice calls=%d, clone sources=%v", clone.GetPlayer(b.PlayerID()).Life(), chooser.calls, clone.StackGraveyardSources())
					}
				}
				g.ResolveTopOfStack()
				if chooser.calls != 1 || !chooser.source.SameObject(saved) || chooser.source.GraveyardVisit() == uuid.Nil {
					t.Fatal("choice lost exact RON graveyard reference")
				}
				if state == "copy" {
					g.ResolveTopOfStack()
					if chooser.calls != 2 {
						t.Fatal("copy did not choose separately")
					}
				}
				g.GetStack().Pop()
				want := 20
				if state == "returned" {
					referer := &mage.StackObject{ID: uuid.New(), IsAbility: true, Targets: []uuid.UUID{card.ID()}}
					g.PushStack(referer)
					current := g.StackGraveyardSources()[0]
					g.GetStack().Pop()
					g.DealDamageToPlayerFromSource(b, 1, current)
					want--
				}
				g.DealDamageToPlayerFromSource(b, 2, saved)
				if b.Life() != want {
					t.Fatal("saved reference lost prevention after stack departure")
				}
				if state == "copy" {
					g.DealDamageToPlayerFromSource(b, 2, saved)
				}
				g.DealDamageToPlayerFromSource(b, 1, saved)
				if b.Life() != want-1 {
					t.Fatal("shield consumption was not local")
				}
				if err := mage.ApplyEffect(g, mage.Sequence(body, mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(2))), uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if b.Life() != want-1 {
					t.Fatal("empty category retained a stale choice")
				}
				if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "unsupported action (ChooseADamageSource)") {
					t.Fatal("incomplete enumeration accepted", err)
				}
			})
		}
	}
}

func TestRONGraveyardDamageSourceInvalid(t *testing.T) {
	for _, declaration := range []string{"ChooseADamageSource()", "ChooseADamageSource(AnyDamageSource())", "ChooseADamageSource(IsColor(Colorless))", "ChooseADamageSource(And([]))", "ChooseADamageSource(Ref_TargetPermanent)"} {
		nodes, err := Parse(strings.NewReader(declaration), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileChosenDamageSourceChoice(nodes[0]); err == nil {
			t.Fatal("invalid source declaration accepted", declaration)
		}
	}
	plan := &chosenDamageSourcePlan{}
	if mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), plan.stackGraveyardChoice(instanceBindings{}))))) == nil {
		t.Fatal("missing instance binding accepted")
	}
}
