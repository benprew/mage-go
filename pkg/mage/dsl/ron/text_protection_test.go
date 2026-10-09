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

func protectionID(t *testing.T, p *mage.Permanent) uuid.UUID {
	t.Helper()
	for _, a := range p.RuntimeAbilities {
		if _, ok := a.(*mage.ProtectionAbility); ok {
			return a.AbilityID()
		}
	}
	t.Fatal("missing protection declaration")
	return uuid.Nil
}

func TestRONTextProtectionRecords(t *testing.T) {
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
					if err := g.RevisePrintedProtection(p, id, change); err != nil {
						t.Fatal(err)
					}
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
				})
			}
		}
	}
	if len(names) != 12 || count != 22 {
		t.Fatalf("coverage: %d names / %d records", len(names), count)
	}
}

func TestRONTextProtectionLifetime(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEA", "Black Knight")
	p := g.PutOnBattlefield(card, a.PlayerID())
	id := protectionID(t, p)
	sibling := g.PutOnBattlefield(repositoryCard(t, "LEA", "Black Knight"), a.PlayerID())
	change, _ := mage.ReplaceColorWord(core.White, core.Red)
	if err := g.RevisePrintedProtection(p, id, change); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	next, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := clone.RevisePrintedProtection(clone.FindPermanent(p.ID()), id, next); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Red)) || !clone.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Blue)) || !sibling.HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("revision state shared")
	}
	phased := g.PhaseOut(p.ID())
	if err := g.RevisePrintedProtection(p, id, next); err == nil {
		t.Fatal("phased source accepted")
	}
	g.PhaseIn(phased)
	if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("phasing lost revision")
	}
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
	effect := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
	effect.SetSourceID(remover.ID())
	g.AddContinuousEffect(effect)
	if g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("removed protection stayed active")
	}
	if err := g.RevisePrintedProtection(g.FindPermanent(p.ID()), id, next); err == nil {
		t.Fatal("removed declaration accepted")
	}
	removedClone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	removedClone.RemoveFromBattlefield(removedClone.FindPermanent(remover.ID()))
	if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Red)) || !removedClone.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("restoration lost revision")
	}
	if err := removedClone.RevisePrintedProtection(removedClone.FindPermanent(p.ID()), id, next); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("restored clone changed original")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if lki := g.LKI(p.ID()); lki == nil || !lki.Snapshot.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("LKI lost revision")
	}
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.RevisePrintedProtection(p, id, next); err == nil {
		t.Fatal("returned source accepted")
	}
	if !returned.HasProtectionFrom(textColorCard(core.White)) || returned.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("return lost printed origin")
	}
}

func TestRONTextProtectionIndependentDeclarations(t *testing.T) {
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Protection(FromColor(Colors([White]))),Protection(FromColor(Colors([White])))`)(), a.PlayerID())
	first, second := p.RuntimeAbilities[0].AbilityID(), p.RuntimeAbilities[1].AbilityID()
	red, _ := mage.ReplaceColorWord(core.White, core.Red)
	blue, _ := mage.ReplaceColorWord(core.White, core.Blue)
	if err := g.RevisePrintedProtection(p, first, red); err != nil {
		t.Fatal(err)
	}
	if err := g.RevisePrintedProtection(g.FindPermanent(p.ID()), second, blue); err != nil {
		t.Fatal(err)
	}
	current := g.FindPermanent(p.ID())
	if current.HasProtectionFrom(textColorCard(core.White)) || !current.HasProtectionFrom(textColorCard(core.Red)) || !current.HasProtectionFrom(textColorCard(core.Blue)) {
		t.Fatal("declarations shared text history")
	}
}

func TestRONTextProtectionInvalid(t *testing.T) {
	for _, rule := range []string{`Protection()`, `Protection(FromColor(Colors([])))`, `Protection(FromColor(Colors([Colorless])))`, `Protection(FromColor(Colors([White])),Flying)`, `Protection(IsCreature)`} {
		records, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),ManaCost:[],Rules:[`+rule+`],CardPT:(Power:1,Toughness:1))`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(records[0]); err == nil {
			t.Fatal("invalid protection accepted", rule)
		}
	}
}

func TestRONTextProtectionCastTiming(t *testing.T) {
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
	if err := g.RevisePrintedProtection(source, id, change); err != nil {
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

func TestRONTextProtectionAttachmentAndGrants(t *testing.T) {
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "White Knight"), a.PlayerID())
	id := protectionID(t, p)
	ward := g.PutOnBattlefield(repositoryCard(t, "LEA", "White Ward"), a.PlayerID())
	g.Attach(ward.ID(), p.ID())
	g.ApplyContinuousEffects()
	grant := g.FindPermanent(p.ID()).RuntimeAbilities[len(g.FindPermanent(p.ID()).RuntimeAbilities)-1]
	change, _ := mage.ReplaceColorWord(core.Black, core.White)
	if err := g.RevisePrintedProtection(p, grant.AbilityID(), change); err == nil {
		t.Fatal("granted protection accepted")
	}
	if err := g.RevisePrintedProtection(p, id, change); err != nil {
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

func TestRONTextProtectionColorSet(t *testing.T) {
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Protection(FromColor(Colors([White,Red])))`)(), a.PlayerID())
	id := protectionID(t, p)
	change, _ := mage.ReplaceColorWord(core.White, core.Blue)
	if err := g.RevisePrintedProtection(p, id, change); err != nil {
		t.Fatal(err)
	}
	current := g.FindPermanent(p.ID())
	if current.HasProtectionFrom(textColorCard(core.White)) || !current.HasProtectionFrom(textColorCard(core.Blue)) || !current.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("color set lost an independent quality")
	}
}
