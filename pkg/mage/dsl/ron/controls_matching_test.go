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

func TestRONControlsMatchingSeasinger(t *testing.T) {
	for _, change := range []string{"none", "island-phase", "island-return", "target-control", "source-untap", "source-return", "source-control", "source-phase", "target-return", "target-phase", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			islandCard := repositoryCard(t, "LEB", "Island")
			island := g.PutOnBattlefield(islandCard, b.PlayerID())
			card := repositoryCard(t, "FEM", "Seasinger")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			victim := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 2, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "island-phase":
				g.PhaseOut(island.ID())
			case "island-return":
				g.RemoveFromBattlefield(island)
				g.PutOnBattlefield(islandCard, b.PlayerID())
			case "target-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.EndOfTurn})
			case "source-untap":
				g.UntapPermanent(source)
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, a.PlayerID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
			case "source-phase":
				g.PhaseOut(source.ID())
			case "target-return":
				victimCard := victim.Card
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			case "target-phase":
				g.PhaseOut(victim.ID())
			case "copy":
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanentIncludingPhased(victim.ID()).ControllerID() != b.PlayerID() && change != "target-control" {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			want := a.PlayerID()
			if change == "island-phase" || change == "source-untap" || change == "source-return" || change == "source-control" || change == "source-phase" || change == "target-return" || change == "target-phase" {
				want = b.PlayerID()
			}
			if g.FindPermanentIncludingPhased(victim.ID()).ControllerID() != want || clone.FindPermanentIncludingPhased(victim.ID()).ControllerID() != want {
				t.Fatal("wrong control result")
			}
			if change == "none" {
				g.PhaseOut(island.ID())
				g.ApplyContinuousEffects()
				if g.FindPermanent(victim.ID()).ControllerID() != a.PlayerID() {
					t.Fatal("target predicate ended a resolved control effect")
				}
				if !g.FindPermanent(source.ID()).HasAttr(core.AttrMayNotUntap) {
					t.Fatal("missing optional untap permission")
				}
				g.UntapPermanent(g.FindPermanent(source.ID()))
				g.ApplyContinuousEffects()
				if g.FindPermanentIncludingPhased(victim.ID()).ControllerID() != b.PlayerID() {
					t.Fatal("untap retained control")
				}
			}
		})
	}
}

func TestRONControlsMatchingOrgg(t *testing.T) {
	g, a, b := ronGame()
	g.SetActivePlayerIndex(0)
	g.SetStep(core.DeclareAttackers)
	card := repositoryCard(t, "FEM", "Orgg")
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	check := func(want bool) {
		t.Helper()
		g.ApplyContinuousEffects()
		if g.FindPermanent(source.ID()).CanDeclareAsAttacker(g) != want {
			t.Fatalf("attack want %v", want)
		}
	}
	check(true)
	defender := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
	check(false)
	if !g.FindPermanent(source.ID()).HasKeyword(core.Trample) {
		t.Fatal("missing trample")
	}
	if mage.CanBlock(source, defender, g) {
		t.Fatal("blocked a creature with power at least three")
	}
	small := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), b.PlayerID())
	if !mage.CanBlock(source, small, g) {
		t.Fatal("could not block a smaller creature")
	}
	g.TapPermanent(defender)
	check(true)
	g.UntapPermanent(g.FindPermanent(defender.ID()))
	check(false)
	g.PhaseOut(defender.ID())
	check(true)
}

func TestRONControlsMatchingInvalid(t *testing.T) {
	for _, predicate := range []string{`ControlsA`, `ControlsA()`, `ControlsA(IsLandType(Island),AnyPermanent)`, `ControlsA(SinglePermanent(ThisPermanent))`, `ControlsA(IsColor(TheChosenColor))`} {
		rule := `Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(ControlledByAPlayer(` + predicate + `))],ActionList([TapPermanent(Ref_TargetPermanent)])))`
		assertConditionalRejected(t, rule)
	}
}

func TestRONControlsMatchingCasting(t *testing.T) {
	for _, name := range []string{"Seasinger", "Orgg"} {
		t.Run(name, func(t *testing.T) {
			g, a, _ := ronGame()
			g.SetActivePlayerIndex(0)
			g.SetStep(core.PrecombatMain)
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			card := repositoryCard(t, "FEM", name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed zones")
			}
			for _, color := range []core.Color{core.Blue, core.Red, core.Colorless} {
				a.ManaPool().Add(color, 10)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil || len(a.Hand()) != 0 {
				t.Fatal("paid cast failed")
			}
		})
	}
}

func TestRONControlsMatchingSeasingerIllegal(t *testing.T) {
	g, a, b := ronGame()
	own := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
	source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Seasinger"), a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	victim := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 2, []uuid.UUID{victim.ID()}); err == nil {
		t.Fatal("illegal controller target")
	}
	if g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
		t.Fatal("illegal target paid cost")
	}
	g.PhaseOut(own.ID())
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("absence did not trigger")
	}
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("absence did not sacrifice")
	}
}

func TestRONControlsMatchingInvalidDefender(t *testing.T) {
	valid := `PermanentRuleEffect(ThisPermanent,[CantAttackIfDefendingPlayer(PlayerPassesFilter(DefendingPlayer,ControlsA(And([IsUntapped,IsCardtype(Creature),PowerIs(GreaterThanOrEqualTo(Integer(3)))]))))])`
	for _, replacement := range [][2]string{
		{"ThisPermanent,", "HostPermanent,"},
		{"DefendingPlayer,", "You,"},
		{"DefendingPlayer,", "DefendingPlayer(),"},
		{"ControlsA(", "ControlsNo("},
		{"IsUntapped,IsCardtype(Creature),PowerIs(GreaterThanOrEqualTo(Integer(3)))", ""},
		{"Integer(3)", "PowerOfPermanent(ThisPermanent)"},
		{"ControlsA(And([IsUntapped,IsCardtype(Creature),PowerIs(GreaterThanOrEqualTo(Integer(3)))]))", "ControlsA()"},
	} {
		assertConditionalRejected(t, strings.Replace(valid, replacement[0], replacement[1], 1))
	}
}

func TestRONControlsMatchingInventory(t *testing.T) {
	files := fstest.MapFS{}
	for _, name := range []string{"Seasinger", "Orgg"} {
		data, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+name+`"`) {
				files[name+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || len(report.Issues) != 0 {
		t.Fatalf("report = %+v", report)
	}
	roles := map[string]bool{}
	for _, signature := range report.Signatures {
		roles[signature.Role+"/"+signature.Name] = true
	}
	for _, key := range []string{"player_filter/ControlsA", "static_player_filter/ControlsA", "permanent_rule_operation/CantAttackIfDefendingPlayer", "defending_player_condition/PlayerPassesFilter", "permanent_filter/IsLandType"} {
		if !roles[key] {
			t.Fatalf("missing role %s", key)
		}
	}
}
