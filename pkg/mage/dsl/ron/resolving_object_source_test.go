package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONResolvingObjectSourcesProtection(t *testing.T) {
	for _, tc := range []struct{ set, name, mana string }{
		{"LEB", "Circle of Protection: Black", "{B}"},
		{"LEA", "Circle of Protection: Blue", "{U}"},
		{"LEA", "Circle of Protection: Green", "{G}"},
		{"LEA", "Circle of Protection: Red", "{R}"},
		{"LEA", "Circle of Protection: White", "{W}"},
		{"ATQ", "Circle of Protection: Artifacts", "{W}"},
		{"LEG", "Greater Realm of Preservation", "{B}"},
	} {
		for _, state := range []string{"live", "departed", "returned", "copy", "clone", "phased", "phase-cycle"} {
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
				binding := mage.BindDamageSource("chosen source")
				bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
				body := mage.Sequence(plan.stackAbilityChoice(bindings), shield(bindings))
				payment := make([]mage.Cost, 0, len(costs))
				for _, cost := range costs {
					payment = append(payment, cost())
				}
				consumer := mage.NewArtifact("RON consumer", tc.mana, mage.WithAbility(mage.Activate(mage.Costs(payment...), body)))
				if err := mage.ValidateCardDefinition(consumer.Name(), consumer); err != nil {
					t.Fatal(err)
				}
				consumer.SetOwner(b.PlayerID())
				permanent := g.PutOnBattlefield(consumer, b.PlayerID())
				card := consumer
				if err := g.ActivateAbilityByIndex(b.PlayerID(), consumer.ID(), 0, nil); err == nil || g.StackSize() != 0 || chooser.calls != 0 {
					t.Fatal("failed payment published a reference")
				}
				b.ManaPool().Add(core.White, 2)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), consumer.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				sources := g.StackAbilityPermanentSources()
				if len(sources) != 1 || chooser.calls != 0 {
					t.Fatal("missing payment or early choice")
				}
				saved := sources[0]
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(g.FindPermanent(card.ID()))
					if state == "returned" {
						b.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
						g.ChangeObjectColor(card.ID(), core.Colorless)
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
						t.Fatal("clone lost reference or shared choice")
					}
				case "phased":
					g.PhaseOut(permanent.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(permanent.ID()))
				}
				g.ResolveTopOfStack()
				wantCalls := 1
				if state == "copy" {
					g.ResolveTopOfStack()
					wantCalls = 2
				}
				if chooser.calls != wantCalls || !chooser.source.SameObject(saved) {
					t.Fatal("resolving ability lost choice")
				}
				if len(g.StackAbilityPermanentSources()) != 0 {
					t.Fatal("completed activation retained source")
				}
				g.DealDamageToPlayerFromSource(b, 2, saved)
				if b.Life() != 20 {
					t.Fatal("shield lost exact resolving choice")
				}
				if state == "copy" {
					g.DealDamageToPlayerFromSource(b, 2, saved)
				}
				g.DealDamageToPlayerFromSource(b, 1, saved)
				if b.Life() != 19 {
					t.Fatal("independent shield consumption failed")
				}
				if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
					t.Fatal("incomplete aggregation accepted", err)
				}
			})
		}
	}
}

func TestRONResolvingObjectSourcesInvalid(t *testing.T) {
	plan := &chosenDamageSourcePlan{}
	binding := mage.BindDamageSource("unwritten")
	bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
	body := mage.Sequence(mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1)), plan.stackAbilityChoice(bindings))
	card := mage.NewArtifact("Invalid order", "", mage.WithAbility(mage.Activate(mage.Costs(), body)))
	if mage.ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("read before choice accepted")
	}
}

func TestRONResolvingObjectSourcesMissingBinding(t *testing.T) {
	plan := &chosenDamageSourcePlan{}
	for _, bindings := range []instanceBindings{{}, {chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: mage.BindDamageSource("")}}} {
		card := mage.NewArtifact("Invalid binding", "", mage.WithAbility(mage.Activate(mage.Costs(), plan.stackAbilityChoice(bindings))))
		if mage.ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("missing or empty binding accepted")
		}
	}
}
