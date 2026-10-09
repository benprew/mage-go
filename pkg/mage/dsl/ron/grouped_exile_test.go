package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONGroupedExileSubjects(t *testing.T) {
	for _, name := range []string{"Bronze Tablet", "Tawnos's Coffin"} {
		t.Run(name, func(t *testing.T) {
			record := staticBaseRecord(t, "ATQ", name)
			rules := record.Fields["Rules"]
			var action Node
			var walk func(Node)
			walk = func(n Node) {
				if n.Name == "Exile" {
					action = n
				}
				for _, a := range n.Args {
					walk(a)
				}
			}
			for _, rule := range rules.Args {
				walk(rule)
			}
			nodes, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([])))])`), "probe.ron")
			if err != nil {
				t.Fatal(err)
			}
			probe := nodes[0]
			pr := probe.Fields["Rules"]
			pr.Args[0].Args[1].Args[1].Args[0].Args = []Node{action}
			probe.Fields["Rules"] = pr
			factory, err := Compile(probe)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			source := factory()
			source.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(source, a.PlayerID())
			hostCard := mage.NewCreature("Host", "", 2, 2)
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			auraCard := mage.NewAura("Aura", "")
			auraCard.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(auraCard, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{host.ID()}); err != nil {
				t.Fatal(err)
			}
			if len(g.GetExile()) != 0 {
				t.Fatal("exile happened before resolution")
			}
			g.ResolveStack()
			if len(g.GetExile()) != 2 || g.FindExiledCard(host.ID()) == nil {
				t.Fatal("wrong simultaneous subjects")
			}
			if name == "Bronze Tablet" {
				if g.FindExiledCard(p.ID()) == nil || g.FindPermanent(aura.ID()) != nil {
					t.Fatal("source or unattached Aura state action wrong")
				}
			} else if g.FindExiledCard(aura.ID()) == nil || g.FindPermanent(p.ID()) == nil {
				t.Fatal("attached Aura selection occurred after host left")
			}
		})
	}
}

func TestRONGroupedExileResults(t *testing.T) {
	for _, tc := range []struct {
		subjects, actions string
		exile, graveyard  int
	}{
		{`Permanent(ThisPermanent),Permanent(Ref_TargetPermanent)`, `PutExiledCardIntoOwnersGraveyard(TheFirstCardExiledThisWay)`, 1, 1},
		{`Permanent(ThisPermanent),Permanent(Ref_TargetPermanent)`, `PutExiledCardIntoOwnersGraveyard(TheSecondCardExiledThisWay)`, 1, 1},
		{`Permanent(ThisPermanent),Permanent(Ref_TargetPermanent)`, `PutExiledCardIntoOwnersGraveyard(ThisExiledCard)`, 1, 1},
		{`Permanent(Ref_TargetPermanent),EachPermanent(AttachedToPermanent(Ref_TargetPermanent))`, `PutExiledCardIntoOwnersGraveyard(TheSinglePermanentExiledThisWay),PutExiledCardsIntoOwnersGraveyards(TheOtherPermanentsExiledThisWay)`, 0, 2},
	} {
		t.Run(tc.actions, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([Exile([`+tc.subjects+`],[]),`+tc.actions+`])))`)
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			hostCard := mage.NewCreature("Host", "", 2, 2)
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			if tc.graveyard == 2 {
				auraCard := mage.NewAura("Aura", "")
				auraCard.SetOwner(b.PlayerID())
				aura := g.PutOnBattlefield(auraCard, b.PlayerID())
				g.Attach(aura.ID(), host.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{host.ID()}); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			g.ResolveStack()
			if len(g.GetExile()) != tc.exile || len(a.Graveyard())+len(b.Graveyard()) != tc.graveyard {
				t.Fatal("wrong result group moved")
			}
			if len(clone.GetExile()) != 0 {
				t.Fatal("pending clone shared results")
			}
			clone.ResolveStack()
			if len(clone.GetExile()) != tc.exile {
				t.Fatal("clone did not produce its own results")
			}
		})
	}
}

