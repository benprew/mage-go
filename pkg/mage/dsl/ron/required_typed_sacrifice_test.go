package ron

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRequiredTypedSacrificeKnowledgeVault(t *testing.T) {
	for _, state := range []string{"present", "missing", "returned", "phased", "phase-cycle", "control", "replaced", "copy"} {
		t.Run(state, func(t *testing.T) {
			record := staticBaseRecord(t, "LEG", "Knowledge Vault")
			rules := record.Fields["Rules"]
			rule := rules.Args[1]
			actions := rule.Args[1].Args[0]
			continuation := actions.Args[1]
			continuation.Args[1] = nameScopeNode(t, `[GainLife(Integer(3))]`)
			actions.Args[1] = continuation
			rule.Args[1].Args[0] = actions
			rules.Args = []Node{rule}
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := factory()
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()) == nil || a.offers != 0 {
				t.Fatal("payment occurred at announcement")
			}
			switch state {
			case "missing", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					b.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "replaced":
				g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), uuid.New())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			paid := state == "present" || state == "phase-cycle" || state == "replaced" || state == "copy"
			for _, game := range []*mage.Game{g, clone} {
				want := 20
				if paid {
					want = 23
				}
				if game.GetPlayer(a.PlayerID()).Life() != want || game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("wrong payer or branch")
				}
				if paid && game.FindPermanentIncludingPhased(p.ID()) != nil {
					t.Fatal("payment subject remains")
				}
				if state == "returned" && game.FindPermanent(p.ID()) == nil {
					t.Fatal("returned source consumed")
				}
				if state == "present" && len(game.GetPlayer(b.PlayerID()).Graveyard()) != 1 {
					t.Fatal("wrong owner")
				}
				if state == "replaced" && len(game.GetExile()) != 1 {
					t.Fatal("replacement lost")
				}
			}
			if a.offers+b.offers != 0 {
				t.Fatal("mandatory payment offered")
			}
		})
	}
}

func TestRONRequiredTypedSacrificeInvalid(t *testing.T) {
	for _, cost := range []string{`SacrificePermanent`, `SacrificePermanent()`, `SacrificePermanent(ThisPermanent())`, `SacrificePermanent(You)`, `SacrificePermanent(Ref_TargetPermanent)`, `SacrificePermanent(Trigger_ThatCreature)`, `SacrificePermanent(SinglePermanent(ThisPermanent))`, `SacrificePermanent(ThisPermanent,ThisPermanent)`} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MustCost(`+cost+`),If(CostWasPaid,[GainLife(Integer(1))])]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", cost)
		}
	}
	for _, body := range []string{
		`MustCost(SacrificePermanent(ThisPermanent))`,
		`MustCost(SacrificePermanent(ThisPermanent)),GainLife(Integer(1)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(SacrificePermanent(ThisPermanent)),If(CostWasPaid(),[GainLife(Integer(1))])`,
		`MustCost(SacrificePermanent(ThisPermanent)),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])`,
		`MustCost(SacrificePermanent(ThisPermanent)),If(CostWasPaid,[])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, rule := range []string{
		`SpellActions(ActionList([MustCost(SacrificePermanent(ThisPermanent)),If(CostWasPaid,[GainLife(Integer(1))])]))`,
		`AsPermanentEnters(ThisPermanent,[MustCost(SacrificePermanent(ThisPermanent))])`,
		`PermanentLayerEffect(ThisPermanent,[MustCost(SacrificePermanent(ThisPermanent))])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}

}

