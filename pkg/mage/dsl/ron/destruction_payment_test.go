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

func TestRONDestructionPaymentCosmicHorror(t *testing.T) {
	for _, state := range []string{"destroy", "pay", "unaffordable", "regenerate", "indestructible", "absent", "returned", "phased", "phase-cycle", "control", "copy", "prevent"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: state == "pay" || state == "unaffordable"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "LEG", "Cosmic Horror")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if p.CurrentPower(g) != 7 || p.CurrentToughness(g) != 7 || !p.HasAttr(core.FirstStrike) {
				t.Fatal("metadata")
			}
			queueRONUpkeep(g, b)
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep")
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 || a.offers != 0 || a.Life() != 20 {
				t.Fatal("early resolution")
			}
			if state == "pay" {
				a.ManaPool().Add(core.Black, 3)
				a.ManaPool().Add(core.Colorless, 3)
			}
			if state == "unaffordable" {
				a.ManaPool().Add(core.Black, 2)
				a.ManaPool().Add(core.Colorless, 3)
			}
			switch state {
			case "regenerate":
				g.AddRegenerationShield(p.ID())
			case "indestructible":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Indestructible)
			case "absent", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			case "prevent":
				g.AddPreventionShield(a.PlayerID(), 2)
			}
			clone := g.Clone()
			g.ResolveStack()
			want := 13
			switch state {
			case "pay", "regenerate", "indestructible", "absent", "returned", "phased", "copy":
				want = 20
			case "prevent":
				want = 15
			}
			if a.Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
			if state == "copy" && b.Life() != 13 {
				t.Fatal("copy controller or stale destruction result")
			}
			survives := state == "pay" || state == "regenerate" || state == "indestructible" || state == "returned"
			if (g.FindPermanent(p.ID()) != nil) != survives {
				t.Fatal("wrong destruction outcome")
			}
			if state == "phased" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("destroyed phased source")
				}
			}
			if state == "pay" && a.ManaPool().TotalMana() != 0 {
				t.Fatal("wrong upkeep payment")
			}
			if state == "unaffordable" && a.ManaPool().TotalMana() != 5 {
				t.Fatal("failed payment spent mana")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.StackSize() == 0 {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			cloneWant := want
			if state == "pay" {
				cloneWant = 13
			}
			if clone.GetPlayer(a.PlayerID()).Life() != cloneWant {
				t.Fatal("clone result")
			}
		})
	}
}

func TestRONDestructionPaymentNestedResults(t *testing.T) {
	for _, success := range []bool{false, true} {
		g, a, b := ronGame()
		card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([MustCost(DestroyPermanent(Ref_TargetPermanent)),If(CostWasPaid,[MustCost(DestroyPermanent(Ref_TargetPermanent)),If(CostWasPaid,[GainLife(Integer(100))]),GainLife(Integer(2))])])) )`)()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		target := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 2), b.PlayerID())
		if !success {
			target.GrantBaseAttr(core.Indestructible)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := 20
		if success {
			want = 22
		}
		if a.Life() != want {
			t.Fatal("nested destruction results interfered")
		}
	}
}

func TestRONDestructionPaymentInvalid(t *testing.T) {
	for _, actions := range []string{
		`MustCost(DestroyPermanent(ThisPermanent)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(DestroyPermanent(Ref_TargetPlayer)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(DestroyPermanent()),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(DestroyPermanent(ThisPermanent())),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(DestroyPermanent(Ref_TargetPermanent)),If(CostWasPaid,[])`,
		`MustCost(DestroyPermanent(Ref_TargetPermanent)),GainLife(Integer(1)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(DestroyPermanent(Ref_TargetPermanent)),IfElse(CostWasPaid,[GainLife(Integer(1))],[])`,
		`MayCost(DestroyPermanent(Ref_TargetPermanent())),If(CostWasPaid,[GainLife(Integer(1))])`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + actions + `])))])`
		nodes, err := Parse(strings.NewReader(source), "invalid-destruction.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", actions)
		}
	}
}

func TestRONDestructionPaymentTargetPolarity(t *testing.T) {
	for _, branch := range []struct {
		text string
		paid bool
	}{
		{`If(CostWasPaid,[GainLife(Integer(2))])`, true},
		{`Unless(CostWasntPaid,[GainLife(Integer(2))])`, true},
		{`Unless(CostWasPaid,[GainLife(Integer(2))])`, false},
		{`If(CostWasntPaid,[GainLife(Integer(2))])`, false},
	} {
		for _, change := range []string{"destroy", "regenerate", "indestructible", "returned", "phased", "shroud"} {
			t.Run(branch.text+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([MustCost(DestroyPermanent(Ref_TargetPermanent)),`+branch.text+`])))`)()
				targetCard := mage.NewCreature("Target", "", 2, 3)
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "regenerate":
					g.AddRegenerationShield(target.ID())
				case "indestructible":
					g.MutablePermanent(target.ID()).GrantBaseAttr(core.Indestructible)
				case "returned":
					g.RemoveFromBattlefield(target)
					g.PutOnBattlefield(targetCard, b.PlayerID())
				case "phased":
					g.PhaseOut(target.ID())
				case "shroud":
					g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
				}
				g.ResolveStack()
				want := 20
				legal := change == "destroy" || change == "regenerate" || change == "indestructible"
				if legal && branch.paid == (change == "destroy") {
					want = 22
				}
				if a.Life() != want {
					t.Fatalf("life=%d want=%d", a.Life(), want)
				}
			})
		}
	}
}

