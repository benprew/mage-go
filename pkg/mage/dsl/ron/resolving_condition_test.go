package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONResolvingConditionWerewolf(t *testing.T) {
	for _, change := range []string{"live", "zero", "lower-after-activation", "source-leaves", "source-returns", "target-leaves", "target-phase", "source-phase", "phase-cycle", "shroud-after", "ability-loss", "control", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Lesser Werewolf")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			victim := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 8), b.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID()})
			g.ExecuteBlockers([]mage.BlockAssignment{{AttackerID: source.ID(), BlockerID: victim.ID()}})
			g.SetStep(core.DeclareBlockers)
			a.ManaPool().Add(core.Black, 1)
			if change == "zero" {
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(-2), mage.Fixed(0))), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().Count(core.Black) != 0 || g.FindPermanent(victim.ID()).CurrentToughness(g) != 8 {
				t.Fatal("payment or early resolution")
			}
			switch change {
			case "lower-after-activation":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(-2), mage.Fixed(0))), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "source-leaves", "source-returns":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if change == "source-returns" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "target-leaves":
				g.BouncePermanentToHand(g.FindPermanent(victim.ID()))
			case "target-phase":
				g.PhaseOut(victim.ID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "shroud-after":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			case "ability-loss":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			if change == "clone" {
				clone.RemoveFromCombat(victim.ID())
			}
			g.ResolveStack()
			want := 8
			if change == "live" || change == "ability-loss" || change == "clone" {
				want = 7
			}
			if change == "copy" {
				want = 6
			}
			if p := g.FindPermanent(victim.ID()); p != nil && p.CurrentToughness(g) != want {
				t.Fatalf("victim toughness %d want %d", p.CurrentToughness(g), want)
			}
			if change == "live" || change == "ability-loss" || change == "clone" {
				checkConditionalPT(t, g, source.ID(), 1, 4)
				g.RunStepWithPriority(core.Cleanup)
				checkConditionalPT(t, g, source.ID(), 2, 4)
				checkConditionalPT(t, g, victim.ID(), 2, 7)
			}
			clone.ResolveStack()
			if change == "clone" {
				checkConditionalPT(t, clone, victim.ID(), 2, 8)
			}
		})
	}
}

func TestRONResolvingConditionEater(t *testing.T) {
	for _, change := range []string{"tapped", "untapped", "tap-after", "untap-after", "source-leaves", "source-returns", "source-phase", "target-leaves", "target-returns", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Eater of the Dead")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			victim := mage.NewCreature("Dead creature", "", 2, 2)
			victim.SetOwner(b.PlayerID())
			b.AddToGraveyard(victim)
			if change != "untapped" && change != "tap-after" {
				g.TapPermanent(source)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			if g.FindExiledCard(victim.ID()) != nil {
				t.Fatal("early exile")
			}
			switch change {
			case "tap-after":
				g.TapPermanent(g.FindPermanent(source.ID()))
			case "untap-after":
				g.UntapPermanent(g.FindPermanent(source.ID()))
			case "source-leaves", "source-returns":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if change == "source-returns" {
					a.RemoveFromHand(card.ID())
					g.TapPermanent(g.PutOnBattlefield(card, a.PlayerID()))
				}
			case "source-phase":
				g.PhaseOut(source.ID())
			case "target-leaves", "target-returns":
				moved, ok := g.MoveFromGraveyard(b.PlayerID(), victim.ID(), core.ZoneHand)
				if !ok {
					t.Fatal("move failed")
				}
				if change == "target-returns" {
					b.AddToGraveyard(moved)
				} else {
					b.AddToHand(moved)
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			if change == "clone" {
				clone.UntapPermanent(clone.FindPermanent(source.ID()))
			}
			g.ResolveStack()
			want := change != "untapped" && change != "untap-after" && change != "target-leaves" && change != "target-returns"
			if (g.FindExiledCard(victim.ID()) != nil) != want {
				t.Fatal("wrong conditional exile")
			}
			if p := g.FindPermanent(source.ID()); p != nil {
				tapped := change == "source-returns" || change == "target-leaves" || change == "target-returns"
				if p.Tapped != tapped {
					t.Fatal("wrong untap identity")
				}
			}
			clone.ResolveStack()
			if change == "clone" && clone.FindExiledCard(victim.ID()) != nil {
				t.Fatal("clone shared condition state")
			}
		})
	}
}

func TestRONResolvingConditionInvalid(t *testing.T) {
	for _, condition := range []string{
		`PermanentPassesFilter`, `PermanentPassesFilter()`, `PermanentPassesFilter(ThisPermanent(),IsTapped)`, `PermanentPassesFilter(Ref_TargetPermanent,IsTapped)`, `PermanentPassesFilter(HostPermanent,IsTapped)`, `PermanentPassesFilter(Trigger_ThatPermanent,IsTapped)`, `PermanentPassesFilter(ThisPermanent,And([]))`, `PermanentPassesFilter(ThisPermanent,Or([]))`, `PermanentPassesFilter(ThisPermanent,IsTapped())`, `PermanentPassesFilter(ThisPermanent,IsEnchanted)`, `PermanentPassesFilter(ThisPermanent,PowerIs(GreaterThanOrEqualTo(ValueX)))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[Activated(PayMana([]),ActionList([If(` + condition + `,[GainLife(Integer(1))])]))])`
		nodes, err := Parse(strings.NewReader(src), "condition.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid condition accepted", condition)
		}
	}
}

