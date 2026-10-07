package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONLastTurnAttackCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		aura      bool
		mana      core.Color
		amount    int
	}{{"LEG", "Giant Turtle", false, core.Green, 3}, {"DRK", "Goblin Rock Sled", false, core.Red, 2}, {"DRK", "Tangle Kelp", true, core.Blue, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			g.SetStep(core.PrecombatMain)
			c := repositoryCard(t, tc.set, tc.name)
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			var targets []uuid.UUID
			var subject *mage.Permanent
			if tc.aura {
				subject = g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 4), b.PlayerID())
				targets = []uuid.UUID{subject.ID()}
			}
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), targets, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(tc.mana, tc.amount)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(c.ID())
			if source == nil {
				t.Fatal("cast did not enter")
			}
			if tc.aura {
				if source.AttachedTo != subject.ID() || !g.FindPermanent(subject.ID()).Tapped {
					t.Fatal("entry tap or attachment")
				}
				g.UntapPermanent(g.FindPermanent(subject.ID()))
				g.SetActivePlayerIndex(1)
			} else {
				subject = source
			}
			subject = g.MutablePermanent(subject.ID())
			subject.RevokeBaseAttr(core.AttrSummonSick)
			if tc.name == "Goblin Rock Sled" {
				if subject.CanDeclareAsAttacker(g) {
					t.Fatal("Mountain restriction missing")
				}
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
				if !g.FindPermanent(subject.ID()).HasAttr(core.Trample) {
					t.Fatal("trample missing")
				}
			}
			actor := a
			if tc.aura {
				actor = b
			}
			if !g.FindPermanent(subject.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("first attack prohibited")
			}
			g.ExecuteAttackers(actor.PlayerID(), []uuid.UUID{subject.ID()})
			if !g.GetCombat().IsAttacking(subject.ID()) {
				t.Fatal("attack not declared")
			}
			g.RunStepWithPriority(core.Cleanup)
			g.RunStepWithPriority(core.Untap)
			if tc.name == "Giant Turtle" {
				if g.FindPermanent(subject.ID()).Tapped || g.FindPermanent(subject.ID()).CanDeclareAsAttacker(g) {
					t.Fatal("Turtle attack restriction")
				}
			} else {
				if !g.FindPermanent(subject.ID()).Tapped {
					t.Fatal("history failed to restrict untap")
				}
				if !g.UntapPermanent(g.FindPermanent(subject.ID())) {
					t.Fatal("other untap prohibited")
				}
			}
			clone := g.Clone()
			clone.RunStepWithPriority(core.Cleanup)
			clone.RunStepWithPriority(core.Untap)
			if !clone.FindPermanent(subject.ID()).CanDeclareAsAttacker(clone) {
				t.Fatal("empty later turn retained attack history")
			}
			if tc.name == "Giant Turtle" && g.FindPermanent(subject.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("clone changed history")
			}
			if tc.aura {
				g.AddContinuousEffect(mage.RemoveAllAbilities(subject.ID(), core.Indefinite))
				g.TapPermanent(g.FindPermanent(subject.ID()))
				g.RunStepWithPriority(core.Untap)
				if !g.FindPermanent(subject.ID()).Tapped {
					t.Fatal("host removal erased Aura restriction")
				}
			}
		})
	}
}