func TestRONDestructionPaymentCosmicNative(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEG", "Cosmic Horror")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 3)
	a.ManaPool().Add(core.Colorless, 2)
	if g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0) == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 5 || g.StackSize() != 0 {
		t.Fatal("casting rollback")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("early entry")
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("casting")
	}
	g.RunStepWithPriority(core.Upkeep)
	if a.Life() != 13 || g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 {
		t.Fatal("native upkeep destruction and damage")
	}
}

func TestRONDestructionPaymentInventoryAndStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Cosmic Horror"`) {
			source = line
		}
	}
	files := fstest.MapFS{"first.ron": {Data: []byte(source)}, "reprint.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("strict reprint load", err)
	}
	nodes, err := Parse(strings.NewReader(source), "real.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.ID() == second.ID() || first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared factory state")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 2 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, s := range report.Signatures {
		if s.Name == "ThisPermanent" && s.Role == "permanent_reference" && strings.Contains(s.Context, "DestroyPermanent") {
			found = true
		}
	}
	if !found {
		t.Fatal("destruction subject semantic role")
	}
	invalid := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MustCost(DestroyPermanent(Ref_TargetPermanent)),If(CostWasPaid,[GainLife(Integer(1))])]))])`
	if LoadInto(registry, fstest.MapFS{"invalid.ron": {Data: []byte(source + "\n" + invalid)}}, "*.ron") == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid load replaced registry")
	}
	infinite := hostUntapRecord(t, "LEG", "Infinite Authority")
	if _, err = Compile(infinite); err == nil || !strings.Contains(err.Error(), "Trigger_ThatCreature") {
		t.Fatal("unsupported nested event binding", err)
	}
}

func TestRONDestructionPaymentDelayedActivation(t *testing.T) {
	g, a, _ := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([MustCost(DestroyPermanent(ThisPermanent)),If(CostWasPaid,[GainLife(Integer(2))])]))]))`)
	card := factory()
	card.SetOwner(a.PlayerID())
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()) == nil || a.Life() != 20 {
		t.Fatal("destruction before delayed event")
	}
	clone := g.Clone()
	g.RunStepWithPriority(core.EndStep)
	if g.FindPermanent(p.ID()) != nil || a.Life() != 22 {
		t.Fatal("delayed result")
	}
	if clone.FindPermanent(p.ID()) == nil || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone isolation")
	}
	clone.BouncePermanentToHand(clone.FindPermanent(p.ID()))
	clone.GetPlayer(a.PlayerID()).RemoveFromHand(card.ID())
	clone.PutOnBattlefield(card, a.PlayerID())
	clone.RunStepWithPriority(core.EndStep)
	if clone.FindPermanent(p.ID()) == nil || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("delayed result followed returned source")
	}
}
