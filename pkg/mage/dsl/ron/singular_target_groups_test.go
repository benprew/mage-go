package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const singularGroupsBody = `Targeted([TargetPermanent(IsCardtype(Creature)), TargetPermanent(IsCardtype(Artifact))], ActionList([TapPermanent(Ref_TargetPermanent1), UntapPermanent(Ref_TargetPermanent2), GainLife(Integer(1))]))`

func singularGroupsFactory(t *testing.T, body string, activation bool) mage.CardFactory {
	t.Helper()
	kind, rule := "Instant", "SpellActions("+body+")"
	if activation {
		kind, rule = "Artifact", "Activated(And([PayMana([ManaCostGeneric(1)]), TapPermanent(ThisPermanent)]),"+body+")"
	}
	src := fmt.Sprintf(`Card(Name:"Groups", Typeline:(Supertypes:[],Cardtypes:[%s],Subtypes:[]), ManaCost:[ManaCostU],Rules:[%s])`, kind, rule)
	nodes, err := Parse(strings.NewReader(src), "groups.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Groups", factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONSingularGroupsGameplay(t *testing.T) {
	for _, activation := range []bool{false, true} {
		for _, change := range []string{"none", "first-illegal", "second-illegal", "all-illegal", "return", "phase", "phase-cycle", "control", "copy"} {
			t.Run(fmt.Sprintf("%v/%s", activation, change), func(t *testing.T) {
				factory := singularGroupsFactory(t, singularGroupsBody, activation)
				g, a, b := ronGame()
				creature := mage.NewCreature("First", "", 2, 3)
				creature.SetOwner(b.PlayerID())
				first := g.PutOnBattlefield(creature, b.PlayerID())
				second := g.PutOnBattlefield(mage.NewArtifact("Second", ""), b.PlayerID())
				g.TapPermanent(second)
				card := factory()
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
				ids := []uuid.UUID{first.ID(), second.ID()}
				if err := announce(ids); err == nil {
					t.Fatal("unpaid action accepted")
				}
				if g.StackSize() != 0 || activation && g.FindPermanent(source.ID()).Tapped || !activation && len(a.Hand()) != 1 {
					t.Fatal("payment rollback")
				}
				a.ManaPool().Add(core.Blue, 1)
				for _, invalid := range [][]uuid.UUID{{first.ID()}, {second.ID(), first.ID()}, {first.ID(), second.ID(), first.ID()}} {
					if err := announce(invalid); err == nil {
						t.Fatal("invalid target list accepted")
					}
					if g.StackSize() != 0 || a.ManaPool().TotalMana() != 1 || activation && g.FindPermanent(source.ID()).Tapped {
						t.Fatal("illegal target spent resources")
					}
				}
				if err := announce(ids); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped || a.Life() != 20 {
					t.Fatal("actions ran before resolution")
				}
				switch change {
				case "first-illegal":
					g.MutablePermanent(first.ID()).GrantBaseAttr(core.Shroud)
				case "second-illegal":
					g.MutablePermanent(second.ID()).GrantBaseAttr(core.Shroud)
				case "all-illegal":
					g.MutablePermanent(first.ID()).GrantBaseAttr(core.Shroud)
					g.MutablePermanent(second.ID()).GrantBaseAttr(core.Shroud)
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
					g.PutOnBattlefield(creature, b.PlayerID())
				case "phase":
					g.PhaseOut(first.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(first.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: second.ID(), TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				if clone.FindPermanentIncludingPhased(first.ID()).Tapped || !clone.FindPermanent(second.ID()).Tapped || clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("resolution changed clone")
				}
				clone.ResolveStack()
				firstLegal := change != "first-illegal" && change != "all-illegal" && change != "return" && change != "phase"
				secondLegal := change != "second-illegal" && change != "all-illegal"
				for _, game := range []*mage.Game{g, clone} {
					if game.FindPermanentIncludingPhased(first.ID()).Tapped != firstLegal {
						t.Fatal("first group identity or legality")
					}
					if game.FindPermanent(second.ID()).Tapped == secondLegal {
						t.Fatal("second group identity or legality")
					}
					life := 21
					if change == "all-illegal" {
						life = 20
					}
					if change == "copy" {
						life = 22
					}
					if game.GetPlayer(a.PlayerID()).Life() != life {
						t.Fatal("independent action or all-illegal resolution")
					}
				}
			})
		}
	}
}

func TestRONSingularGroupsInstancesAndRepeatedObject(t *testing.T) {
	body := `Targeted([TargetPermanent(IsCardtype(Creature)),TargetPermanent(IsCardtype(Artifact)),TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent1),UntapPermanent(Ref_TargetPermanent2),TapPermanent(Ref_TargetPermanent3)]))`
	factory := singularGroupsFactory(t, body, false)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewCreature("Both", "", 2, 3, mage.WithCardType(core.TypeArtifact)), b.PlayerID())
	q := g.PutOnBattlefield(mage.NewArtifact("Other", ""), b.PlayerID())
	for _, ids := range [][]uuid.UUID{{p.ID(), p.ID(), q.ID()}, {p.ID(), q.ID(), p.ID()}} {
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Blue, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, 0); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveTopOfStack()
	if !g.FindPermanent(p.ID()).Tapped || g.FindPermanent(q.ID()).Tapped {
		t.Fatal("second factory read first factory bindings")
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Tapped || !g.FindPermanent(q.ID()).Tapped {
		t.Fatal("separate word target or numbered action order")
	}
}

func TestRONSingularGroupsInvalid(t *testing.T) {
	for _, body := range []string{
		`Targeted([],ActionList([GainLife(Integer(1))]))`,
		`Targeted([TargetPermanent(And([ControlledByAPlayer(SinglePlayer(You)),ControlledByAPlayer(Opponent)])),TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent1)]))`,
		`Targeted([TargetPermanent(AnyPermanent),TargetPlayer(AnyPlayer)],ActionList([GainLife(Integer(1))]))`,
		`Targeted([TargetPermanent(AnyPermanent),NumberTargetPermanents(Integer(2),AnyPermanent)],ActionList([GainLife(Integer(1))]))`,
		`Targeted([TargetPermanent(AnyPermanent),TargetPermanent()],ActionList([GainLife(Integer(1))]))`,
		`Targeted([TargetPermanent(AnyPermanent),TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent)]))`,
		`Targeted([TargetPermanent(AnyPermanent),TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent0)]))`,
		`Targeted([TargetPermanent(AnyPermanent),TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent3)]))`,
		`Targeted([TargetPermanent(AnyPermanent),TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent01)]))`,
		`Targeted([TargetPermanent(AnyPermanent),TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent1())]))`,
		`Targeted([TargetPermanent(AnyPermanent),TargetPermanent(AnyPermanent)],ActionList([PlayerAction(Ref_TargetPermanent1,DrawACard)]))`,
		`Targeted([TargetPermanent(AnyPermanent),TargetPermanent(AnyPermanent)],Targeted([TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent)])))`,
		`ActionList([TapPermanent(Ref_TargetPermanent1)])`,
		`Targeted([TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent1)]))`,
		`Modal_ChooseOne([Targeted([TargetPermanent(AnyPermanent),TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent1)])),ActionList([TapPermanent(Ref_TargetPermanent2)])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(`+body+`)])`), "invalid-groups.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", body)
		}
	}
}

func TestRONSingularGroupsStrictLoadAndConsumers(t *testing.T) {
	src := `Card(Name:"Groups",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(` + singularGroupsBody + `)])`
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fstest.MapFS{"groups.ron": {Data: []byte(src)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	if err := LoadInto(registry, fstest.MapFS{"groups.ron": {Data: []byte(src)}, "bad.ron": {Data: []byte(strings.ReplaceAll(src, "Ref_TargetPermanent1", "Ref_TargetPermanent3"))}}, "*.ron"); err == nil {
		t.Fatal("invalid collection installed")
	}
	if !registry.CardRegistered("Groups") || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("rejected load changed installed registry")
	}
	for _, tc := range []struct{ set, name, want string }{
		{"LEG", "Gauntlets of Chaos", ""},
		{"LEG", "Glyph of Delusion", "supply one indefinite type modification"},
		{"PROMOS", "Arena", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := chosenDiscardRecord(t, tc.set, tc.name)
			if _, err := Compile(record); (tc.want == "" && err != nil) || (tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want))) {
				t.Fatalf("remaining blocker: %v", err)
			}
		})
	}
}

