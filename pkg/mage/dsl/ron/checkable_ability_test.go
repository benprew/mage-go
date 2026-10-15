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

const checkableRemoval = `ChooseACheckableAbility([FirstStrike,Landwalk(IsLandType(Swamp))]),CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[LosesAbility(TheChosenAbility)],UntilEndOfTurn)`

type checkableCombatChooser struct {
	*gametest.TestPlayer
}

func (*checkableCombatChooser) ChooseMode([]string, string) int { return 1 }

func TestRONCheckableAbilityUrborg(t *testing.T) {
	for _, choice := range []int{0, 1, -1, 9} {
		for _, change := range []string{"none", "absent-keywords", "source-left", "source-return", "source-phase", "source-control", "source-disabled", "target-phase", "phase-cycle", "target-return", "shroud", "copy", "installed-phase", "installed-return"} {
			t.Run(fmt.Sprintf("%d/%s", choice, change), func(t *testing.T) {
				a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: choice}
				b := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("B"), choice: 1}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				card := repositoryCard(t, "LEG", "Urborg")
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.PlayLand(a.PlayerID(), card.ID()); err != nil {
					t.Fatal(err)
				}
				source := g.FindPermanent(card.ID())
				if !source.Card.HasSuperType(core.SuperLegendary) {
					t.Fatal("legendary land metadata lost")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().Count(core.Black) != 1 || g.StackSize() != 0 || a.prompts != 0 {
					t.Fatal("mana ability did not resolve immediately")
				}
				g.UntapPermanent(g.FindPermanent(source.ID()))
				opts := []mage.CardOption{mage.WithKeyword(core.DoubleStrike), mage.WithKeyword(core.Islandwalk)}
				if change != "absent-keywords" {
					opts = append(opts, mage.WithKeyword(core.FirstStrike), mage.WithKeyword(core.Swampwalk))
				}
				fixture := mage.NewCreature("Recipient", "", 2, 8, opts...)
				fixture.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(fixture, b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8, mage.WithKeyword(core.FirstStrike), mage.WithKeyword(core.Swampwalk)), b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{source.ID()}); err == nil {
					t.Fatal("noncreature target accepted")
				}
				if g.FindPermanent(source.ID()).Tapped || a.prompts != 0 || g.StackSize() != 0 {
					t.Fatal("invalid announcement paid or chose")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped || a.prompts != 0 || g.StackSize() != 1 {
					t.Fatal("tap cost or resolution choice timing")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err == nil || a.prompts != 0 || g.StackSize() != 1 {
					t.Fatal("tapped source activated again")
				}
				switch change {
				case "source-left", "source-return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if change == "source-return" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "source-phase":
					g.PhaseOut(source.ID())
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "source-disabled":
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				case "target-phase":
					g.PhaseOut(target.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "target-return":
					g.RemoveFromBattlefield(target)
					target = g.PutOnBattlefield(fixture, b.PlayerID())
				case "shroud":
					if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.GrantKeywordModification(mage.This, core.Shroud)), target.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				g.ApplyContinuousEffects()
				clone := g.Clone()
				g.ResolveStack()
				resolved := change != "target-phase" && change != "target-return" && change != "shroud"
				wantPrompts := 0
				if resolved {
					wantPrompts = 1
				}
				if a.prompts != wantPrompts || b.prompts != map[bool]int{true: 1}[change == "copy"] {
					t.Fatalf("choice controllers: A=%d B=%d", a.prompts, b.prompts)
				}
				if change == "installed-phase" {
					g.PhaseIn(g.PhaseOut(target.ID()))
				}
				if change == "installed-return" {
					g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
					target = g.PutOnBattlefield(fixture, b.PlayerID())
					resolved = false
				}
				selected := choice
				if selected < 0 || selected > 1 {
					selected = 0
				}
				p := g.FindPermanentIncludingPhased(target.ID())
				for i, keyword := range []core.Attr{core.FirstStrike, core.Swampwalk} {
					want := change != "absent-keywords"
					if resolved && (selected == i || change == "copy" && i == 1) {
						want = false
					}
					if p.HasKeyword(keyword) != want {
						t.Fatalf("%s=%v want %v", keyword, p.HasKeyword(keyword), want)
					}
					if !g.FindPermanent(other.ID()).HasKeyword(keyword) {
						t.Fatal("other target changed")
					}
				}
				if !p.HasKeyword(core.DoubleStrike) || !p.HasKeyword(core.Islandwalk) {
					t.Fatal("unselected keyword removed")
				}
				if change == "none" {
					if !clone.FindPermanent(target.ID()).HasKeyword(core.FirstStrike) || !clone.FindPermanent(target.ID()).HasKeyword(core.Swampwalk) {
						t.Fatal("clone shared removal")
					}
					clone.ResolveStack()
					if clone.FindPermanent(target.ID()).HasKeyword(core.FirstStrike) {
						t.Fatal("clone lost pending resolution choice")
					}
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				p = g.FindPermanentIncludingPhased(target.ID())
				if change != "absent-keywords" && (!p.HasKeyword(core.FirstStrike) || !p.HasKeyword(core.Swampwalk)) {
					t.Fatal("removal did not expire")
				}
			})
		}
	}
}

func TestRONCheckableAbilityInvalid(t *testing.T) {
	for _, actions := range []string{
		`ChooseACheckableAbility`, `ChooseACheckableAbility([])`, `ChooseACheckableAbility([FirstStrike])`,
		`ChooseACheckableAbility([FirstStrike,FirstStrike]),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseACheckableAbility([FirstStrike(),Flying]),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseACheckableAbility([FirstStrike,Rampage(2)]),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseACheckableAbility([FirstStrike,AnyLandwalk]),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseACheckableAbility([FirstStrike,Flying]),GainLife(Integer(1))`,
		`ChooseACheckableAbility([FirstStrike,Flying]),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(Flying)],UntilEndOfTurn)`,
		`ChooseACheckableAbility([FirstStrike,Flying]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseACheckableAbility([FirstStrike,Flying]),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(TheChosenAbility())],UntilEndOfTurn)`,
		`ChooseACheckableAbility([FirstStrike,Flying]),CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[LosesAbility(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseACheckableAbility([FirstStrike,Flying]),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(TheChosenAbility)],UntilEndOfTurn),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseACheckableAbility([]),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseACheckableAbility([FirstStrike]),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseACheckableAbility([FirstStrike,Flying]),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(TheChosenAbility)],UntilEndOfCombat)`,
		`ChooseACheckableAbility([FirstStrike,Flying]),CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(TheChosenAbility),AdjustPT(1,1)],UntilEndOfTurn)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + actions + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-checkable.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", actions)
		}
	}
	nodes, err := Parse(strings.NewReader("["+strings.ReplaceAll(checkableRemoval, "Ref_TargetPermanent", "ThisPermanent")+"]"), "choice-context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileCheckableAbilityChoice(nodes[0].Args[0], nodes[0].Args[1], ctx); err == nil {
			t.Fatal("unsupported choice context accepted")
		}
	}
}

func TestRONCheckableAbilityInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+checkableRemoval+`])))`)
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory shared targets")
	}
	a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	put := func(name string) *mage.Permanent {
		return g.PutOnBattlefield(mage.NewCreature(name, "", 2, 8, mage.WithKeyword(core.FirstStrike), mage.WithKeyword(core.Swampwalk)), b.PlayerID())
	}
	x, y := put("X"), put("Y")
	for i, card := range []mage.Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{[]uuid.UUID{x.ID(), y.ID()}[i]}, 0); err != nil {
			t.Fatal(err)
		}
	}
	if a.prompts != 0 {
		t.Fatal("announcement chose a keyword")
	}
	a.choice = 1
	g.ResolveTopOfStack()
	a.choice = 0
	g.ResolveStack()
	if a.prompts != 2 || g.FindPermanent(x.ID()).HasKeyword(core.FirstStrike) || !g.FindPermanent(x.ID()).HasKeyword(core.Swampwalk) || !g.FindPermanent(y.ID()).HasKeyword(core.FirstStrike) || g.FindPermanent(y.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("independent resolutions shared a choice or target")
	}
}

func TestRONCheckableAbilityCollection(t *testing.T) {
	node := repositoryRecord(t, "LEG", "Urborg", "checkable.ron")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Urborg", factory()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Urborg"`) {
			record = line
		}
	}
	if registry, err := LoadFS(fstest.MapFS{"urborg.ron": {Data: []byte(record)}}, "*.ron"); err != nil || registry == nil {
		t.Fatalf("complete Urborg strict loading: %v", err)
	}
	src := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + checkableRemoval + `])))])`
	files := fstest.MapFS{"choice.ron": {Data: []byte(src)}}
	if registry, err := LoadFS(files, "*.ron"); err != nil || registry == nil {
		t.Fatalf("valid declaration rejected: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"ChooseACheckableAbility": "action", "FirstStrike": "keyword", "TheChosenAbility": "ability_reference", "Swamp": "land_subtype"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing inventory role: %s/%s", name, role)
		}
	}
}

