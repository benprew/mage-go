package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type intrinsicWordTester struct {
	mage.Player
	words []string
	calls int
}

func (p *intrinsicWordTester) ChooseMode(options []string, _ string) int {
	word := p.words[p.calls%len(p.words)]
	p.calls++
	for i, option := range options {
		if option == word {
			return i
		}
	}
	return 0
}

func TestRONTextIntrinsicObjectRecords(t *testing.T) {
	for _, tc := range intrinsicSpellCases() {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			card := repositoryCard(t, tc.set, tc.name)
			for _, kind := range []string{"color", "land"} {
				from, to := "", "Blue"
				if kind == "color" {
					for _, ability := range card.Abilities() {
						if protection, ok := ability.(*mage.ProtectionAbility); ok {
							from = protection.FromColors[0].String()
						}
					}
				} else {
					to = "Forest"
					for _, pair := range []struct {
						kw   core.Attr
						name string
					}{{core.Plainswalk, "Plains"}, {core.Islandwalk, "Island"}, {core.Swampwalk, "Swamp"}, {core.Mountainwalk, "Mountain"}, {core.Forestwalk, "Forest"}} {
						if card.AttrSeeds()[pair.kw] != 0 {
							from = pair.name
						}
					}
					if from == to {
						to = "Island"
					}
				}
				if from == "" {
					continue
				}
				g, base, b := ronGame()
				a := &intrinsicWordTester{Player: base, words: []string{from, to}}
				g.SetPlayerAt(0, a)
				p := g.PutOnBattlefield(card.Copy(), b.PlayerID())
				name := "Magical Hack"
				if kind == "color" {
					name = "Sleight of Mind"
				}
				record := staticBaseRecord(t, "LEA", name)
				branch := record.Fields["Rules"].Args[0].Args[0].Args[0].Args[1]
				record.Fields["Rules"].Args[0].Args[0] = branch
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				spell := factory()
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err == nil || a.calls != 0 || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("unpaid casting changed state")
				}
				a.ManaPool().Add(core.Blue, 1)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				if a.calls != 0 {
					t.Fatal("early word choices")
				}
				g.ResolveStack()
				p = g.FindPermanent(p.ID())
				if a.calls != 2 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("wrong choice or payment timing")
				}
				if kind == "color" {
					if !p.HasProtectionFrom(textColorCard(core.Blue)) {
						t.Fatal("real RON color change lost")
					}
					color := core.White
					if from == "Black" {
						color = core.Black
					}
					if from == "Red" {
						color = core.Red
					}
					if p.HasProtectionFrom(textColorCard(color)) {
						t.Fatal("original color word survived")
					}
				} else {
					keyword := core.Forestwalk
					if to == "Island" {
						keyword = core.Islandwalk
					}
					if !p.HasKeyword(keyword) {
						t.Fatal("real RON land change lost")
					}
					for kw, count := range card.AttrSeeds() {
						if count > 0 && kw != keyword && !p.HasKeyword(kw) && kw != core.Plainswalk && kw != core.Islandwalk && kw != core.Swampwalk && kw != core.Mountainwalk && kw != core.Forestwalk {
							t.Fatal("ordinary keyword changed")
						}
					}
				}

				if kind == "color" {
					blue := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
					g.DealSimultaneousDamage(mage.PermanentDamageAssignment(p, 1, g.CaptureDamageSource(blue.ID())))
					if g.FindPermanent(p.ID()).Damage != 0 || mage.CanBlock(blue, g.FindPermanent(p.ID()), g) {
						t.Fatal("changed protection lost damage or block legality")
					}
				} else {
					g.PutOnBattlefield(repositoryCard(t, "LEA", to), a.PlayerID())
					if !mage.HasLandwalkEvasion(g.FindPermanent(p.ID()), a.PlayerID(), g) {
						t.Fatal("changed landwalk lost evasion")
					}
				}
				if p.Card.Name() != card.Name() || p.Card.ManaCost().String() != card.ManaCost().String() {
					t.Fatal("name or mana symbols changed")
				}
			}
		})
	}
}

