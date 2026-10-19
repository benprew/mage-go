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

func TestRONDelayedDeathCardWhippoorwill(t *testing.T) {
	for _, change := range []string{"normal", "source-left", "phase-cycle", "returned-pending", "returned-stack", "returned-battlefield", "token", "copy", "waiting-clone", "pending-clone", "stack-clone", "cleanup", "replaced-death", "simultaneous", "phase-before", "returned-before", "shroud-before", "source-left-before", "damage"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			g.SetActivePlayerIndex(0)
			g.SetStep(core.PrecombatMain)
			factory, err := Compile(staticBaseRecord(t, "DRK", "Whippoorwill"))
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			victim := mage.NewCreature("Victim", "", 2, 3)
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			otherCard := mage.NewCreature("Other", "", 2, 3)
			otherCard.SetOwner(b.PlayerID())
			other := g.PutOnBattlefield(otherCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err == nil || g.FindPermanent(source.ID()).Tapped {
				t.Fatal("unpaid activation succeeded")
			}
			a.ManaPool().Add(core.Green, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			if change == "copy" {
				a.targets = []uuid.UUID{other.ID()}
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true) == nil {
					t.Fatal("copy failed")
				}
			}
			if len(g.DelayedTriggers()) != 0 || !g.FindPermanent(source.ID()).Tapped || a.ManaPool().Count(core.Green) != 0 {
				t.Fatal("activation timing or payment")
			}
			switch change {
			case "phase-before":
				g.PhaseOut(p.ID())
			case "returned-before":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				b.RemoveFromHand(victim.ID())
				g.PutOnBattlefield(victim, b.PlayerID())
			case "shroud-before":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			case "source-left-before":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			}
			g.ResolveStack()
			if change == "phase-before" || change == "returned-before" || change == "shroud-before" {
				if len(g.DelayedTriggers()) != 0 {
					t.Fatal("illegal target installed death subscription")
				}
				return
			}
			if source.Card.Name() != "Whippoorwill" || source.CurrentPower(g) != 1 || source.CurrentToughness(g) != 1 {
				t.Fatal("metadata")
			}
			if !g.FindPermanent(p.ID()).HasAttr(core.CantRegenerate) || !g.FindPermanent(p.ID()).HasAttr(core.AttrDamageCantBePreventedOrRedirected) {
				t.Fatal("first action lost")
			}
			if change == "source-left" {
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			}
			if change == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			if change == "token" {
				g.MutablePermanent(p.ID()).IsToken = true
			}
			if change == "cleanup" {
				g.RunStepWithPriority(core.Cleanup)
			}
			if change == "replaced-death" {
				g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), source.ID())
			}
			var clone *mage.Game
			if change == "waiting-clone" {
				clone = g.Clone()
			}
			g.AddRegenerationShield(p.ID())
			switch change {
			case "damage":
				g.AddPreventionShield(p.ID(), 5)
				if err := mage.ApplyEffect(g, mage.RedirectNextDamageToPlayer(mage.This, mage.You, mage.Fixed(3)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.DealDamageToPermanent(g.FindPermanent(p.ID()), 3, other.ID())
				if g.FindPermanent(p.ID()).Damage != 3 || a.Life() != 20 {
					t.Fatal("damage prevented or redirected")
				}
				g.CheckStateBasedActions()
			case "simultaneous":
				g.DestroyPermanents([]*mage.Permanent{g.FindPermanent(p.ID()), g.FindPermanent(source.ID())})
			default:
				g.DestroyPermanent(g.FindPermanent(p.ID()))
			}
			if change == "cleanup" {
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("regeneration prohibition survived cleanup")
				}
				return
			}
			if g.FindPermanent(p.ID()) != nil {
				t.Fatal("regeneration prohibited death")
			}
			wantStack := 0
			if change == "damage" {
				wantStack = 1
			}
			if g.StackSize() != wantStack {
				t.Fatalf("death trigger timing: stack=%d want=%d", g.StackSize(), wantStack)
			}
			if change != "replaced-death" && g.FindExiledCard(victim.ID()) != nil {
				t.Fatal("delayed exile ran before resolution")
			}
			if change == "copy" {
				g.DestroyPermanent(g.FindPermanent(other.ID()))
			}
			if change == "pending-clone" {
				clone = g.Clone()
			}
			if change == "returned-pending" {
				b.RemoveFromGraveyard(victim.ID())
				b.AddToGraveyard(victim)
			}
			g.PutTriggersOnStack()
			if change == "stack-clone" {
				clone = g.Clone()
			}
			if change == "returned-stack" {
				b.RemoveFromGraveyard(victim.ID())
				b.AddToGraveyard(victim)
			}
			if change == "returned-battlefield" {
				b.RemoveFromGraveyard(victim.ID())
				g.PutOnBattlefield(victim, b.PlayerID())
			}
			g.ResolveStack()
			wantExile := !strings.HasPrefix(change, "returned-") && change != "token"
			if (g.FindExiledCard(victim.ID()) != nil) != wantExile {
				t.Fatal("wrong visit exiled")
			}
			if change == "copy" && g.FindExiledCard(other.ID()) == nil {
				t.Fatal("copy lost separate target")
			}
			if change == "returned-battlefield" && g.FindPermanent(victim.ID()) == nil {
				t.Fatal("returned permanent moved")
			}
			if clone != nil {
				if clone.FindExiledCard(victim.ID()) != nil {
					t.Fatal("original changed clone")
				}
				if change == "waiting-clone" {
					clone.DestroyPermanent(clone.FindPermanent(p.ID()))
				}
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if clone.FindExiledCard(victim.ID()) == nil {
					t.Fatal("clone lost saved death")
				}
			}
		})
	}
}

