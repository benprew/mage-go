package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSkipUntapRealCards(t *testing.T) {
	for _, tc := range []struct {
		set, name      string
		generic, green int
		tap            bool
	}{
		{"DRK", "Barl's Cage", 3, 0, false}, {"FEM", "Elvish Hunter", 1, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(c, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			target := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 2), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("unpaid activation accepted")
			}
			if g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Colorless, tc.generic)
			a.ManaPool().Add(core.Green, tc.green)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(source.ID()).Tapped != tc.tap || a.ManaPool().TotalMana() != 0 || g.FindPermanent(target.ID()).Tapped {
				t.Fatal("cost or action timing")
			}
			g.ResolveStack()
			if g.FindPermanent(target.ID()).Tapped {
				t.Fatal("restriction tapped target")
			}
			g.TapPermanent(g.FindPermanent(target.ID()))
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(target.ID()).Tapped {
				t.Fatal("inactive target untapped")
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(target.ID()).Tapped {
				t.Fatal("next untap not suppressed")
			}
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(target.ID()).Tapped {
				t.Fatal("restriction did not expire")
			}
		})
	}
}

func TestRONSkipUntapHomaridWarrior(t *testing.T) {
	for _, change := range []string{"none", "leave-return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "FEM", "Homarid Warrior")
			c.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(c, a.PlayerID())
			a.ManaPool().Add(core.Blue, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).Tapped || g.FindPermanent(p.ID()).HasKeyword(core.Shroud) {
				t.Fatal("actions ran before resolution")
			}
			g.ResolveStack()
			if !g.FindPermanent(p.ID()).Tapped || !g.FindPermanent(p.ID()).HasKeyword(core.Shroud) {
				t.Fatal("ordered source actions failed")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanent(p.ID()).HasKeyword(core.Shroud) {
				t.Fatal("temporary shroud did not expire")
			}
			switch change {
			case "leave-return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(c.ID())
				p = g.PutOnBattlefield(c, a.PlayerID())
				g.TapPermanent(p)
			case "phase":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.RunStepWithPriority(core.Untap)
				g.SetActivePlayerIndex(1)
			}
			clone := g.Clone()
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(p.ID()).Tapped != (change != "leave-return") {
				t.Fatal("source lifetime or next controller untap")
			}
			if !clone.FindPermanentIncludingPhased(p.ID()).Tapped {
				t.Fatal("untap changed clone")
			}
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(p.ID()).Tapped {
				t.Fatal("source skip did not expire")
			}
		})
	}
}

func TestRONSkipUntapTargetLifecycle(t *testing.T) {
	for _, change := range []string{"source-leaves", "target-returns", "phase-before", "phase-cycle", "phase-after", "control-after", "untapped-step", "other-untap", "repeat"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "DRK", "Barl's Cage")
			source := g.PutOnBattlefield(c, a.PlayerID())
			targetCard := mage.NewCreature("Target", "", 2, 2)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			g.TapPermanent(target)
			a.ManaPool().Add(core.Colorless, 6)
			activate := func() {
				t.Helper()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
			}
			activate()
			switch change {
			case "source-leaves":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			case "target-returns":
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				b.RemoveFromHand(targetCard.ID())
				target = g.PutOnBattlefield(targetCard, b.PlayerID())
				g.TapPermanent(target)
			case "phase-before":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "repeat":
				activate()
			}
			g.ResolveStack()
			switch change {
			case "phase-after":
				g.PhaseOut(target.ID())
			case "control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "untapped-step", "other-untap":
				g.UntapPermanent(g.FindPermanent(target.ID()))
			}
			if change == "other-untap" {
				if g.FindPermanent(target.ID()).Tapped {
					t.Fatal("restriction blocked other untap")
				}
				g.TapPermanent(g.FindPermanent(target.ID()))
			}
			if change != "control-after" {
				g.SetActivePlayerIndex(1)
			}
			g.RunStepWithPriority(core.Untap)
			want := change != "target-returns" && change != "phase-before" && change != "untapped-step"
			if g.FindPermanent(target.ID()).Tapped != want {
				t.Fatalf("tapped=%v want %v", g.FindPermanent(target.ID()).Tapped, want)
			}
			g.TapPermanent(g.FindPermanent(target.ID()))
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(target.ID()).Tapped {
				t.Fatal("restrictions did not expire together")
			}
		})
	}
}

func TestRONSkipUntapCopiesAndInstances(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "DRK", "Barl's Cage"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "DRK", "Barl's Cage"), a.PlayerID())
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 2), b.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 2), b.PlayerID())
	a.ManaPool().Add(core.Colorless, 6)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
		t.Fatal(err)
	}
	phase := g.PhaseOut(p.ID())
	if copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true); copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != q.ID() {
		t.Fatal("copy did not retarget independently")
	}
	g.PhaseIn(phase)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	g.TapPermanent(g.FindPermanent(p.ID()))
	g.TapPermanent(g.FindPermanent(q.ID()))
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(p.ID()).Tapped || !g.FindPermanent(q.ID()).Tapped {
		t.Fatal("instance or copy target lost")
	}
	if clone.StackSize() != 3 || clone.FindPermanent(p.ID()).Tapped || clone.FindPermanent(q.ID()).Tapped {
		t.Fatal("live resolution changed clone")
	}
	clone.ResolveStack()
	clone.TapPermanent(clone.FindPermanent(p.ID()))
	clone.TapPermanent(clone.FindPermanent(q.ID()))
	clone.SetActivePlayerIndex(1)
	clone.RunStepWithPriority(core.Untap)
	if !clone.FindPermanent(p.ID()).Tapped || !clone.FindPermanent(q.ID()).Tapped {
		t.Fatal("clone lost typed reads")
	}
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(p.ID()).Tapped || g.FindPermanent(q.ID()).Tapped {
		t.Fatal("copy/repeated restrictions stacked steps")
	}
}

