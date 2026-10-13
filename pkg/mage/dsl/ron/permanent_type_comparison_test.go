package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func typeComparisonRecord(t *testing.T, rules string, aura bool) Node {
	t.Helper()
	typeLine := `Cardtypes:[Artifact],Subtypes:[]`
	if aura {
		typeLine = `Cardtypes:[Enchantment],Subtypes:[Aura]`
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Type comparison",Typeline:(Supertypes:[],`+typeLine+`),ManaCost:[],Rules:[`+rules+`])`), "comparison.ron")
	if err != nil {
		t.Fatal(err)
	}
	return nodes[0]
}

func TestRONPermanentTypeComparisonResolution(t *testing.T) {
	for _, restricted := range []bool{false, true} {
		for _, change := range []string{"normal", "live-type", "source-return", "source-phase", "member-phase", "late-entry", "control", "clone", "copy"} {
			t.Run(change+map[bool]string{false: "/all", true: "/restricted"}[restricted], func(t *testing.T) {
				comparison := `SharesACardtypeWithPermanent(ThisPermanent)`
				if restricted {
					comparison = `SharesACardtypeWithPermanentFromAmongCardtypes(ThisPermanent,[Artifact,Creature,Land])`
				}
				f, err := Compile(typeComparisonRecord(t, `Activated(PayMana([ManaCostGeneric(1)]),ActionList([TapEachPermanent(And([`+comparison+`,ControlledByAPlayer(Opponent)]))]))`, false))
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				card := f()
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				artifact := g.PutOnBattlefield(mage.NewArtifact("Member", ""), b.PlayerID())
				creature := g.PutOnBattlefield(mage.NewCreature("Creature", "", 1, 5), b.PlayerID())
				land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || artifact.Tapped || g.StackSize() != 0 {
					t.Fatal("failed payment changed state")
				}
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if artifact.Tapped || creature.Tapped {
					t.Fatal("comparison ran at announcement")
				}
				wantArtifact, wantCreature := true, false
				switch change {
				case "live-type":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeCreature)), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					wantCreature = true
				case "source-return":
					g.BouncePermanentToHand(source)
					a.RemoveFromHand(card.ID())
					returned := g.PutOnBattlefield(card, a.PlayerID())
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeCreature)), returned.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "source-phase":
					g.PhaseOut(source.ID())
				case "member-phase":
					g.PhaseOut(artifact.ID())
					wantArtifact = false
				case "late-entry":
					artifact = g.PutOnBattlefield(mage.NewArtifact("Late", ""), b.PlayerID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: artifact.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					wantArtifact = false
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				if change == "clone" {
					clone.PhaseOut(artifact.ID())
				}
				g.ResolveStack()
				if clone.FindPermanentIncludingPhased(artifact.ID()).Tapped {
					t.Fatal("clone shares resolution")
				}
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					want := wantArtifact && (game != clone || change != "clone")
					if game.FindPermanentIncludingPhased(artifact.ID()).Tapped != want || game.FindPermanent(creature.ID()).Tapped != wantCreature || game.FindPermanent(land.ID()).Tapped {
						t.Fatal("wrong current membership or reference visit")
					}
				}
			})
		}
	}
}

func TestRONPermanentTypeComparisonTargetsAndHost(t *testing.T) {
	for _, host := range []bool{false, true} {
		ref := `Ref_TargetPermanent`
		body := `Targeted([TargetPermanent(AnyPermanent)],ActionList([TapEachPermanent(SharesACardtypeWithPermanent(` + ref + `))]))`
		if host {
			body = `ActionList([TapEachPermanent(SharesACardtypeWithPermanent(HostPermanent))])`
		}
		rules := `Activated(PayMana([ManaCostGeneric(1)]),` + body + `)`
		if host {
			rules = `EnchantPermanent(AnyPermanent),` + rules
		}
		f, err := Compile(typeComparisonRecord(t, rules, host))
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		card := f()
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		member := g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 5), b.PlayerID())
		other := g.PutOnBattlefield(mage.NewArtifact("Other", ""), b.PlayerID())
		var targets []uuid.UUID
		index := 0
		if host {
			g.Attach(source.ID(), member.ID())
		} else {
			targets = []uuid.UUID{member.ID()}
		}
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, targets); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if !member.Tapped || other.Tapped || source.Tapped {
			t.Fatal("wrong singular comparison binding")
		}
	}
}

