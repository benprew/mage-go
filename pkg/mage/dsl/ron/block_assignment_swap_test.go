package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONBlockAssignmentSwapSorrow(t *testing.T) {
	for _, change := range []string{"ordinary", "tapped", "flying", "target-return", "target-phase", "target-control", "attacker-death", "attacker-leaves", "clone", "source-leaves", "phase-cycle", "copy"} {
		t.Run(change, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			first := g.PutOnBattlefield(mage.NewCreature("First attacker", "", 2, 10), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second attacker", "", 3, 10), a.PlayerID())
			left := g.PutOnBattlefield(mage.NewCreature("First blocker", "", 1, 10), b.PlayerID())
			right := g.PutOnBattlefield(mage.NewCreature("Second blocker", "", 1, 10), b.PlayerID())
			for _, p := range []*mage.Permanent{first, second} {
				p.RevokeBaseAttr(core.AttrSummonSick)
			}
			card := repositoryCard(t, "DRK", "Sorrow's Path")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			g.SetStep(core.PrecombatMain)
			if err := g.PlayLand(a.PlayerID(), card.ID()); err != nil {
				t.Fatal(err)
			}
			land := g.FindPermanent(card.ID())
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
			g.SetStep(core.DeclareBlockers)
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: left.ID(), AttackerID: first.ID()}, {BlockerID: right.ID(), AttackerID: second.ID()}})

			if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, []uuid.UUID{left.ID(), left.ID()}); err == nil || land.Tapped || g.StackSize() != 0 {
				t.Fatal("duplicate targets paid tap cost")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, []uuid.UUID{first.ID(), right.ID()}); err == nil || land.Tapped {
				t.Fatal("own target paid tap cost")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, []uuid.UUID{left.ID(), right.ID()}); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			if !land.Tapped || g.StackSize() != 2 {
				t.Fatal("tap trigger not above activation")
			}
			switch change {
			case "tapped":
				g.TapPermanent(left)
			case "flying":
				second.GrantBaseAttr(core.Flying)
			case "target-return":
				g.RemoveFromBattlefield(left)
				g.PutOnBattlefield(left.Card, b.PlayerID())
			case "target-phase":
				g.PhaseOut(left.ID())
			case "target-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: land.ID(), TargetID: left.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "attacker-death":
				first.Damage = 8
			case "attacker-leaves":
				g.RemoveFromBattlefield(first)
			case "source-leaves":
				g.RemoveFromBattlefield(land)
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(left.ID()))
			}
			original := g
			if change == "clone" {
				g = g.Clone()
			}
			g.ResolveTopOfStack()
			if g.StackSize() != 1 {
				t.Fatal("tap trigger did not resolve before swap")
			}
			if change == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if g.GetCombat().GroupFor(second.ID()).BlockerIDs[0] != left.ID() {
					t.Fatal("copy did not swap")
				}
			}
			g.ResolveStack()
			if g.GetPlayer(a.PlayerID()).Life() != 18 {
				t.Fatal("tap trigger did not damage controller")
			}
			if change != "attacker-death" && change != "attacker-leaves" && g.FindPermanent(first.ID()).Damage != 2 {
				t.Fatal("tap trigger did not damage creatures")
			}
			want := change == "ordinary" || change == "clone" || change == "source-leaves" || change == "attacker-death" || change == "attacker-leaves"
			group := g.GetCombat().GroupFor(second.ID())
			swapped := false
			for _, id := range group.BlockerIDs {
				swapped = swapped || id == left.ID()
			}
			if swapped != want {
				t.Fatalf("swap = %v, want %v", swapped, want)
			}
			if (change == "attacker-death" || change == "attacker-leaves") && g.GetCombat().IsBlocking(right.ID()) {
				t.Fatal("empty assignment kept blocker in combat")
			}
			if g.FindPermanent(right.ID()).Damage != 0 || (change != "target-control" && g.FindPermanent(left.ID()) != nil && g.FindPermanent(left.ID()).Damage != 0) {
				t.Fatal("tap trigger damaged opposing creatures")
			}
			if change == "ordinary" {
				g.ExecuteCombatDamage()
				if g.FindPermanent(left.ID()).Damage != 3 || g.FindPermanent(right.ID()).Damage != 2 {
					t.Fatal("combat damage used the old assignments")
				}
			}
			if change == "clone" && (original.GetPlayer(a.PlayerID()).Life() != 20 || original.GetCombat().GroupFor(first.ID()).BlockerIDs[0] != left.ID()) {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestRONBlockAssignmentSwapInvalid(t *testing.T) {
	declaration := `NumberTargetGroupPermanents(Integer(2),And([IsBlocking,IsCardtype(Creature),ControlledByAPlayer(Opponent)]),ControlledByTheSamePlayer)`
	action := `TrySwappingBlockAssignmentsOfTwoBlockingCreatures(Ref_TargetPermanents_0,Ref_TargetPermanents_1)`
	for _, bad := range []struct{ declaration, action string }{
		{strings.Replace(declaration, "Integer(2)", "Integer(1)", 1), action},
		{strings.Replace(declaration, "Integer(2)", "ValueX", 1), action},
		{strings.Replace(declaration, "ControlledByTheSamePlayer", "ControlledByTheSamePlayer()", 1), action},
		{strings.Replace(declaration, "Opponent", "AnyPlayer", 1), action},
		{declaration, `TrySwappingBlockAssignmentsOfTwoBlockingCreatures()`},
		{declaration, `TrySwappingBlockAssignmentsOfTwoBlockingCreatures(Ref_TargetPermanents_0,Ref_TargetPermanents_0)`},
		{declaration, `TrySwappingBlockAssignmentsOfTwoBlockingCreatures(Ref_TargetPermanents_0,ThisPermanent)`},
		{declaration, `TrySwappingBlockAssignmentsOfTwoBlockingCreatures(Ref_TargetPermanents_0(),Ref_TargetPermanents_1)`},
		{`NumberTargetPermanents(Integer(2),IsCardtype(Creature))`, action},
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Land],Subtypes:[]),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([`+bad.declaration+`],ActionList([`+bad.action+`])))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid pair", bad)
		}
	}
}