func TestRONSkipUntapInvalid(t *testing.T) {
	for _, rule := range []string{
		`PermanentDoesntUntapDuringControllersNextUntap(ThisPermanent)`,
		`Activated(PermanentDoesntUntapDuringControllersNextUntap(ThisPermanent),ActionList([]))`,
		`Activated(PayMana([]),ActionList([PermanentDoesntUntapDuringControllersNextUntap]))`,
		`Activated(PayMana([]),ActionList([PermanentDoesntUntapDuringControllersNextUntap(ThisPermanent,ThisPermanent)]))`,
		`Activated(PayMana([]),ActionList([PermanentDoesntUntapDuringControllersNextUntap(ThisPermanent())]))`,
		`Activated(PayMana([]),ActionList([PermanentDoesntUntapDuringControllersNextUntap(HostPermanent)]))`,
		`Activated(PayMana([]),ActionList([PermanentDoesntUntapDuringControllersNextUntap(Ref_TargetPermanent)]))`,
		`Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PermanentDoesntUntapDuringControllersNextUntap(Ref_TargetPlayer)])))`,
		`Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([]))),Activated(PayMana([]),ActionList([PermanentDoesntUntapDuringControllersNextUntap(Ref_TargetPermanent)]))`,
		`SpellActions(ActionList([PermanentDoesntUntapDuringControllersNextUntap(ThisPermanent)]))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONSkipUntapInventory(t *testing.T) {
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(`Card(Name:"Inventory",Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([PermanentDoesntUntapDuringControllersNextUntap(Ref_TargetPermanent)])))])`)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range report.Signatures {
		if entry.Name == "Ref_TargetPermanent" && entry.Role == "permanent_reference" {
			return
		}
	}
	t.Fatal("missing typed permanent reference role")
}

func TestRONSkipUntapSharedContexts(t *testing.T) {
	for _, context := range []string{"spell", "trigger", "host", "departed-host-source"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := ronGame()
			target := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 2), b.PlayerID())
			action := `PermanentDoesntUntapDuringControllersNextUntap(Ref_TargetPermanent)`
			types := "Instant"
			rule := `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))`
			if context == "trigger" {
				types = "Artifact"
				rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))`
			}
			if strings.Contains(context, "host") {
				types = "Enchantment"
				rule = `EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),ActionList([PermanentDoesntUntapDuringControllersNextUntap(HostPermanent)]))`
			}
			var factory mage.CardFactory
			if strings.Contains(context, "host") {
				nodes, err := Parse(strings.NewReader(`Card(Name:"Host",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rule+`])`), "host.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err = Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
			} else {
				factory = permanentScopeFactory(t, types, rule)
			}
			c := factory()
			c.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			switch context {
			case "spell":
				a.AddToHand(c)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			case "trigger":
				g.PutOnBattlefield(c, a.PlayerID())
				g.PutTriggersOnStack()
			default:
				source := g.PutOnBattlefield(c, a.PlayerID())
				g.Attach(source.ID(), target.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if context == "departed-host-source" {
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					a.RemoveFromHand(c.ID())
					other := g.PutOnBattlefield(mage.NewCreature("Other host", "", 2, 2), a.PlayerID())
					returned := g.PutOnBattlefield(c, a.PlayerID())
					g.Attach(returned.ID(), other.ID())
				}
			}
			if g.StackSize() != 1 {
				t.Fatal("missing resolving action")
			}
			g.ResolveStack()
			g.TapPermanent(g.FindPermanent(target.ID()))
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(target.ID()).Tapped {
				t.Fatal("shared context lost its subject")
			}
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(target.ID()).Tapped {
				t.Fatal("shared context skip did not expire")
			}
		})
	}
}

func TestRONSkipUntapIllegalAnnouncementRollback(t *testing.T) {
	for _, scenario := range []string{"land", "phased", "shroud", "missing", "wrong-player", "sick"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Elvish Hunter"), a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			card := mage.NewCreature("Target", "", 2, 2)
			if scenario == "land" {
				card = mage.NewLand("Land")
			}
			target := g.PutOnBattlefield(card, b.PlayerID())
			id := target.ID()
			player := a.PlayerID()
			switch scenario {
			case "phased":
				g.PhaseOut(target.ID())
			case "shroud":
				target.GrantBaseAttr(core.Shroud)
			case "missing":
				id = uuid.New()
			case "wrong-player":
				player = b.PlayerID()
			case "sick":
				source.GrantBaseAttr(core.AttrSummonSick)
			}
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(player, source.ID(), 0, []uuid.UUID{id}); err == nil {
				t.Fatal("illegal activation accepted")
			}
			if g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 {
				t.Fatal("illegal activation changed costs or stack")
			}
		})
	}
}
