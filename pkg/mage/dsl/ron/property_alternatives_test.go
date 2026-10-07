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

func TestRONPropertyAlternativesBloodLust(t *testing.T) {
	for _, toughness := range []int{1, 2, 4, 5, 8} {
		for _, change := range []string{"live", "raise", "shroud", "return", "phase", "phase-cycle", "clone", "copy"} {
			t.Run(fmt.Sprintf("%d/%s", toughness, change), func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", "Blood Lust")
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				victim := mage.NewCreature("Victim", "", 2, toughness)
				victim.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(victim, b.PlayerID())
				a.ManaPool().Add(core.Red, 1)
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				checkConditionalPT(t, g, p.ID(), 2, toughness)
				base := toughness
				switch change {
				case "raise":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(0), mage.Fixed(4))), p.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					base += 4
				case "shroud":
					g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				case "return":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					b.RemoveFromHand(victim.ID())
					g.PutOnBattlefield(victim, b.PlayerID())
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				if change == "clone" {
					clone.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				}
				g.ResolveStack()
				power, result := 6, max(1, base-4)
				if change == "return" || change == "phase" || change == "shroud" {
					power, result = 2, toughness
				}
				if change == "copy" {
					power, result = 10, max(1, result-4)
				}
				if change == "phase" {
					g.PhaseIn([]uuid.UUID{p.ID()})
				}
				checkConditionalPT(t, g, p.ID(), power, result)
				clone.ResolveStack()
				if change == "clone" {
					checkConditionalPT(t, clone, p.ID(), 2, toughness)
				}
				g.RunStepWithPriority(core.Cleanup)
				checkConditionalPT(t, g, p.ID(), 2, toughness)
			})
		}
	}
}

func TestRONPropertyAlternativesErgRaiders(t *testing.T) {
	for _, change := range []string{"old", "new", "new-control-away", "haste-new", "attacked", "attacked-after", "opponent-step", "control", "control-back", "leave", "return", "phase-before", "phase-after", "phase-cycle", "clone", "copy", "prevent"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "Erg Raiders")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if change != "new" && change != "new-control-away" && change != "haste-new" {
				g.SetTurn(3)
			}
			if change == "haste-new" {
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Haste)
			}
			if change == "attacked" {
				p.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			}
			if change == "phase-before" {
				g.PhaseOut(p.ID())
			}
			if change == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			player := a.PlayerID()
			if change == "opponent-step" {
				player = b.PlayerID()
				g.SetActivePlayerIndex(1)
			}
			g.SetStep(core.EndStep)
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: player})
			g.PutTriggersOnStack()
			wantStack := 1
			if change == "attacked" || change == "opponent-step" || change == "phase-before" {
				wantStack = 0
			}
			if g.StackSize() != wantStack {
				t.Fatalf("stack %d want %d", g.StackSize(), wantStack)
			}
			switch change {
			case "attacked-after":
				p.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			case "control", "control-back", "new-control-away":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				if change == "control-back" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phase-after":
				g.PhaseOut(p.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "prevent":
				g.AddPreventionShield(a.PlayerID(), 2)
			}
			clone := g.Clone()
			if change == "clone" {
				clone.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				clone.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			want := 18
			if wantStack == 0 || change == "attacked-after" || change == "new" || change == "new-control-away" || change == "haste-new" || change == "control-back" || change == "prevent" {
				want = 20
			}
			if a.Life() != want {
				t.Fatalf("life %d want %d", a.Life(), want)
			}
			if change == "copy" && b.Life() != 18 {
				t.Fatal("copy lost controller")
			}
			clone.ResolveStack()
			cloneWant := want
			if change == "clone" {
				cloneWant = 20
			}
			if clone.GetPlayer(a.PlayerID()).Life() != cloneWant {
				t.Fatal("clone shared condition")
			}
		})
	}
}