func TestRONGroupedExileInvalid(t *testing.T) {
	prefix := `Exile([Permanent(ThisPermanent),EachPermanent(IsCardtype(Creature))],[])`
	for _, actions := range []string{
		`PutExiledCardIntoOwnersGraveyard(TheFirstCardExiledThisWay),` + prefix,
		prefix + `,PutExiledCardIntoOwnersGraveyard(TheFirstCardExiledThisWay)`,
		prefix + `,PutExiledCardIntoOwnersGraveyard(TheOtherPermanentsExiledThisWay)`,
		prefix + `,PutExiledCardIntoOwnersGraveyard(TheSinglePermanentExiledThisWay())`,
		`If(PermanentPassesFilter(ThisPermanent,IsTapped),[` + prefix + `]),PutExiledCardIntoOwnersGraveyard(TheSinglePermanentExiledThisWay)`,
		`Exile([Permanent(ThisPermanent),TheTopCardOfLibrary],[])`,
		`Exile([Permanent(ThisPermanent),Permanent(ThisPermanent)], [ExilesFaceDown])`,
		`Exile([Permanent(ThisPermanent),EachPermanent(AttachedToPermanent(Ref_TargetPermanent))],[])`,
		`Exile([Permanent(ThisPermanent),Permanent(ThisPermanent)],[]),PutExiledCardIntoOwnersGraveyard(ThisExiledCard)`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+actions+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", actions)
		}
	}
}

func TestRONGroupedExileCardReferences(t *testing.T) {
	for _, name := range []string{"TheFirstCardExiledThisWay", "TheSecondCardExiledThisWay", "ThisExiledCard"} {
		n := Node{Kind: Symbol, Name: name}
		ctx := scope{groupedExile: &groupedExilePlan{singular: []bool{true, true}, source: 0}}
		plan, err := compileCardRef(n, ctx)
		if err != nil {
			t.Fatal(err)
		}
		first, second := mage.BindCards("first"), mage.BindCards("second")
		bindings := instanceBindings{groupedExiles: map[*groupedExilePlan][]*mage.CardsBinding{ctx.groupedExile: {first, second}}}
		effects := mage.Sequence(mage.ExilePermanentGroupsInto(mage.PermanentExileGroup{Subjects: mage.SingletonPermanent(mage.This), Result: first}, mage.PermanentExileGroup{Subjects: mage.MatchingPermanents(mage.IsCreature), Result: second}), mage.ReturnCardToHand(plan.build(bindings)))
		card := mage.NewArtifact("Valid", "", mage.WithAbility(mage.Activate(mage.Costs(), effects)))
		if err := mage.ValidateCardDefinition("Valid", card); err != nil {
			t.Fatal(err)
		}
		if _, err := compileCardRef(n, scope{}); err == nil {
			t.Fatal("unbound result accepted")
		}
	}
}

