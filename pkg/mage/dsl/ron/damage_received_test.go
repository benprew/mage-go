package ron

import (
	"fmt"
	"maps"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func livingArtifactDamageClause(t *testing.T) mage.Card {
	t.Helper()
	n := hostUntapRecord(t, "LEB", "Living Artifact")
	n.Fields = maps.Clone(n.Fields)
	rules := n.Fields["Rules"]
	rules.Args = rules.Args[:2:2]
	n.Fields["Rules"] = rules
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	c := factory()
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRONDamageReceivedFungusaur(t *testing.T) {
	for _, change := range []string{"normal", "zero", "prevented", "partial", "other", "twice", "lethal", "departure", "return", "phase-before", "phase-after", "phase-cycle", "control", "removed", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEB", "Fungusaur")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			otherCard := mage.NewCreature("Other", "", 1, 8)
			otherCard.SetOwner(b.PlayerID())
			other := g.PutOnBattlefield(otherCard, b.PlayerID())
			if change == "phase-before" {
				g.PhaseOut(p.ID())
			}
			if change == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			if change == "prevented" {
				g.AddPreventionShield(p.ID(), 1)
			}
			if change == "partial" {
				g.AddPreventionShield(p.ID(), 1)
			}
			amount := 1
			if change == "zero" {
				amount = 0
			}
			if change == "partial" || change == "lethal" {
				amount = 2
			}
			victim := p
			if change == "other" {
				victim = other
			}
			g.DealDamageToPermanent(victim, amount, other.ID())
			want := 1
			if change == "zero" || change == "prevented" || change == "other" || change == "phase-before" || change == "removed" {
				want = 0
			}
			if change == "twice" {
				g.DealDamageToPermanent(p, 1, other.ID())
				want = 2
			}
			if p.Counters[core.P1P1] != 0 {
				t.Fatal("counter added before resolution")
			}
			switch change {
			case "lethal":
				g.CheckStateBasedActions()
			case "departure", "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				if change == "return" {
					g.PutOnBattlefield(c, b.PlayerID())
				}
			case "phase-after", "phase-cycle":
				g.PhaseOut(p.ID())
				if change == "phase-cycle" {
					g.PhaseIn([]uuid.UUID{p.ID()})
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.PutTriggersOnStack()
			if g.StackSize() != want {
				t.Fatalf("stack %d, want %d", g.StackSize(), want)
			}
			clone := g.Clone()
			if change == "copy" {
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				want++
			}
			g.ResolveStack()
			if change == "lethal" || change == "departure" {
				if g.FindPermanent(p.ID()) != nil {
					t.Fatal("absent source restored")
				}
				if change == "lethal" && len(a.Graveyard()) != 1 {
					t.Fatal("lethal damage did not kill source before trigger")
				}
				return
			}
			if change == "return" || change == "phase-after" {
				want = 0
			}
			got := g.FindPermanentIncludingPhased(p.ID())
			if got.Counters[core.P1P1] != want {
				t.Fatalf("counters %d, want %d", got.Counters[core.P1P1], want)
			}
			if clone.FindPermanentIncludingPhased(p.ID()).Counters[core.P1P1] != 0 {
				t.Fatal("clone changed with original")
			}
			clone.ResolveStack()
			cloneWant := want
			if change == "copy" {
				cloneWant--
			}
			if clone.FindPermanentIncludingPhased(p.ID()).Counters[core.P1P1] != cloneWant {
				t.Fatal("clone lost source identity")
			}
		})
	}
}

func TestRONDamageReceivedControllerClause(t *testing.T) {
	for _, change := range []string{"normal", "zero", "prevented", "partial", "opponent", "twice", "life-loss", "control-before", "control-after", "departure", "return", "phase-before", "phase-after", "removed", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := livingArtifactDamageClause(t)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			host := g.PutOnBattlefield(mage.NewArtifact("Host", ""), b.PlayerID())
			g.Attach(p.ID(), host.ID())
			if change == "control-before" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: host.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if change == "phase-before" {
				g.PhaseOut(p.ID())
			}
			if change == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			victim := a
			if change == "opponent" || change == "control-before" {
				victim = b
			}
			if change == "prevented" {
				g.AddPreventionShield(victim.PlayerID(), 3)
			}
			if change == "partial" {
				g.AddPreventionShield(victim.PlayerID(), 2)
			}
			amount := 3
			if change == "zero" {
				amount = 0
			}
			if change == "life-loss" {
				a.LoseLife(3)
			} else {
				g.DealDamageToPlayer(victim, amount, host.ID())
			}
			want, count := 3, 1
			if change == "partial" {
				want = 1
			}
			if change == "zero" || change == "prevented" || change == "opponent" || change == "life-loss" || change == "phase-before" || change == "removed" {
				want, count = 0, 0
			}
			if change == "twice" {
				g.DealDamageToPlayer(victim, 2, host.ID())
				want, count = 5, 2
			}
			if p.Counters[core.Vitality] != 0 {
				t.Fatal("damage amount resolved early")
			}
			switch change {
			case "control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: host.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase-after":
				g.PhaseOut(p.ID())
			case "departure", "return":
				g.RemoveFromBattlefield(p)
				if change == "return" {
					p = g.PutOnBattlefield(c, b.PlayerID())
					g.Attach(p.ID(), host.ID())
				}
			}
			g.PutTriggersOnStack()
			if g.StackSize() != count {
				t.Fatalf("stack %d, want %d", g.StackSize(), count)
			}
			clone := g.Clone()
			if change == "copy" {
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				want *= 2
			}
			g.ResolveStack()
			if change == "departure" {
				if g.FindPermanent(p.ID()) != nil {
					t.Fatal("departed Aura restored")
				}
				return
			}
			if change == "return" || change == "phase-after" {
				want = 0
			}
			if g.FindPermanentIncludingPhased(p.ID()).Counters[core.Vitality] != want {
				t.Fatalf("vitality counters, want %d", want)
			}
			if clone.FindPermanentIncludingPhased(p.ID()).Counters[core.Vitality] != 0 {
				t.Fatal("shared counter state")
			}
			clone.ResolveStack()
			if change == "copy" {
				want /= 2
			}
			if clone.FindPermanentIncludingPhased(p.ID()).Counters[core.Vitality] != want {
				t.Fatal("clone lost amount")
			}
		})
	}
	n := hostUntapRecord(t, "LEB", "Living Artifact")
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Living Artifact", factory()); err != nil {
		t.Fatal(err)
	}
}