func TestRONLastTurnAttackInvalid(t *testing.T) {
	for _, rule := range []string{
		`If(PermanentPassesFilter(ThisPermanent,AttackedDuringLastPlayersTurn(You())),[PermanentRuleEffect(ThisPermanent,[CantAttack])])`,
		`If(PermanentPassesFilter(ThisPermanent,AttackedDuringLastPlayersTurn(Opponent)),[PermanentRuleEffect(ThisPermanent,[CantAttack])])`,
		`If(PermanentPassesFilter(ThisPermanent,AttackedDuringLastPlayersTurn()),[PermanentRuleEffect(ThisPermanent,[CantAttack])])`,
		`If(PermanentPassesFilter(Ref_TargetPermanent,AttackedDuringLastPlayersTurn(You)),[PermanentRuleEffect(ThisPermanent,[CantAttack])])`,
		`If(PermanentPassesFilter(ThisPermanent,AttackedDuringLastPlayersTurn(You)),[PermanentRuleEffect(ThisPermanent,[CantAttack()])])`,
		`If(PermanentPassesFilter(ThisPermanent,AttackedDuringLastPlayersTurn(You)),[PermanentRuleEffect(ThisPermanent,[CantAttack,DoesntUntapDuringControllersUntap])])`,
		`If(PermanentPassesFilter(HostPermanent,AttackedDuringLastPlayersTurn(ItsController)),[PermanentRuleEffect(ThisPermanent,[CantAttack])])`,
	} {
		text := fmt.Sprintf(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[%s])`, rule)
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONLastTurnAttackLifecycle(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		aura      bool
	}{{"LEG", "Giant Turtle", false}, {"DRK", "Goblin Rock Sled", false}, {"DRK", "Tangle Kelp", true}} {
		for _, change := range []string{"none", "phase-cycle", "subject-return", "control", "source-phase", "source-remove", "source-leaves", "source-return", "host-remove", "move", "two"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				g.SetStep(core.PrecombatMain)
				c := repositoryCard(t, tc.set, tc.name)
				c.SetOwner(a.PlayerID())
				var source, subject *mage.Permanent
				hostCard := mage.NewCreature("Host", "", 2, 8)
				hostCard.SetOwner(b.PlayerID())
				actor := a.PlayerID()
				if tc.aura {
					subject = g.PutOnBattlefield(hostCard, b.PlayerID())
					source = g.PutOnBattlefield(c, a.PlayerID())
					g.Attach(source.ID(), subject.ID())
					g.ResolveStack()
					actor = b.PlayerID()
					g.SetActivePlayerIndex(1)
				} else {
					source = g.PutOnBattlefield(c, a.PlayerID())
					subject = source
				}
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), a.PlayerID())
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
				g.MutablePermanent(subject.ID()).RevokeBaseAttr(core.AttrSummonSick)
				g.UntapPermanent(g.FindPermanent(subject.ID()))
				g.ExecuteAttackers(actor, []uuid.UUID{subject.ID()})
				if !g.GetCombat().IsAttacking(subject.ID()) {
					t.Fatal("history attack failed")
				}
				g.RunStepWithPriority(core.Cleanup)
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8), actor)
				g.MutablePermanent(other.ID()).RevokeBaseAttr(core.AttrSummonSick)
				clone := g.Clone()
				restricted := true
				switch change {
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(subject.ID()))
				case "subject-return":
					card := c
					if tc.aura {
						card = hostCard
					}
					g.BouncePermanentToHand(g.FindPermanent(subject.ID()))
					g.GetPlayer(actor).RemoveFromHand(card.ID())
					subject = g.PutOnBattlefield(card, actor)
					g.MutablePermanent(subject.ID()).RevokeBaseAttr(core.AttrSummonSick)
					if tc.aura {
						g.Attach(source.ID(), subject.ID())
					}
					restricted = false
				case "control":
					newController := b.PlayerID()
					if tc.aura {
						newController = a.PlayerID()
					}
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: subject.ID(), ControllerID: newController, Duration: core.Indefinite})
					g.MutablePermanent(subject.ID()).RevokeBaseAttr(core.AttrSummonSick)
					if newController == a.PlayerID() {
						g.SetActivePlayerIndex(0)
					} else {
						g.SetActivePlayerIndex(1)
					}
					restricted = false
				case "source-phase":
					g.PhaseOut(source.ID())
					restricted = false
				case "source-remove":
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
					restricted = false
				case "source-return":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					a.RemoveFromHand(c.ID())
					source = g.PutOnBattlefield(c, a.PlayerID())
					if tc.aura {
						g.Attach(source.ID(), subject.ID())
						g.ResolveStack()
					} else {
						subject = source
						g.MutablePermanent(subject.ID()).RevokeBaseAttr(core.AttrSummonSick)
						restricted = false
					}
				case "source-leaves":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					restricted = false
				case "host-remove":
					if tc.aura {
						g.AddContinuousEffect(mage.RemoveAllAbilities(subject.ID(), core.Indefinite))
					} else {
						g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
						restricted = false
					}
				case "move":
					if tc.aura {
						g.Attach(source.ID(), other.ID())
						restricted = false
					}
				case "two":
					second := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), actor)
					if tc.aura {
						g.Attach(second.ID(), other.ID())
					} else {
						g.MutablePermanent(second.ID()).RevokeBaseAttr(core.AttrSummonSick)
						if !g.FindPermanent(second.ID()).CanDeclareAsAttacker(g) {
							t.Fatal("second factory inherited history")
						}
					}
				}
				if tc.name == "Giant Turtle" {
					if p := g.FindPermanent(subject.ID()); p != nil {
						g.UntapPermanent(p)
						if g.FindPermanent(subject.ID()).CanDeclareAsAttacker(g) == restricted {
							t.Fatal("live attack restriction", change)
						}
					}
				} else {
					if p := g.FindPermanent(subject.ID()); p != nil {
						g.TapPermanent(p)
					}
					g.TapPermanent(g.FindPermanent(other.ID()))
					g.RunStepWithPriority(core.Untap)
					if change == "source-phase" && !tc.aura {
						restricted = true
					}
					if p := g.FindPermanent(subject.ID()); p != nil && p.Tapped != restricted {
						t.Fatal("live untap restriction", change)
					}
					if g.FindPermanent(other.ID()).Tapped != (change == "control") {
						t.Fatal("other subject did not follow its controller untap")
					}
				}
				if tc.name == "Giant Turtle" {
					clone.UntapPermanent(clone.FindPermanent(c.ID()))
					if clone.FindPermanent(c.ID()).CanDeclareAsAttacker(clone) {
						t.Fatal("clone lost original history")
					}
				} else {
					clone.TapPermanent(clone.FindPermanent(subject.ID()))
					clone.RunStepWithPriority(core.Untap)
					if !clone.FindPermanent(subject.ID()).Tapped {
						t.Fatal("clone lost original history")
					}
				}
			})
		}
	}
}

func TestRONLastTurnAttackInventoryAndStrictLoad(t *testing.T) {
	var records []string
	for _, tc := range []struct{ set, name string }{{"LEG", "Giant Turtle"}, {"DRK", "Goblin Rock Sled"}, {"DRK", "Tangle Kelp"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+tc.name+`"`) {
				records = append(records, line)
			}
		}
	}
	if len(records) != 3 {
		t.Fatal("missing complete records")
	}
	files := fstest.MapFS{"cards.ron": {Data: []byte(strings.Join(records, "\n"))}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 3 || report.UniqueCards != 3 || len(report.Issues) != 0 {
		t.Fatal("complete records rejected", report.Issues)
	}
	found := false
	for _, s := range report.Signatures {
		if s.Name == "ItsController" && s.Role == "player_reference" {
			found = true
		}
	}
	if !found {
		t.Fatal("history player role missing")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatal("test registry", err)
	}
	for _, tc := range []struct{ set, name string }{{"LEG", "Giant Turtle"}, {"DRK", "Goblin Rock Sled"}, {"DRK", "Tangle Kelp"}} {
		c := repositoryCard(t, tc.set, tc.name)
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("published partial registry")
	}
}

