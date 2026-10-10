package ron

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextModalSpellContinuationRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Blue Elemental Blast", "Red Elemental Blast", "Healing Salve"} {
			for mode := range 2 {
				t.Run(fmt.Sprintf("%s/%s/%d", set, name, mode), func(t *testing.T) {
					g, a, b := ronModalGame(mode)
					var target uuid.UUID
					color := core.Red
					if name == "Red Elemental Blast" {
						color = core.Blue
					}
					if name == "Healing Salve" {
						target = b.PlayerID()
					} else {
						victim := mage.NewCreature("Victim", "{R}", 2, 2)
						victim.SetColorOverride([]core.Color{color})
						if mode == 0 {
							target = castRONSpell(t, g, b, victim).ID
						} else {
							target = g.PutOnBattlefield(victim, b.PlayerID()).ID()
						}
					}
					card := repositoryCard(t, set, name)
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					a.ManaPool().Add(core.Blue, 1)
					a.ManaPool().Add(core.Red, 1)
					a.ManaPool().Add(core.White, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
						t.Fatal(err)
					}
					obj := g.StackPeek()
					paid := a.ManaPool().TotalMana()
					change, _ := mage.ReplaceColorWord(color, core.Green)
					if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
						t.Fatal(err)
					}
					if name != "Healing Salve" {
						change, _ = mage.ReplaceColorWord(core.Green, color)
						if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
							t.Fatal(err)
						}
					}
					current := g.StackPeek()
					if current.ModeChoice != mode || len(current.ModalTargets[1-mode]) != 0 || current.Targets[0] != target || a.ManaPool().TotalMana() != paid || current.SourceID != card.ID() {
						t.Fatal("revision changed announcement or payment")
					}
					a.mode = 1 - mode
					clone := g.Clone()
					for _, game := range []*mage.Game{g, clone} {
						game.ResolveTopOfStack()
						if name == "Healing Salve" {
							player := game.GetPlayer(b.PlayerID())
							if mode == 0 && player.Life() != 23 {
								t.Fatal("life mode lost player binding")
							}
							if mode == 1 {
								game.DealDamageToPlayer(player, 5, uuid.New())
								if player.Life() != 18 {
									t.Fatal("prevention mode lost recipient binding")
								}
							}
						} else if game.FindPermanent(target) != nil || game.GetStack().FindByID(target) != nil {
							t.Fatal("chosen mode lost its target binding")
						}
					}
					if a.prompts != 1 {
						t.Fatal("revision selected a new mode")
					}
					if len(a.Graveyard()) != 1 || a.Graveyard()[0].Name() != name || a.Graveyard()[0].ManaCost().String() != card.ManaCost().String() {
						t.Fatal("revision changed physical card")
					}
				})
			}
		}
	}
}

func TestRONTextModalSpellContinuationLegality(t *testing.T) {
	for _, state := range []string{"illegal", "legal", "return", "phase", "phase-cycle"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronModalGame(1)
			p := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
			card := repositoryCard(t, "LEA", "Blue Elemental Blast")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			before := g.Clone()
			id := g.StackPeek().ID
			change, _ := mage.ReplaceColorWord(core.Red, core.Green)
			if err := g.RevisePrintedSpellText(id, change); err != nil {
				t.Fatal(err)
			}
			if state != "illegal" {
				change, _ = mage.ReplaceColorWord(core.Green, core.Red)
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
			g.ResolveTopOfStack()
			want := state == "illegal" || state == "return" || state == "phase"
			if (g.FindPermanentIncludingPhased(p.ID()) != nil) != want {
				t.Fatal("revised target legality or exact visit failed")
			}
			before.ResolveTopOfStack()
			if before.FindPermanent(p.ID()) != nil {
				t.Fatal("revision changed prior clone")
			}
			if err := g.RevisePrintedSpellText(id, change); err == nil {
				t.Fatal("departed stack visit accepted")
			}
		})
	}
}

