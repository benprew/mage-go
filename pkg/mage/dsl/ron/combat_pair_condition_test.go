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

func TestRONCombatPairConditionNative(t *testing.T) {
	for _, group := range []string{"alone", "two-walls", "nonwall", "unrelated"} {
		t.Run(group, func(t *testing.T) {
			g, a, b := ronGame()
			wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Caltrops"), b.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 0, 8, mage.WithKeyword(core.Shroud)), a.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			opts := []mage.CardOption{mage.WithSubTypes("Wall")}
			if group == "nonwall" {
				opts = nil
			}
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 8, opts...), b.PlayerID())
			if wall.CanDeclareAsAttacker(g) || wall.CurrentPower(g) != 2 || wall.CurrentToughness(g) != 1 {
				t.Fatal("printed characteristics")
			}
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			assignments := []mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}}
			if group == "two-walls" || group == "nonwall" {
				assignments = append(assignments, mage.BlockAssignment{BlockerID: other.ID(), AttackerID: attacker.ID()})
			}
			g.ExecuteBlockers(assignments)
			if g.FindPermanent(wall.ID()).HasKeyword(core.Banding) != (group == "two-walls") {
				t.Fatal("wrong blocking condition")
			}
			g.ExecuteCombatDamage()
			if g.FindPermanent(attacker.ID()).Damage != 2 {
				t.Fatal("printed combat damage")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanent(wall.ID()).HasKeyword(core.Banding) {
				t.Fatal("banding did not expire")
			}
		})
	}
}

func caltropsPair(t *testing.T) (*mage.Game, *mage.BasePlayer, *mage.BasePlayer, *mage.Permanent, *mage.Permanent, *mage.Permanent) {
	t.Helper()
	g, a, b := ronGame()
	wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Caltrops"), a.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 0, 8), b.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other wall", "", 0, 8, mage.WithSubTypes("Wall")), a.PlayerID())
	g.GetCombat().AddAttackerIncarnation(attacker.ID(), g.CaptureDamageSource(attacker.ID()).Incarnation(), a.PlayerID())
	for _, p := range []*mage.Permanent{wall, other} {
		g.GetCombat().AddBlockerIncarnation(p.ID(), g.CaptureDamageSource(p.ID()).Incarnation(), attacker.ID())
	}
	return g, a, b, wall, attacker, other
}

func TestRONCombatPairConditionRechecks(t *testing.T) {
	for _, change := range []string{"none", "other-removed", "other-left-before-placement", "nonwall-before", "nonwall-after", "nonwall-removed", "subtype-before", "subtype-after", "other-return", "other-phase", "attacker-return", "attacker-phase", "attacker-removed", "source-return", "source-phase", "source-cycle", "source-left", "source-control", "abilities", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, wall, attacker, other := caltropsPair(t)
			addNonwall := func() *mage.Permanent {
				p := g.PutOnBattlefield(mage.NewCreature("Nonwall", "", 0, 8), a.PlayerID())
				g.GetCombat().AddBlockerIncarnation(p.ID(), g.CaptureDamageSource(p.ID()).Incarnation(), attacker.ID())
				return p
			}
			var nonwall *mage.Permanent
			if change == "nonwall-before" {
				nonwall = addNonwall()
			}
			if change == "subtype-before" {
				g.AddContinuousEffect(mage.SetSubtypes(other.ID(), mage.SubtypeCreature, core.Indefinite, "Goblin"))
				g.ApplyContinuousEffects()
			}
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: wall.ID(), TargetID: attacker.ID(), PlayerID: a.PlayerID()})
			if change == "other-left-before-placement" {
				g.RemoveFromBattlefield(other)
			}
			if change == "nonwall-before" {
				g.RemoveFromCombat(nonwall.ID())
			}
			g.PutTriggersOnStack()
			detected := change != "nonwall-before" && change != "subtype-before"
			if (g.StackSize() == 1) != detected {
				t.Fatal("event-time check", g.StackSize())
			}
			switch change {
			case "other-removed":
				g.RemoveFromCombat(other.ID())
			case "nonwall-after":
				addNonwall()
			case "nonwall-removed":
				nonwall = addNonwall()
				g.RemoveFromCombat(nonwall.ID())
			case "subtype-after":
				g.AddContinuousEffect(mage.SetSubtypes(other.ID(), mage.SubtypeCreature, core.Indefinite, "Goblin"))
				g.ApplyContinuousEffects()
			case "other-return":
				card := other.Card
				g.RemoveFromBattlefield(other)
				other = g.PutOnBattlefield(card, a.PlayerID())
				g.GetCombat().AddBlockerIncarnation(other.ID(), g.CaptureDamageSource(other.ID()).Incarnation(), attacker.ID())
			case "other-phase":
				g.PhaseIn(g.PhaseOut(other.ID()))
			case "attacker-return":
				card := attacker.Card
				g.RemoveFromBattlefield(attacker)
				attacker = g.PutOnBattlefield(card, b.PlayerID())
				g.GetCombat().AddAttackerIncarnation(attacker.ID(), g.CaptureDamageSource(attacker.ID()).Incarnation(), a.PlayerID())
				g.GetCombat().AddBlockerIncarnation(other.ID(), g.CaptureDamageSource(other.ID()).Incarnation(), attacker.ID())
			case "attacker-phase":
				g.PhaseIn(g.PhaseOut(attacker.ID()))
			case "attacker-removed":
				g.RemoveFromCombat(attacker.ID())
			case "source-return":
				card := wall.Card
				g.RemoveFromBattlefield(wall)
				wall = g.PutOnBattlefield(card, a.PlayerID())
			case "source-phase":
				g.PhaseOut(wall.ID())
			case "source-cycle":
				g.PhaseIn(g.PhaseOut(wall.ID()))
			case "source-left":
				g.RemoveFromBattlefield(wall)
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: attacker.ID(), TargetID: wall.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "abilities":
				if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.RemoveAbilities(mage.This)), wall.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.StackSize() != boolStack(detected, change == "copy") {
				t.Fatal("clone changed original stack")
			}
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				if change == "source-phase" {
					game.PhaseIn([]uuid.UUID{wall.ID()})
				}
				current := game.FindPermanent(wall.ID())
				want := change == "none" || change == "nonwall-removed" || change == "other-return" || change == "source-cycle" || change == "source-control" || change == "abilities" || change == "copy"
				if current != nil && current.HasKeyword(core.Banding) != want {
					t.Fatal("resolution check or exact source visit")
				}
			}
		})
	}
}

