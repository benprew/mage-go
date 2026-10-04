package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStaticUntapBasaltMonolith(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "LEB", "Basalt Monolith")
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	q := g.PutOnBattlefield(repositoryCard(t, "LEB", "Basalt Monolith"), b.PlayerID())
	g.SetStep(core.PrecombatMain)
	if !mage.IsActivatedManaAbility(c.Abilities()[1]) {
		t.Fatal("tap ability is not a mana ability")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 0 || a.ManaPool().Count(core.Colorless) != 3 || !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("mana did not resolve immediately")
	}
	g.TapPermanent(g.FindPermanent(q.ID()))
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(p.ID()).Tapped || !g.FindPermanent(q.ID()).Tapped {
		t.Fatal("untap step ignored restriction or inactive player")
	}
	g.SetStep(core.PrecombatMain)
	a.ManaPool().Clear()
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err == nil {
		t.Fatal("unpaid untap accepted")
	}
	if a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 || !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 || !g.FindPermanent(p.ID()).Tapped || g.StackSize() != 1 {
		t.Fatal("untap timing or payment")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.FindPermanent(p.ID()).Tapped || !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("clone resolution changed original")
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Tapped || !g.FindPermanent(q.ID()).Tapped {
		t.Fatal("untap redirected to another instance")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("restriction expired after one step")
	}
	if !g.UntapPermanent(g.FindPermanent(p.ID())) {
		t.Fatal("other untap effect prohibited")
	}
}

func TestRONStaticUntapBrassMan(t *testing.T) {
	for _, scenario := range []string{"pay", "decline", "unaffordable", "opponent", "control", "return", "phase", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline"}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			c := repositoryCard(t, "ARN", "Brass Man")
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.TapPermanent(p)
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(p.ID()).Tapped || a.offers != 0 {
				t.Fatal("untap step bypassed restriction or offered payment")
			}
			if scenario != "unaffordable" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			player := a.PlayerID()
			if scenario == "opponent" {
				player = b.PlayerID()
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: player})
			g.PutTriggersOnStack()
			if scenario == "opponent" {
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep triggered")
				}
				return
			}
			if g.StackSize() != 1 || a.offers != 0 {
				t.Fatal("payment must wait for resolution")
			}
			current := p.ID()
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(c.ID())
				current = g.PutOnBattlefield(c, a.PlayerID()).ID()
				g.TapPermanent(g.FindPermanent(current))
			case "phase":
				g.PhaseOut(p.ID())
			case "copy":
				b.ManaPool().Add(core.Colorless, 1)
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			wantTapped := scenario == "decline" || scenario == "unaffordable" || scenario == "return" || scenario == "phase"
			if g.FindPermanentIncludingPhased(current).Tapped != wantTapped {
				t.Fatal("paid branch or source incarnation")
			}
			if scenario != "decline" && scenario != "unaffordable" && a.ManaPool().TotalMana() != 0 {
				t.Fatal("trigger controller did not pay")
			}
			if scenario == "copy" && (b.offers != 1 || b.ManaPool().TotalMana() != 0) {
				t.Fatal("copied trigger did not use copy controller")
			}
			if scenario != "copy" && b.offers != 0 {
				t.Fatal("payment followed new source controller")
			}
			if !wantTapped {
				g.TapPermanent(g.FindPermanent(current))
				if scenario == "control" {
					g.SetActivePlayerIndex(1)
				}
				g.RunStepWithPriority(core.Untap)
				if !g.FindPermanent(current).Tapped {
					t.Fatal("paid untap erased static restriction")
				}
			}
		})
	}
}

