package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronBasicLandChooser struct {
	*mage.BasePlayer
	choice                      int
	entryCalls, resolutionCalls int
}

var basicLandChoiceNames = []string{"Plains", "Island", "Swamp", "Mountain", "Forest"}
var basicLandChoiceKeywords = []core.Attr{core.Plainswalk, core.Islandwalk, core.Swampwalk, core.Mountainwalk, core.Forestwalk}

func (p *ronBasicLandChooser) ChooseString(options []string, _ string) string {
	p.entryCalls++
	if !slices.Equal(options, basicLandChoiceNames) {
		panic("invalid entry options")
	}
	return options[p.choice]
}

func (p *ronBasicLandChooser) ChooseMode(options []string, _ string) int {
	p.resolutionCalls++
	if !slices.Equal(options, basicLandChoiceNames) {
		panic("invalid resolution options")
	}
	return p.choice
}

func TestRONBasicLandChoiceTerrain(t *testing.T) {
	for index, landType := range basicLandChoiceNames {
		t.Run(landType, func(t *testing.T) {
			a := &ronBasicLandChooser{BasePlayer: mage.NewBasePlayer("A"), choice: index}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			host := g.PutOnBattlefield(repositoryCard(t, "LEA", "Tropical Island"), b.PlayerID())
			aura := repositoryCard(t, "LEA", "Phantasmal Terrain")
			aura.SetOwner(a.PlayerID())
			a.AddToHand(aura)
			if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0); err == nil || a.entryCalls != 0 || g.StackSize() != 0 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.Blue, 2)
			if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil || a.ManaPool().TotalMana() != 2 {
				t.Fatal("illegal target spent mana")
			}
			if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if a.entryCalls != 0 {
				t.Fatal("entry choice at announcement")
			}
			g.ResolveStack()
			p := g.FindPermanent(host.ID())
			if a.entryCalls != 1 || a.resolutionCalls != 0 || !p.HasSubType(landType) || len(p.SubTypes()) != 1 || p.Card.HasSuperType(core.SuperBasic) {
				t.Fatal("land type, supertype, or entry choice")
			}
			cost := []string{"{W}", "{U}", "{B}", "{R}", "{G}"}[index]
			if !g.CanAfford(b.PlayerID(), core.ParseManaCost(cost), nil) {
				t.Fatal("intrinsic mana missing")
			}
			clone := g.Clone()
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(aura.ID()).ChosenBasicLandType != core.BasicLandType(index+1) {
				t.Fatal("clone lost saved land choice")
			}
			g.PhaseIn(g.PhaseOut(host.ID()))
			g.ApplyContinuousEffects()
			if a.entryCalls != 1 || !g.FindPermanent(host.ID()).HasSubType(landType) {
				t.Fatal("phase cycle repeated entry")
			}
			phased := g.PhaseOut(aura.ID())
			g.ApplyContinuousEffects()
			if len(g.FindPermanent(host.ID()).SubTypes()) != 2 {
				t.Fatal("phased Aura changed host")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			if a.entryCalls != 1 || !g.FindPermanent(host.ID()).HasSubType(landType) {
				t.Fatal("source phase cycle lost choice")
			}
			second := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
			g.Attach(aura.ID(), second.ID())
			if !g.FindPermanent(second.ID()).HasSubType(landType) || !g.FindPermanent(host.ID()).HasSubType("Forest") || a.entryCalls != 1 {
				t.Fatal("reattachment or choice persistence")
			}
			g.BouncePermanentToHand(g.FindPermanent(aura.ID()))
			if !g.FindPermanent(second.ID()).HasSubType("Mountain") || !clone.FindPermanent(host.ID()).HasSubType(landType) {
				t.Fatal("departure or clone isolation")
			}
		})
	}
}