func TestRONSingularGroupsCopyRetarget(t *testing.T) {
	for _, activation := range []bool{false, true} {
		t.Run(fmt.Sprint(activation), func(t *testing.T) {
			g, a, b := ronGame()
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 3), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewArtifact("Second", ""), b.PlayerID())
			g.TapPermanent(second)
			replacement1 := g.PutOnBattlefield(mage.NewCreature("Replacement1", "", 2, 3), b.PlayerID())
			replacement2 := g.PutOnBattlefield(mage.NewArtifact("Replacement2", ""), b.PlayerID())
			g.TapPermanent(replacement2)
			card := singularGroupsFactory(t, singularGroupsBody, activation)()
			card.SetOwner(a.PlayerID())
			a.ManaPool().Add(core.Blue, 1)
			if activation {
				source := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID(), second.ID()}); err != nil {
					t.Fatal(err)
				}
				g.PhaseOut(source.ID())
			} else {
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID(), second.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			}
			phased := append(g.PhaseOut(first.ID()), g.PhaseOut(second.ID())...)
			copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
			if copied == nil || len(copied.Targets) != 2 || copied.Targets[0] != replacement1.ID() || copied.Targets[1] != replacement2.ID() {
				t.Fatalf("retargeted groups: %v", copied)
			}
			g.PhaseIn(phased)
			g.ResolveTopOfStack()
			if g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped || !g.FindPermanent(replacement1.ID()).Tapped || g.FindPermanent(replacement2.ID()).Tapped {
				t.Fatal("copy followed original groups")
			}
			g.ResolveStack()
			if !g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped {
				t.Fatal("original followed copied groups")
			}
		})
	}
}