func TestRONDelayedDeathCardDeclarations(t *testing.T) {
	for _, ref := range []string{"Ref_TargetPermanent", "Trigger_ThatPermanent"} {
		src := `Card(Name:"Watch",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + deathCaptureAction("WhenAPermanentDies", `Exile([Permanent(`+ref+`)],[])`) + `])))])`
		factory, err := Compile(parseTemporaryRedirectCard(t, src))
		if err != nil {
			t.Fatal(err)
		}
		if err := mage.ValidateCardDefinition("Watch", factory()); err != nil {
			t.Fatal(err)
		}
	}
	for _, body := range []string{
		`Exile([Permanent(Trigger_ThatPermanent)], [ExilesFaceDown])`,
		`Exile([Permanent(Trigger_ThatPermanent())], [])`,
		`Exile([Permanent(Ref_TargetPermanent),Permanent(Ref_TargetPermanent)], [])`,
		`Exile([Permanent(Trigger_ThatPermanent)], [], You)`,
	} {
		action := deathCaptureAction("WhenAPermanentDies", body)
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))])`
		if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || registry != nil {
			t.Fatalf("invalid declaration accepted: %s", body)
		}
	}
}

func TestRONDelayedDeathCardScope(t *testing.T) {
	action := deathCaptureAction("WhenAPermanentDies", `Exile([CardInGraveyards(Trigger_ThatGraveyardCard)],[])`)
	for _, context := range []string{"spell", "activation", "nested"} {
		body := `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `]))`
		if context == "nested" {
			body = `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + action + `]))]))`
		}
		rule := `SpellActions(` + body + `)`
		kind := "Instant"
		if context == "activation" {
			rule = `Activated(PayMana([]),` + body + `)`
			kind = "Artifact"
		}
		src := `Card(Name:"Watch",Typeline:(Supertypes:[],Cardtypes:[` + kind + `],Subtypes:[]),Rules:[` + rule + `])`
		factory, err := Compile(parseTemporaryRedirectCard(t, src))
		if err != nil {
			t.Fatal(context, err)
		}
		g, a, b := ronGame()
		victim := mage.NewCreature("Victim", "", 2, 3)
		victim.SetOwner(b.PlayerID())
		p := g.PutOnBattlefield(victim, b.PlayerID())
		card := factory()
		card.SetOwner(a.PlayerID())
		if context == "activation" {
			source := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
		} else {
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
		}
		g.ResolveStack()
		if context == "nested" {
			g.RunStepWithPriority(core.EndStep)
			g.ResolveStack()
		}
		g.PutPermanentIntoGraveyard(g.FindPermanent(p.ID()))
		g.PutTriggersOnStack()
		g.ResolveStack()
		if g.FindExiledCard(victim.ID()) == nil {
			t.Fatal(context, "lost explicit death card")
		}

	}
	for _, event := range []string{"WhenAPermanentLeavesTheBattlefield", "WhenAPermanentDies"} {
		actions := deathCaptureAction(event, `GainLife(Integer(1))`) + `,Exile([CardInGraveyards(Trigger_ThatGraveyardCard)],[])`
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + actions + `])))])`
		if _, err := Compile(parseTemporaryRedirectCard(t, src)); err == nil {
			t.Fatal("death card escaped its body")
		}
	}
	src := `Card(Name:"Bad entry",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[AsPermanentEnters(ActionList([` + action + `]))])`
	if _, err := Compile(parseTemporaryRedirectCard(t, src)); err == nil {
		t.Fatal("entry replacement accepted delayed action")
	}
	nodes, err := Parse(strings.NewReader(action), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{resolving: true, drawReplacementBody: true}, {}} {
		ctx.permanentTarget = &permanentTargetPlan{name: "Ref_TargetPermanent"}
		if _, err := compileDepartureDelay(nodes[0], ctx); err == nil {
			t.Fatal("forbidden delayed context accepted")
		}
	}
}

func TestRONDelayedDeathCardLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Whippoorwill"`) {
			record = line
			break
		}
	}
	if record == "" {
		t.Fatal("missing complete record")
	}
	files := fstest.MapFS{"whippoorwill.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([Exile([CardInGraveyards(Trigger_ThatGraveyardCard)],[])]))])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("collection published despite unbound death card")
	}
}