func TestRONResolvingConditionRollbackAndCasting(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		generic   int
	}{{"LEG", "Lesser Werewolf", 3}, {"DRK", "Eater of the Dead", 4}} {
		for _, paid := range []bool{false, true} {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if paid {
				a.ManaPool().Add(core.Colorless, tc.generic)
				a.ManaPool().Add(core.Black, 1)
			}
			err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
			if !paid {
				if err == nil || len(a.Hand()) != 1 || !g.Stack().IsEmpty() {
					t.Fatal("unpaid cast changed state")
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("casting payment or entry timing")
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("real RON creature did not enter")
			}
		}
	}
	for _, reason := range []string{"unpaid", "before-blockers", "wrong-step", "shroud", "phase", "wrong-type"} {
		g, a, b := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Lesser Werewolf"), a.PlayerID())
		victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 8), b.PlayerID())
		source.RevokeBaseAttr(core.AttrSummonSick)
		g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID()})
		if reason != "before-blockers" {
			g.ExecuteBlockers([]mage.BlockAssignment{{AttackerID: source.ID(), BlockerID: victim.ID()}})
		}
		g.SetStep(core.DeclareBlockers)
		if reason == "wrong-step" {
			g.SetStep(core.CombatDamage)
		}
		if reason == "shroud" {
			g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
		}
		if reason == "phase" {
			g.PhaseOut(victim.ID())
		}
		if reason == "wrong-type" {
			victim = g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
		}
		if reason != "unpaid" {
			a.ManaPool().Add(core.Black, 1)
		}
		before := a.ManaPool().TotalMana()
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err == nil {
			t.Fatal("invalid activation accepted", reason)
		}
		if a.ManaPool().TotalMana() != before || !g.Stack().IsEmpty() {
			t.Fatal("invalid activation changed state", reason)
		}
	}
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Eater of the Dead"), a.PlayerID())
	artifact := mage.NewArtifact("Dead artifact", "")
	artifact.SetOwner(b.PlayerID())
	b.AddToGraveyard(artifact)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{artifact.ID()}); err == nil {
		t.Fatal("noncreature target accepted")
	}
	if !g.Stack().IsEmpty() || len(b.Graveyard()) != 1 {
		t.Fatal("invalid graveyard target changed state")
	}
}