func TestRONPermanentTypeComparisonInvalid(t *testing.T) {
	for _, expr := range []string{
		`SharesACardtypeWithPermanent`, `SharesACardtypeWithPermanent()`,
		`SharesACardtypeWithPermanent(ThisPermanent,ThisPermanent)`,
		`SharesACardtypeWithPermanent(Ref_TargetPermanent)`, `SharesACardtypeWithPermanent(HostPermanent)`,
		`SharesACardtypeWithPermanent(ActionPlayer)`, `SharesACardtypeWithPermanent(TheChosenPermanent)`,
		`SharesACardtypeWithPermanentFromAmongCardtypes(ThisPermanent,[])`,
		`SharesACardtypeWithPermanentFromAmongCardtypes(ThisPermanent,[Artifact,Artifact])`,
		`SharesACardtypeWithPermanentFromAmongCardtypes(ThisPermanent,[Aura])`,
		`SharesACardtypeWithPermanentFromAmongCardtypes(ThisPermanent,Artifact)`,
	} {
		n := typeComparisonRecord(t, `Activated(PayMana([ManaCostGeneric(1)]),ActionList([TapEachPermanent(`+expr+`)]))`, false)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", expr)
		}
	}
	for _, name := range []string{"Gauntlets of Chaos", "Enchantment Alteration"} {
		if _, err := Compile(staticBaseRecord(t, "LEG", name)); err == nil {
			t.Fatalf("incomplete record accepted: %s", name)
		}
	}
}

func TestRONPermanentTypeComparisonRepositoryExpressions(t *testing.T) {
	for _, name := range []string{"Gauntlets of Chaos", "Enchantment Alteration"} {
		t.Run(name, func(t *testing.T) {
			record := staticBaseRecord(t, "LEG", name)
			var expression Node
			var find func(Node)
			find = func(n Node) {
				if n.Name == "SharesACardtypeWithPermanent" || n.Name == "SharesACardtypeWithPermanentFromAmongCardtypes" {
					expression = n
				}
				for _, child := range n.Args {
					find(child)
				}
				for _, child := range n.Fields {
					find(child)
				}
			}
			find(record)
			if expression.Kind != Tuple {
				t.Fatal("missing repository comparison")
			}
			targetPlan := &permanentTargetPlan{name: "Ref_TargetPermanent1"}
			ctx := scope{permanent: true, aura: name == "Enchantment Alteration", permanentGroups: map[string]*permanentTargetPlan{"Ref_TargetPermanent1": targetPlan}}
			build, err := compilePermanentTypeComparison(expression, ctx)
			if err != nil {
				t.Fatal(err)
			}
			for _, invalid := range []scope{{permanent: true, enteringSource: true}, {permanent: true, drawReplacementBody: true}} {
				if _, err := compilePermanentTypeComparison(expression, invalid); err == nil {
					t.Fatal("nonresolving comparison accepted")
				}
			}
			g, a, b := ronGame()
			target := mage.DeclarePermanentTarget("first", mage.AnyPermanent)
			bindings := initializeActionBindings(instanceBindings{})
			bindings.permanents[targetPlan] = target
			ability := mage.Activate(mage.Costs(), mage.TapEach(build(bindings)), mage.WithTargets(target))
			card := mage.NewArtifact("Expression fixture", "", mage.WithAbility(ability))
			if ctx.aura {
				card = mage.NewEnchantment("Expression fixture", "", mage.WithSubTypes("Aura"), mage.WithAbility(ability))
			}
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			member := g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 5), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewLand("Other"), b.PlayerID())
			if ctx.aura {
				g.Attach(source.ID(), member.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{member.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if !member.Tapped || other.Tapped || source.Tapped {
				t.Fatal("repository expression used wrong type or binding")
			}
		})
	}
}

func TestRONPermanentTypeComparisonDelayedChoice(t *testing.T) {
	for _, change := range []string{"normal", "source-departure", "choice-return", "choice-phase", "copy"} {
		t.Run(change, func(t *testing.T) {
			f, err := Compile(typeComparisonRecord(t, `Activated(PayMana([]),ActionList([ChooseAPermanent(IsCardtype(Creature)),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(And([ControlledByAPlayer(Opponent),SharesACardtypeWithPermanent(TheChosenPermanent)]))]))]))`, false))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := f()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			chosenCard := mage.NewCreature("Chosen", "", 1, 5)
			chosenCard.SetOwner(a.PlayerID())
			chosen := g.PutOnBattlefield(chosenCard, a.PlayerID())
			member := g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 5), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewArtifact("Other", ""), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			switch change {
			case "source-departure":
				g.BouncePermanentToHand(source)
			case "choice-return":
				g.BouncePermanentToHand(chosen)
				a.RemoveFromHand(chosenCard.ID())
				returned := g.PutOnBattlefield(chosenCard, a.PlayerID())
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeArtifact)), returned.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "choice-phase":
				g.PhaseOut(chosen.ID())
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				for range 3 {
					game.WaitingDelayedPermanentSources()
					if game.FindPermanent(member.ID()).Tapped {
						t.Fatal("waiting query ran body")
					}
				}
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if !game.FindPermanent(member.ID()).Tapped || game.FindPermanent(other.ID()).Tapped {
					t.Fatal("delay lost choice or followed returned visit")
				}
			}
		})
	}
}

