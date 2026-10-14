package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const permanentLoopPayment = `ActionForEachPermanent(IsCardtype(Creature),[PlayerMayCost(ControllerOfPermanent(EachablePermanent),PayMana([ManaCostGeneric(1)])),Unless(CostWasPaid,[TapPermanent(EachablePermanent)])])`

func TestRONPermanentLoopPayment(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+permanentLoopPayment+`]))`)
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 10), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 10), a.PlayerID())
	third := g.PutOnBattlefield(mage.NewCreature("Third", "", 1, 10), b.PlayerID())
	phased := g.PutOnBattlefield(mage.NewCreature("Phased", "", 1, 10), b.PlayerID())
	g.PhaseOut(phased.ID())
	a.ManaPool().Add(core.White, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if first.Tapped || second.Tapped || third.Tapped || a.ManaPool().TotalMana() != 1 {
		t.Fatal("loop ran before resolution")
	}
	clone := g.Clone()
	clone.SetPlayerAt(0, &ronRepeatedPaymentClonePlayer{clone.GetPlayer(a.PlayerID())})
	clone.SetPlayerAt(1, &ronRepeatedPaymentClonePlayer{clone.GetPlayer(b.PlayerID())})
	for _, game := range []*mage.Game{g, clone} {
		game.ResolveStack()
		if game.FindPermanent(first.ID()).Tapped || !game.FindPermanent(second.ID()).Tapped || !game.FindPermanent(third.ID()).Tapped || game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
			t.Fatalf("member payment or controller crossed iterations: first=%v second=%v third=%v mana=%d player=%T", game.FindPermanent(first.ID()).Tapped, game.FindPermanent(second.ID()).Tapped, game.FindPermanent(third.ID()).Tapped, game.GetPlayer(a.PlayerID()).ManaPool().TotalMana(), game.GetPlayer(a.PlayerID()))
		}
		game.PhaseIn([]uuid.UUID{phased.ID()})
		if game.FindPermanent(phased.ID()).Tapped {
			t.Fatal("phased permanent entered the collection")
		}
	}
}

func TestRONPermanentLoopNested(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([ActionForEachPermanent(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]),[ActionForEachPermanent(And([IsCardtype(Creature),ControlledByAPlayer(Opponent)]),[TapPermanent(EachablePermanent)]),UntapPermanent(EachablePermanent)])]))`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	outer := g.PutOnBattlefield(mage.NewCreature("Outer", "", 1, 10), a.PlayerID())
	outer.Tapped = true
	inner := g.PutOnBattlefield(mage.NewCreature("Inner", "", 1, 10), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if outer.Tapped || !inner.Tapped {
		t.Fatal("nested member shadowed its caller")
	}
}

func TestRONPermanentLoopInvalid(t *testing.T) {
	for _, body := range []string{
		`ActionForEachPermanent`, `ActionForEachPermanent()`, `ActionForEachPermanent(AnyPermanent)`,
		`ActionForEachPermanent(AnyPermanent,[],[])`, `ActionForEachPermanent(And([]),[])`,
		`ActionForEachPermanent(EachablePermanent,[])`, `ActionForEachPermanent(AnyPermanent,TapPermanent(EachablePermanent))`,
		`ActionForEachPermanent(AnyPermanent,[TapPermanent(EachablePermanent())])`,
		`ActionForEachPermanent(AnyPermanent,[TapPermanent(Ref_TargetPermanent)])`,
		`ActionForEachPermanent(AnyPermanent,[CreatePermanentLayerEffect(EachablePermanent,[AddAbility([Activated(PayMana([]),ActionList([TapPermanent(EachablePermanent)]))])])])`,
		`ActionForEachPermanent(AnyPermanent,[PlayerMayCost(ControllerOfPermanent(EachablePermanent),PayMana([]))])`,
		`ActionForEachPermanent(AnyPermanent,[]),TapPermanent(EachablePermanent)`,
		`TapPermanent(EachablePermanent),ActionForEachPermanent(AnyPermanent,[])`,
		`ActionForEachPermanent(AnyPermanent,[ChooseAPermanent(AnyPermanent)]),TapPermanent(TheChosenPermanent)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "loop.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid loop accepted", body)
		}
	}
	nodes, err := Parse(strings.NewReader(`ActionForEachPermanent(AnyPermanent,[])`), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("invalid loop context accepted")
		}
	}
}

