package ron

import (
	"reflect"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextDefinitionGameplay(t *testing.T) {
	original := repositoryCard(t, "LEA", "Black Knight")
	change, err := mage.ReplaceColorWord(core.White, core.Red)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := mage.BuildTextChangedDefinition(original, change)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(changed, a.PlayerID())
	if !p.HasProtectionFrom(textColorCard(core.Red)) || p.HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("protection word did not change")
	}
	if changed.Name() != original.Name() || changed.ManaCost().String() != original.ManaCost().String() || !slices.Contains(p.Colors(), core.Black) {
		t.Fatal("printed exclusions changed")
	}
	other := g.PutOnBattlefield(original, b.PlayerID())
	if !other.HasProtectionFrom(textColorCard(core.White)) || other.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("original changed")
	}
	clone := g.Clone()
	if !clone.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("clone lost changed rules")
	}
	rebuilt, err := mage.RebuildCardDefinition(changed)
	if err != nil {
		t.Fatal(err)
	}
	fresh := g.PutOnBattlefield(rebuilt, a.PlayerID())
	if !fresh.HasProtectionFrom(textColorCard(core.White)) || fresh.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("text entered printed provenance")
	}
	landChange, err := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err != nil {
		t.Fatal(err)
	}
	wraith, err := mage.BuildTextChangedDefinition(repositoryCard(t, "LEA", "Bog Wraith"), landChange)
	if err != nil {
		t.Fatal(err)
	}
	walking := g.PutOnBattlefield(wraith, a.PlayerID())
	if !walking.HasKeyword(core.Islandwalk) || walking.HasKeyword(core.Swampwalk) {
		t.Fatal("landwalk did not change")
	}
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	if !mage.HasLandwalkEvasion(walking, b.PlayerID(), g) {
		t.Fatal("changed landwalk is not active")
	}
}

func TestRONTextDefinitionOrdering(t *testing.T) {
	first, _ := mage.ReplaceColorWord(core.White, core.Red)
	second, _ := mage.ReplaceColorWord(core.Red, core.Green)
	card, err := mage.BuildTextChangedDefinition(repositoryCard(t, "LEA", "Black Knight"), first)
	if err != nil {
		t.Fatal(err)
	}
	card, err = mage.BuildTextChangedDefinition(card, second)
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(card, a.PlayerID())
	if !p.HasProtectionFrom(textColorCard(core.Green)) || p.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("substitution order lost")
	}
	same, err := mage.BuildTextChangedDefinition(card)
	if err != nil {
		t.Fatal(err)
	}
	if same.ID() == card.ID() {
		t.Fatal("prototype identity reused")
	}
}

func textColorCard(color core.Color) mage.Card {
	card := mage.NewCreature("color", map[core.Color]string{core.White: "{W}", core.Blue: "{U}", core.Black: "{B}", core.Red: "{R}", core.Green: "{G}"}[color], 1, 1)
	card.SetColorOverride([]core.Color{color})
	return card
}

func TestRONTextDefinitionRoles(t *testing.T) {
	colorChange, _ := mage.ReplaceColorWord(core.Black, core.Red)
	landChange, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	record := staticBaseRecord(t, "LEA", "Black Ward")
	saved := cloneTextNode(record)
	transformed, err := transformTextDefinition(record, []mage.TextChange{colorChange})
	if err != nil {
		t.Fatal(err)
	}
	if transformed.Fields["Name"].String != "Black Ward" || reflect.DeepEqual(transformed, record) || !reflect.DeepEqual(record, saved) {
		t.Fatal("name or input AST changed")
	}
	card, err := mage.BuildTextChangedDefinition(repositoryCard(t, "LEA", "Island"), landChange)
	if err != nil {
		t.Fatal(err)
	}
	if card.Name() != "Island" || !card.HasSubType("Forest") || card.HasSubType("Island") {
		t.Fatal("type line and card name roles were mixed")
	}
	node := nameScopeNode(t, `IsNamed(NamedCard("Black Knight"))`)
	transformedName, err := transformTextWords(node, "permanent_filter", colorChange)
	if err != nil || transformedName.Args[0].Args[0].String != "Black Knight" {
		t.Fatal("named-card operand changed", err)
	}
	if _, err := transformTextWords(nameScopeNode(t, `Unknown(Black)`), "rule", colorChange); err == nil {
		t.Fatal("unknown role accepted")
	}
	invalid := staticBaseRecord(t, "LEA", "Magical Hack")
	if _, err := Compile(invalid); err == nil {
		t.Fatal("incomplete card accepted")
	}
	if _, err := transformTextDefinition(invalid, []mage.TextChange{landChange}); err != nil {
		t.Fatal(err)
	}
}