func TestRONBlockAssignmentSwapLoadAndScopes(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, `Card(Name: "Sorrow's Path",`) {
			record = line
		}
	}
	files := fstest.MapFS{"DRK.ron": &fstest.MapFile{Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Sorrow's Path")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Sorrow's Path")
	if err != nil || first.ID() == second.ID() {
		t.Fatal("shared factory state")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	for name, role := range map[string]string{"ControlledByTheSamePlayer": "target_group_restriction", "Ref_TargetPermanents_0": "paired_target_member", "Ref_TargetPermanents_1": "paired_target_member"} {
		found := false
		for _, signature := range report.Signatures {
			found = found || signature.Name == name && signature.Role == role
		}
		if !found {
			t.Fatal("missing semantic role", name)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([MissingAction]))])`)}
	if LoadInto(registry, files, "*.ron") == nil {
		t.Fatal("published partial collection")
	}
	if _, err := registry.CreateCard("Sorrow's Path"); err != nil {
		t.Fatal("failed load erased registry")
	}
	action := `TrySwappingBlockAssignmentsOfTwoBlockingCreatures(Ref_TargetPermanents_0,Ref_TargetPermanents_1)`
	nodes, err := Parse(strings.NewReader(action), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	pair := &permanentTargetsPlan{name: "Ref_TargetPermanents"}
	for _, ctx := range []scope{{pairedTargets: pair, permanentTargets: pair}, {resolving: true, pairedTargets: pair, permanentTargets: pair, enteringSource: true}, {resolving: true, pairedTargets: pair, permanentTargets: pair, drawReplacementBody: true}, {resolving: true, pairedTargets: pair, permanentTargets: &permanentTargetsPlan{name: "other"}}} {
		if _, err := compileBlockAssignmentSwap(nodes[0], ctx); err == nil {
			t.Fatal("accepted invalid body context")
		}
	}
}

func TestRONBlockAssignmentSwapOtherTap(t *testing.T) {
	for _, change := range []string{"ordinary", "source-control", "phased-recipient", "replacement"} {
		t.Run(change, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Sorrow's Path"), a.PlayerID())
			own := g.PutOnBattlefield(mage.NewCreature("Own", "", 1, 10), a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 10), b.PlayerID())
			if change == "phased-recipient" {
				g.PhaseOut(own.ID())
			}
			if change == "replacement" {
				g.AddPreventionShield(own.ID(), 1)
			}
			g.TapPermanent(source)
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("external tap did not trigger")
			}
			if change == "source-control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			want := 2
			if change == "replacement" {
				want = 1
			}
			if change == "phased-recipient" {
				want = 0
			}
			own = g.FindPermanentIncludingPhased(own.ID())
			if a.Life() != 18 || b.Life() != 20 || own.Damage != want || g.FindPermanent(other.ID()).Damage != 0 {
				t.Fatal("wrong saved trigger controller or damage recipients")
			}
			g.TapPermanent(source)
			g.ResolveStack()
			if a.Life() != 18 || b.Life() != 20 {
				t.Fatal("already-tapped source triggered again")
			}
		})
	}
}

func TestRONBlockAssignmentSwapIndependentBindings(t *testing.T) {
	for _, sameSource := range []bool{false, true} {
		t.Run(map[bool]string{false: "two-factories", true: "two-activations"}[sameSource], func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			attackers, blockers := make([]*mage.Permanent, 4), make([]*mage.Permanent, 4)
			ids := make([]uuid.UUID, 4)
			assignments := make([]mage.BlockAssignment, 4)
			for i := range attackers {
				attackers[i] = g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 20), a.PlayerID())
				attackers[i].RevokeBaseAttr(core.AttrSummonSick)
				blockers[i] = g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 20), b.PlayerID())
				ids[i] = attackers[i].ID()
				assignments[i] = mage.BlockAssignment{BlockerID: blockers[i].ID(), AttackerID: ids[i]}
			}
			g.ExecuteAttackers(a.PlayerID(), ids)
			g.SetStep(core.DeclareBlockers)
			g.ExecuteBlockers(assignments)
			first := g.PutOnBattlefield(repositoryCard(t, "DRK", "Sorrow's Path"), a.PlayerID())
			second := first
			if !sameSource {
				second = g.PutOnBattlefield(repositoryCard(t, "DRK", "Sorrow's Path"), a.PlayerID())
			}
			for i, source := range []*mage.Permanent{first, second} {
				if i == 1 && sameSource {
					g.UntapPermanent(source)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{blockers[i*2].ID(), blockers[i*2+1].ID()}); err != nil {
					t.Fatal(err)
				}
				g.PutTriggersOnStack()
			}
			g.ResolveStack()
			for i, attacker := range attackers {
				if g.GetCombat().GroupFor(attacker.ID()).BlockerIDs[0] != blockers[i^1].ID() {
					t.Fatal("pending pairs shared target state")
				}
			}
			if a.Life() != 16 {
				t.Fatal("separate tap triggers were lost")
			}
		})
	}
}

func TestRONBlockAssignmentSwapResolvingScopes(t *testing.T) {
	body := `Targeted([NumberTargetGroupPermanents(Integer(2),And([IsBlocking,IsCardtype(Creature),ControlledByAPlayer(Opponent)]),ControlledByTheSamePlayer)],ActionList([TrySwappingBlockAssignmentsOfTwoBlockingCreatures(Ref_TargetPermanents_1,Ref_TargetPermanents_0)]))`
	for _, rule := range []string{`Activated(PayMana([]),` + body + `)`, `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),` + body + `)`} {
		factory := permanentScopeFactory(t, "Artifact", rule)
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Pair spell",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(`+body+`)])`), "spell.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err != nil {
		t.Fatal(err)
	}
}