func TestRONPermanentLoopTidalFlats(t *testing.T) {
	for _, state := range []string{"paid", "declined", "short", "mana-source", "flying", "not-attacking", "phased", "source-leaves", "copy", "foreign-copy", "control", "cleanup", "return-blocker", "pending-clone", "installed-clone"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state != "declined"}
			g := mage.NewGame(a, b)
			g.SetStep(core.DeclareBlockers)
			g.SetTurn(2)
			card := repositoryCard(t, "FEM", "Tidal Flats")
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 10), b.PlayerID())
			blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 10), a.PlayerID())
			unrelated := g.PutOnBattlefield(mage.NewCreature("Unrelated", "", 2, 10), a.PlayerID())
			g.GetCombat().AddAttackerIncarnation(attacker.ID(), g.CaptureDamageSource(attacker.ID()).Incarnation(), b.PlayerID())
			g.GetCombat().AddBlockerIncarnation(blocker.ID(), g.CaptureDamageSource(blocker.ID()).Incarnation(), attacker.ID())
			if state == "flying" {
				attacker.GrantBaseAttr(core.Flying)
			}
			if state == "not-attacking" {
				g.GetCombat().RemoveFromCombat(attacker.ID())
			}
			if state == "phased" {
				g.PhaseOut(attacker.ID())
			}
			a.ManaPool().Add(core.Blue, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("failed activation did not roll back")
			}
			a.ManaPool().Add(core.Blue, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if blocker.HasAttr(core.FirstStrike) || b.offers != 0 {
				t.Fatal("loop executed at announcement")
			}
			if state == "paid" {
				b.ManaPool().Add(core.Red, 1)
			}
			if state == "declined" {
				b.ManaPool().Add(core.Red, 1)
			}
			if state == "mana-source" {
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
			}
			if state == "source-leaves" {
				g.RemoveFromBattlefield(source)
			}
			if state == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if state == "copy" || state == "foreign-copy" {
				controller := a.PlayerID()
				if state == "foreign-copy" {
					controller = b.PlayerID()
				}
				if g.CopyStackObjectDirect(g.StackPeek(), controller, false) == nil {
					t.Fatal("ability copy failed")
				}
			}
			if state == "pending-clone" {
				g = g.Clone()
				g.SetPlayerAt(0, &ronRepeatedPaymentClonePlayer{g.GetPlayer(a.PlayerID())})
				g.SetPlayerAt(1, &counterPaymentClonePayer{Player: g.GetPlayer(b.PlayerID()), accept: true})
			}
			g.ResolveStack()
			g.ApplyContinuousEffects()
			want := state != "paid" && state != "mana-source" && state != "flying" && state != "not-attacking" && state != "phased"
			if g.FindPermanent(blocker.ID()).HasAttr(core.FirstStrike) != want || g.FindPermanent(unrelated.ID()).HasAttr(core.FirstStrike) {
				t.Fatal("wrong blocking pair or payment result")
			}
			if state == "paid" && b.ManaPool().TotalMana() != 0 {
				t.Fatal("payment did not spend mana")
			}
			if state == "declined" && b.ManaPool().TotalMana() != 1 {
				t.Fatal("declined payment spent mana")
			}
			if state == "installed-clone" {
				g = g.Clone()
				g.ApplyContinuousEffects()
				if !g.FindPermanent(blocker.ID()).HasAttr(core.FirstStrike) {
					t.Fatal("clone lost grant")
				}
			}
			if state == "return-blocker" {
				p := g.FindPermanent(blocker.ID())
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, a.PlayerID())
				g.ApplyContinuousEffects()
				if g.FindPermanent(blocker.ID()).HasAttr(core.FirstStrike) {
					t.Fatal("grant followed returned blocker")
				}
			}
			if state == "cleanup" {
				g.RunStepWithPriority(core.Cleanup)
				g.ApplyContinuousEffects()
				if g.FindPermanent(blocker.ID()).HasAttr(core.FirstStrike) {
					t.Fatal("grant survived cleanup")
				}
			}
		})
	}
}

