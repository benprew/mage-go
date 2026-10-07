package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNextUntapsTelekinesis(t *testing.T) {
	for _, change := range []string{"none", "return", "phase-before", "phase-after", "control", "shroud", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Telekinesis")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			creature := mage.NewCreature("Target", "", 2, 20)
			creature.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(creature, b.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 20), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("rollback")
			}
			a.ManaPool().Add(core.Blue, 2)
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{land.ID()}, 0); err == nil {
				t.Fatal("noncreature target accepted")
			}
			if a.ManaPool().TotalMana() != 2 || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("illegal target spent resources")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).Tapped {
				t.Fatal("action before resolution")
			}
			switch change {
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				b.RemoveFromHand(creature.ID())
				p = g.PutOnBattlefield(creature, b.PlayerID())
			case "phase-before":
				g.PhaseOut(p.ID())
			case "shroud":
				g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
			case "copy":
				ids := g.PhaseOut(p.ID())
				copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), true)
				if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != q.ID() {
					t.Fatal("copy targets")
				}
				g.PhaseIn(ids)
			}
			clone := g.Clone()
			g.ResolveStack()
			legal := change != "return" && change != "phase-before" && change != "shroud"
			if g.FindPermanentIncludingPhased(p.ID()).Tapped != legal {
				t.Fatal("tap legality")
			}
			if clone.FindPermanentIncludingPhased(p.ID()).Tapped {
				t.Fatal("clone changed")
			}
			if change == "phase-before" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			if sourceCombatPrevented(g, p.ID(), a.PlayerID()) != legal {
				t.Fatal("combat prevention")
			}
			if sourceCombatPrevented(g, q.ID(), a.PlayerID()) != (change == "copy") {
				t.Fatal("copy prevention")
			}
			g.RunStepWithPriority(core.Cleanup)
			if sourceCombatPrevented(g, p.ID(), a.PlayerID()) {
				t.Fatal("prevention expiry")
			}
			if change == "phase-after" {
				g.PhaseOut(p.ID())
			}
			if change == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			} else {
				g.SetActivePlayerIndex(1)
			}
			for step := range 3 {
				if live := g.FindPermanent(p.ID()); live != nil {
					g.TapPermanent(live)
				}
				if change == "copy" {
					g.TapPermanent(g.FindPermanent(q.ID()))
				}
				g.RunStepWithPriority(core.Untap)
				if change == "copy" && g.FindPermanent(q.ID()).Tapped != (step < 2) {
					t.Fatal("copy untap count")
				}
				if g.FindPermanent(p.ID()).Tapped != (legal && step < 2) {
					t.Fatalf("untap step %d", step)
				}
			}
		})
	}
}

func TestRONNextUntapsSporeCloud(t *testing.T) {
	g, a, b := ronGame()
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 20), a.PlayerID())
	blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 20), b.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 20), b.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
	spell := repositoryCard(t, "FEM", "Spore Cloud")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Green, 2)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(blocker.ID()).Tapped {
		t.Fatal("tap before resolution")
	}
	g.ResolveStack()
	if !g.FindPermanent(blocker.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
		t.Fatal("blocking tap")
	}
	for _, p := range []*mage.Permanent{attacker, blocker, other} {
		if !sourceCombatPrevented(g, p.ID(), a.PlayerID()) {
			t.Fatal("global combat prevention")
		}
	}
	late := g.PutOnBattlefield(mage.NewCreature("Late", "", 2, 20), a.PlayerID())
	g.DealDamageToPlayer(b, 1, attacker.ID())
	if b.Life() != 19 {
		t.Fatal("combat prevention blocked noncombat damage")
	}
	g.RunStepWithPriority(core.Cleanup)
	if sourceCombatPrevented(g, attacker.ID(), b.PlayerID()) {
		t.Fatal("global prevention survived cleanup")
	}
	clone := g.Clone()
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(attacker.ID()).Tapped {
		t.Fatal("attacker restriction")
	}
	g.TapPermanent(g.FindPermanent(late.ID()))
	g.SetActivePlayerIndex(1)
	g.TapPermanent(g.FindPermanent(other.ID()))
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(blocker.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
		t.Fatal("captured blockers")
	}
	if !clone.FindPermanent(blocker.ID()).Tapped {
		t.Fatal("clone changed")
	}
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(blocker.ID()).Tapped {
		t.Fatal("one-step expiry")
	}
	g.SetActivePlayerIndex(0)
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(attacker.ID()).Tapped || g.FindPermanent(late.ID()).Tapped {
		t.Fatal("captured attackers expiry or late member")
	}
}

