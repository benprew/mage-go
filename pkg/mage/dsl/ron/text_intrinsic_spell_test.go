package ron

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func intrinsicSpellCases() []struct{ set, name string } {
	cases := make([]struct{ set, name string }, 0, 34)
	cases = append(cases, []struct{ set, name string }{
		{"LEG", "Cat Warriors"}, {"LEG", "Devouring Deep"}, {"DRK", "Knights of Thorn"},
		{"LEG", "Lost Soul"}, {"DRK", "Marsh Goblins"}, {"LEG", "Mountain Yeti"},
		{"ARN", "Repentant Blacksmith"}, {"LEG", "Righteous Avengers"}, {"LEG", "Segovian Leviathan"}, {"LEG", "Wall of Light"},
	}...)
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Black Knight", "White Knight", "Bog Wraith", "Shanodin Dryads"} {
			cases = append(cases, struct{ set, name string }{set, name})
		}
	}
	return cases
}

func TestRONTextIntrinsicSpellRecords(t *testing.T) {
	for _, tc := range intrinsicSpellCases() {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
				a.ManaPool().Add(color, 10)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			paid := a.ManaPool().TotalMana()
			changes := []mage.TextChange{}
			wants := map[core.Attr]bool{}
			for _, pair := range []struct {
				attr core.Attr
				land core.BasicLandType
			}{
				{core.Plainswalk, core.PlainsLandType}, {core.Islandwalk, core.IslandLandType}, {core.Swampwalk, core.SwampLandType}, {core.Mountainwalk, core.MountainLandType}, {core.Forestwalk, core.ForestLandType},
			} {
				if card.AttrSeeds()[pair.attr] == 0 {
					continue
				}
				to := core.IslandLandType
				next := core.Islandwalk
				if to == pair.land {
					to, next = core.ForestLandType, core.Forestwalk
				}
				change, _ := mage.ReplaceBasicLandWord(pair.land, to)
				changes = append(changes, change)
				wants[next] = true
			}
			for _, ability := range card.Abilities() {
				if protection, ok := ability.(*mage.ProtectionAbility); ok {
					change, _ := mage.ReplaceColorWord(protection.FromColors[0], core.Blue)
					changes = append(changes, change)
				}
			}
			if err := g.RevisePrintedSpellText(g.StackPeek().ID, changes...); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil || a.ManaPool().TotalMana() != paid || g.StackPeek().Card != card {
				t.Fatal("revision changed entry timing, payment or physical card")
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				p := game.FindPermanent(card.ID())
				if p == nil || p.Card.Owner() != a.PlayerID() || p.ControllerID() != a.PlayerID() || p.CurrentPower(game) != card.Power() || p.CurrentToughness(game) != card.Toughness() {
					t.Fatal("entry lost physical identity or characteristics")
				}
				for kw := range wants {
					land := "Island"
					if kw == core.Forestwalk {
						land = "Forest"
					}
					game.PutOnBattlefield(repositoryCard(t, "LEA", land), b.PlayerID())
					if !p.HasKeyword(kw) || !mage.HasLandwalkEvasion(game.FindPermanent(p.ID()), b.PlayerID(), game) {
						t.Fatal("entry lost revised landwalk evasion")
					}
				}
				for kw := range card.AttrSeeds() {
					if kw == core.Plainswalk || kw == core.Islandwalk || kw == core.Swampwalk || kw == core.Mountainwalk || kw == core.Forestwalk {
						if p.HasKeyword(kw) {
							t.Fatal("entry retained the former landwalk")
						}
					}
				}
				for _, kw := range []core.Attr{core.FirstStrike, core.Banding, core.Defender} {
					if p.HasKeyword(kw) != (card.AttrSeeds()[kw] > 0) {
						t.Fatal("entry changed an independent keyword")
					}
				}
				for _, ability := range card.Abilities() {
					if protection, ok := ability.(*mage.ProtectionAbility); ok && (!p.HasProtectionFrom(textColorCard(core.Blue)) || p.HasProtectionFrom(textColorCard(protection.FromColors[0]))) {
						t.Fatal("entry lost revised protection")
					}
				}
				values, err := game.CaptureCopiableValues(p)
				if err != nil {
					t.Fatal(err)
				}
				copied, err := values.Build()
				if err != nil {
					t.Fatal(err)
				}
				sibling := game.PutOnBattlefield(copied, a.PlayerID())
				for kw := range wants {
					if sibling.HasKeyword(kw) {
						t.Fatal("ordinary text entered copiable landwalk")
					}
				}
				for _, ability := range card.Abilities() {
					if protection, ok := ability.(*mage.ProtectionAbility); ok && (!sibling.HasProtectionFrom(textColorCard(protection.FromColors[0])) || sibling.HasProtectionFrom(textColorCard(core.Blue))) {
						t.Fatal("ordinary text entered copiable protection")
					}
				}
			}
			g.PhaseIn(g.PhaseOut(card.ID()))
			g.ApplyContinuousEffects()
			for kw := range wants {
				if !g.FindPermanent(card.ID()).HasKeyword(kw) {
					t.Fatal("phasing lost continuation")
				}
			}
			g.BouncePermanentToHand(g.FindPermanent(card.ID()))
			lki := g.LKI(card.ID())
			if lki == nil || len(a.Hand()) != 1 || a.Hand()[0].ID() != card.ID() {
				t.Fatal("departure lost printed card or LKI")
			}
			for kw := range wants {
				if !lki.Snapshot.HasKeyword(kw) {
					t.Fatal("LKI lost continued landwalk")
				}
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			for kw := range wants {
				if g.FindPermanent(card.ID()).HasKeyword(kw) {
					t.Fatal("recast retained former text")
				}
			}
			for _, ability := range card.Abilities() {
				if protection, ok := ability.(*mage.ProtectionAbility); ok && !g.FindPermanent(card.ID()).HasProtectionFrom(textColorCard(protection.FromColors[0])) {
					t.Fatal("recast lost printed protection")
				}
			}
		})
	}
}