func boolStack(detected, copied bool) int {
	if !detected {
		return 0
	}
	if copied {
		return 2
	}
	return 1
}

func TestRONCombatPairConditionInvalid(t *testing.T) {
	raw := hostUntapRecord(t, "LEG", "Wall of Caltrops")
	for _, tc := range []struct{ from, to string }{
		{"GreaterThanOrEqualTo(Integer(1))", "GreaterThanOrEqualTo"},
		{"GreaterThanOrEqualTo(Integer(1))", "GreaterThanOrEqualTo(Integer(-1))"},
		{"GreaterThanOrEqualTo(Integer(1))", "GreaterThanOrEqualTo(ValueX)"},
		{"GreaterThanOrEqualTo(Integer(1))", "LessThan(Integer(1))"},
		{"Trigger_ThatOtherCreature", "Trigger_TheBlockingCreature"},
		{"Trigger_ThatCreature", "Ref_TargetPermanent"},
		{"Trigger_ThatCreature", "Trigger_ThatCreature()"},
		{"IsBlockingAttacker(Trigger_ThatOtherCreature)", "IsBlockingAttacker()"},
		{"IsBlockingAttacker(Trigger_ThatOtherCreature)", "OnTheBattlefield"},
		{"Other(Trigger_ThatCreature)", "Other(You)"},
		{"IsCreatureType(Wall)", "IsCreatureType(Island)"},
		{"And([Other(Trigger_ThatCreature), IsCreatureType(Wall), IsCardtype(Creature)])", "And([])"},
	} {
		t.Run(tc.to, func(t *testing.T) {
			data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
			if err != nil {
				t.Fatal(err)
			}
			record := strings.Split(string(data), "\n")[raw.Position.Line-1]
			nodes, err := Parse(strings.NewReader(strings.ReplaceAll(record, tc.from, tc.to)), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("accepted invalid condition")
			}
		})
	}
	for _, condition := range []string{"And([])", "Or([])", "NoPermanentsPassFilter(IsCardtype(Creature))", "NumPermanentsPassFilter(GreaterThanOrEqualTo(Integer(1)),IsCardtype(Creature))"} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:0,Toughness:8),Rules:[TriggerI(WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature)),`+condition+`,ActionList([TapPermanent(ThisPermanent)]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid", condition)
		}
	}
}

func TestRONCombatPairConditionCastingLoadAndInventory(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEG", "Wall of Caltrops")
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid cast did not roll back")
	}
	a.ManaPool().Add(core.White, 1)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("printed payment or entry")
	}
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	record := strings.Split(string(data), "\n")[hostUntapRecord(t, "LEG", "Wall of Caltrops").Position.Line-1]
	files := fstest.MapFS{"wall.ron": {Data: []byte(record)}, "reprint.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("complete load", err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(record, "Trigger_ThatOtherCreature", "Ref_TargetPermanent"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Wall of Caltrops") {
		t.Fatal("invalid collection published")
	}
	delete(files, "invalid.ron")
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 2 {
		t.Fatal("inventory", err)
	}
	for _, signature := range report.Signatures {
		if signature.Name == "IsBlockingAttacker" && signature.Role != "permanent_filter" {
			t.Fatal("relation role", signature.Role)
		}
		if signature.Name == "GreaterThanOrEqualTo" && signature.Role != "value_comparison" {
			t.Fatal("comparison role", signature.Role)
		}
	}
}