func TestRONTextModalSpellContinuationCopy(t *testing.T) {
	g, a, b := ronModalGame(1)
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	card := repositoryCard(t, "LEA", "Blue Elemental Blast")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{red.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.mode = 0
	a.targets = []uuid.UUID{green.ID()}
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copied == nil || copied.ModeChoice != 1 || copied.Targets[0] != green.ID() || copied.ModalTargets[1][0] != green.ID() || original.Targets[0] != red.ID() || a.prompts != 1 {
		t.Fatal("copy lost revised declarations or mode")
	}
	change, _ = mage.ReplaceColorWord(core.Green, core.Red)
	if err := g.RevisePrintedSpellText(original.ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(green.ID()) != nil || g.FindPermanent(red.ID()) == nil {
		t.Fatal("copy shared later original revision")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(red.ID()) != nil {
		t.Fatal("original lost its independent revision")
	}
}

func TestRONTextModalSpellContinuationResults(t *testing.T) {
	for mode := range 2 {
		g, a, _ := ronModalGame(mode)
		card := permanentScopeFactory(t, "Instant", `SpellActions(Modal_ChooseOne([ActionList([DrawACard,RevealTheCardDrawnThisWay,DiscardTheCardDrawnThisWay,GainLife(Integer(2))]),ActionList([DrawACard,RevealTheCardDrawnThisWay,DiscardTheCardDrawnThisWay,GainLife(Integer(4))])]))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.AddToLibrary(mage.NewLand("Drawn"))
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 3); err != nil {
			t.Fatal(err)
		}
		change, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
		if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
			t.Fatal(err)
		}
		if g.StackPeek().XValue != 3 {
			t.Fatal("revision changed X")
		}
		clone := g.Clone()
		for _, game := range []*mage.Game{g, clone} {
			game.ResolveTopOfStack()
			p := game.GetPlayer(a.PlayerID())
			if len(p.Hand()) != 0 || len(p.Graveyard()) != 2 || p.Life() != 22+2*mode {
				t.Fatal("mode-local results or clone frames failed")
			}
		}
	}
}

func TestRONTextModalSpellContinuationPartialLegality(t *testing.T) {
	g, a, b := ronModalGame(1)
	card := permanentScopeFactory(t, "Instant", `SpellActions(Modal_ChooseOne([ActionList([GainLife(Integer(1))]),Targeted([TargetPermanent(IsColor(Red)),TargetPermanent(IsColor(Green))],ActionList([TapPermanent(Ref_TargetPermanent1),TapPermanent(Ref_TargetPermanent2)]))]))`)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{red.ID(), green.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(red.ID()).Tapped || !g.FindPermanent(green.ID()).Tapped {
		t.Fatal("partly illegal mode targets did not resolve independently")
	}
}

func TestRONTextModalSpellContinuationLandWords(t *testing.T) {
	for mode := range 2 {
		g, a, b := ronModalGame(mode)
		card := permanentScopeFactory(t, "Instant", `SpellActions(Modal_ChooseOne([ActionList([DestroyEachPermanent(IsLandType(Plains))]),ActionList([TapEachPermanent(IsLandType(Plains))])]))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		plains := g.PutOnBattlefield(repositoryCard(t, "LEA", "Plains"), b.PlayerID())
		forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		change, _ := mage.ReplaceBasicLandWord(core.PlainsLandType, core.ForestLandType)
		if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
			t.Fatal(err)
		}
		g.ResolveTopOfStack()
		if g.FindPermanent(plains.ID()) == nil || g.FindPermanent(plains.ID()).Tapped {
			t.Fatal("printed collection still used")
		}
		if mode == 0 && g.FindPermanent(forest.ID()) != nil {
			t.Fatal("revised destruction collection lost land word")
		}
		if mode == 1 && !g.FindPermanent(forest.ID()).Tapped {
			t.Fatal("revised tap collection lost land word")
		}
	}
}

func TestRONTextModalSpellContinuationSpellLegality(t *testing.T) {
	g, a, b := ronModalGame(0)
	victim := castRONSpell(t, g, b, repositoryCard(t, "LEA", "Shivan Dragon"))
	card := repositoryCard(t, "LEA", "Blue Elemental Blast")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID}, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(victim.ID) == nil {
		t.Fatal("spell mode ignored revised legality")
	}
}