func TestRONGroupedExileTimingAndLegality(t *testing.T) {
	for _, state := range []string{"present", "source-phased", "source-returned", "source-control", "target-phased", "target-returned", "copy"} {
		t.Run(state, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(And([PayMana([ManaCostGeneric(3)]),TapPermanent(ThisPermanent)]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([Exile([Permanent(ThisPermanent),Permanent(Ref_TargetPermanent)],[]),PutExiledCardIntoOwnersGraveyard(TheSecondCardExiledThisWay)])))`)
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			targetCard := mage.NewCreature("Subject", "", 2, 2)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err == nil || p.Tapped || len(g.GetExile()) != 0 {
				t.Fatal("failed payment changed live state")
			}
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if !p.Tapped || a.ManaPool().TotalMana() != 0 || len(g.GetExile()) != 0 {
				t.Fatal("announcement timing failed")
			}
			switch state {
			case "source-phased":
				g.PhaseOut(p.ID())
			case "source-returned":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "target-phased":
				g.PhaseOut(target.ID())
			case "target-returned":
				g.BouncePermanentToHand(target)
				b.RemoveFromHand(targetCard.ID())
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				illegal := state == "target-phased" || state == "target-returned"
				if (len(game.GetPlayer(b.PlayerID()).Graveyard()) == 1) == illegal {
					t.Fatal("target visit legality failed")
				}
				wantSourceExiled := !illegal && state != "source-phased" && state != "source-returned"
				if (game.FindExiledCard(card.ID()) != nil) != wantSourceExiled {
					t.Fatal("source identity or fizzle failed")
				}
			}
		})
	}
}

func TestRONGroupedExileContexts(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(ActionList([Exile([EachPermanent(IsCardtype(Creature)),EachPermanent(IsCardtype(Artifact))],[])]))`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([Exile([Permanent(ThisPermanent),EachPermanent(IsCardtype(Creature))],[])]))`,
		`Activated(PayMana([]),ActionList([Exile([Permanent(ThisPermanent),EachPermanent(IsCardtype(Creature))],[]),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PutExiledCardIntoOwnersGraveyard(TheSinglePermanentExiledThisWay)]))]))`,
	} {
		types := "Artifact"
		if strings.HasPrefix(rule, "SpellActions") {
			types = "Sorcery"
		}
		n := nameScopeNode(t, `Card(Name:"Context",Typeline:(Supertypes:[],Cardtypes:[`+types+`],Subtypes:[]),Rules:[`+rule+`])`)
		factory, err := Compile(n)
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		c := factory()
		c.SetOwner(a.PlayerID())
		creature := mage.NewCreature("Subject", "", 2, 2)
		creature.SetOwner(b.PlayerID())
		g.PutOnBattlefield(creature, b.PlayerID())
		if types == "Sorcery" {
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
		} else {
			p := g.PutOnBattlefield(c, a.PlayerID())
			if strings.HasPrefix(rule, "Activated") {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
		}
		g.ResolveStack()
		if g.FindExiledCard(creature.ID()) == nil {
			t.Fatal("context did not exile at resolution")
		}
		if strings.HasPrefix(rule, "Activated") {
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.ResolveStack()
			if g.FindExiledCard(c.ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("saved delayed result failed")
			}
		}
	}
}

func TestRONGroupedExileDuplicateSubjects(t *testing.T) {
	spellFactory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([Exile([Permanent(Ref_TargetPermanent),Permanent(Ref_TargetPermanent)],[])])))`)
	g, a, b := ronGame()
	victim := mage.NewCreature("Subject", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(victim, b.PlayerID())
	spell := spellFactory()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.GetExile()) != 1 || g.CardsPutIntoExileThisTurn() != 1 {
		t.Fatal("duplicate target caused multiple moves")
	}
	delayedFactory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([Exile([Permanent(ThisPermanent),Permanent(ThisPermanent)],[])]))]))`)
	source := delayedFactory()
	source.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("duplicate groups changed waiting source identity")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if len(g.GetExile()) != 2 || g.CardsPutIntoExileThisTurn() != 2 {
		t.Fatal("delayed duplicate groups moved twice")
	}
}

func TestRONGroupedExileOtherTransport(t *testing.T) {
	for _, destination := range []string{"hand", "battlefield"} {
		action := `ReturnEachExiledCardToOwnersHand(TheOtherPermanentsExiledThisWay)`
		if destination == "battlefield" {
			action = `PutEachExiledCardOntoTheBattlefield(TheOtherPermanentsExiledThisWay,[EntersUnderOwnersControl])`
		}
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([Exile([Permanent(ThisPermanent),EachPermanent(IsCardtype(Creature))],[]),`+action+`]))`)
		g, a, b := ronGame()
		card := factory()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		c := mage.NewCreature("Subject", "", 2, 2)
		c.SetOwner(b.PlayerID())
		g.PutOnBattlefield(c, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if len(g.GetExile()) != 1 || g.FindExiledCard(card.ID()) == nil {
			t.Fatal("transport consumed the source group")
		}
		if destination == "hand" {
			if len(b.Hand()) != 1 || len(a.Hand()) != 0 {
				t.Fatal("wrong hand owner")
			}
		} else {
			subject := g.FindPermanent(c.ID())
			if subject == nil || subject.ControllerID() != b.PlayerID() {
				t.Fatal("wrong entry controller")
			}
		}
	}
}
