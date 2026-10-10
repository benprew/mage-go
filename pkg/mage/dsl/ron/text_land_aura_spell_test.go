package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type landAuraSpellPlayer struct {
	mage.Player
	choice string
	target uuid.UUID
}

func (p *landAuraSpellPlayer) ChooseString([]string, string) string { return p.choice }

func (p *landAuraSpellPlayer) ChooseTargets([]uuid.UUID, int, int, *mage.Game) []uuid.UUID {
	return []uuid.UUID{p.target}
}

func landAuraSpellFactory(t *testing.T, rules string) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Land Aura",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rules+`])`), "land-aura.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONTextLandAuraSpellRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Evil Presence", "Phantasmal Terrain"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				chooser := &ronBasicLandChooser{BasePlayer: a, choice: 2}
				g.SetPlayerAt(0, chooser)
				host := g.PutOnBattlefield(repositoryCard(t, "LEA", "Tropical Island"), b.PlayerID())
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Black, 1)
				a.ManaPool().Add(core.Blue, 2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				paid := a.ManaPool().TotalMana()
				change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.MountainLandType)
				if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
					t.Fatal(err)
				}
				if chooser.entryCalls != 0 || a.ManaPool().TotalMana() != paid || g.StackPeek().Card != card || g.StackPeek().SourceID != card.ID() {
					t.Fatal("revision changed announcement or physical identity")
				}
				clone := g.Clone()
				clone.SetPlayerAt(0, &landAuraSpellPlayer{Player: clone.GetPlayer(a.PlayerID()), choice: "Swamp"})
				g.ResolveStack()
				clone.ResolveStack()
				want := "Mountain"
				if name == "Phantasmal Terrain" {
					want = "Swamp"
					if g.FindPermanent(card.ID()).ChosenBasicLandType != core.SwampLandType {
						t.Fatal("text change altered the later entry choice")
					}
				}
				for _, game := range []*mage.Game{g, clone} {
					p := game.FindPermanent(card.ID())
					land := game.FindPermanent(host.ID())
					if p == nil || p.AttachedTo != host.ID() || !land.HasSubType(want) || len(land.SubTypes()) != 1 || p.Card.ManaCost().String() != card.ManaCost().String() {
						t.Fatal("entry lost text, attachment or protected mana symbols")
					}
				}
				calls := chooser.entryCalls
				g.PhaseIn(g.PhaseOut(card.ID()))
				g.ApplyContinuousEffects()
				if chooser.entryCalls != calls || !g.FindPermanent(host.ID()).HasSubType(want) {
					t.Fatal("phasing lost text or repeated entry")
				}
				g.BouncePermanentToHand(g.FindPermanent(card.ID()))
				if len(a.Hand()) != 1 || a.Hand()[0].ID() != card.ID() || g.LKI(card.ID()) == nil || !g.FindPermanent(host.ID()).HasSubType("Forest") || !clone.FindPermanent(host.ID()).HasSubType(want) {
					t.Fatal("departure lost printed identity, LKI or clone isolation")
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
					t.Fatal("recast without mana succeeded")
				}
				a.ManaPool().Add(core.Black, 1)
				a.ManaPool().Add(core.Blue, 2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if !g.FindPermanent(host.ID()).HasSubType("Swamp") {
					t.Fatal("returned Aura retained the former word change")
				}
			})
		}
	}
}

func TestRONTextLandAuraSpellLegality(t *testing.T) {
	for _, state := range []string{"legal", "illegal", "returned", "phased", "phase-cycle"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := landAuraSpellFactory(t, `EnchantPermanent(IsLandType(Swamp)),PermanentLayerEffect(HostPermanent,[SetLandType(Swamp)])`)()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			host := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
			if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
				t.Fatal(err)
			}
			if state != "illegal" {
				change, _ = mage.ReplaceBasicLandWord(core.ForestLandType, core.SwampLandType)
				if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
					t.Fatal(err)
				}
			}
			switch state {
			case "returned":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(host.Card, b.PlayerID())
			case "phased", "phase-cycle":
				ids := g.PhaseOut(host.ID())
				if state == "phase-cycle" {
					g.PhaseIn(ids)
				}
			}
			g.ResolveStack()
			want := state == "legal" || state == "phase-cycle"
			if (g.FindPermanent(card.ID()) != nil) != want {
				t.Fatal("revised targets lost legality or incarnation checks")
			}
		})
	}
}

