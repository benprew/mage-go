package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextSpellActionRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
			for _, category := range []string{"spell", "permanent"} {
				t.Run(set+"/"+name+"/"+category, func(t *testing.T) {
					g, base, b := ronGame()
					words := []string{"Swamp", "Island"}
					if name == "Sleight of Mind" {
						words = []string{"Red", "Blue"}
					}
					a := &intrinsicWordTester{Player: base, words: words}
					g.SetPlayerAt(0, a)
					var target uuid.UUID
					if category == "spell" {
						g.SetActivePlayerIndex(1)
						subject := repositoryCard(t, "LEA", "White Knight")
						if name == "Magical Hack" {
							subject = repositoryCard(t, "LEA", "Bog Wraith")
						} else {
							words[0] = "Black"
						}
						target = castRONSpell(t, g, b, subject).ID
					} else {
						subject := repositoryCard(t, "LEA", "Bog Wraith")
						if name == "Sleight of Mind" {
							words[0] = "Black"
							subject = repositoryCard(t, "LEA", "White Knight")
						}
						target = g.PutOnBattlefield(subject, b.PlayerID()).ID()
					}
					card := repositoryCard(t, set, name)
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err == nil || a.calls != 0 {
						t.Fatal("failed payment made choices")
					}
					a.ManaPool().Add(core.Blue, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
						t.Fatal(err)
					}
					if a.calls != 0 {
						t.Fatal("announcement made word or mode choices")
					}
					g.ResolveTopOfStack()
					if a.calls != 2 || a.ManaPool().TotalMana() != 0 {
						t.Fatal("wrong choice or payment timing")
					}
					clone := g.Clone()
					g.ResolveStack()
					clone.ResolveStack()
					for _, game := range []*mage.Game{g, clone} {
						var p *mage.Permanent
						for _, candidate := range game.AllBattlefield() {
							if candidate.Card.Name() == "White Knight" || candidate.Card.Name() == "Bog Wraith" {
								p = candidate
							}
						}
						if p == nil {
							t.Fatal("subject did not enter")
						}
						if name == "Magical Hack" {
							if !p.HasKeyword(core.Islandwalk) || p.HasKeyword(core.Swampwalk) {
								t.Fatal("land word did not change")
							}
						} else if !p.HasProtectionFrom(textColorCard(core.Blue)) || p.HasProtectionFrom(textColorCard(core.Black)) {
							t.Fatal("color word did not change")
						}
					}
				})
			}
		}
	}
}

func TestRONTextSpellActionInvalid(t *testing.T) {
	valid := `ChooseTwoColorWords,CreateSpellEffect(Ref_TargetSpell,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`
	for _, body := range []string{
		strings.TrimPrefix(valid, "ChooseTwoColorWords,"),
		strings.Replace(valid, "Ref_TargetSpell", "Ref_TargetPermanent", 1),
		strings.Replace(valid, "TheSecondChosenColorWord", "TheFirstChosenColorWord", 1),
		strings.Replace(valid, "ChooseTwoColorWords", "ChooseTwoBasicLandTypes", 1),
		`ChooseTwoColorWords,CreateSpellEffect(Ref_TargetSpell,[])`,
		strings.Replace(valid, ")])", "),SetColor(SimpleColorList([Blue]))])", 1),
		strings.Replace(valid, "CreateSpellEffect(Ref_TargetSpell,", "CreateSpellEffect(", 1),
	} {
		rule := `SpellActions(Targeted([TargetSpell(AnySpell)],ActionList([` + body + `])))`
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatal("invalid spell text declaration accepted", body)
		}
	}
}

type spellTextActionPlayer struct {
	intrinsicWordTester
	targets []uuid.UUID
}

func (p *spellTextActionPlayer) ChooseTargets([]uuid.UUID, int, int, *mage.Game) []uuid.UUID {
	return append([]uuid.UUID(nil), p.targets...)
}

func castTextConsumer(t *testing.T, g *mage.Game, player mage.Player, name string, target uuid.UUID) *mage.StackObject {
	t.Helper()
	card := repositoryCard(t, "LEA", name)
	player.AddToHand(card)
	player.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(player.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
		t.Fatal(err)
	}
	return g.StackPeek()
}