func TestRONTextDefinitionFiltersAndIsolation(t *testing.T) {
	change, _ := mage.ReplaceColorWord(core.Black, core.Red)
	original := staticBaseRecord(t, "LEA", "Bad Moon")
	factory, err := Compile(original)
	if err != nil {
		t.Fatal(err)
	}
	original.Fields["Rules"].Args[0].Args[0].Args[0].Args[0].Args[0].Name = "Green"
	changed, err := mage.BuildTextChangedDefinition(factory(), change)
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	g.PutOnBattlefield(changed, a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	black := g.PutOnBattlefield(textColorCard(core.Black), a.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
	g.ApplyContinuousEffects()
	if red.CurrentPower(g) != 2 || black.CurrentPower(g) != 1 || green.CurrentPower(g) != 1 {
		t.Fatal("filter substitution or retained AST isolation failed")
	}
	printed, err := mage.RebuildCardDefinition(changed)
	if err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(printed, a.PlayerID())
	g.ApplyContinuousEffects()
	if black.CurrentPower(g) != 2 || green.CurrentPower(g) != 1 {
		t.Fatal("printed factory changed")
	}
}

func TestRONTextDefinitionSubstitutionDeclarations(t *testing.T) {
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		record := staticBaseRecord(t, "LEA", name)
		operation := record.Fields["Rules"].Args[0].Args[0].Args[0].Args[0].Args[1].Args[0].Args[1].Args[1].Args[0]
		choice := record.Fields["Rules"].Args[0].Args[0].Args[0].Args[0].Args[1].Args[0].Args[0]
		ctx := scope{player: playerReferencePlan{}}
		choose, plan, err := compileTextWordChoice(choice, ctx)
		if err != nil {
			t.Fatal(err)
		}
		if plan.color {
			ctx.colorWords = plan
		} else {
			ctx.landWords = plan
		}
		build, err := compileTextSubstitution(operation, ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileTextSubstitution(operation, scope{}); err == nil {
			t.Fatal("unbound words accepted")
		}
		invalid := cloneTextNode(operation)
		invalid.Args[1] = invalid.Args[0]
		if _, err := compileTextSubstitution(invalid, ctx); err == nil {
			t.Fatal("same word reference accepted")
		}
		invalid = cloneTextNode(operation)
		invalid.Args = invalid.Args[:1]
		if _, err := compileTextSubstitution(invalid, ctx); err == nil {
			t.Fatal("invalid arity accepted")
		}
		invalid = cloneTextNode(operation)
		invalid.Args[0] = Node{Kind: Tuple, Name: invalid.Args[0].Name}
		if _, err := compileTextSubstitution(invalid, ctx); err == nil {
			t.Fatal("tuple word reference accepted")
		}
		invalid = cloneTextNode(operation)
		if plan.color {
			invalid.Args[0].Name = "TheFirstChosenLandType"
		} else {
			invalid.Args[0].Name = "TheFirstChosenColorWord"
		}
		if _, err := compileTextSubstitution(invalid, ctx); err == nil {
			t.Fatal("cross-domain reference accepted")
		}
		invalid = cloneTextNode(operation)
		invalid.Name = "UnknownSubstitution"
		if _, err := compileTextSubstitution(invalid, ctx); err == nil {
			t.Fatal("unknown operation accepted")
		}
		g, a, _ := ronGame()
		bindings := initializeActionBindings(instanceBindings{})
		effectCtx := &mage.EffectContext{Game: g, Controller: a.PlayerID()}
		ref := build(bindings)
		if _, ok := ref.Resolve(effectCtx); ok {
			t.Fatal("missing choice supplied substitution")
		}
		if err := choose(bindings).Apply(effectCtx); err != nil {
			t.Fatal(err)
		}
		change, ok := ref.Resolve(effectCtx)
		if !ok {
			t.Fatal("real operation did not read its typed choice")
		}
		if _, ok := build(initializeActionBindings(instanceBindings{})).Resolve(effectCtx); ok {
			t.Fatal("independent declaration inherited choice")
		}
		name := "Plains"
		if plan.color {
			name = "Black Knight"
		}
		changed, err := mage.BuildTextChangedDefinition(repositoryCard(t, "LEA", name), change)
		if err != nil {
			t.Fatal(err)
		}
		p := g.PutOnBattlefield(changed, a.PlayerID())
		if plan.color {
			if !p.HasProtectionFrom(textColorCard(core.Blue)) || p.HasProtectionFrom(textColorCard(core.White)) {
				t.Fatal("lowered color substitution did not change real protection")
			}
		} else if changed.Name() != "Plains" || !changed.HasSubType("Island") || changed.HasSubType("Plains") {
			t.Fatal("lowered land substitution did not change real type line")
		}
	}
}

func TestRONTextDefinitionSpellAndTrigger(t *testing.T) {
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	spell, err := mage.BuildTextChangedDefinition(repositoryCard(t, "LEA", "Blue Elemental Blast"), change)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronModalGame(1)
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{red.ID()}, 0); err == nil {
		t.Fatal("old color target remains legal")
	}
	if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("illegal target spent payment")
	}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{green.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	if g.FindPermanent(green.ID()) != nil || g.FindPermanent(red.ID()) == nil || clone.FindPermanent(green.ID()) == nil {
		t.Fatal("changed spell filter or clone isolation failed")
	}
	clone.ResolveStack()
	if clone.FindPermanent(green.ID()) != nil {
		t.Fatal("clone lost changed target declarations")
	}
	landChange, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	lifetap, err := mage.BuildTextChangedDefinition(repositoryCard(t, "LEA", "Lifetap"), landChange)
	if err != nil {
		t.Fatal(err)
	}
	g, base, other := ronGame()
	g.PutOnBattlefield(lifetap, base.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), other.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), other.PlayerID())
	g.TapPermanent(forest)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("old land word triggered")
	}
	g.TapPermanent(island)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if base.Life() != 21 {
		t.Fatal("changed trigger did not gain life")
	}
}

func TestRONTextDefinitionMetadataAndTypeSet(t *testing.T) {
	change, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	card, err := mage.BuildTextChangedDefinition(repositoryCard(t, "LEA", "Tropical Island"), change)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(card.SubTypes(), []string{"Forest"}) {
		t.Fatal("subtype set contains duplicate words", card.SubTypes())
	}
	colorChange, _ := mage.ReplaceColorWord(core.White, core.Red)
	record := staticBaseRecord(t, "LEA", "Black Knight")
	record.Fields["ColorIndicator"] = Node{Kind: List, Args: []Node{{Kind: Symbol, Name: "White"}}}
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := mage.BuildTextChangedDefinition(factory(), colorChange)
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(changed, a.PlayerID())
	if !slices.Equal(p.Colors(), []core.Color{core.White}) || !p.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("indicator and rules text roles mixed")
	}
}
