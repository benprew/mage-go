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

func TestRONTextProtectionLayerRecords(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
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
			rules := record.Fields["Rules"].Args
			for _, rule := range rules {
				if rule.Name != "Protection" {
					continue
				}
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				card := factory()
				names[card.Name()] = true
				count++
				t.Run(filepath.Base(file)+"/"+card.Name(), func(t *testing.T) {
					g, a, b := ronGame()
					p := g.PutOnBattlefield(card, a.PlayerID())
					id := protectionID(t, p)
					old, err := protectionColors(rule.Args[0])
					if err != nil {
						t.Fatal(err)
					}
					change, _ := mage.ReplaceColorWord(old[0], core.Blue)
					if err := g.AddPrintedProtectionTextEffect(p, id, core.EndOfTurn, change); err != nil {
						t.Fatal(err)
					}
					g.ApplyContinuousEffects()
					g.ApplyContinuousEffects()
					current := g.FindPermanent(p.ID())
					if protectionID(t, current) != id || current.HasProtectionFrom(textColorCard(old[0])) || !current.HasProtectionFrom(textColorCard(core.Blue)) {
						t.Fatal("printed protection was not revised")
					}
					blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
					other := g.PutOnBattlefield(mage.NewCreature("Other", "", 3, 3), a.PlayerID())
					g.DealSimultaneousDamage(mage.PermanentDamageAssignment(current, 1, g.CaptureDamageSource(blue.ID())), mage.PermanentDamageAssignment(other, 1, g.CaptureDamageSource(blue.ID())))
					if g.FindPermanent(p.ID()).Damage != 0 || g.FindPermanent(other.ID()).Damage != 1 {
						t.Fatal("shared prevention did not use revised protection")
					}
					if g.FindPermanent(p.ID()).CanBeTargetedBy(blue.Card, b.PlayerID(), g) {
						t.Fatal("matching target source accepted")
					}
					if mage.CanBlock(blue, g.FindPermanent(p.ID()), g) {
						t.Fatal("matching blocker accepted")
					}
					g.GetEffects().RemoveEndOfTurn()
					g.ApplyContinuousEffects()
					if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(old[0])) || g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Blue)) {
						t.Fatal("expiry lost printed protection")
					}
				})
			}
		}
	}
	if len(names) != 12 || count != 22 {
		t.Fatalf("coverage: %d names / %d records", len(names), count)
	}
}

func TestRONTextProtectionLayerOrderingAndExpiry(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Black Knight")
	p := g.PutOnBattlefieldAttacking(card, a.PlayerID(), b.PlayerID())
	id := protectionID(t, p)
	first, _ := mage.ReplaceColorWord(core.White, core.Blue)
	second, _ := mage.ReplaceColorWord(core.Blue, core.Red)
	if err := g.AddPrintedProtectionTextEffect(p, id, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedProtectionTextEffect(g.FindPermanent(p.ID()), id, core.Indefinite, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Red)) || g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("ordered text effects failed")
	}
	blocker := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: p.ID()}})
	if g.IsBlockingInCombat(blocker.ID()) {
		t.Fatal("combat ignored layer-3 protection")
	}
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.White)) || clone.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("expiry did not recompute from baseline")
	}
	if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("clone expiry changed original")
	}
	if !g.PutOnBattlefield(card.Copy(), a.PlayerID()).HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("text changed copy information")
	}
}

func TestRONTextProtectionLayerRemovalAndPhasing(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Black Knight"), a.PlayerID())
	id := protectionID(t, p)
	first, _ := mage.ReplaceColorWord(core.White, core.Blue)
	if err := g.AddPrintedProtectionTextEffect(p, id, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	aura := g.PutOnBattlefield(repositoryCard(t, "LEA", "Blue Ward"), a.PlayerID())
	g.Attach(aura.ID(), p.ID())
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	removal := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	if g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Blue)) {
		t.Fatal("layer 6 removal failed")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.White)) || !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Blue)) {
		t.Fatal("expiry changed grant or restoration")
	}
	if err := g.AddPrintedProtectionTextEffect(g.FindPermanent(p.ID()), id, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	phased := g.PhaseOut(p.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("phased expiry did not restore baseline")
	}
}

func TestRONTextProtectionLayerIdentityAndBaseline(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEA", "Black Knight")
	p := g.PutOnBattlefield(card, a.PlayerID())
	id := protectionID(t, p)
	blue, _ := mage.ReplaceColorWord(core.White, core.Blue)
	red, _ := mage.ReplaceColorWord(core.Blue, core.Red)
	if err := g.RevisePrintedProtection(p, id, blue); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedProtectionTextEffect(g.FindPermanent(p.ID()), id, core.EndOfTurn, red); err != nil {
		t.Fatal(err)
	}
	if err := g.RevisePrintedProtection(g.FindPermanent(p.ID()), id, red); err == nil {
		t.Fatal("direct revision bypassed layered baseline")
	}
	sibling := g.PutOnBattlefield(card.Copy(), a.PlayerID())
	if !sibling.HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("copy inherited noncopiable text")
	}
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Blue)) {
		t.Fatal("expiry lost prior direct revision")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if lki := g.LKI(p.ID()); lki == nil || !lki.Snapshot.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("departure LKI lost text")
	}
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.ApplyContinuousEffects()
	if !returned.HasProtectionFrom(textColorCard(core.White)) || returned.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("returned object inherited layer")
	}
	if err := g.AddPrintedProtectionTextEffect(p, id, core.Indefinite, blue); err == nil {
		t.Fatal("old incarnation accepted")
	}
}

