package ron

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const dependentTargetBody = `Targeted([TargetPermanent(ControlledByAPlayer(SinglePlayer(You))),TargetPermanent(And([ControlledByAPlayer(Opponent),SharesACardtypeWithPermanentFromAmongCardtypes(Ref_TargetPermanent1,[Artifact,Creature,Land])]))],ActionList([TapPermanent(Ref_TargetPermanent1),TapPermanent(Ref_TargetPermanent2),GainLife(Integer(1))]))`

func TestRONTargetDependencyGameplay(t *testing.T) {
	for _, activation := range []bool{false, true} {
		f := singularGroupsFactory(t, dependentTargetBody, activation)
		g, a, b := ronGame()
		first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 5), a.PlayerID())
		second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 5), b.PlayerID())
		other := g.PutOnBattlefield(mage.NewLand("Other"), b.PlayerID())
		card := f()
		card.SetOwner(a.PlayerID())
		var source *mage.Permanent
		if activation {
			source = g.PutOnBattlefield(card, a.PlayerID())
		} else {
			a.AddToHand(card)
		}
		announce := func(ids []uuid.UUID) error {
			if activation {
				return g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids)
			}
			return g.CastSpellByID(a.PlayerID(), card.ID(), ids, 0)
		}
		a.ManaPool().Add(core.Blue, 1)
		if err := announce([]uuid.UUID{first.ID(), other.ID()}); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
			t.Fatal("invalid dependent target changed state")
		}
		if err := announce([]uuid.UUID{first.ID(), second.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if !first.Tapped || !second.Tapped || other.Tapped || a.Life() != 21 {
			t.Fatal("dependent RON binding failed")
		}
	}
}

func TestRONTargetDependencyGauntletsPrerequisite(t *testing.T) {
	record := staticBaseRecord(t, "LEG", "Gauntlets of Chaos")
	if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "PermanentsChangedControlThisWay") {
		t.Fatalf("remaining blocker: %v", err)
	}
	rules := record.Fields["Rules"]
	ability := rules.Args[0]
	targeted := ability.Args[1]
	body := targeted.Args[1]
	actions := body.Args[0]
	// The exchange outcome still blocks the complete record.
	actions.Args = actions.Args[:1]
	body.Args[0] = actions
	targeted.Args[1] = body
	ability.Args[1] = targeted
	rules.Args[0] = ability
	record.Fields["Rules"] = rules
	f, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"normal", "invalid", "type-gain", "return", "phase", "phase-cycle", "copy", "source-return", "control", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := opponentTargetGame()
			card := f()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			firstCard := mage.NewCreature("First", "", 1, 5)
			firstCard.SetOwner(a.PlayerID())
			first := g.PutOnBattlefield(firstCard, a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 5), b.PlayerID())
			newFirst := g.PutOnBattlefield(mage.NewLand("New first"), a.PlayerID())
			newSecond := g.PutOnBattlefield(mage.NewLand("New second"), b.PlayerID())
			ids := []uuid.UUID{first.ID(), second.ID()}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids); err == nil || g.FindPermanent(source.ID()) == nil || g.StackSize() != 0 {
				t.Fatal("unpaid announcement changed state")
			}
			a.ManaPool().Add(core.Colorless, 5)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID(), newSecond.ID()}); err == nil || a.ManaPool().TotalMana() != 5 || g.FindPermanent(source.ID()) == nil {
				t.Fatal("invalid pair consumed sacrifice or mana")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(source.ID()) != nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("actual Gauntlets cost was not paid")
			}
			want := true
			switch scenario {
			case "invalid":
				g.MutablePermanent(second.ID()).GrantBaseAttr(core.Shroud)
				want = false
			case "type-gain":
				for _, p := range []*mage.Permanent{first, second} {
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeLand)), p.ID(), p.ControllerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
			case "return":
				g.BouncePermanentToHand(first)
				a.RemoveFromHand(firstCard.ID())
				g.PutOnBattlefield(firstCard, a.PlayerID())
				want = false
			case "phase":
				g.PhaseOut(first.ID())
				want = false
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "source-return":
				a.RemoveFromGraveyard(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: first.ID(), TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = false
			case "copy":
				a.queue = [][]uuid.UUID{{newFirst.ID()}, {newSecond.ID()}}
				original := g.StackPeek()
				copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
				if copied == nil || !slices.Equal(copied.Targets, []uuid.UUID{newFirst.ID(), newSecond.ID()}) || !slices.Equal(original.Targets, ids) {
					t.Fatal("copy did not keep separate pair")
				}
				if len(a.candidates) != 2 || !slices.Equal(a.candidates[1], []uuid.UUID{newSecond.ID()}) {
					t.Fatal("copy candidates used original first target")
				}
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.PhaseOut(second.ID())
			}
			g.ResolveStack()
			if clone.FindPermanentIncludingPhased(second.ID()).ControllerID() != b.PlayerID() {
				t.Fatal("clone shared control")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				changed := game.FindPermanentIncludingPhased(second.ID()).ControllerID() == a.PlayerID()
				if changed != (want && (scenario != "clone" || game != clone)) {
					t.Fatal("dependent legality used wrong visit or control")
				}
			}
			if scenario == "copy" && (g.FindPermanent(newFirst.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(newSecond.ID()).ControllerID() != a.PlayerID()) {
				t.Fatal("copied exchange lost pair")
			}
		})
	}
}

