package ron

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type landAuraObjectPlayer struct {
	intrinsicWordTester
	entries int
}

func (p *landAuraObjectPlayer) ChooseString([]string, string) string {
	p.entries++
	return "Swamp"
}

func TestRONTextLandAuraObjectRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Evil Presence", "Phantasmal Terrain"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				g, base, b := ronGame()
				a := &landAuraObjectPlayer{intrinsicWordTester: intrinsicWordTester{Player: base, words: []string{"Swamp", "Island"}}}
				g.SetPlayerAt(0, a)
				host := g.PutOnBattlefield(repositoryCard(t, "LEA", "Tropical Island"), b.PlayerID())
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Black, 1)
				a.ManaPool().Add(core.Blue, 2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				id := textStaticID(t, p)
				choice := p.ChosenBasicLandType
				entries := a.entries
				calls := a.calls
				castStaticActivationWordBranch(t, g, a, p, "Magical Hack")
				if a.calls != calls {
					t.Fatal("announcement made word choices")
				}
				g.ResolveStack()
				want := "Island"
				if name == "Phantasmal Terrain" {
					want = "Swamp"
				}
				check := func(game *mage.Game, subtype string) {
					t.Helper()
					game.ApplyContinuousEffects()
					land := game.FindPermanent(host.ID())
					if land == nil || !land.HasSubType(subtype) || len(land.SubTypes()) != 1 {
						t.Fatalf("wanted %s host, got %v", subtype, land)
					}
				}
				check(g, want)
				p = g.FindPermanent(card.ID())
				if textStaticID(t, p) != id || p.ChosenBasicLandType != choice || a.calls != calls+2 || p.Card.Name() != name || p.Card.ManaCost().String() != card.ManaCost().String() {
					t.Fatal("object text changed identity, entry choice or protected words")
				}
				mana := core.Blue
				if name == "Phantasmal Terrain" {
					mana = core.Black
				}
				if err := g.ActivateAbilityByIndex(b.PlayerID(), host.ID(), 0, nil); err != nil || b.ManaPool().Count(mana) != 1 {
					t.Fatal("host intrinsic mana did not use its final land type", err)
				}
				clone := g.Clone()
				change, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
				if err := g.AddPrintedLandAuraObjectTextEffect(p, core.EndOfTurn, change); err != nil {
					t.Fatal(err)
				}
				if name == "Evil Presence" {
					check(g, "Mountain")
				}
				check(clone, want)
				phase := g.PhaseOut(card.ID())
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				g.PhaseIn(phase)
				check(g, want)
				if a.calls != calls+2 || a.entries != entries || g.FindPermanent(card.ID()).ChosenBasicLandType != choice {
					t.Fatal("phasing repeated entry or resolution choices")
				}
				g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.EndOfTurn))
				check(g, want)
				if len(g.FindPermanent(card.ID()).RuntimeAbilities) != 0 {
					t.Fatal("layer 6 removal failed")
				}
				g.GetEffects().RemoveEndOfTurn()
				check(g, want)
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
				check(g, want)
				values, err := g.CaptureCopiableValues(g.FindPermanent(card.ID()))
				if err != nil {
					t.Fatal(err)
				}
				printed, err := values.Build()
				if err != nil {
					t.Fatal(err)
				}
				second := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
				copyAura := g.PutOnBattlefield(printed, a.PlayerID())
				g.Attach(copyAura.ID(), second.ID())
				g.ApplyContinuousEffects()
				if !g.FindPermanent(second.ID()).HasSubType("Swamp") {
					t.Fatal("ordinary text entered copy values")
				}
				g.BouncePermanentToHand(g.FindPermanent(card.ID()))
				if g.LKI(card.ID()) == nil || !g.FindPermanent(host.ID()).HasSubType("Forest") {
					t.Fatal("departure lost LKI or left the host changed")
				}
				a.words = []string{"Swamp"}
				a.ManaPool().Add(core.Black, 1)
				a.ManaPool().Add(core.Blue, 2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				check(g, "Swamp")
				check(clone, want)
			})
		}
	}
}

func TestRONTextLandAuraObjectTargetVisit(t *testing.T) {
	for _, state := range []string{"returned", "phased", "shroud", "phase-cycle"} {
		t.Run(state, func(t *testing.T) {
			g, base, _ := ronGame()
			a := &intrinsicWordTester{Player: base, words: []string{"Swamp", "Island"}}
			g.SetPlayerAt(0, a)
			host := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
			p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Evil Presence"), a.PlayerID())
			g.Attach(p.ID(), host.ID())
			castStaticActivationWordBranch(t, g, a, p, "Magical Hack")
			switch state {
			case "returned":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(p.Card, a.PlayerID())
				g.Attach(p.ID(), host.ID())
			case "phased", "phase-cycle":
				ids := g.PhaseOut(p.ID())
				if state == "phase-cycle" {
					g.PhaseIn(ids)
				}
			case "shroud":
				g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
			}
			g.ResolveStack()
			wantCalls := 0
			want := "Swamp"
			if state == "phase-cycle" {
				wantCalls, want = 2, "Island"
			}
			if a.calls != wantCalls {
				t.Fatal("target visit did not control resolution choices")
			}
			if state != "phased" && !g.FindPermanent(host.ID()).HasSubType(want) {
				t.Fatal("target visit did not control text changes")
			}
		})
	}
}