func TestRONCombatPairConditionIndependentInstances(t *testing.T) {
	g, a, b := ronGame()
	walls := make([]*mage.Permanent, 0, 2)
	others := make([]*mage.Permanent, 0, 2)
	for range 2 {
		wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Caltrops"), a.PlayerID())
		attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 0, 8), b.PlayerID())
		other := g.PutOnBattlefield(mage.NewCreature("Other wall", "", 0, 8, mage.WithSubTypes("Wall")), a.PlayerID())
		g.GetCombat().AddAttackerIncarnation(attacker.ID(), g.CaptureDamageSource(attacker.ID()).Incarnation(), a.PlayerID())
		for _, p := range []*mage.Permanent{wall, other} {
			g.GetCombat().AddBlockerIncarnation(p.ID(), g.CaptureDamageSource(p.ID()).Incarnation(), attacker.ID())
		}
		g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: wall.ID(), TargetID: attacker.ID(), PlayerID: a.PlayerID()})
		walls = append(walls, wall)
		others = append(others, other)
	}
	g.RemoveFromCombat(others[0].ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("independent pending events lost")
	}
	clone := g.Clone()
	clone.RemoveFromCombat(others[1].ID())
	clone.ResolveStack()
	g.ResolveStack()
	for i, wall := range walls {
		if g.FindPermanent(wall.ID()).HasKeyword(core.Banding) != (i == 1) || clone.FindPermanent(wall.ID()).HasKeyword(core.Banding) {
			t.Fatal("instance, event or clone frame crossed")
		}
	}
}

func TestRONCombatPairConditionDirectionsAndHost(t *testing.T) {
	for _, direction := range []string{"WhenACreatureBlocksACreature", "WhenACreatureBecomesBlockedByACreature", "union", "host"} {
		event := direction + `(SinglePermanent(ThisPermanent),IsCardtype(Creature))`
		if direction == "union" {
			event = `Or([WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature)),WhenACreatureBecomesBlockedByACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature))])`
		}
		relation, kind := `IsBlockingAttacker(Trigger_ThatOtherCreature)`, core.EvtDeclaredBlocker
		if direction == "WhenACreatureBecomesBlockedByACreature" {
			relation = `IsBlockedByDefender(Trigger_TheBlockingCreature)`
			kind = core.EvtCreatureBlockedBy
		}
		if direction == "union" {
			kind = core.EvtCombatPair
		}
		typeline := `Cardtypes:[Creature],Subtypes:[]`
		rules := ""
		if direction == "host" {
			typeline = `Cardtypes:[Enchantment],Subtypes:[Aura]`
			rules = `EnchantPermanent(IsCardtype(Creature)),`
			event = `WhenACreatureBlocksACreature(SinglePermanent(HostPermanent),IsCardtype(Creature))`
		}
		rule := `TriggerI(` + event + `,Or([NumPermanentsPassFilter(GreaterThanOrEqualTo(Integer(1)),IsCardtype(Creature),` + relation + `),NumPermanentsPassFilter(GreaterThanOrEqualTo(Integer(2)),IsCardtype(Creature),` + relation + `)]),ActionList([PlayerAction(You,GainLife(Integer(1)))]))`
		nodes, err := Parse(strings.NewReader(`Card(Name:"Condition",Typeline:(Supertypes:[],`+typeline+`),CardPT:(Power:0,Toughness:8),Rules:[`+rules+rule+`])`), "condition.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		subject := source
		if direction == "host" {
			subject = g.PutOnBattlefield(mage.NewCreature("Host", "", 0, 8), a.PlayerID())
			g.Attach(source.ID(), subject.ID())
		}
		opponent := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 0, 8), b.PlayerID())
		attacker, blocker := opponent, subject
		if kind == core.EvtCreatureBlockedBy {
			attacker, blocker = subject, opponent
		}
		g.GetCombat().AddAttackerIncarnation(attacker.ID(), g.CaptureDamageSource(attacker.ID()).Incarnation(), b.PlayerID())
		g.GetCombat().AddBlockerIncarnation(blocker.ID(), g.CaptureDamageSource(blocker.ID()).Incarnation(), attacker.ID())
		g.FireEvent(core.GameEvent{Type: kind, SourceID: subject.ID(), TargetID: opponent.ID(), PlayerID: a.PlayerID()})
		if direction == "host" {
			newHost := g.PutOnBattlefield(mage.NewCreature("New host", "", 0, 8), a.PlayerID())
			g.Attach(source.ID(), newHost.ID())
		}
		g.PutTriggersOnStack()
		g.ResolveStack()
		if a.Life() != 21 {
			t.Fatal("direction or saved host")
		}
	}
}