func TestRONTargetDependencyScopes(t *testing.T) {
	for _, context := range []string{"entry", "modal", "opponent"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := opponentTargetGame()
			body := dependentTargetBody
			rules := `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + body + `)`
			if context == "modal" {
				rules = `Activated(PayMana([]),Modal_ChooseOne([` + body + `,ActionList([GainLife(Integer(2))])]))`
			}
			if context == "opponent" {
				body = strings.Replace(body, `TargetPermanent(And([ControlledByAPlayer(Opponent)`, `TargetPermanentOfAPlayersChoice(And([ControlledByAPlayer(Opponent)`, 1)
				body = strings.Replace(body, `[Artifact,Creature,Land])]))]`, `[Artifact,Creature,Land])]),Opponent)]`, 1)
				rules = `Activated(PayMana([]),` + body + `)`
			}
			n := typeComparisonRecord(t, rules, false)
			f, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 5), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 5), b.PlayerID())
			g.PutOnBattlefield(mage.NewLand("Other"), b.PlayerID())
			a.queue = [][]uuid.UUID{{first.ID()}, {second.ID()}}
			b.queue = [][]uuid.UUID{{second.ID()}}
			card := f()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if context == "entry" {
				g.PutTriggersOnStack()
			} else {
				var targets []uuid.UUID
				if context == "opponent" {
					targets = []uuid.UUID{first.ID()}
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if !first.Tapped || !second.Tapped || a.Life() != 21 {
				t.Fatal("scoped acquisition failed")
			}
			if context == "opponent" && (len(b.candidates) != 1 || !slices.Equal(b.candidates[0], []uuid.UUID{second.ID()})) {
				t.Fatal("opponent saw incompatible candidates")
			}
		})
	}
}

func TestRONTargetDependencyInvalid(t *testing.T) {
	for _, expr := range []string{
		`SharesACardtypeWithPermanent`,
		`SharesACardtypeWithPermanent()`,
		`SharesACardtypeWithPermanent(Ref_TargetPermanent1,Ref_TargetPermanent1)`,
		`SharesACardtypeWithPermanentFromAmongCardtypes(Ref_TargetPermanent1)`,
		`SharesACardtypeWithPermanentFromAmongCardtypes(Ref_TargetPermanent1,Creature)`,
		`SharesACardtypeWithPermanentFromAmongCardtypes(Ref_TargetPermanent1,[])`,
		`SharesACardtypeWithPermanentFromAmongCardtypes(Ref_TargetPermanent1,[Creature,Creature])`,
		`SharesACardtypeWithPermanentFromAmongCardtypes(Ref_TargetPermanent1,[Aura])`,
		`SharesACardtypeWithPermanentFromAmongCardtypes(Ref_TargetPermanent2,[Creature])`,
		`SharesACardtypeWithPermanent(Ref_TargetPermanent3)`,
		`SharesACardtypeWithPermanent(ActionPlayer)`,
		`SharesACardtypeWithPermanent(TheChosenPermanent)`,
		`SharesACardtypeWithPermanent(ThisPermanent)`,
		`SharesACardtypeWithPermanent(HostPermanent)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[],Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent),TargetPermanent(` + expr + `)],ActionList([TapPermanent(Ref_TargetPermanent2)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", expr)
		}
	}
}

func TestRONTargetDependencyInstancesAndUnrestricted(t *testing.T) {
	body := strings.Replace(dependentTargetBody, `SharesACardtypeWithPermanentFromAmongCardtypes(Ref_TargetPermanent1,[Artifact,Creature,Land])`, `SharesACardtypeWithPermanent(Ref_TargetPermanent1)`, 1)
	f := singularGroupsFactory(t, body, true)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 5), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 5), b.PlayerID())
	otherFirst := g.PutOnBattlefield(mage.NewLand("Other first"), a.PlayerID())
	otherSecond := g.PutOnBattlefield(mage.NewLand("Other second"), b.PlayerID())
	pairs := [][]uuid.UUID{{first.ID(), second.ID()}, {otherFirst.ID(), otherSecond.ID()}}
	for _, pair := range pairs {
		card := f()
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, pair); err != nil {
			t.Fatal(err)
		}
	}
	latest := g.StackPeek()
	g.ResolveStackObject(latest)
	if first.Tapped || second.Tapped || !otherFirst.Tapped || !otherSecond.Tapped {
		t.Fatal("factory instances share targets")
	}
	g.ResolveStack()
	if !first.Tapped || !second.Tapped {
		t.Fatal("earlier instance lost bindings")
	}
}