func TestRONRequiredTypedSacrificeContexts(t *testing.T) {
	for _, context := range []string{"activated", "triggered", "delayed", "target"} {
		for _, condition := range []string{"If(CostWasPaid", "Unless(CostWasPaid", "If(CostWasntPaid", "Unless(CostWasntPaid"} {
			t.Run(context+condition, func(t *testing.T) {
				subject := "ThisPermanent"
				if context == "target" {
					subject = "Ref_TargetPermanent"
				}
				body := fmt.Sprintf("MustCost(SacrificePermanent(%s)),%s,[GainLife(Integer(3))])", subject, condition)
				rule := "Activated(PayMana([]),ActionList([" + body + "]))"
				if context == "triggered" {
					rule = "TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([" + body + "]))"
				}
				if context == "delayed" {
					rule = "Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([" + body + "]))]))"
				}
				if context == "target" {
					rule = "Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Artifact))],ActionList([" + body + "])))"
				}
				g, a, _ := ronGame()
				card := permanentScopeFactory(t, "Artifact", rule)()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				var targets []uuid.UUID
				if context == "target" {
					target := g.PutOnBattlefield(mage.NewArtifact("Target", ""), a.PlayerID())
					targets = []uuid.UUID{target.ID()}
				}
				if context == "triggered" {
					g.PutTriggersOnStack()
				} else if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
				if context == "delayed" {
					g.ResolveStack()
					saved := g.CaptureDamageSource(p.ID())
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(saved) || g.FindPermanent(p.ID()) == nil || a.Life() != 20 {
						t.Fatal("waiting query changed state or lost source")
					}
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					g.PutTriggersOnStack()
				}
				g.ResolveStack()
				want := 20
				if condition == "If(CostWasPaid" || condition == "Unless(CostWasntPaid" {
					want = 23
				}
				if a.Life() != want {
					t.Fatalf("life=%d want=%d", a.Life(), want)
				}
				subjectID := p.ID()
				if context == "target" {
					subjectID = targets[0]
					if g.FindPermanent(p.ID()) == nil {
						t.Fatal("source paid target cost")
					}
				}
				if g.FindPermanent(subjectID) != nil {
					t.Fatal("subject remains")
				}
			})
		}
	}
}

func TestRONRequiredTypedSacrificeLocalResults(t *testing.T) {
	g, a, _ := ronGame()
	rule := `Activated(PayMana([]),ActionList([MustCost(SacrificePermanent(ThisPermanent)),If(CostWasPaid,[MustCost(SacrificePermanent(ThisPermanent)),Unless(CostWasPaid,[GainLife(Integer(2))]),GainLife(Integer(3))])]))`
	card := permanentScopeFactory(t, "Artifact", rule)()
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 25 || g.FindPermanent(p.ID()) != nil {
		t.Fatal("nested payment overwrote outer result")
	}
}

func TestRONRequiredTypedSacrificeTargetLegality(t *testing.T) {
	for _, state := range []string{"opponent", "returned", "phased", "type"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			rule := `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([MustCost(SacrificePermanent(Ref_TargetPermanent)),Unless(CostWasPaid,[GainLife(Integer(3))])])))`
			source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
			card := mage.NewCreature("Target", "", 1, 1)
			owner := a.PlayerID()
			if state == "opponent" {
				owner = b.PlayerID()
			}
			card.SetOwner(owner)
			target := g.PutOnBattlefield(card, owner)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "returned":
				g.BouncePermanentToHand(target)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				g.PhaseOut(target.ID())
			case "type":
				g.AddContinuousEffect(mage.FuncContinuousEffect(core.LayerType, core.EndOfTurn, func(game *mage.Game, _ uuid.UUID) error {
					game.GetEffects().RevokeAttr(target.ID(), core.AttrIsCreature)
					return nil
				}))
				g.ApplyContinuousEffects()
			}
			g.ResolveStack()
			want := 20
			if state == "opponent" {
				want = 23
			}
			if a.Life() != want || g.FindPermanentIncludingPhased(target.ID()) == nil || g.FindPermanent(source.ID()) == nil {
				t.Fatalf("life=%d want=%d target=%v source=%v", a.Life(), want, g.FindPermanentIncludingPhased(target.ID()) != nil, g.FindPermanent(source.ID()) != nil)
			}
		})
	}
}