func TestRONTextLandAuraSpellCopiesAndLayers(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
	card := repositoryCard(t, "LEA", "Evil Presence")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	duplicate := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	if duplicate == nil {
		t.Fatal("spell copy failed")
	}
	change, _ = mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.RevisePrintedSpellText(duplicate.ID, change); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveTopOfStack()
	if !g.FindPermanent(duplicate.SourceID).Card.IsToken() || !g.FindPermanent(first.ID()).HasSubType("Forest") {
		t.Fatal("copied spell lost its independent text or token entry")
	}
	g.ResolveTopOfStack()
	if !g.FindPermanent(first.ID()).HasSubType("Island") {
		t.Fatal("copy revision changed the original spell")
	}
	clone.ResolveStack()
	if !clone.FindPermanent(first.ID()).HasSubType("Island") {
		t.Fatal("cloned stack lost continuation")
	}
	p := g.FindPermanent(card.ID())
	change, _ = mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
	if err := g.AddPrintedStaticLandTypeTextEffect(p, p.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(first.ID()).HasSubType("Mountain") {
		t.Fatal("later text layer lost the continued baseline")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if !g.FindPermanent(first.ID()).HasSubType("Island") {
		t.Fatal("expiry restored physical text instead of continued text")
	}
	values, err := g.CaptureCopiableValues(g.FindPermanent(card.ID()))
	if err != nil {
		t.Fatal(err)
	}
	built, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	printed := g.PutOnBattlefield(built, a.PlayerID())
	g.Attach(printed.ID(), second.ID())
	if !g.FindPermanent(second.ID()).HasSubType("Swamp") {
		t.Fatal("ordinary spell text entered copiable values")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.EndOfTurn))
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(card.ID()).RuntimeAbilities) != 0 || !g.FindPermanent(first.ID()).HasSubType("Island") {
		t.Fatal("layer 6 removal undid the earlier land-type layer")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if !g.FindPermanent(first.ID()).HasSubType("Island") {
		t.Fatal("ability restoration lost continued text")
	}
}

func TestRONTextLandAuraSpellChoiceCopies(t *testing.T) {
	g, a, b := ronGame()
	first := &ronBasicLandChooser{BasePlayer: a, choice: 4}
	second := &ronBasicLandChooser{BasePlayer: b, choice: 3}
	g.SetPlayerAt(0, first)
	g.SetPlayerAt(1, second)
	host := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	card := repositoryCard(t, "LEA", "Phantasmal Terrain")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	duplicate := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	g.ResolveTopOfStack()
	if second.entryCalls != 1 || first.entryCalls != 0 || g.FindPermanent(duplicate.SourceID).ChosenBasicLandType != core.MountainLandType || !g.FindPermanent(host.ID()).HasSubType("Mountain") {
		t.Fatal("copy did not choose with its own controller at entry")
	}
	g.ResolveTopOfStack()
	if first.entryCalls != 1 || g.FindPermanent(card.ID()).ChosenBasicLandType != core.ForestLandType || !g.FindPermanent(host.ID()).HasSubType("Forest") {
		t.Fatal("original inherited a copied choice or rewrote the saved choice")
	}
}

func TestRONTextLandAuraSpellRetarget(t *testing.T) {
	g, a, b := ronGame()
	swamp := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	card := landAuraSpellFactory(t, `EnchantPermanent(IsLandType(Swamp)),PermanentLayerEffect(HostPermanent,[SetLandType(Swamp)])`)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{swamp.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.SetPlayerAt(1, &landAuraSpellPlayer{Player: b, target: forest.ID()})
	duplicate := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if duplicate == nil || len(duplicate.Targets) != 1 || duplicate.Targets[0] != forest.ID() {
		t.Fatal("copy retargeting did not use the revised enchant declaration")
	}
	g.ResolveStack()
	if p := g.FindPermanent(duplicate.SourceID); p == nil || p.AttachedTo != forest.ID() || !p.Card.IsToken() || g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 {
		t.Fatal("copy or original lost its independent target legality")
	}
}

func TestRONTextLandAuraSpellBoundaries(t *testing.T) {
	for _, rules := range []string{
		`EnchantPermanent(IsCardtype(Land)),PermanentLayerEffect(HostPermanent,[SetLandType(Goblin)])`,
		`EnchantPermanent(IsCardtype(Land)),PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType)])`,
		`EnchantPermanent(IsCardtype(Land)),AsPermanentEnters(ThisPermanent,[ChooseABasicLandType,ChooseABasicLandType]),PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType)])`,
		`EnchantPermanent(IsCardtype(Land)),AsPermanentEnters(ThisPermanent,[ChooseABasicLandType()]),PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rules+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid land Aura declaration: %s", rules)
		}
	}
	for _, name := range []string{"Clone", "Copy Artifact", "Wild Growth"} {
		g, a, b := ronGame()
		card := repositoryCard(t, "LEA", name)
		if name == "Clone" || name == "Copy Artifact" {
			card = textCopyMixedRecord(t, "LEA", name)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Colorless, 4)
		a.ManaPool().Add(core.Blue, 1)
		a.ManaPool().Add(core.Green, 1)
		var targets []uuid.UUID
		if name == "Wild Growth" {
			land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
			targets = []uuid.UUID{land.ID()}
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
			t.Fatal(err)
		}
		before := g.StackPeek()
		change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
		if err := g.RevisePrintedSpellText(before.ID, change); err == nil || g.StackPeek() != before {
			t.Fatal("unsupported entry context changed live state")
		}
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		if _, err := Compile(staticBaseRecord(t, "LEA", name)); err == nil {
			t.Fatal("incomplete text spell accepted")
		}
	}
}