func TestRONTextIntrinsicObjectLifecycle(t *testing.T) {
	g, a, b := ronGame()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Multiple",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[FirstStrike,Protection(FromColor(Colors([White,Red]))),Protection(FromColor(Colors([White]))),Landwalk(IsLandType(Swamp)),Landwalk(IsLandType(Swamp))],CardPT:(Power:2,Toughness:3))`), "multiple.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.GrantKeywordModification(mage.This, core.Swampwalk)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	originalIDs := make([]uuid.UUID, 0, len(p.RuntimeAbilities))
	for _, ability := range p.RuntimeAbilities {
		originalIDs = append(originalIDs, ability.AbilityID())
	}
	whiteBlue, _ := mage.ReplaceColorWord(core.White, core.Blue)
	swampForest, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
	if err := g.AddPrintedIntrinsicCreatureTextEffect(p, core.Indefinite, whiteBlue, swampForest); err != nil {
		t.Fatal(err)
	}
	blueGreen, _ := mage.ReplaceColorWord(core.Blue, core.Green)
	forestIsland, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.AddPrintedIntrinsicCreatureTextEffect(g.FindPermanent(p.ID()), core.EndOfTurn, blueGreen, forestIsland); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	check := func(game *mage.Game, color core.Color, kw core.Attr) {
		t.Helper()
		q := game.FindPermanent(p.ID())
		if !q.HasProtectionFrom(textColorCard(color)) || !q.HasProtectionFrom(textColorCard(core.Red)) || !q.HasKeyword(kw) || !q.HasKeyword(core.FirstStrike) {
			t.Fatal("sibling or ordered words lost")
		}
		for i, ability := range q.RuntimeAbilities {
			if ability.AbilityID() != originalIDs[i] {
				t.Fatal("declaration identity changed")
			}
		}
	}
	check(g, core.Green, core.Islandwalk)
	if !g.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("text changed a granted keyword")
	}
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	check(clone, core.Blue, core.Forestwalk)
	check(g, core.Green, core.Islandwalk)
	g.PhaseOut(p.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.ApplyContinuousEffects()
	check(g, core.Blue, core.Forestwalk)
	g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
	if g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Blue)) || g.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("layer 6 removal lost")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	check(g, core.Blue, core.Forestwalk)
	values, err := g.CaptureCopiableValues(g.FindPermanent(p.ID()))
	if err != nil {
		t.Fatal(err)
	}
	copyCard, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	copyPermanent := g.PutOnBattlefield(copyCard, b.PlayerID())
	if !copyPermanent.HasProtectionFrom(textColorCard(core.White)) || !copyPermanent.HasKeyword(core.Swampwalk) {
		t.Fatal("copy inherited text layer")
	}
	stale := g.FindPermanent(p.ID())
	g.BouncePermanentToHand(stale)
	if !g.LKI(p.ID()).Snapshot.HasProtectionFrom(textColorCard(core.Blue)) || !g.LKI(p.ID()).Snapshot.HasKeyword(core.Forestwalk) {
		t.Fatal("LKI lost words")
	}
	returned := g.PutOnBattlefield(a.Hand()[0], a.PlayerID())
	if err := g.AddPrintedIntrinsicCreatureTextEffect(stale, core.Indefinite, whiteBlue); err == nil {
		t.Fatal("returned incarnation accepted")
	}
	if !returned.HasProtectionFrom(textColorCard(core.White)) || !returned.HasKeyword(core.Swampwalk) {
		t.Fatal("return inherited words")
	}
}

func TestRONTextIntrinsicObjectDeclarations(t *testing.T) {
	prefix := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([`
	for _, body := range []string{
		`CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheFirstChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceLandTypeVariableWithNewLandTypeVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPlayer,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord),SetColor([Red])])`,
	} {
		nodes, err := Parse(strings.NewReader(prefix+body+`])))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", body)
		}
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		if _, err := Compile(staticBaseRecord(t, "LEA", name)); err != nil {
			t.Fatal("complete text consumer failed", name, err)
		}
	}
}

func TestRONTextIntrinsicObjectAtomicAndIsolation(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Zombie Master"}, {"LEG", "Kobold Overlord"}, {"LEA", "Serra Angel"}} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		before := g.FindPermanent(p.ID())
		change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
		if err := g.AddPrintedIntrinsicCreatureTextEffect(p, core.Indefinite, change); err == nil {
			t.Fatal("unsupported complete object accepted", tc.name)
		}
		if g.FindPermanent(p.ID()) != before {
			t.Fatal("rejection mutated source")
		}
	}
	g, base, b := ronGame()
	a := &intrinsicWordTester{Player: base, words: []string{"Forest", "Island", "Swamp", "Forest"}}
	g.SetPlayerAt(0, a)
	first := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), b.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEA", "Shanodin Dryads"), b.PlayerID())
	rule := `SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([ChooseTwoBasicLandTypes,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceLandTypeVariableWithNewLandTypeVariable(TheFirstChosenLandType,TheSecondChosenLandType)])])))`
	factory := permanentScopeFactory(t, "Instant", rule)
	for _, p := range []*mage.Permanent{first, second} {
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
			t.Fatal(err)
		}
	}
	clone := g.Clone()
	g.ResolveStack()
	g.ResolveStack()
	if !g.FindPermanent(second.ID()).HasKeyword(core.Islandwalk) || !g.FindPermanent(first.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("pending pair results crossed instances")
	}
	clone.ResolveStack()
	clone.ResolveStack()
	if !g.FindPermanent(second.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("cloned resolution changed original")
	}
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(first.ID()))
	returned := g.PutOnBattlefield(b.Hand()[0], b.PlayerID())
	calls := a.calls
	g.ResolveStack()
	if calls != a.calls || !returned.HasKeyword(core.Swampwalk) {
		t.Fatal("illegal returned target chose or changed text")
	}
}