func TestRONDamageReceivedBatch(t *testing.T) {
	g, a, b := ronGame()
	c1, c2 := livingArtifactDamageClause(t), livingArtifactDamageClause(t)
	first, second := g.PutOnBattlefield(c1, a.PlayerID()), g.PutOnBattlefield(c2, b.PlayerID())
	host := g.PutOnBattlefield(mage.NewArtifact("Host", ""), a.PlayerID())
	g.Attach(first.ID(), host.ID())
	g.Attach(second.ID(), host.ID())
	g.AddPreventionShield(a.PlayerID(), 1)
	src := g.CaptureDamageSource(host.ID())
	g.DealSimultaneousDamage(mage.PlayerDamageAssignment(a, 3, src), mage.PlayerDamageAssignment(a, 2, src), mage.PlayerDamageAssignment(b, 4, src))
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.PutTriggersOnStack()
		if game.StackSize() != 3 {
			t.Fatalf("stack %d", game.StackSize())
		}
		game.ResolveStack()
		if game.FindPermanent(first.ID()).Counters[core.Vitality] != 4 || game.FindPermanent(second.ID()).Counters[core.Vitality] != 4 {
			t.Fatal("event amounts or instances shared")
		}
	}
}

func TestRONDamageReceivedCastAndCombat(t *testing.T) {
	a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	c := repositoryCard(t, "LEB", "Fungusaur")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid cast changed state")
	}
	a.ManaPool().Add(core.Green, 4)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("early entry")
	}
	g.ResolveStack()
	p := g.FindPermanent(c.ID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	blocker := mage.NewCreature("Blocker", "", 1, 5)
	blocker.SetOwner(b.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	a.SetAttackers(1, []string{"Fungusaur"})
	b.SetBlockers(1, map[string]string{"Blocker": "Fungusaur"})
	g.RunStepWithPriority(core.DeclareAttackers)
	g.RunStepWithPriority(core.DeclareBlockers)
	g.RunStepWithPriority(core.CombatDamage)
	p = g.FindPermanent(c.ID())
	if p == nil || p.Counters[core.P1P1] != 1 || p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 3 || p.Damage != 1 {
		t.Fatal("native combat did not execute Fungusaur")
	}
}

func TestRONDamageReceivedReprintsAndInventory(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		files[set+".ron"] = &fstest.MapFile{Data: damageReceivedRecord(t, set, "Fungusaur")}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("reprints counted separately")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 6 {
		t.Fatalf("report: %v", err)
	}
	for _, s := range report.Signatures {
		if s.Name == "Trigger_ThatCreatureOrPlaneswalker" && s.Role != "permanent_reference" {
			t.Fatal("recipient role")
		}
		if s.Name == "ThisPermanent" && s.Role != "permanent_reference" {
			t.Fatal("event subject role")
		}
	}
	files["living.ron"] = &fstest.MapFile{Data: damageReceivedRecord(t, "LEB", "Living Artifact")}
	complete, err := LoadFS(files, "*.ron")
	if err != nil || complete == nil || len(complete.RegisteredCardNames()) != 2 {
		t.Fatalf("supported records failed loading: %v", err)
	}
	files["unsupported.ron"] = &fstest.MapFile{Data: damageReceivedRecord(t, "LEG", "Mirror Universe")}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("partial registry published")
	}
}