func TestRONBasicLandChoiceSlug(t *testing.T) {
	for index, landType := range basicLandChoiceNames {
		t.Run(landType, func(t *testing.T) {
			a := &ronBasicLandChooser{BasePlayer: mage.NewBasePlayer("A"), choice: index}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := repositoryCard(t, "LEG", "Giant Slug")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Black, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 || !p.HasSubType("Slug") {
				t.Fatal("metadata")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || g.StackSize() != 0 {
				t.Fatal("unpaid activation")
			}
			a.ManaPool().Add(core.Colorless, 5)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.resolutionCalls != 0 || a.entryCalls != 0 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("choice before upkeep or wrong cost")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep matched")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.resolutionCalls != 0 {
				t.Fatal("delayed trigger timing")
			}
			g.ResolveStack()
			if a.resolutionCalls != 1 || !g.FindPermanent(p.ID()).HasKeyword(basicLandChoiceKeywords[index]) || mage.HasLandwalkEvasion(g.FindPermanent(p.ID()), b.PlayerID(), g) {
				t.Fatal("chosen landwalk")
			}
			land := g.PutOnBattlefield(repositoryCard(t, "LEA", landType), b.PlayerID())
			if !mage.HasLandwalkEvasion(g.FindPermanent(p.ID()), b.PlayerID(), g) {
				t.Fatal("landwalk legality")
			}
			g.PhaseOut(land.ID())
			if mage.HasLandwalkEvasion(g.FindPermanent(p.ID()), b.PlayerID(), g) {
				t.Fatal("phased land grants evasion")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if g.FindPermanent(p.ID()).HasKeyword(basicLandChoiceKeywords[index]) {
				t.Fatal("landwalk did not expire")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("one-shot trigger repeated")
			}
		})
	}
}

func TestRONBasicLandChoiceInvalid(t *testing.T) {
	choice := `ChooseABasicLandType`
	body := `CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Landwalk(IsLandTypeVariable(TheChosenLandType))])],UntilEndOfTurn)`
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[ChooseABasicLandType()])`,
		`AsPermanentEnters(ThisPermanent,[ChooseABasicLandType]),AsPermanentEnters(ThisPermanent,[ChooseABasicLandType])`,
		`PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseABasicLandType]),PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType())])`,
		`AsPermanentEnters(ThisPermanent,[ChooseABasicLandType]),PermanentLayerEffect(ThisPermanent,[SetLandTypeVariable(TheChosenLandType)])`,
		`Activated(PayMana([]),ActionList([` + choice + `]))`,
		`Activated(PayMana([]),ActionList([` + choice + `(),` + body + `]))`,
		`Activated(PayMana([]),ActionList([` + body + `,` + choice + `]))`,
		`Activated(PayMana([]),ActionList([` + choice + `,GainLife(Integer(1)),` + body + `]))`,
		`Activated(PayMana([]),ActionList([` + choice + `,` + body + `,` + body + `]))`,
		`Activated(PayMana([]),ActionList([` + choice + `,` + strings.Replace(body, "TheChosenLandType", "TheChosenColor", 1) + `]))`,
		`Activated(PayMana([]),ActionList([` + choice + `,` + strings.Replace(body, "ThisPermanent", "Ref_TargetPermanent", 1) + `]))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONBasicLandChoiceStrictLoading(t *testing.T) {
	src := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType)]),EnchantPermanent(IsCardtype(Land)),AsPermanentEnters(ThisPermanent,[ChooseABasicLandType])])`
	files := fstest.MapFS{"choice.ron": {Data: []byte(src)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(src, `Name:"Choice"`, `Name:"Invalid"`), "ChooseABasicLandType]", "ChooseABasicLandType()]"))}
	if err = LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("strict rejection changed registry")
	}
}

func TestRONBasicLandChoiceSlugIdentity(t *testing.T) {
	for _, change := range []string{"control", "depart", "return", "phase", "phase-cycle", "remove-abilities", "copy", "counter"} {
		t.Run(change, func(t *testing.T) {
			a := &ronBasicLandChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 2}
			b := &ronBasicLandChooser{BasePlayer: mage.NewBasePlayer("B"), choice: 4}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := repositoryCard(t, "LEG", "Giant Slug")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(repositoryCard(t, "LEG", "Giant Slug"), a.PlayerID())
			a.ManaPool().Add(core.Colorless, 5)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "depart", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(card.ID())
					p = g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "remove-abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			queueRONUpkeep(g, b.BasePlayer)
			if g.StackSize() != 0 {
				t.Fatal("saved controller changed")
			}
			queueRONUpkeep(g, a.BasePlayer)
			if g.StackSize() != 1 {
				t.Fatal("source change canceled schedule")
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if change == "counter" {
				g.GetStack().RemoveByID(g.StackPeek().ID)
			}
			clone := g.Clone()
			ca := &ronBasicLandChooser{BasePlayer: clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, choice: 1}
			cb := &ronBasicLandChooser{BasePlayer: clone.GetPlayer(b.PlayerID()).(*mage.SearchPlayer).BasePlayer, choice: 0}
			clone.SetPlayerAt(0, ca)
			clone.SetPlayerAt(1, cb)
			g.ResolveStack()
			want := change != "depart" && change != "return" && change != "phase" && change != "counter"
			if live := g.FindPermanent(p.ID()); live != nil && live.HasKeyword(core.Swampwalk) != want {
				t.Fatal("wrong original incarnation")
			}
			if g.FindPermanent(other.ID()).HasKeyword(core.Swampwalk) {
				t.Fatal("another factory inherited choice")
			}
			if a.resolutionCalls != 1 && change != "counter" || change == "counter" && a.resolutionCalls != 0 {
				t.Fatal("choice timing after source change")
			}
			if (b.resolutionCalls == 1) != (change == "copy") {
				t.Fatal("copy chooser")
			}
			if change == "copy" && !g.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) {
				t.Fatal("copy lost source binding")
			}
			clone.ResolveStack()
			if live := clone.FindPermanent(p.ID()); live != nil && live.HasKeyword(core.Islandwalk) != want {
				t.Fatal("clone lost schedule or shared choice")
			}
			if live := g.FindPermanent(p.ID()); live != nil && live.HasKeyword(core.Islandwalk) {
				t.Fatal("clone choice changed original")
			}
		})
	}
}

func TestRONBasicLandChoiceTerrainIdentity(t *testing.T) {
	for _, change := range []string{"illegal-resolution", "control", "remove-abilities", "copy", "return"} {
		t.Run(change, func(t *testing.T) {
			a := &ronBasicLandChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 2}
			b := &ronBasicLandChooser{BasePlayer: mage.NewBasePlayer("B"), choice: 4}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
			card := repositoryCard(t, "LEA", "Phantasmal Terrain")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{land.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if change == "illegal-resolution" {
				g.MutablePermanent(land.ID()).GrantBaseAttr(core.Shroud)
				g.ResolveStack()
				if a.entryCalls != 0 || g.FindPermanent(card.ID()) != nil {
					t.Fatal("illegal Aura made entry choice")
				}
				return
			}
			var copied uuid.UUID
			if change == "copy" {
				obj := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
				if obj == nil {
					t.Fatal("copy failed")
				}
				copied = obj.SourceID
			}
			g.ResolveStack()
			if change == "copy" {
				if a.entryCalls != 1 || b.entryCalls != 1 || g.FindPermanent(copied).ChosenBasicLandType != core.ForestLandType || g.FindPermanent(card.ID()).ChosenBasicLandType != core.SwampLandType {
					t.Fatal("spell copies shared choice")
				}
				g.BouncePermanentToHand(g.FindPermanent(card.ID()))
				if !g.FindPermanent(land.ID()).HasSubType("Forest") {
					t.Fatal("independent copied Aura choice")
				}
				return
			}
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "remove-abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.Indefinite))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(card.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
				g.Attach(card.ID(), land.ID())
			}
			want := "Swamp"
			if change == "return" {
				want = "Forest"
			}
			if !g.FindPermanent(land.ID()).HasSubType(want) || a.entryCalls != 1 || (b.entryCalls == 1) != (change == "return") {
				t.Fatal("choice lifetime or active source")
			}
		})
	}
}

func TestRONBasicLandChoiceInventoryAndReprints(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		card := repositoryCard(t, set, "Phantasmal Terrain")
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(set, err)
		}
	}
	src := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Land)),AsPermanentEnters(ThisPermanent,[ChooseABasicLandType]),PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType)])])`
	report, err := InspectFS(fstest.MapFS{"choice.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool {
		return s.Name == "TheChosenLandType" && s.Role == "basic_land_type_choice_reference"
	}) {
		t.Fatal("missing chosen land reference role")
	}
}

func TestRONBasicLandChoiceIndependentSchedules(t *testing.T) {
	a := &ronBasicLandChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 1}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	p := g.PutOnBattlefield(repositoryCard(t, "LEG", "Giant Slug"), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 10)
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	queueRONUpkeep(g, a.BasePlayer)
	if g.StackSize() != 2 || a.resolutionCalls != 0 {
		t.Fatal("independent activations merged schedules")
	}
	g.ResolveTopOfStack()
	a.choice = 4
	g.ResolveTopOfStack()
	if a.resolutionCalls != 2 || !g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) || !g.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("one choice replaced another invocation")
	}
}