func TestRONStaticUntapColossus(t *testing.T) {
	for _, scenario := range []string{"legal", "unpaid", "wrong-step", "wrong-turn", "wrong-controller", "phase", "return", "copy", "control"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "ATQ", "Colossus of Sardia")
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			if !p.HasAttr(core.Trample) || p.CurrentPower(g) != 9 || p.CurrentToughness(g) != 9 {
				t.Fatal("creature characteristics")
			}
			g.TapPermanent(p)
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("colossus untapped normally")
			}
			g.SetStep(core.Upkeep)
			payer := a
			if scenario == "wrong-step" {
				g.SetStep(core.PrecombatMain)
			}
			if scenario == "wrong-turn" {
				g.SetActivePlayerIndex(1)
			}
			if scenario == "wrong-controller" {
				payer = b
			}
			if scenario == "phase" {
				g.PhaseOut(p.ID())
			}
			amount := 9
			if scenario == "unpaid" {
				amount = 8
			}
			payer.ManaPool().Add(core.Colorless, amount)
			err := g.ActivateAbilityByIndex(payer.PlayerID(), p.ID(), 1, nil)
			if scenario == "unpaid" || scenario == "wrong-step" || scenario == "wrong-turn" || scenario == "wrong-controller" || scenario == "phase" {
				if err == nil {
					t.Fatal("illegal activation succeeded")
				}
				if payer.ManaPool().TotalMana() != amount || g.StackSize() != 0 || !g.FindPermanentIncludingPhased(p.ID()).Tapped {
					t.Fatal("illegal activation changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if payer.ManaPool().TotalMana() != 0 || g.StackSize() != 1 || !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("announcement or payment")
			}
			current := p.ID()
			switch scenario {
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(c.ID())
				current = g.PutOnBattlefield(c, a.PlayerID()).ID()
				g.TapPermanent(g.FindPermanent(current))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.SetStep(core.EndStep)
			g.SetActivePlayerIndex(1)
			g.ResolveStack()
			if g.FindPermanent(current).Tapped != (scenario == "return") {
				t.Fatal("resolution timing or incarnation")
			}
		})
	}
}

func TestRONStaticUntapLifecycle(t *testing.T) {
	for _, scenario := range []string{"control", "phase-in", "remove-abilities", "return"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEB", "Basalt Monolith")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Basalt Monolith"), a.PlayerID())
			g.TapPermanent(p)
			g.TapPermanent(other)
			current := p.ID()
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.SetActivePlayerIndex(1)
			case "phase-in":
				g.PhaseOut(p.ID())
			case "remove-abilities":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(c.ID())
				current = g.PutOnBattlefield(c, a.PlayerID()).ID()
				g.TapPermanent(g.FindPermanent(current))
			}
			clone := g.Clone()
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(current).Tapped != (scenario != "remove-abilities") {
				t.Fatal("static restriction lifecycle")
			}
			if !g.FindPermanent(other.ID()).Tapped || !clone.FindPermanentIncludingPhased(current).Tapped {
				t.Fatal("another instance or clone changed")
			}
			if scenario == "remove-abilities" {
				g.RunStepWithPriority(core.Cleanup)
				g.TapPermanent(g.FindPermanent(current))
				g.RunStepWithPriority(core.Untap)
				if !g.FindPermanent(current).Tapped {
					t.Fatal("restriction did not return at cleanup")
				}
			}
		})
	}
}

func TestRONStaticUntapInvalid(t *testing.T) {
	for _, rule := range []string{
		`PermanentRuleEffect(ThisPermanent)`,
		`PermanentRuleEffect(ThisPermanent, [])`,
		`PermanentRuleEffect(ThisPermanent, [DoesntUntapDuringControllersUntap()])`,
		`PermanentRuleEffect(ThisPermanent(), [DoesntUntapDuringControllersUntap])`,
		`PermanentRuleEffect(HostPermanent, [DoesntUntapDuringControllersUntap])`,
		`PermanentRuleEffect(Ref_TargetPermanent, [DoesntUntapDuringControllersUntap])`,
		`PermanentRuleEffect(You, [DoesntUntapDuringControllersUntap])`,
		`PermanentRuleEffect(ThisPermanent, [DoesntUntapDuringControllersUntap, MayChooseNotToUntapDuringUntap])`,
		`Activated(PayMana([]), ActionList([DoesntUntapDuringControllersUntap]))`,
		`Activated(PermanentRuleEffect(ThisPermanent, [DoesntUntapDuringControllersUntap]), ActionList([UntapPermanent(ThisPermanent)]))`,
		`If(PermanentPassesFilter(ThisPermanent, IsTapped), [PermanentRuleEffect(ThisPermanent, [DoesntUntapDuringControllersUntap])])`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid", Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid spell", Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[PermanentRuleEffect(ThisPermanent,[DoesntUntapDuringControllersUntap])])`), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil {
		t.Fatal("accepted permanent rule on instant")
	}
}

func TestRONStaticUntapNativeUpkeep(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	c := repositoryCard(t, "ARN", "Brass Man")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.TapPermanent(p)
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("native untap ignored restriction")
	}
	a.ManaPool().Add(core.Colorless, 1)
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(p.ID()).Tapped || a.offers != 1 || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
		t.Fatal("native upkeep did not offer and resolve payment")
	}
}
