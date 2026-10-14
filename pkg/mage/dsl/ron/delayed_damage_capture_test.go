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

const delayedDamageAction = `CreateTriggerUntil(WhenAPermanentDealsDamageToAPermanent(And([IsAttacking,IsCardtype(Creature)]),SinglePermanent(Ref_TargetPermanent)),ActionList([GainLife(Trigger_ThatMuch)]),UntilEndOfTurn)`

func TestRONDelayedDamageCaptureGlyph(t *testing.T) {
	for _, change := range []string{"normal", "not-attacking", "attacking-wall", "idle-wall", "prevent", "copy", "clone", "illegal", "return", "phase", "phase-cycle", "expire"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 20, mage.WithSubTypes("Wall")), b.PlayerID())
			attackerCard := mage.NewCreature("Attacker", "", 3, 20)
			if change == "attacking-wall" || change == "idle-wall" {
				attackerCard = mage.NewCreature("Other Wall", "", 3, 20, mage.WithSubTypes("Wall"))
			}
			attacker := g.PutOnBattlefield(attackerCard, a.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			spell := repositoryCard(t, "LEG", "Glyph of Life")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if len(g.DelayedTriggers()) != 0 {
				t.Fatal("installed before resolution")
			}
			if change == "illegal" {
				wall.GrantBaseAttr(core.Shroud)
			}
			if change == "copy" && g.CopySpellOnStack(spell.ID(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			if change == "illegal" {
				if len(g.DelayedTriggers()) != 0 {
					t.Fatal("illegal target subscribed")
				}
				return
			}
			if change == "return" {
				g.RemoveFromBattlefield(wall)
				wall = g.PutOnBattlefield(wall.Card, b.PlayerID())
			}
			if change == "phase" {
				g.PhaseOut(wall.ID())
			}
			if change == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(wall.ID()))
			}
			if change == "expire" {
				g.RunStepWithPriority(core.Cleanup)
			}
			if change != "not-attacking" && change != "idle-wall" {
				g.GetCombat().AddAttackerIncarnation(attacker.ID(), g.CaptureDamageSource(attacker.ID()).Incarnation(), b.PlayerID())
			}
			if change == "prevent" {
				g.AddPreventionShield(wall.ID(), 1)
			}
			clone := g.Clone()
			g.DealDamageToPermanent(g.FindPermanentIncludingPhased(wall.ID()), 3, attacker.ID())
			g.PutTriggersOnStack()
			g.ResolveStack()
			gain := 3
			if change == "prevent" {
				gain = 2
			}
			if change == "not-attacking" || change == "idle-wall" || change == "phase" || change == "return" || change == "expire" {
				gain = 0
			}
			if a.Life() != 20+gain {
				t.Fatalf("life=%d want %d", a.Life(), 20+gain)
			}
			if change == "copy" && b.Life() != 23 {
				t.Fatal("spell copy controller lost")
			}
			if change == "clone" {
				if clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("clone shared state")
				}
				clone.DealDamageToPermanent(clone.FindPermanent(wall.ID()), 2, attacker.ID())
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 22 {
					t.Fatal("clone lost subscription")
				}
			}
		})
	}
}

func TestRONDelayedDamageCaptureInvalid(t *testing.T) {
	valid := `Card(Name:"Watch",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + delayedDamageAction + `])))])`
	for _, bad := range []string{
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfCombat", 1),
		strings.Replace(valid, "SinglePermanent(Ref_TargetPermanent)", "AnyPermanent", 1),
		strings.Replace(valid, "ActionList([GainLife(Trigger_ThatMuch)])", "ActionList([])", 1),
		strings.Replace(valid, "Trigger_ThatMuch", "Trigger_ThatMuch()", 1),
		strings.Replace(valid, "Trigger_ThatMuch", "Trigger_ThatPlayer", 1),
		strings.Replace(valid, "SinglePermanent(Ref_TargetPermanent)", "SinglePermanent(Ref_TargetPermanent,ThisPermanent)", 1),
		strings.Replace(valid, "IsAttacking", "IsAttacking()", 1),
		strings.Replace(valid, delayedDamageAction, delayedDamageAction+`,GainLife(Trigger_ThatMuch)`, 1),
		strings.Replace(valid, delayedDamageAction, `GainLife(Trigger_ThatMuch)`, 1),
	} {
		nodes, err := Parse(strings.NewReader(bad), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	nodes, err := Parse(strings.NewReader(valid), "good.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err = mage.ValidateCardDefinition("Watch", factory()); err != nil {
		t.Fatal(err)
	}
}

func TestRONDelayedDamageCaptureLoad(t *testing.T) {
	raw, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var glyph string
	for line := range strings.SplitSeq(string(raw), "\n") {
		if strings.Contains(line, `Name: "Glyph of Life"`) {
			glyph = line
		}
	}
	files := fstest.MapFS{"glyph.ron": &fstest.MapFile{Data: []byte(glyph)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("complete record not loaded")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([GainLife(Trigger_ThatMuch)]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("strict load changed registry")
	}
}

func TestRONDelayedDamageCaptureInstancesAndNested(t *testing.T) {
	body := strings.Replace(delayedDamageAction, "GainLife(Trigger_ThatMuch)", "CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(Trigger_ThatMuch)]))", 1)
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+body+`])))`)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	wall := g.PutOnBattlefield(mage.NewCreature("Watched", "", 0, 20), b.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 20), a.PlayerID())
	g.GetCombat().AddAttackerIncarnation(attacker.ID(), g.CaptureDamageSource(attacker.ID()).Incarnation(), b.PlayerID())
	for _, p := range []*mage.Permanent{first, second} {
		if err := g.ActivateAbilityByIndex(p.ControllerID(), p.ID(), 0, []uuid.UUID{wall.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	g.DealDamageToPermanent(wall, 2, attacker.ID())
	g.DealDamageToPermanent(wall, 4, attacker.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("nested delay ran early")
	}
	if len(g.DelayedTriggers()) != 6 {
		t.Fatal("independent subscriptions or nested frames lost")
	}
	g.RemoveFromBattlefield(first)
	g.RemoveFromBattlefield(second)
	g.RemoveFromBattlefield(wall)
	clone := g.Clone()
	g.RunStepWithPriority(core.EndStep)
	if a.Life() != 26 || b.Life() != 26 {
		t.Fatal("nested delayed amounts or controllers shared")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone shared nested execution")
	}
	clone.RunStepWithPriority(core.EndStep)
	if clone.GetPlayer(a.PlayerID()).Life() != 26 || clone.GetPlayer(b.PlayerID()).Life() != 26 {
		t.Fatal("clone lost nested frames")
	}
}
