package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingEmptyLibrarySourceProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name, cost string
		amount          int
	}{
		{"LEB", "Circle of Protection: Black", "{B}", 1},
		{"LEA", "Circle of Protection: Blue", "{U}", 1},
		{"LEA", "Circle of Protection: Green", "{G}", 1},
		{"LEA", "Circle of Protection: Red", "{R}", 1},
		{"LEA", "Circle of Protection: White", "{W}", 1},
		{"ATQ", "Circle of Protection: Artifacts", "", 2},
		{"LEG", "Greater Realm of Preservation", "{B}", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := staticBaseRecord(t, tc.set, tc.name)
			activation := record.Fields["Rules"].Args[0]
			actions := activation.Args[1].Args[0].Args
			plan, err := compileChosenDamageSourceChoice(actions[0])
			if err != nil {
				t.Fatal(err)
			}
			shield, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
			if err != nil {
				t.Fatal(err)
			}
			binding := mage.BindDamageSource("empty-library source")
			bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
			costs, _, err := compileCosts(activation.Args[0])
			if err != nil {
				t.Fatal(err)
			}
			body := mage.Sequence(plan.waitingEmptyLibraryChoice(bindings), shield(bindings))
			protection := mage.NewEnchantment(tc.name, "", mage.WithAbility(mage.Activate(mage.Costs(costs[0]()), body)))
			if err := mage.ValidateCardDefinition(protection.Name(), protection); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			sourceCard := mage.NewArtifact("Replacement", tc.cost)
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(*mage.Game, uuid.UUID) { t.Fatal("source query ran callback") })
			saved := g.CaptureDamageSource(source.ID())
			p := g.PutOnBattlefield(protection, b.PlayerID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil || g.StackSize() != 0 {
				t.Fatal("unpaid activation changed stack")
			}
			if tc.set == "LEG" {
				b.ManaPool().Add(core.White, 1)
				b.ManaPool().Add(core.Colorless, 1)
			} else {
				b.ManaPool().Add(core.Colorless, tc.amount)
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			clone.ResolveStack()
			clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 2, saved)
			if clone.GetPlayer(b.PlayerID()).Life() != 20 || b.Life() != 20 || g.StackSize() != 2 {
				t.Fatal("clone shield changed original")
			}
			g.ResolveStack()
			g.DealDamageToPlayerFromSource(b, 2, saved)
			if b.Life() != 20 {
				t.Fatal("RON protection failed")
			}
			g.DealDamageToPlayerFromSource(b, 1, saved)
			if b.Life() != 20 {
				t.Fatal("independent copied shield missing")
			}
			g.DealDamageToPlayerFromSource(b, 1, saved)
			if b.Life() != 19 {
				t.Fatal("shield not consumed")
			}
			g.PhaseOut(source.ID())
			if err := mage.ApplyEffect(g, mage.Sequence(plan.waitingEmptyLibraryChoice(bindings), shield(bindings), mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1))), uuid.Nil, b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if b.Life() != 19 {
				t.Fatal("empty query retained a choice")
			}
			if _, err := Compile(record); err != nil {
				t.Fatal("complete source-choice record rejected", err)
			}
		})
	}
}

func TestRONWaitingEmptyLibrarySourceInvalid(t *testing.T) {
	for _, declaration := range []string{"ChooseADamageSource", "ChooseADamageSource()", "ChooseADamageSource(AnyDamageSource())", "ChooseADamageSource(IsColor(Colorless))", "ChooseADamageSource(IsColor(Red,Blue))", "ChooseADamageSource(And([]))", "ChooseADamageSource(ThisPermanent)"} {
		nodes, err := Parse(strings.NewReader(declaration), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileChosenDamageSourceChoice(nodes[0]); err == nil {
			t.Fatal("invalid declaration", declaration)
		}
	}
	plan := &chosenDamageSourcePlan{}
	binding := mage.BindDamageSource("empty-library")
	bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
	for _, effect := range []mage.Effect{
		plan.waitingEmptyLibraryChoice(instanceBindings{}),
		mage.Sequence(mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1)), plan.waitingEmptyLibraryChoice(bindings)),
	} {
		if mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), effect)))) == nil {
			t.Fatal("invalid binding accepted")
		}
	}
}

func TestRONWaitingEmptyLibrarySourceTimingAndRechecks(t *testing.T) {
	record := staticBaseRecord(t, "LEA", "Circle of Protection: Red")
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
	binding := mage.BindDamageSource("resolution source")
	bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
	_, a, b := ronGame()
	chooser := &ronSourceChoicePlayer{BasePlayer: b}
	g := mage.NewGame(a, chooser)
	protection := g.PutOnBattlefield(mage.NewEnchantment("Protection", "", mage.WithAbility(mage.Activate(mage.Costs(costs[0]()), mage.Sequence(plan.waitingEmptyLibraryChoice(bindings), shield(bindings))))), b.PlayerID())
	source := g.PutOnBattlefield(mage.NewArtifact("Source", "{R}"), a.PlayerID())
	b.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), protection.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 0 || len(g.WaitingEmptyLibraryPermanentSources()) != 0 {
		t.Fatal("choice ran at announcement")
	}
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(*mage.Game, uuid.UUID) { t.Fatal("source query ran callback") })
	g.ResolveStack()
	if chooser.calls != 1 || !chooser.source.SameObject(g.CaptureDamageSource(source.ID())) {
		t.Fatal("choice did not read resolution-time registration")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Blue)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayerFromSource(b, 1, g.CaptureDamageSource(source.ID()))
	if b.Life() != 19 {
		t.Fatal("property mismatch prevented damage")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Red)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		recipient := game.GetPlayer(b.PlayerID())
		saved := game.CaptureDamageSource(source.ID())
		game.DealSimultaneousDamage(mage.PlayerDamageAssignment(recipient, 2, saved), mage.PlayerDamageAssignment(recipient, 3, saved))
		if recipient.Life() != 19 {
			t.Fatal("one shield did not prevent the simultaneous event")
		}
		game.DealDamageToPlayerFromSource(recipient, 1, saved)
		if recipient.Life() != 18 {
			t.Fatal("matching damage did not consume shield")
		}
	}
}