func TestRONTextLandAuraObjectBoundaries(t *testing.T) {
	for _, rules := range []string{
		`EnchantPermanent(IsCardtype(Land)),PermanentLayerEffect(HostPermanent,[SetLandType(Goblin)])`,
		`EnchantPermanent(IsCardtype(Land)),PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType)])`,
		`EnchantPermanent(IsCardtype(Land)),AsPermanentEnters(ThisPermanent,[ChooseABasicLandType,ChooseABasicLandType]),PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType)])`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rules+`])`)); err == nil {
			t.Fatal("invalid land Aura accepted", rules)
		}
	}
	for _, rules := range []string{
		`EnchantPermanent(IsLandType(Swamp)),PermanentLayerEffect(HostPermanent,[SetLandType(Swamp)])`,
		`EnchantPermanent(IsCardtype(Creature)),PermanentLayerEffect(HostPermanent,[SetLandType(Swamp)])`,
	} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(landAuraSpellFactory(t, rules)(), a.PlayerID())
		before := g.FindPermanent(p.ID())
		change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
		if err := g.AddPrintedLandAuraObjectTextEffect(p, core.Indefinite, change); err == nil || g.FindPermanent(p.ID()) != before {
			t.Fatal("unsupported enchant context changed live state")
		}
	}
}

func TestRONTextLandAuraObjectInventory(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		records, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			if !landAuraSpellTextRules(record.Fields["Rules"].Args) {
				continue
			}
			for _, reverse := range []bool{false, true} {
				if reverse {
					slices.Reverse(record.Fields["Rules"].Args)
				}
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				g, a, _ := ronGame()
				p := g.PutOnBattlefield(factory(), a.PlayerID())
				change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
				if err := g.AddPrintedLandAuraObjectTextEffect(p, core.Indefinite, change); err != nil {
					t.Fatal(file, record.Fields["Name"].String, err)
				}
			}
			counts[record.Fields["Name"].String]++
		}
	}
	if !maps.Equal(counts, map[string]int{"Evil Presence": 6, "Phantasmal Terrain": 6}) {
		t.Fatal("unmapped records", counts)
	}
}

func TestRONTextLandAuraObjectPendingBindings(t *testing.T) {
	g, base, b := ronGame()
	a := &intrinsicWordTester{Player: base, words: []string{"Swamp", "Island", "Swamp", "Mountain"}}
	g.SetPlayerAt(0, a)
	firstHost := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	secondHost := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	first := g.PutOnBattlefield(repositoryCard(t, "LEA", "Evil Presence"), b.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEA", "Evil Presence"), b.PlayerID())
	g.Attach(first.ID(), firstHost.ID())
	g.Attach(second.ID(), secondHost.ID())
	castStaticActivationWordBranch(t, g, a, first, "Magical Hack")
	castStaticActivationWordBranch(t, g, a, second, "Magical Hack")
	clone := g.Clone()
	clonedPlayer := &intrinsicWordTester{Player: clone.PlayerAt(0), words: []string{"Swamp", "Forest", "Swamp", "Plains"}}
	clone.SetPlayerAt(0, clonedPlayer)
	g.ResolveStack()
	clone.ResolveStack()
	if !g.FindPermanent(firstHost.ID()).HasSubType("Mountain") || !g.FindPermanent(secondHost.ID()).HasSubType("Island") || !clone.FindPermanent(firstHost.ID()).HasSubType("Plains") || !clone.FindPermanent(secondHost.ID()).HasSubType("Forest") || a.calls != 4 || clonedPlayer.calls != 4 {
		t.Fatal("pending results crossed declarations or cloned frames")
	}
}

func TestRONTextLandAuraObjectOrderAndAttachment(t *testing.T) {
	g, a, b := ronGame()
	host := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	card := repositoryCard(t, "LEA", "Evil Presence")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	swampIsland, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, swampIsland); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	islandMountain, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
	mountainPlains, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.PlainsLandType)
	if err := g.AddPrintedLandAuraObjectTextEffect(p, core.EndOfTurn, islandMountain); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedLandAuraObjectTextEffect(g.FindPermanent(p.ID()), core.Indefinite, mountainPlains); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(host.ID()).HasSubType("Plains") {
		t.Fatal("ordered substitutions did not compose")
	}
	clone := g.Clone()
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasSubType("Island") || !clone.FindPermanent(host.ID()).HasSubType("Plains") {
		t.Fatal("expiry lost entry baseline or clone isolation")
	}
	other := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
	g.Attach(p.ID(), other.ID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasSubType("Forest") || !g.FindPermanent(other.ID()).HasSubType("Island") {
		t.Fatal("changed Aura did not follow its current attachment")
	}
	g.BouncePermanentToHand(g.FindPermanent(other.ID()))
	if lki := g.LKI(other.ID()); lki == nil || !lki.Snapshot.HasSubType("Island") {
		t.Fatal("host departure did not keep its final land type")
	}
	returned := g.PutOnBattlefield(other.Card, a.PlayerID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(returned.ID()).HasSubType("Mountain") {
		t.Fatal("returned host inherited a former attachment")
	}
}
