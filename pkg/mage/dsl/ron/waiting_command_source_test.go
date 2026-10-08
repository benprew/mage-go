package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingCommandSourceProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black}, {"LEA", "Circle of Protection: Blue", core.Blue},
		{"LEA", "Circle of Protection: Green", core.Green}, {"LEA", "Circle of Protection: Red", core.Red},
		{"LEA", "Circle of Protection: White", core.White}, {"ATQ", "Circle of Protection: Artifacts", core.White},
		{"LEG", "Greater Realm of Preservation", core.Black},
	} {
		for _, state := range []string{"live", "depart", "return", "hidden"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
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
				card := mage.NewArtifact("Command subject", map[core.Color]string{core.White: "{W}", core.Blue: "{U}", core.Black: "{B}", core.Red: "{R}", core.Green: "{G}"}[tc.color])
				visit, err := g.AddCommandObject(card, false)
				if err != nil {
					t.Fatal(err)
				}
				saved := g.CommandDamageSources()[0]
				seed := mage.BindDamageSource("seed")
				if err := mage.ApplyEffect(g, mage.Sequence(mage.ChooseDamageSourceFromCommandObjects(mage.You, mage.DamageSourceFilter{}, seed), mage.PreventNextDamageFromSelectedSource(mage.FromDamageSource(seed), mage.You, mage.DamageSourceFilter{})), uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}

				binding := mage.BindDamageSource("chosen")
				bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
				body := mage.Sequence(plan.waitingCommandChoice(bindings), shield(bindings))
				damage := mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(2))
				if err := mage.ValidateCardDefinition("Consumer", mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(), body)))); err != nil {
					t.Fatal(err)
				}
				for _, game := range []*mage.Game{g, g.Clone()} {
					if state == "depart" || state == "return" {
						removed, _ := game.RemoveCommandObject(visit)
						if state == "return" {
							if _, err := game.AddCommandObject(removed, false); err != nil {
								t.Fatal(err)
							}
						}
					}
					if err := mage.ApplyEffect(game, body, uuid.Nil, b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					switch state {
					case "return":
						game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, game.CommandDamageSources()[0])
					case "hidden":
						game.SetCommandObjectFaceDown(visit, true)
						game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, saved)
						game.SetCommandObjectFaceDown(visit, false)
					}
					want := 20
					if state == "return" || state == "hidden" {
						want--
					}
					game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 2, saved)
					if game.GetPlayer(b.PlayerID()).Life() != want {
						t.Fatal("choice prevention failed")
					}
					game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, saved)
					if game.GetPlayer(b.PlayerID()).Life() != want-1 {
						t.Fatal("shield did not consume")
					}

					game.GetEffects().ClearReplacementsEndOfTurn()
					for _, source := range game.CommandDamageSources() {
						game.RemoveCommandObject(source.CommandVisit())
					}
					if err := mage.ApplyEffect(game, mage.Sequence(body, damage), uuid.Nil, b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if game.GetPlayer(b.PlayerID()).Life() != want-1 {
						t.Fatal("empty choice retained previous binding")
					}
				}
				if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "unsupported action (ChooseADamageSource)") {
					t.Fatal("incomplete source enumeration accepted", err)
				}
				if err := mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), plan.waitingCommandChoice(instanceBindings{}))))); err == nil {
					t.Fatal("missing binding accepted")
				}
			})
		}
	}
}

func TestRONWaitingCommandSourceInvalid(t *testing.T) {
	for _, declaration := range []string{"ChooseADamageSource()", "ChooseADamageSource(IsColor(Colorless))", "ChooseADamageSource(And([]))", "ChooseADamageSource(Ref_TargetPermanent)"} {
		nodes, err := Parse(strings.NewReader(declaration), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileChosenDamageSourceChoice(nodes[0]); err == nil {
			t.Fatal("invalid source choice accepted", declaration)
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
	binding := mage.BindDamageSource("command source")
	bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
	consumer := mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1))
	for _, effect := range []mage.ResolvingEffect{mage.Sequence(consumer, plan.waitingCommandChoice(bindings)), plan.waitingCommandChoice(instanceBindings{}), mage.ChooseDamageSourceFromWaitingCommandReferences(nil, mage.DamageSourceFilter{}, binding)} {
		if mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), effect)))) == nil {
			t.Fatal("invalid binding declaration accepted")
		}
	}
}
func TestRONWaitingCommandSourceTimingAndCopies(t *testing.T) {
	record := staticBaseRecord(t, "LEA", "Circle of Protection: Red")
	rule := record.Fields["Rules"].Args[0]
	costs, _, err := compileCosts(rule.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	actions := rule.Args[1].Args[0].Args
	plan, err := compileChosenDamageSourceChoice(actions[0])
	if err != nil {
		t.Fatal(err)
	}
	shield, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
	if err != nil {
		t.Fatal(err)
	}
	_, a, b := ronGame()
	chooser := &ronSourceChoicePlayer{BasePlayer: b}
	g := mage.NewGame(a, chooser)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	binding := mage.BindDamageSource("command source")
	bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
	nativeCosts := make([]mage.Cost, len(costs))
	for i, cost := range costs {
		nativeCosts[i] = cost()
	}
	fixture := mage.NewEnchantment("Command choice fixture", "", mage.WithAbility(mage.Activate(mage.Costs(nativeCosts...), mage.Sequence(plan.waitingCommandChoice(bindings), shield(bindings)))))
	permanent := g.PutOnBattlefield(fixture, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), permanent.ID(), 0, nil); err == nil || g.StackSize() != 0 || chooser.calls != 0 {
		t.Fatal("unpaid activation changed choices or stack")
	}
	b.ManaPool().Add(core.Red, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), permanent.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 0 {
		t.Fatal("source choice ran at activation")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	card := mage.NewArtifact("Later command object", "{R}")
	visit, err := g.AddCommandObject(card, false)
	if err != nil {
		t.Fatal(err)
	}

	saved := g.CommandDamageSources()[0]
	seed := mage.BindDamageSource("seed")
	if err := mage.ApplyEffect(g, mage.Sequence(mage.ChooseDamageSourceFromCommandObjects(mage.You, mage.DamageSourceFilter{}, seed), mage.PreventNextDamageFromSelectedSource(mage.FromDamageSource(seed), mage.You, mage.DamageSourceFilter{})), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveCommandObject(visit)
	clone := g.Clone()
	g.ResolveStack()
	if chooser.calls != 2 {
		t.Fatal("copies did not choose at resolution")
	}
	for _, game := range []*mage.Game{g, clone} {
		if game == clone {
			game.ResolveStack()
		}
		for range 2 {
			game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 2, saved)
		}
		if game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("copied choices shared shield consumption")
		}
		game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, saved)
		if game.GetPlayer(b.PlayerID()).Life() != 19 {
			t.Fatal("copied shields did not consume independently")
		}
	}
}
