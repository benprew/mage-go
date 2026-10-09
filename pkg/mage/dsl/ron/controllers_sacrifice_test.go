package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONControllersSacrificeGameplay(t *testing.T) {
	record := staticBaseRecord(t, "ATQ", "Golgothian Sylex")
	rule := record.Fields["Rules"].Args[0]
	rule.Args[1].Args[0].Args[0].Args[0] = nameScopeNode(t, `And([IsNonToken,IsCardtype(Creature)])`)
	factory := staticBaseSubrule(t, record, rule)
	for _, state := range []string{"normal", "control", "phased", "clone", "source-left", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			card := mage.NewCreature("Subject", "", 1, 1)
			card.SetOwner(a.PlayerID())
			victim := g.PutOnBattlefield(card, b.PlayerID())
			token := g.PutOnBattlefield(mage.NewToken("Subject", 1, 1, []core.CardType{core.TypeCreature}, nil), b.PlayerID())
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(victim.ID()) == nil || !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("timing or payment")
			}
			late := g.PutOnBattlefield(mage.NewCreature("Late", "", 1, 1), a.PlayerID())
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "phased":
				g.PhaseOut(victim.ID())
			case "source-left":
				g.BouncePermanentToHand(source)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			if state == "clone" {
				clone.PhaseOut(victim.ID())
			}
			g.ResolveStack()
			if (g.FindPermanentIncludingPhased(victim.ID()) != nil) != (state == "phased") || g.FindPermanent(late.ID()) != nil || g.FindPermanent(token.ID()) == nil {
				t.Fatal("current collection membership")
			}
			if state != "phased" && len(a.Graveyard()) != 2 {
				t.Fatal("sacrifices did not use owners")
			}
			if clone.FindPermanentIncludingPhased(victim.ID()) == nil || clone.FindPermanent(late.ID()) == nil {
				t.Fatal("original changed clone")
			}
			clone.ResolveStack()
			if (clone.FindPermanentIncludingPhased(victim.ID()) != nil) != (state == "phased" || state == "clone") {
				t.Fatal("clone lost selection")
			}
		})
	}
}

func TestRONControllersSacrificeDeclarations(t *testing.T) {
	for _, action := range []string{
		`ControllersSacrificeEachPermanent`,
		`ControllersSacrificeEachPermanent()`,
		`ControllersSacrificeEachPermanent(AnyPermanent,AnyPermanent)`,
		`ControllersSacrificeEachPermanent(SinglePlayer(You))`,
		`ControllersSacrificeEachPermanent(Trigger_ThosePermanents)`,
		`ControllersSacrificeEachPermanent(SharesANameOriginallyPrintedInAntiquities())`,
	} {
		t.Run(action, func(t *testing.T) {
			record := staticBaseRecord(t, "ATQ", "Golgothian Sylex")
			record.Fields["Rules"] = nameScopeNode(t, `[Activated(TapPermanent(ThisPermanent),ActionList([`+action+`]))]`)
			if _, err := Compile(record); err == nil {
				t.Fatal("invalid declaration accepted")
			}
		})
	}
	record := staticBaseRecord(t, "ARN", "City in a Bottle")
	if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "WhenAnyNumberOfPermanentsAreOnTheBattlefield") {
		t.Fatal("unsupported state trigger changed", err)
	}
}

func TestRONControllersSacrificeTargets(t *testing.T) {
	for _, state := range []string{"normal", "returned", "phased", "cycle", "shroud", "control", "all-illegal"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature)),TargetPermanent(IsCardtype(Creature))],ActionList([ControllersSacrificeEachPermanent(Or([SinglePermanent(Ref_TargetPermanent2),SinglePermanent(Ref_TargetPermanent1),SinglePermanent(Ref_TargetPermanent2)]))])))`)
			spell := factory()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			card := mage.NewCreature("First", "", 1, 1)
			card.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(card, b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 1), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{first.ID(), second.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "returned":
				g.BouncePermanentToHand(first)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phased":
				g.PhaseOut(first.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "shroud":
				g.MutablePermanent(first.ID()).GrantBaseAttr(core.Shroud)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: second.ID(), TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "all-illegal":
				g.PhaseOut(first.ID())
				g.PhaseOut(second.ID())
			}
			g.ResolveStack()
			legal := state == "normal" || state == "cycle" || state == "control"
			if (g.FindPermanentIncludingPhased(first.ID()) == nil) != legal || (g.FindPermanentIncludingPhased(second.ID()) == nil) != (state != "all-illegal") {
				t.Fatal("target legality or identity")
			}
		})
	}
}

func TestRONControllersSacrificeRollbackAndWaiting(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(And([PayMana([ManaCostGeneric(2)]),TapPermanent(ThisPermanent)]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([ControllersSacrificeEachPermanent(SinglePermanent(ThisPermanent))]))]))`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || source.Tapped || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("failed payment changed resources")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	saved := g.CaptureDamageSource(source.ID())
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	for range 3 {
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || !got[0].SameObject(saved) || g.FindPermanent(source.ID()) == nil {
			t.Fatal("waiting query executed or lost source")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("saved controller blocked current controller sacrifice")
	}
}

func TestRONControllersSacrificeAtomicRoles(t *testing.T) {
	for _, rules := range []string{
		`Activated(ControllersSacrificeEachPermanent(AnyPermanent),ActionList([]))`,
		`EachPermanentLayerEffect(AnyPermanent,[ControllersSacrificeEachPermanent(AnyPermanent)])`,
		`Activated(PayMana([]),ActionList([ControllersSacrificeEachPermanent(SinglePlayer(You))]))`,
	} {
		text := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rules + `])`
		if registry, err := LoadFS(fstest.MapFS{"a.ron": {Data: []byte(text)}}, "*.ron"); err == nil || registry != nil {
			t.Fatal("invalid role published registry")
		}
	}
	text := `Card(Name:"Good",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([ControllersSacrificeEachPermanent(AnyPermanent)]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(text)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "AnyPermanent" && sig.Role == "permanent_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing collection inventory role")
	}
}
