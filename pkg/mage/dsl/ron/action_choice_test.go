package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONActionChoiceSequence(t *testing.T) {
	for _, mode := range []int{-1, 0, 1, 8} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: mode}
			b := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("B"), choice: 1}
			g := mage.NewGame(a, b)
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([CreateValueX(Integer(2)),PlayerActions(Ref_TargetPlayer,[ChooseAnAction([[GainLife(ValueX),LoseLife(Integer(1))],[CreateValueX(Integer(5)),GainLife(ValueX),ChooseAnAction([[GainLife(Integer(10))],[GainLife(Integer(20))]])]])]),ChooseAnAction([[GainLife(ValueX),GainLife(Integer(1))],[GainLife(Integer(7))]]),GainLife(ValueX)])))`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if a.prompts != 0 || b.prompts != 0 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("choice ran before resolution")
			}
			g.ResolveStack()
			want := 25
			if mode == 1 {
				want = 29
			}
			if a.Life() != want || b.Life() != 45 || a.prompts != 1 || b.prompts != 2 {
				t.Fatalf("wrong sequence or scope: life %d/%d prompts %d/%d", a.Life(), b.Life(), a.prompts, b.prompts)
			}
		})
	}
}

func TestRONActionChoicePaymentAndDraw(t *testing.T) {
	for _, mode := range []int{0, 1} {
		a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: mode}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),ActionList([ChooseAnAction([[PlayerMustCost(You,DiscardACard),IfElse(CostWasPaid,[GainLife(Integer(2))],[GainLife(Integer(3))])],[DrawACard,DiscardTheCardDrawnThisWay]]),GainLife(Integer(1))]))`)
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		seedRONDrawLibrary(a.BasePlayer)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || a.prompts != 0 || g.StackSize() != 0 {
			t.Fatal("failed payment executed a choice")
		}
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		wantLife, wantGraveyard := 24, 0
		if mode == 1 {
			wantLife, wantGraveyard = 21, 1
		}
		if a.Life() != wantLife || len(a.Graveyard()) != wantGraveyard || len(a.Hand()) != 0 || a.prompts != 1 {
			t.Fatal("choice body lost payment or draw-result context")
		}
	}
}

func TestRONActionChoiceSanctuaryContext(t *testing.T) {
	for _, mode := range []int{0, 1} {
		record := staticBaseRecord(t, "LEA", "Island Sanctuary")
		if _, err := Compile(record); err == nil {
			t.Fatal("unsupported attack restriction must remain rejected")
		}
		choice := record.Fields["Rules"].Args[0].Args[1].Args[0]
		body, err := Parse(strings.NewReader(`GainLife(Integer(4))`), "restriction-substitute.ron")
		if err != nil {
			t.Fatal(err)
		}
		choice.Args[0].Args[0].Args = body
		factory, err := Compile(record)
		if err != nil {
			t.Fatal(err)
		}
		a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: mode}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.PutOnBattlefield(factory(), a.PlayerID())
		seedRONDrawLibrary(a.BasePlayer)
		seedRONDrawLibrary(b)
		g.SetStep(core.Upkeep)
		g.PlayerDrawCard(a)
		if len(a.Hand()) != 1 || a.prompts != 0 {
			t.Fatal("replacement ran outside the draw step")
		}
		g.SetStep(core.Draw)
		g.PlayerDrawCard(b)
		if len(b.Hand()) != 1 || a.prompts != 0 {
			t.Fatal("replacement changed the other player's draw")
		}
		g.PlayerDrawCard(a)
		wantLife, wantHand := 24, 1
		if mode == 1 {
			wantLife, wantHand = 20, 2
		}
		if a.Life() != wantLife || len(a.Hand()) != wantHand || a.prompts != 1 || g.StackSize() != 0 {
			t.Fatal("replacement choice timing, branch or draw ancestry")
		}
	}
}