func TestRONTextIntrinsicSpellCopies(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Bog Wraith")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 1)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	first, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, first); err != nil {
		t.Fatal(err)
	}
	copySpell := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.RevisePrintedSpellText(copySpell.ID, second); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	third, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.MountainLandType)
	if err := clone.RevisePrintedSpellText(copySpell.ID, third); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	if !g.FindPermanent(card.ID()).HasKeyword(core.Islandwalk) || !clone.FindPermanent(card.ID()).HasKeyword(core.Islandwalk) || !g.FindPermanent(copySpell.SourceID).HasKeyword(core.Forestwalk) || !clone.FindPermanent(copySpell.SourceID).HasKeyword(core.Mountainwalk) {
		t.Fatal("copy or clone shared spell rules")
	}
	copied := g.FindPermanent(copySpell.SourceID)
	if !copied.Card.IsToken() || copied.ControllerID() != b.PlayerID() {
		t.Fatal("copy lost token entry or controller")
	}
	values, err := g.CaptureCopiableValues(copied)
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	if printed.AttrSeeds()[core.Swampwalk] != 1 || printed.AttrSeeds()[core.Forestwalk] != 0 {
		t.Fatal("token copy values included ordinary text")
	}
}

func TestRONTextIntrinsicSpellLayers(t *testing.T) {
	g, a, _ := ronGame()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Intrinsic",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:3,Toughness:3),Rules:[FirstStrike,Landwalk(IsLandType(Swamp)),Landwalk(IsLandType(Swamp)),Protection(FromColor(Colors([White,Red])))])`), "intrinsic.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	island, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	blue, _ := mage.ReplaceColorWord(core.White, core.Blue)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, island, blue); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	id := landwalkID(t, p)
	protection := protectionID(t, p)
	forest, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	green, _ := mage.ReplaceColorWord(core.Blue, core.Green)
	if err := g.AddPrintedBasicLandwalkTextEffect(p, id, core.EndOfTurn, forest); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedProtectionTextEffect(g.FindPermanent(p.ID()), protection, core.EndOfTurn, green); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	current := g.FindPermanent(p.ID())
	if !current.HasKeyword(core.Forestwalk) || !current.HasKeyword(core.Islandwalk) || current.HasKeyword(core.Swampwalk) || !current.HasProtectionFrom(textColorCard(core.Green)) || !current.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("later layers mixed intrinsic declarations")
	}
	clone := g.Clone()
	g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
	if g.FindPermanent(p.ID()).HasKeyword(core.FirstStrike) || g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Green)) {
		t.Fatal("ability removal failed")
	}
	g.PhaseOut(p.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.ApplyContinuousEffects()
	current = g.FindPermanent(p.ID())
	if !current.HasKeyword(core.FirstStrike) || !current.HasKeyword(core.Islandwalk) || current.HasKeyword(core.Forestwalk) || !current.HasProtectionFrom(textColorCard(core.Blue)) || current.HasProtectionFrom(textColorCard(core.Green)) {
		t.Fatal("restoration lost continued baseline")
	}
	if !clone.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) || !clone.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Green)) {
		t.Fatal("expiry changed clone")
	}
	grant := g.PutOnBattlefield(repositoryCard(t, "ARN", "Fishliver Oil"), a.PlayerID())
	g.Attach(grant.ID(), p.ID())
	if err := g.AddPrintedBasicLandwalkTextEffect(g.FindPermanent(p.ID()), g.FindPermanent(p.ID()).RuntimeAbilities[len(g.FindPermanent(p.ID()).RuntimeAbilities)-1].AbilityID(), core.EndOfTurn, forest); err == nil {
		t.Fatal("grant accepted as a printed rule")
	}
}