func TestRONTextSpellActionCopyAndClone(t *testing.T) {
	for _, fromSpell := range []bool{false, true} {
		g, base, b := ronGame()
		a := &spellTextActionPlayer{intrinsicWordTester: intrinsicWordTester{Player: base, words: []string{"Black", "Blue", "Black", "Green"}}}
		g.SetPlayerAt(0, a)
		g.SetActivePlayerIndex(1)
		spellCard := repositoryCard(t, "LEA", "White Knight")
		spell := castRONSpell(t, g, b, spellCard)
		p := g.PutOnBattlefield(repositoryCard(t, "LEA", "White Knight"), b.PlayerID())
		first, second := p.ID(), spell.ID
		if fromSpell {
			first, second = second, first
		}
		original := castTextConsumer(t, g, a, "Sleight of Mind", first)
		a.targets = []uuid.UUID{second}
		copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
		if copied == nil || copied.Targets[0] != second || original.Targets[0] != first || a.calls != 0 {
			t.Fatal("copy changed original target or chose words early")
		}
		clone := g.Clone()
		clonePlayer := &intrinsicWordTester{Player: clone.GetPlayer(a.PlayerID()), words: []string{"Black", "Red", "Black", "White"}}
		clone.SetPlayerAt(0, clonePlayer)
		g.ResolveTopOfStack()
		g.ResolveTopOfStack()
		g.ResolveStack()
		clone.ResolveStack()
		for _, tc := range []struct {
			game           *mage.Game
			copy, original core.Color
		}{{g, core.Blue, core.Green}, {clone, core.Red, core.White}} {
			permanentColor, spellColor := tc.original, tc.copy
			if fromSpell {
				permanentColor, spellColor = tc.copy, tc.original
			}
			if !tc.game.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(permanentColor)) || !tc.game.FindPermanent(spellCard.ID()).HasProtectionFrom(textColorCard(spellColor)) {
				t.Fatal("copy category or independent word binding lost")
			}
		}
		if a.calls != 4 || clonePlayer.calls != 4 {
			t.Fatal("copy or clone repeated or omitted choices")
		}
	}
}

func TestRONTextSpellActionLegality(t *testing.T) {
	for _, state := range []string{"countered", "recast", "phased", "phase-cycle", "returned", "shroud"} {
		t.Run(state, func(t *testing.T) {
			g, base, b := ronGame()
			a := &intrinsicWordTester{Player: base, words: []string{"Swamp", "Island"}}
			g.SetPlayerAt(0, a)
			card := repositoryCard(t, "LEA", "Bog Wraith")
			if state == "recast" {
				card = repositoryCard(t, "LEA", "Dark Ritual")
			}
			var target uuid.UUID
			var p *mage.Permanent
			if state == "countered" || state == "recast" {
				g.SetActivePlayerIndex(1)
				target = castRONSpell(t, g, b, card).ID
			} else {
				p = g.PutOnBattlefield(card, b.PlayerID())
				target = p.ID()
			}
			castTextConsumer(t, g, a, "Magical Hack", target)
			switch state {
			case "countered", "recast":
				g.CounterStackObject(target)
				if state == "recast" {
					b.RemoveFromGraveyard(card.ID())
					castRONSpell(t, g, b, card)
					g.ResolveTopOfStack()
				}
			case "phased", "phase-cycle":
				ids := g.PhaseOut(p.ID())
				if state == "phase-cycle" {
					g.PhaseIn(ids)
				}
			case "returned":
				g.BouncePermanentToHand(p)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "shroud":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), p.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.ApplyContinuousEffects()
			}
			g.ResolveTopOfStack()
			want := 0
			if state == "phase-cycle" {
				want = 2
				if !g.FindPermanent(card.ID()).HasKeyword(core.Islandwalk) {
					t.Fatal("phase cycle lost the text change")
				}
			}
			if a.calls != want {
				t.Fatal("illegal target made choices", a.calls)
			}
			if state == "returned" {
				if !g.FindPermanent(card.ID()).HasKeyword(core.Swampwalk) {
					t.Fatal("old target followed a new visit")
				}
			}
		})
	}
}

