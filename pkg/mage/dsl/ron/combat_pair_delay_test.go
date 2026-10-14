package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONCombatPairDelayFlotilla(t *testing.T) {
	for _, scenario := range []string{"paid", "declined", "short", "control", "source-left", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "declined"}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "FEM", "Goblin Flotilla")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 1, 8), b.PlayerID())
			if scenario == "paid" || scenario == "declined" {
				a.ManaPool().Add(core.Red, 1)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.offers != 0 {
				t.Fatal("combat trigger timing")
			}
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-left":
				g.RemoveFromBattlefield(p)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			want := 1
			if scenario == "paid" || scenario == "source-left" {
				want = 0
			}
			if scenario == "copy" {
				want = 2
			}
			if len(g.DelayedTriggers()) != want {
				t.Fatalf("delays=%d want %d", len(g.DelayedTriggers()), want)
			}
			if scenario == "declined" && (a.offers != 1 || a.ManaPool().Count(core.Red) != 1) {
				t.Fatal("declined payment spent mana or omitted choice")
			}
			if scenario == "paid" && a.ManaPool().Count(core.Red) != 0 {
				t.Fatal("payment missing")
			}
			for range 2 {
				g.FireEvent(core.GameEvent{Type: core.EvtCombatPair, SourceID: p.ID(), TargetID: q.ID()})
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 2*want {
				t.Fatal("pair multiplicity")
			}
			g.ResolveStack()
			if q.HasKeyword(core.FirstStrike) != (want > 0) {
				t.Fatal("opponent grant missing")
			}
			g.RunStepWithPriority(core.EndCombat)
			if len(g.DelayedTriggers()) != 0 || q.HasKeyword(core.FirstStrike) != (want > 0) {
				t.Fatal("duration mismatch")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if q.HasKeyword(core.FirstStrike) {
				t.Fatal("first strike survived cleanup")
			}
		})
	}
}

func TestRONCombatPairDelayNative(t *testing.T) {
	for _, attack := range []bool{false, true} {
		a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
		g := mage.NewGame(a, b)
		g.SetTurn(1)
		p := g.PutOnBattlefield(repositoryCard(t, "FEM", "Goblin Flotilla"), a.PlayerID())
		q := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 2, 8), b.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		q.RevokeBaseAttr(core.AttrSummonSick)
		if attack {
			a.SetAttackers(1, []string{"Goblin Flotilla"})
			b.SetBlockers(1, map[string]string{"Opponent": "Goblin Flotilla"})
		} else {
			g.SetActivePlayerIndex(1)
			b.SetAttackers(1, []string{"Opponent"})
			a.SetBlockers(1, map[string]string{"Goblin Flotilla": "Opponent"})
		}
		g.RunStepWithPriority(core.BeginCombat)
		g.RunStepWithPriority(core.DeclareAttackers)
		g.RunStepWithPriority(core.DeclareBlockers)
		if !q.HasKeyword(core.FirstStrike) || p.HasKeyword(core.FirstStrike) {
			t.Fatal("native pair capture")
		}
		g.RunStepWithPriority(core.FirstStrikeDamage)
		if g.FindPermanent(p.ID()) != nil || q.Damage != 0 {
			t.Fatal("first strike did not kill before normal damage")
		}
		g.RunStepWithPriority(core.CombatDamage)
		g.RunStepWithPriority(core.EndCombat)
		if len(g.DelayedTriggers()) != 0 || !q.HasKeyword(core.FirstStrike) {
			t.Fatal("combat duration")
		}
	}
}

func TestRONCombatPairDelayInvalid(t *testing.T) {
	record := repositoryRecord(t, "FEM", "Goblin Flotilla", "FEM.lines.ron")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(factory().Name(), factory()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	raw := ""
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Goblin Flotilla"`) {
			raw = line
			break
		}
	}
	if raw == "" {
		t.Fatal("record missing")
	}
	for _, bad := range []string{
		strings.Replace(raw, "UntilEndOfCombat", "UntilEndOfCombat()", 1),
		strings.Replace(raw, "UntilEndOfCombat", "UntilEndOfTurn", 1),
		strings.Replace(raw, "Trigger_ThatOtherCreature", "Trigger_TheBlockingCreature", 1),
		strings.Replace(raw, "SinglePermanent(ThisPermanent)", "SinglePermanent(HostPermanent)", 1),
		strings.Replace(raw, "WhenACreatureBlocksACreature", "WhenACreatureBecomesBlockedByACreature", 1),
		strings.Replace(raw, "ActionList([CreatePermanentLayerEffectUntil(Trigger_ThatOtherCreature, [AddAbility([FirstStrike])], UntilEndOfTurn)])", "ActionList([])", 1),
	} {
		registry := mage.NewRegistry()
		if err := LoadInto(registry, fstest.MapFS{"bad.ron": {Data: []byte(bad)}}, "*.ron"); err == nil {
			t.Fatal("invalid delayed pair accepted")
		}
	}
}

func TestRONCombatPairDelayInstancesAndStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	raw := ""
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Goblin Flotilla"`) {
			raw = line
			break
		}
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fstest.MapFS{"flotilla.ron": {Data: []byte(raw)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 || !registry.CardRegistered("Goblin Flotilla") {
		t.Fatal("strict load missing card")
	}
	bad := strings.Replace(raw, "Trigger_ThatOtherCreature", "Ref_TargetPermanent", 1)
	if err := LoadInto(registry, fstest.MapFS{"bad.ron": {Data: []byte(bad)}}, "*.ron"); err == nil || !registry.CardRegistered("Goblin Flotilla") {
		t.Fatal("atomic rejection lost registry")
	}
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	p := g.PutOnBattlefield(repositoryCard(t, "FEM", "Goblin Flotilla"), a.PlayerID())
	r := g.PutOnBattlefield(repositoryCard(t, "FEM", "Goblin Flotilla"), b.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 8), b.PlayerID())
	g.RunStepWithPriority(core.BeginCombat)
	if len(g.DelayedTriggers()) != 2 {
		t.Fatal("factory instances merged")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtCombatPair, SourceID: p.ID(), TargetID: q.ID()})
	g.FireEvent(core.GameEvent{Type: core.EvtCombatPair, SourceID: r.ID(), TargetID: p.ID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	g.ResolveStack()
	if !g.FindPermanent(p.ID()).HasKeyword(core.FirstStrike) || !g.FindPermanent(q.ID()).HasKeyword(core.FirstStrike) || g.FindPermanent(r.ID()).HasKeyword(core.FirstStrike) {
		t.Fatal("event frames merged")
	}
	if clone.FindPermanent(p.ID()).HasKeyword(core.FirstStrike) {
		t.Fatal("clone shared grants")
	}
	clone.ResolveStack()
	g.RunStepWithPriority(core.EndCombat)
	if len(g.DelayedTriggers()) != 0 || len(clone.DelayedTriggers()) != 2 {
		t.Fatal("clone shared expiry")
	}
	a.accept = true
	a.ManaPool().Add(core.Red, 1)
	g.RunStepWithPriority(core.BeginCombat)
	if len(g.DelayedTriggers()) != 1 || a.ManaPool().Count(core.Red) != 0 {
		t.Fatal("additional combat did not choose again")
	}
	g.RunStepWithPriority(core.EndCombat)
	if len(g.DelayedTriggers()) != 0 {
		t.Fatal("additional combat retained subscription")
	}
}