func TestRONPropertyAlternativesInvalid(t *testing.T) {
	for _, action := range []string{
		`IfElse(PermanentPassesFilter(ThisPermanent,IsTapped),[],[GainLife(Integer(1))])`,
		`IfElse(PermanentPassesFilter(ThisPermanent,IsTapped),[GainLife(Integer(1))],[])`,
		`IfElse(PermanentPassesFilter(ThisPermanent,IsTapped),[GainLife(Integer(1))])`,
		`IfElse(PermanentPassesFilter(ThisPermanent,IsTapped),[GainLife(Integer(1))],[UntapPermanent(Ref_TargetPermanent)])`,
		`Unless(PermanentPassesFilter(Ref_TargetPermanent,IsTapped),[GainLife(Integer(1))])`,
		`Unless(PermanentPassesFilter(ThisPermanent,IsTapped),[])`,
		`Unless(PermanentPassesFilter(ThisPermanent,CameUnderControlOfPlayerThisTurn()),[GainLife(Integer(1))])`,
		`Unless(PermanentPassesFilter(ThisPermanent,CameUnderControlOfPlayerThisTurn(Trigger_ThatPlayer)),[GainLife(Integer(1))])`,
		`IfElse(PermanentPassesFilter(ThisPermanent,CameUnderControlOfPlayerThisTurn(You)),[GainLife(Integer(1))],[GainLife(Integer(2))])`,
		`Unless(PermanentPassesFilter(ThisPermanent,IsAttacking),[GainLife(Integer(1))])`,
		`Unless(PermanentPassesFilter(ThisPermanent,And([ControlledByAPlayer(SinglePlayer(You)),NotControlledSinceBeginningOfTurn()])),[GainLife(Integer(1))])`,
		`Unless(PermanentPassesFilter(ThisPermanent,And([ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer)),NotControlledSinceBeginningOfTurn])),[GainLife(Integer(1))])`,
		`Unless(PermanentPassesFilter(ThisPermanent,And([ControlledByAPlayer(SinglePlayer(You)),NotControlledSinceBeginningOfTurn,IsTapped])),[GainLife(Integer(1))])`,
		`Unless(PermanentPassesFilter(ThisPermanent,IsTapped),[AddMana(ManaProduceB)])`,
		`IfElse(PermanentPassesFilter(ThisPermanent,IsTapped),[AddMana(ManaProduceB)],[GainLife(Integer(1))])`,
		`IfElse(PermanentPassesFilter(ThisPermanent,IsTapped),[GainLife(Integer(1))],[AddMana(ManaProduceB)])`,
		`IfElse(PermanentPassesFilter(ThisPermanent,IsTapped),[MayCost(PayMana([])),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])],[Unless(CostWasPaid,[GainLife(Integer(3))])])`,
		`Unless(PermanentPassesFilter(ThisPermanent,IsTapped),[MayCost(PayMana([]))])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[Activated(PayMana([]),ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", action)
		}
	}
}

func TestRONPropertyAlternativesNestedAndInventory(t *testing.T) {
	src := `Card(Name:"Branches",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([IfElse(PermanentPassesFilter(Ref_TargetPermanent,IsTapped),[UntapPermanent(Ref_TargetPermanent),GainLife(Integer(2))],[TapPermanent(Ref_TargetPermanent),Unless(PermanentPassesFilter(Ref_TargetPermanent,IsUntapped),[GainLife(Integer(7))])])])))] )`
	registry, err := LoadFS(fstest.MapFS{"branches.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, tapped := range []bool{false, true} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(mage.NewArtifact("Subject", ""), a.PlayerID())
		if tapped {
			g.TapPermanent(p)
		}
		card, err := registry.CreateCard("Branches")
		if err != nil {
			t.Fatal(err)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := 27
		if tapped {
			want = 22
		}
		if a.Life() != want || g.FindPermanent(p.ID()).Tapped == tapped {
			t.Fatal("branch read repeated or nested condition read too early")
		}
	}
	report, err := InspectFS(fstest.MapFS{"branches.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	for _, signature := range report.Signatures {
		if (signature.Name == "IfElse" || signature.Name == "Unless") && signature.Role != "action" {
			t.Fatal("alternative role")
		}
		if signature.Name == "PermanentPassesFilter" && signature.Role != "condition" {
			t.Fatal("condition role")
		}
	}
}

func TestRONPropertyAlternativesPaymentAndRetarget(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		generic   int
		color     core.Color
	}{
		{"LEG", "Blood Lust", 1, core.Red}, {"ARN", "Erg Raiders", 1, core.Black},
	} {
		for _, paid := range []bool{false, true} {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			var targets []uuid.UUID
			if tc.name == "Blood Lust" {
				p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 5), a.PlayerID())
				targets = []uuid.UUID{p.ID()}
			}
			if paid {
				a.ManaPool().Add(core.Colorless, tc.generic)
				a.ManaPool().Add(tc.color, 1)
			}
			err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0)
			if !paid {
				if err == nil || !g.Stack().IsEmpty() || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("unpaid cast changed state")
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 {
				t.Fatal("announcement payment")
			}
			g.ResolveStack()
			if tc.name == "Erg Raiders" {
				checkConditionalPT(t, g, card.ID(), 2, 3)
			}
		}
	}
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetTurn(2)
	g.SetStep(core.PrecombatMain)
	first, second := repositoryCard(t, "LEG", "Blood Lust"), repositoryCard(t, "LEG", "Blood Lust")
	if first.Abilities()[0].(*mage.SpellAbility).Targets()[0] == second.Abilities()[0].(*mage.SpellAbility).Targets()[0] {
		t.Fatal("factory shares targets")
	}
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 5), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 2), b.PlayerID())
	first.SetOwner(a.PlayerID())
	a.AddToHand(first)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	b.targets = []uuid.UUID{q.ID()}
	if g.CopyStackObjectDirect(original, b.PlayerID(), true) == nil || original.Targets[0] != p.ID() {
		t.Fatal("copy changed original target")
	}
	g.ResolveStack()
	checkConditionalPT(t, g, p.ID(), 6, 1)
	checkConditionalPT(t, g, q.ID(), 7, 1)
}
