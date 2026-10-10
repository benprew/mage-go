package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextSpellContinuationRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Terror", "Flashfires", "Tsunami", "Swords to Plowshares", "Hurricane"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				spell := repositoryCard(t, set, name)
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				a.ManaPool().Add(core.Colorless, 10)
				a.ManaPool().Add(core.Black, 1)
				a.ManaPool().Add(core.Red, 1)
				a.ManaPool().Add(core.Green, 1)
				a.ManaPool().Add(core.White, 1)
				creature := mage.NewCreature("Target", "{R}", 3, 6)
				p := g.PutOnBattlefield(creature, b.PlayerID())
				land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
				var targets []uuid.UUID
				if name == "Terror" || name == "Swords to Plowshares" {
					targets = []uuid.UUID{p.ID()}
				}
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), targets, 2); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				before := a.ManaPool().TotalMana()
				change, _ := mage.ReplaceColorWord(core.Black, core.Green)
				if name == "Flashfires" {
					change, _ = mage.ReplaceBasicLandWord(core.PlainsLandType, core.ForestLandType)
				}
				if name == "Tsunami" {
					change, _ = mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
				}
				if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != before || g.StackPeek().XValue != 2 || g.StackPeek().SourceID != spell.ID() {
					t.Fatal("revision changed payment, X or source identity")
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					switch name {
					case "Flashfires", "Tsunami":
						if game.FindPermanent(land.ID()) != nil || game.FindPermanent(p.ID()) == nil {
							t.Fatal("land word or collection selection failed")
						}
					case "Terror", "Swords to Plowshares":
						if game.FindPermanent(p.ID()) != nil {
							t.Fatal("fresh target binding lost the announced target")
						}
					case "Hurricane":
						if game.GetPlayer(b.PlayerID()).Life() != 18 {
							t.Fatal("announced X did not reach damage")
						}
					}
				}
				if name == "Swords to Plowshares" && b.Life() != 23 {
					t.Fatal("resolution lost pre-departure power or controller")
				}
				if spell.ManaCost().String() != g.GetPlayer(a.PlayerID()).Graveyard()[0].ManaCost().String() {
					t.Fatal("revision changed the physical card")
				}
			})
		}
	}
}

func TestRONTextSpellContinuationLegality(t *testing.T) {
	for _, state := range []string{"illegal", "legal", "return", "phase", "phase-cycle"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			spell := repositoryCard(t, "LEA", "Terror")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Black, 2)
			p := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			id := g.StackPeek().ID
			before := g.Clone()
			change, _ := mage.ReplaceColorWord(core.Black, core.Red)
			if err := g.RevisePrintedSpellText(id, change); err != nil {
				t.Fatal(err)
			}
			if state != "illegal" {
				change, _ = mage.ReplaceColorWord(core.Red, core.Green)
				if err := g.RevisePrintedSpellText(id, change); err != nil {
					t.Fatal(err)
				}
			}
			switch state {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, b.PlayerID())
			case "phase", "phase-cycle":
				g.PhaseOut(p.ID())
				if state == "phase-cycle" {
					g.PhaseIn([]uuid.UUID{p.ID()})
				}
			}
			g.ResolveStack()
			wantPresent := state == "illegal" || state == "return" || state == "phase"
			if (g.FindPermanentIncludingPhased(p.ID()) != nil) != wantPresent {
				t.Fatal("text legality or exact target visit was lost")
			}
			before.ResolveStack()
			if before.FindPermanent(p.ID()) != nil {
				t.Fatal("revision changed a prior clone")
			}
			if err := g.RevisePrintedSpellText(id, change); err == nil {
				t.Fatal("departed stack visit accepted")
			}
		})
	}
}

func TestRONTextSpellContinuationResults(t *testing.T) {
	g, a, _ := ronGame()
	spell := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([DrawACard,RevealTheCardDrawnThisWay,DiscardTheCardDrawnThisWay,DrawACard,RevealTheCardDrawnThisWay,DiscardTheCardDrawnThisWay,GainLife(Integer(2))]))`)()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	for range 2 {
		a.AddToLibrary(mage.NewLand("Drawn"))
	}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Black, core.Red)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		player := game.GetPlayer(a.PlayerID())
		if len(player.Hand()) != 0 || len(player.Graveyard()) != 3 || player.Life() != 22 {
			t.Fatal("fresh resolution result frames were not independent")
		}
	}
}

func TestRONTextSpellContinuationPartialLegality(t *testing.T) {
	g, a, b := ronGame()
	spell := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsColor(Red)),TargetPermanent(IsColor(Green))],ActionList([TapPermanent(Ref_TargetPermanent1),TapPermanent(Ref_TargetPermanent2)])))`)()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{red.ID(), green.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(red.ID()).Tapped || !g.FindPermanent(green.ID()).Tapped {
		t.Fatal("partly illegal groups did not resolve independently")
	}
}

func TestRONTextSpellContinuationSpellTarget(t *testing.T) {
	g, a, b := ronGame()
	bolt := repositoryCard(t, "LEA", "Lightning Bolt")
	bolt.SetOwner(a.PlayerID())
	a.AddToHand(bolt)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), bolt.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	id := g.StackPeek().ID
	spell := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetSpell(IsColor(Red))],ActionList([CounterSpell(Ref_TargetSpell)])))`)()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{id}, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if b.Life() != 17 {
		t.Fatal("spell target did not use revised color legality")
	}
}

func TestRONTextSpellContinuationBoundaries(t *testing.T) {
	for _, rule := range []string{
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))),SpellActions(ActionList([GainLife(Integer(1))]))`,
	} {
		g, a, _ := ronGame()
		spell := permanentScopeFactory(t, "Instant", rule)()
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, 1), a.PlayerID())
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		before := g.StackPeek()
		change, _ := mage.ReplaceColorWord(core.Black, core.Red)
		if err := g.RevisePrintedSpellText(before.ID, change); err == nil || g.StackPeek() != before {
			t.Fatal("unsupported continuation changed the live object")
		}
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		if _, err := Compile(staticBaseRecord(t, "LEA", name)); err == nil {
			t.Fatal("partial text spell accepted")
		}
	}
}