func TestRONActionChoiceInvalid(t *testing.T) {
	for _, body := range []string{
		`ChooseAnAction([[GainLife(Integer(1))],[]])`,
		`ChooseAnAction([[GainLife(Integer(1))],[UnknownAction]])`,
		`ChooseAnAction([[CreateValueX(Integer(1))],[GainLife(ValueX)]])`,
		`ChooseAnAction([[CreateValueX(Integer(1))],[GainLife(Integer(1))]]),GainLife(ValueX)`,
		`ChooseAnAction([[DrawACard],[DiscardTheCardDrawnThisWay]])`,
		`ChooseAnAction([[DrawACard],[GainLife(Integer(1))]]),DiscardTheCardDrawnThisWay`,
		`ChooseAnAction([[PlayerMustCost(You,DiscardACard)],[If(CostWasPaid,[GainLife(Integer(1))])]])`,
		`ChooseAnAction([[TapPermanent(Ref_TargetPermanent)],[GainLife(Integer(1))]])`,
		`ChooseAnAction([[GainLife(Integer(1))],[Targeted([TargetPlayer(AnyPlayer)],ActionList([GainLife(Integer(1))]))]])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid-action-choice.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid branch accepted", body)
		}
	}
}

func TestRONActionChoiceReplacementAncestry(t *testing.T) {
	a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 0}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	factory := permanentScopeFactory(t, "Enchantment", `ReplaceWouldDraw(PlayerWouldDrawDuringTheirDrawStep(You),[ChooseAnAction([[DrawACard],[GainLife(Integer(1))]])])`)
	for range 2 {
		g.PutOnBattlefield(factory(), a.PlayerID())
	}
	seedRONDrawLibrary(a.BasePlayer)
	g.SetStep(core.Draw)
	g.PlayerDrawCard(a)
	if len(a.Hand()) != 1 || a.prompts != 3 || a.Life() != 20 || g.StackSize() != 0 {
		t.Fatalf("a generated draw repeated a source or lost another source: hand=%d prompts=%d life=%d stack=%d", len(a.Hand()), a.prompts, a.Life(), g.StackSize())
	}
	g.PlayerDrawCard(a)
	if len(a.Hand()) != 2 || a.prompts != 6 {
		t.Fatal("a later draw reused the prior event ancestry")
	}
}

func TestRONActionChoiceDelayedIdentity(t *testing.T) {
	for _, state := range []string{"live", "phased", "returned", "control", "copy", "clone"} {
		for _, mode := range []int{0, 1} {
			t.Run(fmt.Sprintf("%s/%d", state, mode), func(t *testing.T) {
				a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: mode}
				b := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("B"), choice: 1}
				g := mage.NewGame(a, b)
				nodes, err := Parse(strings.NewReader(`Card(Name:"Choice source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([ChooseAnAction([[GainLife(PowerOfPermanent(ThisPermanent)),GainLife(Integer(2))],[GainLife(Integer(1))]])]))]))])`), "choice-source.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				card := factory()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.MutablePermanent(p.ID()).Counters[core.P1P1] = 3
				saved := g.CaptureDamageSource(p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveStack()
				if a.prompts != 0 || b.prompts != 0 {
					t.Fatal("choice ran when the delay was created")
				}
				switch state {
				case "phased":
					g.PhaseOut(p.ID())
				case "returned":
					g.BouncePermanentToHand(p)
					a.RemoveFromHand(card.ID())
					p = g.PutOnBattlefield(card, b.PlayerID())
					g.MutablePermanent(p.ID()).Counters[core.P1P1] = 9
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				for range 2 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(saved) || a.prompts != 0 || b.prompts != 0 {
						t.Fatal("query lost an unselected body's source or made a choice")
					}
				}
				game := g
				if state == "clone" {
					game = g.Clone()
				}
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				want := 21
				if mode == 0 || state == "clone" {
					want = 27
				}
				if game.GetPlayer(a.PlayerID()).Life() != want || len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatalf("wrong saved body or source: life %d want %d", game.GetPlayer(a.PlayerID()).Life(), want)
				}
				if state == "copy" && (b.Life() != 21 || b.prompts != 1) {
					t.Fatal("copy did not use its own chooser")
				}
				if state == "clone" && (a.Life() != 20 || a.prompts != 0 || len(g.WaitingDelayedPermanentSources()) != 1) {
					t.Fatal("clone shares choice or frame storage")
				}
			})
		}
	}
}