func TestRONTextProtectionLayerInvalid(t *testing.T) {
	for _, scenario := range []string{"nil", "id", "empty", "change", "land", "duration", "face-down", "phased", "departed", "removed", "grant"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Black Knight"), a.PlayerID())
			id := protectionID(t, p)
			change, _ := mage.ReplaceColorWord(core.White, core.Blue)
			changes := []mage.TextChange{change}
			duration := core.EndOfTurn
			input := p
			switch scenario {
			case "nil":
				input = nil
			case "id":
				id = protectionID(t, g.PutOnBattlefield(repositoryCard(t, "LEA", "Black Knight"), a.PlayerID()))
			case "empty":
				changes = nil
			case "change":
				changes = []mage.TextChange{{}}
			case "land":
				change, _ = mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
				changes = []mage.TextChange{change}
			case "duration":
				duration = core.WhileOnBattlefield
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "phased":
				g.PhaseOut(p.ID())
			case "departed":
				g.BouncePermanentToHand(p)
			case "removed":
				effect := mage.RemoveAllAbilities(p.ID(), core.EndOfTurn)
				g.AddContinuousEffect(effect)
			case "grant":
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = mage.WrapGrantedAbility(p.RuntimeAbilities[0])
			}
			if err := g.AddPrintedProtectionTextEffect(input, id, duration, changes...); err == nil {
				t.Fatal("invalid installation accepted")
			}
			if present := g.FindPermanent(p.ID()); present != nil && !present.FaceDown && present.HasProtectionFrom(textColorCard(core.Blue)) {
				t.Fatal("failed installation changed text")
			}
		})
	}
	for _, rule := range []string{`Protection()`, `Protection(FromColor(Colors([Unknown])))`, `Protection(FromColor(Colors([White])),Flying)`, `Protection(FromColor(Colors([])))`} {
		records, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(records[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}

func TestRONTextProtectionLayerDuplicateDeclarations(t *testing.T) {
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Protection(FromColor(Colors([White]))),Protection(FromColor(Colors([White])))`)(), a.PlayerID())
	first, second := p.RuntimeAbilities[0].AbilityID(), p.RuntimeAbilities[1].AbilityID()
	blue, _ := mage.ReplaceColorWord(core.White, core.Blue)
	red, _ := mage.ReplaceColorWord(core.White, core.Red)
	if err := g.AddPrintedProtectionTextEffect(p, first, core.EndOfTurn, blue); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedProtectionTextEffect(g.FindPermanent(p.ID()), second, core.Indefinite, red); err != nil {
		t.Fatal(err)
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	current := g.FindPermanent(p.ID())
	if !current.HasProtectionFrom(textColorCard(core.White)) || !current.HasProtectionFrom(textColorCard(core.Red)) || current.HasProtectionFrom(textColorCard(core.Blue)) {
		t.Fatal("sibling lifetimes mixed")
	}
}

func TestRONTextProtectionLayerCastTiming(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Black Knight")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("creature entered before resolution")
	}
	g.ResolveTopOfStack()
	source := g.FindPermanent(card.ID())
	id := protectionID(t, source)
	spell := repositoryCard(t, "LEA", "Unsummon")
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	b.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{source.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.White, core.Blue)
	if err := g.AddPrintedProtectionTextEffect(source, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveTopOfStack()
	if g.FindPermanent(source.ID()) == nil || !g.FindPermanent(source.ID()).HasKeyword(core.FirstStrike) {
		t.Fatal("resolution did not recheck protection or changed another printed rule")
	}
	if clone.StackSize() != 1 {
		t.Fatal("clone shared stack resolution")
	}
	clone.ResolveTopOfStack()
	if clone.FindPermanent(source.ID()) == nil {
		t.Fatal("cloned legality lost revised protection")
	}
	second := repositoryCard(t, "LEA", "Unsummon")
	second.SetOwner(b.PlayerID())
	b.AddToHand(second)
	b.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(b.PlayerID(), second.ID(), []uuid.UUID{source.ID()}, 0); err == nil {
		t.Fatal("new illegal target accepted")
	}
	if b.ManaPool().TotalMana() != 1 {
		t.Fatal("illegal announcement consumed mana")
	}
}

func TestRONTextProtectionLayerAttachmentAndGrants(t *testing.T) {
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "White Knight"), a.PlayerID())
	id := protectionID(t, p)
	ward := g.PutOnBattlefield(repositoryCard(t, "LEA", "White Ward"), a.PlayerID())
	g.Attach(ward.ID(), p.ID())
	g.ApplyContinuousEffects()
	grant := g.FindPermanent(p.ID()).RuntimeAbilities[len(g.FindPermanent(p.ID()).RuntimeAbilities)-1]
	change, _ := mage.ReplaceColorWord(core.Black, core.White)
	if err := g.AddPrintedProtectionTextEffect(p, grant.AbilityID(), core.EndOfTurn, change); err == nil {
		t.Fatal("granted protection accepted")
	}
	if err := g.AddPrintedProtectionTextEffect(p, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(ward.ID()) != nil {
		t.Fatal("printed protection incorrectly exempted the Ward")
	}
	if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("grant departure removed printed protection")
	}
}

func TestRONTextProtectionLayerColorSet(t *testing.T) {
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Protection(FromColor(Colors([White,Red])))`)(), a.PlayerID())
	id := protectionID(t, p)
	change, _ := mage.ReplaceColorWord(core.White, core.Blue)
	if err := g.AddPrintedProtectionTextEffect(p, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	current := g.FindPermanent(p.ID())
	if current.HasProtectionFrom(textColorCard(core.White)) || !current.HasProtectionFrom(textColorCard(core.Blue)) || !current.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("color set lost an independent quality")
	}
}