func TestRONNextUntapsInvalidAndInventory(t *testing.T) {
	for _, body := range []string{
		`PermanentDoesntUntapDuringControllersNextNumberUntaps`,
		`PermanentDoesntUntapDuringControllersNextNumberUntaps(ThisPermanent)`,
		`PermanentDoesntUntapDuringControllersNextNumberUntaps(ThisPermanent, Integer(-1))`,
		`PermanentDoesntUntapDuringControllersNextNumberUntaps(ThisPermanent, ValueX)`,
		`PermanentDoesntUntapDuringControllersNextNumberUntaps(ThisPermanent, Integer(You))`,
		`PermanentDoesntUntapDuringControllersNextNumberUntaps(ThisPermanent(), Integer(2))`,
		`PermanentDoesntUntapDuringControllersNextNumberUntaps(Ref_TargetPermanent, Integer(2))`,
		`PermanentDoesntUntapDuringControllersNextNumberUntaps(Trigger_ThatPermanent, Integer(2))`,
		`PermanentDoesntUntapDuringControllersNextNumberUntaps(HostPermanent, Integer(2))`,
		`EachPermanentDoesntUntapDuringControllersNextUntap()`,
		`EachPermanentDoesntUntapDuringControllersNextUntap(And([]))`,
		`EachPermanentDoesntUntapDuringControllersNextUntap(Or([]))`,
		`EachPermanentDoesntUntapDuringControllersNextUntap(You)`,
		`EachPermanentDoesntUntapDuringControllersNextUntap(Ref_TargetPermanent)`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([%s]))])`, body)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", body)
		}
	}
	src := `Card(Name: "Probe", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([PermanentDoesntUntapDuringControllersNextNumberUntaps(ThisPermanent, Integer(2)), EachPermanentDoesntUntapDuringControllersNextUntap(Or([IsAttacking, IsBlocking]))]))])`
	report, err := InspectFS(fstest.MapFS{"probe.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"ThisPermanent": "permanent_reference", "Integer": "value", "Or": "permanent_filter", "IsAttacking": "permanent_filter", "IsBlocking": "permanent_filter"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}

func TestRONNextUntapsSharedContextsAndInstances(t *testing.T) {
	for _, context := range []string{"activation", "entry", "upkeep"} {
		t.Run(context, func(t *testing.T) {
			body := `ActionList([PermanentDoesntUntapDuringControllersNextNumberUntaps(ThisPermanent, Integer(2))])`
			rule := `Activated(PayMana([]), ` + body + `)`
			if context == "entry" {
				rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ` + body + `)`
			}
			if context == "upkeep" {
				rule = `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ` + body + `)`
			}
			factory := permanentScopeFactory(t, "Artifact", rule)
			g, a, b := ronGame()
			first := g.PutOnBattlefield(factory(), a.PlayerID())
			second := g.PutOnBattlefield(factory(), b.PlayerID())
			switch context {
			case "activation":
				if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			case "upkeep":
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			}
			g.ResolveStack()
			for step := range 3 {
				g.TapPermanent(g.FindPermanent(first.ID()))
				g.TapPermanent(g.FindPermanent(second.ID()))
				g.RunStepWithPriority(core.Untap)
				if g.FindPermanent(first.ID()).Tapped != (step < 2) {
					t.Fatal("source count")
				}
				g.SetActivePlayerIndex(1)
				g.RunStepWithPriority(core.Untap)
				if g.FindPermanent(second.ID()).Tapped != (context == "entry" && step < 2) {
					t.Fatal("shared source state")
				}
				g.SetActivePlayerIndex(0)
			}
		})
	}
}