func TestRONDamageReceivedInvalid(t *testing.T) {
	for i, rule := range []string{
		`TriggerA(WhenAPermanentIsDealtDamage,ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPermanentIsDealtDamage(SinglePermanent(ThisPermanent),AnyPlayer),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPermanentIsDealtDamage(ThisPermanent),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPermanentIsDealtDamage(SinglePermanent(ThisPermanent())),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPermanentIsDealtDamage(SinglePermanent(HostPermanent)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPermanentIsDealtDamage(SinglePermanent(Ref_TargetPermanent)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerIsDealtDamage(AnyPlayer),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerIsDealtDamage(SinglePlayer(Opponent)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerIsDealtDamage(SinglePlayer(You())),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerIsDealtDamage(SinglePlayer(You)),ActionList([PutCounters([ACounterOfTypeOnPermanent(PTCounter(1,1),Trigger_ThatCreatureOrPlaneswalker)])]))`,
		`TriggerA(WhenAPermanentIsDealtDamage(SinglePermanent(ThisPermanent)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`,
		`TriggerA(WhenAPermanentIsDealtDamage(SinglePermanent(ThisPermanent)),ActionList([GainLife(Trigger_AmountOfDamageDealt())]))`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([PutCounters([NumberCountersOfTypeOnPermanent(Trigger_AmountOfDamageDealt,VitalityCounter,ThisPermanent)])]))`,
		`Activated(PayMana([]),ActionList([PutCounters([ACounterOfTypeOnPermanent(PTCounter(1,1),Trigger_ThatCreatureOrPlaneswalker)])]))`,
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			n, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(n[0]); err == nil {
				t.Fatal("accepted invalid event declaration")
			}
		})
	}
}

func damageReceivedRecord(t *testing.T, set, name string) []byte {
	t.Helper()
	n := hostUntapRecord(t, set, name)
	data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	return []byte(strings.Split(string(data), "\n")[n.Position.Line-1])
}

func TestRONDamageReceivedFinalRecipient(t *testing.T) {
	g, a, b := ronGame()
	creature := repositoryCard(t, "LEB", "Fungusaur")
	creature.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(creature, a.PlayerID())
	aura := livingArtifactDamageClause(t)
	aura.SetOwner(a.PlayerID())
	living := g.PutOnBattlefield(aura, a.PlayerID())
	host := g.PutOnBattlefield(mage.NewArtifact("Host", ""), b.PlayerID())
	g.Attach(living.ID(), host.ID())
	g.SetCreatureDamageRedirect(p.ID(), a.PlayerID())
	g.AddPreventionShield(a.PlayerID(), 1)
	g.DealDamageToPermanent(p, 2, host.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Counters[core.P1P1] != 0 || g.FindPermanent(p.ID()).Damage != 0 || a.Life() != 19 || g.FindPermanent(living.ID()).Counters[core.Vitality] != 1 {
		t.Fatal("triggers did not use final recipient and actual damage")
	}
}

func TestRONDamageReceivedPermanentAmountsAndInstances(t *testing.T) {
	rule := `TriggerA(WhenAPermanentIsDealtDamage(SinglePermanent(ThisPermanent)),ActionList([PutCounters([NumberCountersOfTypeOnPermanent(Trigger_AmountOfDamageDealt,VitalityCounter,Trigger_ThatCreatureOrPlaneswalker)])]))`
	nodes, err := Parse(strings.NewReader(`Card(Name:"Amount fixture",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:8),Rules:[`+rule+`])`), "amount.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	first, second := g.PutOnBattlefield(factory(), a.PlayerID()), g.PutOnBattlefield(factory(), b.PlayerID())
	first.GrantBaseAttr(core.Indestructible)
	second.GrantBaseAttr(core.Indestructible)
	src1, src2 := g.CaptureDamageSource(first.ID()), g.CaptureDamageSource(second.ID())
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(first, 2, src1), mage.PermanentDamageAssignment(first, 3, src2), mage.PermanentDamageAssignment(second, 4, src1))
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.PutTriggersOnStack()
		if game.StackSize() != 3 {
			t.Fatalf("stack %d", game.StackSize())
		}
		game.ResolveStack()
		if game.FindPermanent(first.ID()).Counters[core.Vitality] != 5 || game.FindPermanent(second.ID()).Counters[core.Vitality] != 4 {
			t.Fatal("recipient instances or packet amounts shared")
		}
	}
}