func TestRONLastTurnAttackKelpEntryTimingAndLegality(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "DRK", "Tangle Kelp")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{land.ID()}, 0); err == nil {
		t.Fatal("noncreature target accepted")
	}
	if a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("illegal target changed resources")
	}
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 4), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(host.ID()).Tapped {
		t.Fatal("host tapped during announcement")
	}
	g.ResolveTopOfStack()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.FindPermanent(host.ID()).Tapped {
		t.Fatal("entry tap did not wait for trigger resolution")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if !clone.FindPermanent(host.ID()).Tapped || g.FindPermanent(host.ID()).Tapped {
		t.Fatal("pending trigger clone isolation")
	}
	g.ResolveStack()
	if !g.FindPermanent(host.ID()).Tapped {
		t.Fatal("entry tap failed")
	}
}

func TestRONLastTurnAttackConditionalBodyReuse(t *testing.T) {
	src := `Card(Name:"Conditional attack",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),If(PermanentPassesFilter(ThisPermanent,HasACounterOfType(NetCounter)),[PermanentRuleEffect(HostPermanent,[CantAttack])])])`
	nodes, err := Parse(strings.NewReader(src), "condition.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	c := factory()
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 4), b.PlayerID())
	g.MutablePermanent(host.ID()).RevokeBaseAttr(core.AttrSummonSick)
	source := g.PutOnBattlefield(c, a.PlayerID())
	g.Attach(source.ID(), host.ID())
	if !g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("false condition prohibited attack")
	}
	g.MutablePermanent(source.ID()).Counters[core.Net] = 1
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("true condition permitted attack")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(host.ID(), core.Indefinite))
	if g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("host removal erased Aura rule")
	}
	clone := g.Clone()
	clone.MutablePermanent(source.ID()).Counters[core.Net] = 0
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(host.ID()).CanDeclareAsAttacker(clone) || g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("condition clone state")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
	if !g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("source removal retained rule")
	}
}