func TestRONResolvingConditionInstancesAndRetarget(t *testing.T) {
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetTurn(2)
	g.SetStep(core.PrecombatMain)
	source := `Card(Name:"Condition",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([If(PermanentPassesFilter(Ref_TargetPermanent,And([PowerIs(GreaterThanOrEqualTo(Integer(2))),DoesntHaveAbility(Flying)])),[TapPermanent(Ref_TargetPermanent),GainLife(Integer(1))])])))] )`
	nodes, err := Parse(strings.NewReader(source), "condition.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.SpellAbility).Targets()[0] == second.Abilities()[0].(*mage.SpellAbility).Targets()[0] {
		t.Fatal("factory shares target declarations")
	}
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 8), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 8), b.PlayerID())
	first.SetOwner(a.PlayerID())
	a.AddToHand(first)
	if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	b.targets = []uuid.UUID{q.ID()}
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), true)
	if copied == nil || copied.Targets[0] != q.ID() || original.Targets[0] != p.ID() {
		t.Fatal("copy changed original target")
	}
	clone := g.Clone()
	if err := mage.ApplyEffect(clone, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(1), mage.Fixed(0))), q.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	if !g.FindPermanent(p.ID()).Tapped || g.FindPermanent(q.ID()).Tapped || a.Life() != 21 || b.Life() != 20 {
		t.Fatal("condition lost per-instance targets or controller")
	}
	if !clone.FindPermanent(q.ID()).Tapped || clone.GetPlayer(b.PlayerID()).Life() != 21 {
		t.Fatal("clone lost current condition read")
	}
}

func TestRONResolvingConditionCombatAndInventory(t *testing.T) {
	record := hostUntapRecord(t, "PROMOS", "Sewers of Estark")
	actions := record.Fields["Rules"].Args[0].Args[0].Args[1].Args[0].Args
	for _, index := range []int{0, 1} {
		condition := actions[index].Args[0]
		conditionText := "IsAttacking"
		if index == 1 {
			conditionText = "IsBlocking"
		}
		source := `Card(Name:"Combat condition",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([If(PermanentPassesFilter(Ref_TargetPermanent,` + conditionText + `),[GainLife(Integer(2))])])))] )`
		nodes, err := Parse(strings.NewReader(source), "combat-condition.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes[0].Fields["Rules"].Args[0].Args[0].Args[1].Args[0].Args[0].Args[0] = condition
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		for _, exit := range []bool{false, true} {
			g, a, b := ronGame()
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 8), a.PlayerID())
			blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 8), b.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			g.ExecuteBlockers([]mage.BlockAssignment{{AttackerID: attacker.ID(), BlockerID: blocker.ID()}})
			target := attacker
			if index == 1 {
				target = blocker
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if exit {
				g.RemoveFromCombat(target.ID())
			}
			g.ResolveStack()
			want := 22
			if exit {
				want = 20
			}
			if a.Life() != want {
				t.Fatal("combat branch used stale membership")
			}
		}
	}
}

func TestRONResolvingConditionInventory(t *testing.T) {
	source := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([If(PermanentPassesFilter(Ref_TargetPermanent,PowerIs(GreaterThanOrEqualTo(Integer(1)))),[TapPermanent(Ref_TargetPermanent)])])))] )`
	report, err := InspectFS(fstest.MapFS{"condition.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	for _, signature := range report.Signatures {
		if signature.Name == "PermanentPassesFilter" && signature.Role != "condition" {
			t.Fatal("wrong condition role")
		}
		if signature.Name == "Ref_TargetPermanent" && signature.Role != "permanent_reference" {
			t.Fatal("wrong reference role")
		}
		if signature.Name == "PowerIs" && signature.Role != "permanent_filter" {
			t.Fatal("wrong property role")
		}
	}
}

func TestRONResolvingConditionRepeatedWerewolf(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Lesser Werewolf"), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 8), b.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{AttackerID: source.ID(), BlockerID: victim.ID()}})
	g.SetStep(core.DeclareBlockers)
	a.ManaPool().Add(core.Black, 3)
	for range 3 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	if a.ManaPool().TotalMana() != 0 || g.FindPermanent(victim.ID()).CurrentToughness(g) != 8 {
		t.Fatal("announcement evaluated the condition")
	}
	g.ResolveStack()
	checkConditionalPT(t, g, source.ID(), 0, 4)
	checkConditionalPT(t, g, victim.ID(), 2, 6)
}

func TestRONResolvingConditionRejectsConditionalManaActivation(t *testing.T) {
	for _, body := range []string{`AddMana(ManaProduceB)`, `If(PermanentPassesFilter(ThisPermanent,IsUntapped),[AddMana(ManaProduceB)])`} {
		source := `Card(Name:"Unsupported mana condition",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([If(PermanentPassesFilter(ThisPermanent,IsUntapped),[` + body + `])]))])`
		nodes, err := Parse(strings.NewReader(source), "mana-condition.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("conditional mana activation needs classification and solver planning")
		}
	}
}