func TestRONTextSpellActionOrderAndDeparture(t *testing.T) {
	g, base, b := ronGame()
	a := &intrinsicWordTester{Player: base, words: []string{"Plains", "Forest", "Forest", "Island"}}
	g.SetPlayerAt(0, a)
	g.SetActivePlayerIndex(1)
	card := repositoryCard(t, "LEA", "Flashfires")
	subject := castRONSpell(t, g, b, card)
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	castTextConsumer(t, g, a, "Magical Hack", subject.ID)
	g.ResolveTopOfStack()
	clone := g.Clone()
	castTextConsumer(t, g, a, "Magical Hack", subject.ID)
	g.ResolveTopOfStack()
	g.ResolveStack()
	clone.ResolveStack()
	if g.FindPermanent(island.ID()) != nil || g.FindPermanent(forest.ID()) == nil || clone.FindPermanent(forest.ID()) != nil || clone.FindPermanent(island.ID()) == nil {
		t.Fatal("ordered changes or clone isolation failed")
	}
	plains := g.PutOnBattlefield(repositoryCard(t, "LEA", "Plains"), b.PlayerID())
	b.RemoveFromGraveyard(card.ID())
	castRONSpell(t, g, b, card)
	g.ResolveStack()
	if g.FindPermanent(plains.ID()) != nil || g.FindPermanent(forest.ID()) == nil || a.calls != 4 {
		t.Fatal("physical recast kept changed text or repeated choices")
	}
}

func TestRONTextSpellActionMixedList(t *testing.T) {
	g, base, b := ronGame()
	a := &intrinsicWordTester{Player: base, words: []string{"Red", "Blue", "Swamp", "Forest"}}
	g.SetPlayerAt(0, a)
	g.SetActivePlayerIndex(1)
	factory, err := Compile(nameScopeNode(t, `Card(Name:"Mixed",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[Protection(FromColor(Colors([Red]))),Landwalk(IsLandType(Swamp))],CardPT:(Power:2,Toughness:2))`))
	if err != nil {
		t.Fatal(err)
	}
	subject := factory()
	subjectID := castRONSpell(t, g, b, subject).ID
	rule := `SpellActions(Targeted([TargetSpell(AnySpell)],ActionList([ChooseTwoColorWords,ChooseTwoBasicLandTypes,CreateSpellEffect(Ref_TargetSpell,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord),ReplaceLandTypeVariableWithNewLandTypeVariable(TheFirstChosenLandType,TheSecondChosenLandType)])])))`
	consumer := permanentScopeFactory(t, "Instant", rule)()
	a.AddToHand(consumer)
	if err := g.CastSpellByID(a.PlayerID(), consumer.ID(), []uuid.UUID{subjectID}, 0); err != nil {
		t.Fatal(err)
	}
	if a.calls != 0 {
		t.Fatal("mixed list made choices early")
	}
	g.ResolveStack()
	p := g.FindPermanent(subject.ID())
	if p == nil || !p.HasKeyword(core.Forestwalk) || p.HasKeyword(core.Swampwalk) || !p.HasProtectionFrom(textColorCard(core.Blue)) || p.HasProtectionFrom(textColorCard(core.Red)) || a.calls != 4 {
		t.Fatal("mixed list lost a lexical pair")
	}
}

func TestRONTextSpellActionRevisedTargetLegality(t *testing.T) {
	g, base, b := ronGame()
	a := &intrinsicWordTester{Player: base, words: []string{"Black", "Green"}}
	g.SetPlayerAt(0, a)
	p := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	subject := castRONSpell(t, g, b, repositoryCard(t, "LEA", "Terror"), p.ID())
	castTextConsumer(t, g, a, "Sleight of Mind", subject.ID)
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if a.calls != 2 || g.FindPermanent(p.ID()) == nil {
		t.Fatal("changed spell target restriction was not checked")
	}
}