func TestRONPermanentTypeComparisonTargetLegality(t *testing.T) {
	for _, change := range []string{"normal", "shroud", "phase", "return", "second-illegal", "all-illegal"} {
		t.Run(change, func(t *testing.T) {
			f := singularGroupsFactory(t, `Targeted([TargetPermanent(IsCardtype(Creature)),TargetPermanent(IsCardtype(Artifact))],ActionList([TapEachPermanent(And([SharesACardtypeWithPermanent(Ref_TargetPermanent1),ControlledByAPlayer(Opponent)])),GainLife(Integer(1))]))`, false)
			g, a, b := ronGame()
			refCard := mage.NewCreature("Reference", "", 1, 5)
			refCard.SetOwner(a.PlayerID())
			ref := g.PutOnBattlefield(refCard, a.PlayerID())
			second := g.PutOnBattlefield(mage.NewArtifact("Second", ""), b.PlayerID())
			member := g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 5), b.PlayerID())
			card := f()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{ref.ID(), second.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			wantMember, wantLife := true, 21
			switch change {
			case "shroud":
				g.MutablePermanent(ref.ID()).GrantBaseAttr(core.Shroud)
				wantMember = false
			case "phase":
				g.PhaseOut(ref.ID())
				wantMember = false
			case "return":
				g.BouncePermanentToHand(ref)
				a.RemoveFromHand(refCard.ID())
				g.PutOnBattlefield(refCard, a.PlayerID())
				wantMember = false
			case "second-illegal":
				g.MutablePermanent(second.ID()).GrantBaseAttr(core.Shroud)
			case "all-illegal":
				g.MutablePermanent(ref.ID()).GrantBaseAttr(core.Shroud)
				g.MutablePermanent(second.ID()).GrantBaseAttr(core.Shroud)
				wantMember, wantLife = false, 20
			}
			g.ResolveStack()
			if member.Tapped != wantMember || a.Life() != wantLife {
				t.Fatal("illegal target supplied information or stopped independent actions")
			}
		})
	}
}