func TestRONTextIntrinsicSpellProtectionGameplay(t *testing.T) {
	g, a, b := ronGame()

	card := repositoryCard(t, "LEA", "Black Knight")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.White, core.Blue)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.AddEntersTappedRule(func(p *mage.Permanent) bool { return p.HasType(core.TypeCreature) })
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if !p.Tapped {
		t.Fatal("continuation bypassed the entry rule")
	}
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 3, 3), a.PlayerID())
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(p, 1, g.CaptureDamageSource(blue.ID())), mage.PermanentDamageAssignment(other, 1, g.CaptureDamageSource(blue.ID())))
	if g.FindPermanent(p.ID()).Damage != 0 || g.FindPermanent(other.ID()).Damage != 1 || mage.CanBlock(blue, g.FindPermanent(p.ID()), g) {
		t.Fatal("protection lost damage or blocking behavior")
	}
	spell := repositoryCard(t, "LEA", "Unsummon")
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	b.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err == nil || b.ManaPool().TotalMana() != 1 {
		t.Fatal("illegal target accepted or paid")
	}
	aura := g.PutOnBattlefield(repositoryCard(t, "LEA", "Flight"), b.PlayerID())
	g.Attach(aura.ID(), p.ID())
	g.CheckStateBasedActions()
	if g.FindPermanent(aura.ID()).AttachedTo != uuid.Nil {
		t.Fatal("protection permitted a matching Aura")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if !g.LKI(p.ID()).Snapshot.HasProtectionFrom(textColorCard(core.Blue)) {
		t.Fatal("departure LKI lost protection")
	}
}

func TestRONTextIntrinsicSpellBoundaries(t *testing.T) {
	for _, rules := range []string{`Protection()`, `Protection(FromColor(Colors([])))`, `Protection(FromColor(Colors([Colorless])))`, `Landwalk(IsLandType(Unknown))`, `Landwalk(IsLandTypeVariable(TheChosenLandType))`, `FirstStrike()`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rules+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rules)
		}
	}
	for _, name := range []string{"Clone", "Sol'kanar the Swamp King"} {
		set := "LEA"
		if name == "Sol'kanar the Swamp King" {
			set = "LEG"
		}
		g, a, _ := ronGame()
		card := repositoryCard(t, set, name)
		if name == "Clone" {
			card = textCopyMixedRecord(t, set, name)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		for _, color := range []core.Color{core.Blue, core.Black, core.Red, core.Colorless} {
			a.ManaPool().Add(color, 10)
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		before := g.StackPeek()
		change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
		if err := g.RevisePrintedSpellText(before.ID, change); err == nil || g.StackPeek() != before {
			t.Fatal("unsupported rules changed stack state")
		}
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		if _, err := Compile(staticBaseRecord(t, "LEA", name)); err == nil {
			t.Fatal("incomplete text spell accepted")
		}
	}
}

func TestRONTextIntrinsicSpellInventory(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	expected := make(map[string]bool)
	for _, tc := range intrinsicSpellCases() {
		expected[tc.set+"/"+tc.name] = true
	}
	count := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		records, err := Parse(bytes.NewReader(data), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			types := record.Fields["Typeline"].Fields["Cardtypes"].Args
			if len(types) != 1 || types[0].Name != "Creature" || !intrinsicCreatureSpellTextRules(record.Fields["Rules"].Args) {
				continue
			}
			key := strings.TrimSuffix(filepath.Base(file), ".lines.ron") + "/" + record.Fields["Name"].String
			if !expected[key] {
				t.Fatal("intrinsic spell record lacks a gameplay mapping", key)
			}
			delete(expected, key)
			count++
		}
	}
	if count != 34 || len(expected) != 0 {
		t.Fatalf("inventory has %d records with %d missing mappings", count, len(expected))
	}
}