func TestRONCheckableAbilityCombat(t *testing.T) {
	for _, double := range []bool{false, true} {
		for _, afterFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("double=%v/after-first=%v", double, afterFirst), func(t *testing.T) {
				a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Urborg"), a.PlayerID())
				opts := []mage.CardOption{mage.WithKeyword(core.FirstStrike)}
				if double {
					opts = append(opts, mage.WithKeyword(core.DoubleStrike))
				}
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 8, opts...), a.PlayerID())
				g.MutablePermanent(attacker.ID()).RevokeBaseAttr(core.AttrSummonSick)
				a.SetAttackers(1, []string{"Attacker"})
				g.RunStepWithPriority(core.BeginCombat)
				g.RunStepWithPriority(core.DeclareAttackers)
				g.RunStepWithPriority(core.DeclareBlockers)
				if afterFirst {
					g.RunStepWithPriority(core.FirstStrikeDamage)
					if b.Life() != 18 {
						t.Fatal("first strike damage missing")
					}
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{attacker.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if g.FindPermanent(attacker.ID()).HasKeyword(core.FirstStrike) {
					t.Fatal("first strike retained")
				}
				if !afterFirst {
					g.RunStepWithPriority(core.FirstStrikeDamage)
				}
				g.RunStepWithPriority(core.CombatDamage)
				want := 18
				if double {
					want = 16
				}
				if b.Life() != want {
					t.Fatalf("life=%d want=%d", b.Life(), want)
				}
			})
		}
	}
	for _, beforeBlocks := range []bool{false, true} {
		t.Run(fmt.Sprintf("swampwalk/before-blocks=%v", beforeBlocks), func(t *testing.T) {
			a := &checkableCombatChooser{TestPlayer: gametest.NewTestPlayer("A")}
			b := gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Urborg"), a.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 8, mage.WithKeyword(core.Swampwalk)), a.PlayerID())
			g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 8), b.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), b.PlayerID())
			g.MutablePermanent(attacker.ID()).RevokeBaseAttr(core.AttrSummonSick)
			a.SetAttackers(1, []string{"Attacker"})
			b.SetBlockers(1, map[string]string{"Blocker": "Attacker"})
			g.RunStepWithPriority(core.BeginCombat)
			g.RunStepWithPriority(core.DeclareAttackers)
			if !mage.HasLandwalkEvasion(attacker, b.PlayerID(), g) {
				t.Fatal("swampwalk allowed a block")
			}
			remove := func() {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{attacker.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			if beforeBlocks {
				remove()
			}
			g.RunStepWithPriority(core.DeclareBlockers)
			if !beforeBlocks {
				remove()
			}
			g.RunStepWithPriority(core.CombatDamage)
			want := 18
			if beforeBlocks {
				want = 20
			}
			if b.Life() != want {
				t.Fatalf("life=%d want=%d", b.Life(), want)
			}
		})
	}
}
