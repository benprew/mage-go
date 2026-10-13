package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func staticBaseRecord(t *testing.T, set, name string) Node {
	t.Helper()
	return repositoryRecord(t, set, name, set+".lines.ron")
}

// Test fixtures keep exact subrules because the complete cards still have gaps.
func staticBaseSubrule(t *testing.T, record Node, rules ...Node) mage.CardFactory {
	t.Helper()
	record.Fields["Rules"] = Node{Kind: List, Args: rules}
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(record.Fields["Name"].String, factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONStaticBaseValueRealBranches(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := staticBaseRecord(t, set, "Gaea's Liege")
			factory := staticBaseSubrule(t, record, record.Fields["Rules"].Args[0])
			g, a, b := ronGame()
			for range 2 {
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid source cast")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed payment changed state")
			}
			a.RemoveFromHand(card.ID())
			castRONSpell(t, g, a, card)
			pending := g.Clone()
			pending.ResolveStack()
			checkConditionalPT(t, pending, card.ID(), 2, 2)
			g.ResolveStack()
			checkConditionalPT(t, g, card.ID(), 2, 2)
			other := g.PutOnBattlefield(factory(), b.PlayerID())
			checkConditionalPT(t, g, other.ID(), 0, 0)
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
			checkConditionalPT(t, g, other.ID(), 1, 1)
			clone := g.Clone()
			phased := g.PhaseOut(land.ID())
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, other.ID(), 0, 0)
			checkConditionalPT(t, clone, other.ID(), 1, 1)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			checkConditionalPT(t, g, card.ID(), 1, 1)
			g.MutablePermanent(card.ID()).AddCounter(core.P1P1, 1)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(2), mage.Fixed(4)), mage.SwitchPowerToughness(mage.This)), card.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, card.ID(), 6, 4)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, card.ID(), 2, 2)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), card.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, card.ID(), 1, 1)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, card.ID(), 2, 2)
			g.MutablePermanent(card.ID()).RevokeBaseAttr(core.AttrSummonSick)
			g.SetActivePlayerIndex(1)
			g.SetStep(core.DeclareAttackers)
			g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{card.ID()})
			if !g.GetCombat().IsAttacking(card.ID()) {
				t.Fatal("attack not declared")
			}
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, card.ID(), 1, 1)
		})
	}
	record := staticBaseRecord(t, "DRK", "Angry Mob")
	branch := record.Fields["Rules"].Args[1]
	for i, want := range []int{4, 2} {
		g, a, b := ronGame()
		for range 2 {
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), b.PlayerID())
		}
		factory := staticBaseSubrule(t, record, branch.Args[i+1].Args[0])
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		checkConditionalPT(t, g, p.ID(), want, want)
	}
}

func TestRONStaticBaseValueHost(t *testing.T) {
	record := staticBaseRecord(t, "LEB", "Animate Artifact")
	rules := record.Fields["Rules"].Args
	layer := rules[1].Args[1].Args[0]
	layer.Args[1].Args = layer.Args[1].Args[2:]
	factory := staticBaseSubrule(t, record, rules[0], layer)
	g, a, b := ronGame()
	hostCard := mage.NewCreature("Host", "{3}", 1, 2, mage.WithCardType(core.TypeArtifact))
	hostCard.SetOwner(b.PlayerID())
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	card := factory()
	castRONSpell(t, g, a, card, host.ID())
	g.ResolveStack()
	checkConditionalPT(t, g, host.ID(), 3, 3)
	other := g.PutOnBattlefield(mage.NewCreature("Other", "{5}", 2, 4, mage.WithCardType(core.TypeArtifact)), a.PlayerID())
	clone := g.Clone()
	g.Attach(card.ID(), other.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, host.ID(), 1, 2)
	checkConditionalPT(t, g, other.ID(), 5, 5)
	checkConditionalPT(t, clone, host.ID(), 3, 3)
	phased := g.PhaseOut(card.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, other.ID(), 2, 4)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, other.ID(), 5, 5)
	g.Attach(card.ID(), host.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, host.ID(), 3, 3)
	g.BouncePermanentToHand(g.FindPermanent(host.ID()))
	b.RemoveFromHand(hostCard.ID())
	g.PutOnBattlefield(hostCard, b.PlayerID())
	checkConditionalPT(t, g, host.ID(), 1, 2)
	g.Attach(card.ID(), other.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, other.ID(), 5, 5)
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	checkConditionalPT(t, g, other.ID(), 2, 4)
}

func TestRONStaticBaseValueInvalid(t *testing.T) {
	for _, body := range []string{
		`PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth()])`,
		`PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(Integer(2),Integer(3))])`,
		`PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(ValueX)])`,
		`PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(PowerOfPermanent(ThisPermanent))])`,
		`PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(ManaValueOfPermanent(Ref_TargetPermanent))])`,
		`PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(Plus(Integer(2)))])`,
		`PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(Integer(2)),AdjustPT(1,1)])`,
		`PermanentLayerEffect(HostPermanent,[SetPowerAndToughnessBoth(Integer(2))])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[SetPowerAndToughnessBoth(Integer(2))])`,
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(Integer(2)),AddAbility([Flying])])])`,
	} {
		assertConditionalRejected(t, body)
	}
}

func TestRONStaticBaseValueInventory(t *testing.T) {
	source := `Card(Name:"Base",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:2),Rules:[PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(Plus(Integer(2),ManaValueOfPermanent(ThisPermanent)))])])`
	files := fstest.MapFS{"base.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = registry.CreateCard("Base"); err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, sig := range report.Signatures {
		if sig.Role == "unknown" {
			t.Fatalf("unknown %s", sig.Name)
		}
	}
}

func TestRONStaticBaseValueArithmeticAndCondition(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(Plus(Integer(5),Minus(Twice(Integer(3)),HalfRoundedDown(Integer(-3)))))])])`), "arithmetic.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory := staticBaseSubrule(t, staticBaseRecord(t, "LEB", "Gaea's Liege"), nodes[0])
	g, a, _ := ronGame()
	card := factory()
	p := g.PutOnBattlefield(card, a.PlayerID())
	checkConditionalPT(t, g, p.ID(), 13, 13)
	for range 3 {
		g.ApplyContinuousEffects()
		checkConditionalPT(t, g, p.ID(), 13, 13)
	}
	clone := g.Clone()
	g.MutablePermanent(p.ID()).Tapped = true
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, p.ID(), card.Power(), card.Toughness())
	checkConditionalPT(t, clone, p.ID(), 13, 13)
	g.MutablePermanent(p.ID()).Tapped = false
	phased := g.PhaseOut(p.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, p.ID(), 13, 13)
}

func TestRONStaticBaseValueCompleteRecordsStayRejected(t *testing.T) {
	for _, tc := range []struct{ set, name, gap string }{
		{"LEB", "Gaea's Liege", "ControlledByPlayer"},
	} {
		if _, err := Compile(staticBaseRecord(t, tc.set, tc.name)); err == nil || !strings.Contains(err.Error(), tc.gap) {
			t.Fatalf("%s gap: %v", tc.name, err)
		}
	}
}