func TestRONPermanentLoopDelayedMembers(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([ActionForEachPermanent(IsCardtype(Creature),[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapPermanent(EachablePermanent)]))])]))`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 10), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 10), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.RemoveFromBattlefield(first)
	g.PutOnBattlefield(first.Card, a.PlayerID())
	g.RemoveFromBattlefield(source)
	for _, game := range []*mage.Game{g, g.Clone()} {
		game.RunStepWithPriority(core.EndStep)
		if game.FindPermanent(first.ID()).Tapped || !game.FindPermanent(second.ID()).Tapped {
			t.Fatal("delayed iteration lost its own member visit")
		}
	}
}

func TestRONPermanentLoopInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Loop",Rules:[SpellActions(ActionList([`+permanentLoopPayment+`]))])`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := make(map[string]*InventorySignature)
	walkInventory(nodes[0], "Loop", "", "card", "", false, entries)
	found := false
	for _, entry := range entries {
		if entry.Name == "EachablePermanent" && entry.Role == "permanent_reference" {
			found = true
		}
	}
	if !found {
		t.Fatal("loop member lost its permanent reference role")
	}
}

func TestRONPermanentLoopRetainedController(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([ActionForEachPermanent(IsCardtype(Creature),[PutPermanentIntoItsOwnersHand(EachablePermanent),PlayerAction(ControllerOfPermanent(EachablePermanent),GainLife(Integer(1)))])]))`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("A member", "", 1, 10), a.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("B member", "", 1, 10), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("departed members lost their separate controllers")
	}
}

func TestRONPermanentLoopPaymentRollback(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([ActionForEachPermanent(IsCardtype(Creature),[PlayerMayCost(ControllerOfPermanent(EachablePermanent),PayMana([ManaCostR])),Unless(CostWasPaid,[TapPermanent(EachablePermanent)])])]))`)
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 10), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 10), a.PlayerID())
	a.ManaPool().Add(core.White, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !first.Tapped || !second.Tapped || a.ManaPool().TotalMana() != 1 {
		t.Fatal("failed member payments spent resources or kept a paid result")
	}
}

func TestRONPermanentLoopTidalFlatsMultiple(t *testing.T) {
	a := mage.NewBasePlayer("A")
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	g.SetStep(core.DeclareBlockers)
	source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Tidal Flats"), a.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First attacker", "", 1, 10), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second attacker", "", 1, 10), b.PlayerID())
	one := g.PutOnBattlefield(mage.NewCreature("First blocker", "", 1, 10), a.PlayerID())
	two := g.PutOnBattlefield(mage.NewCreature("Second blocker", "", 1, 10), a.PlayerID())
	three := g.PutOnBattlefield(mage.NewCreature("Third blocker", "", 1, 10), a.PlayerID())
	for _, p := range []*mage.Permanent{first, second} {
		g.GetCombat().AddAttackerIncarnation(p.ID(), g.CaptureDamageSource(p.ID()).Incarnation(), b.PlayerID())
	}
	for _, pair := range [][2]*mage.Permanent{{one, first}, {two, second}, {three, second}} {
		g.GetCombat().AddBlockerIncarnation(pair[0].ID(), g.CaptureDamageSource(pair[0].ID()).Incarnation(), pair[1].ID())
	}
	a.ManaPool().Add(core.Blue, 2)
	b.ManaPool().Add(core.Green, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.ApplyContinuousEffects()
	if one.HasAttr(core.FirstStrike) || !two.HasAttr(core.FirstStrike) || !three.HasAttr(core.FirstStrike) || b.ManaPool().TotalMana() != 0 {
		t.Fatal("per-attacker payment or plural blocker grant crossed members")
	}
}