func TestRONSingularGroupsDelayedLKI(t *testing.T) {
	body := `Targeted([TargetPermanent(IsCardtype(Creature)),TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent1,[AdjustPTX(PlusX,Zero,PowerOfPermanent(Ref_TargetPermanent2))],UntilEndOfTurn)]))]))`
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 2, 20), a.PlayerID())
	info := mage.NewCreature("Information", "", 4, 20)
	info.SetOwner(b.PlayerID())
	second := g.PutOnBattlefield(info, b.PlayerID())
	spell := singularGroupsFactory(t, body, false)()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{first.ID(), second.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
	returned := g.PutOnBattlefield(info, b.PlayerID())
	returned.AddCounter(core.P1P1, 5)
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
		game.ResolveStack()
		if game.FindPermanent(first.ID()).CurrentPower(game) != 6 || game.FindPermanent(second.ID()).CurrentPower(game) != 9 {
			t.Fatal("delayed groups lost identity or retained information")
		}
	}
	g.RunStepWithPriority(core.Cleanup)
	if g.FindPermanent(first.ID()).CurrentPower(g) != 2 || clone.FindPermanent(first.ID()).CurrentPower(clone) != 6 {
		t.Fatal("delayed cleanup or clone isolation")
	}
}

func TestRONSingularGroupsTriggeredAndModal(t *testing.T) {
	for _, context := range []string{"entry", "modal"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(mage.NewCreature("Creature", "", 2, 3), b.PlayerID())
			q := g.PutOnBattlefield(mage.NewArtifact("Artifact", ""), b.PlayerID())
			g.TapPermanent(q)
			if context == "entry" {
				factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),`+singularGroupsBody+`)`)
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			} else {
				factory := singularGroupsFactory(t, `Modal_ChooseOne([`+singularGroupsBody+`,ActionList([GainLife(Integer(9))])])`, false)
				card := factory()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Blue, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID(), q.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if !g.FindPermanent(p.ID()).Tapped || g.FindPermanent(q.ID()).Tapped || a.Life() != 21 {
				t.Fatal("context lost numbered target bindings")
			}
		})
	}
}
