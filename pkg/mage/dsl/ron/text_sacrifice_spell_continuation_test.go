package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextSacrificeSpellContinuationRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI", "ARN"} {
		name, color := "Sacrifice", core.Black
		if set == "ARN" {
			name, color = "Metamorphosis", core.Green
		}
		for _, cost := range []string{"", "{X}{3}{U}"} {
			t.Run(set+"/"+name+"/"+cost, func(t *testing.T) {
				a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
				b := mage.NewBasePlayer("Bob")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				spell := repositoryCard(t, set, name)
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				paid := mage.NewCreature("Payment", cost, 1, 1, mage.WithKeyword(core.Indestructible))
				paid.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(paid, a.PlayerID())
				a.choice = p.ID()
				a.ManaPool().Add(color, 1)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				id := g.StackPeek().ID
				if a.ManaPool().TotalMana() != 0 || len(b.Graveyard()) != 1 {
					t.Fatal("payment was not retained")
				}
				b.RemoveFromGraveyard(paid.ID())
				returned := g.PutOnBattlefield(paid, b.PlayerID())
				g.PhaseOut(returned.ID())
				before := g.Clone()
				change, _ := mage.ReplaceColorWord(core.Black, core.Red)
				if err := g.RevisePrintedSpellText(id, change); err != nil {
					t.Fatal(err)
				}
				copyObj := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
				if copyObj == nil {
					t.Fatal("copy failed")
				}
				change, _ = mage.ReplaceColorWord(core.Red, core.Blue)
				if err := g.RevisePrintedSpellText(copyObj.ID, change); err != nil {
					t.Fatal(err)
				}
				if err := g.RevisePrintedSpellText(id, change); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				want := 0
				if cost != "" {
					want = 4
				}
				if name == "Metamorphosis" {
					want++
				}
				for _, game := range []*mage.Game{g, clone} {
					for _, player := range []mage.Player{game.GetPlayer(a.PlayerID()), game.GetPlayer(b.PlayerID())} {
						pool := player.ManaPool()
						if pool.TotalMana() != want {
							t.Fatalf("saved payment got %d, want %d", pool.TotalMana(), want)
						}
						if name == "Sacrifice" && pool.Count(core.Black) != want {
							t.Fatal("mana symbol changed with a color word")
						}
						if name == "Metamorphosis" && want > 0 {
							manaCost := core.ManaCost{Generic: want}
							creature := core.SpellPaymentContext{IsCreature: true}
							noncreature := core.SpellPaymentContext{}
							if !pool.CanPay(manaCost, &creature) || pool.CanPay(manaCost, &noncreature) || pool.CanPay(manaCost, nil) {
								t.Fatal("restricted mana lost its rule")
							}
						}
					}
					if game.FindPermanentIncludingPhased(returned.ID()) == nil || len(game.GetPlayer(b.PlayerID()).Graveyard()) != 0 {
						t.Fatal("revision paid the sacrifice again")
					}
					if err := game.RevisePrintedSpellText(id, change); err == nil {
						t.Fatal("departed visit accepted")
					}
				}
				before.ResolveStack()
				if before.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != want || before.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 {
					t.Fatal("revision changed an earlier clone")
				}
			})
		}
	}
}

func TestRONTextSacrificeSpellContinuationFreshResults(t *testing.T) {
	g, a, _ := ronGame()
	spell := permanentScopeFactory(t, "Instant", `CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))),SpellActions(ActionList([DrawACard,RevealTheCardDrawnThisWay,DiscardTheCardDrawnThisWay,GainLife(ManaValueOfTheSacrificedPermanent)]))`)()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	g.PutOnBattlefield(mage.NewCreature("Payment", "{3}", 1, 1), a.PlayerID())
	a.AddToLibrary(mage.NewLand("Drawn"))
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
		if player.Life() != 23 || len(player.Hand()) != 0 || len(player.Graveyard()) != 3 {
			t.Fatal("payment and fresh resolution bindings were mixed")
		}
	}
}

func TestRONTextSacrificeSpellContinuationLegality(t *testing.T) {
	for _, state := range []string{"legal", "illegal", "return", "phase", "phase-cycle"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			spell := permanentScopeFactory(t, "Instant", `CastEffect(AdditionalCastingCost(SacrificeAPermanent(And([IsCardtype(Creature),IsColor(Red)])))),SpellActions(Targeted([TargetPermanent(IsColor(Red))],ActionList([TapPermanent(Ref_TargetPermanent),GainLife(ManaValueOfTheSacrificedPermanent)])))`)()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			g.PutOnBattlefield(mage.NewCreature("Payment", "{2}{R}", 1, 1), a.PlayerID())
			target := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			change, _ := mage.ReplaceColorWord(core.Red, core.Green)
			if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
				t.Fatal(err)
			}
			if state != "illegal" {
				change, _ = mage.ReplaceColorWord(core.Green, core.Red)
				if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
					t.Fatal(err)
				}
			}
			switch state {
			case "return":
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(target.Card, b.PlayerID())
			case "phase", "phase-cycle":
				g.PhaseOut(target.ID())
				if state == "phase-cycle" {
					g.PhaseIn([]uuid.UUID{target.ID()})
				}
			}
			g.ResolveStack()
			want := 20
			if state == "legal" || state == "phase-cycle" {
				want = 23
			}
			if a.Life() != want {
				t.Fatalf("target legality lost: got life %d, want %d", a.Life(), want)
			}
		})
	}
}

func TestRONTextSacrificeSpellContinuationInstances(t *testing.T) {
	a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
	b := mage.NewBasePlayer("Bob")
	g := mage.NewGame(a, b)
	factory, err := Compile(staticBaseRecord(t, "LEA", "Sacrifice"))
	if err != nil {
		t.Fatal(err)
	}
	for _, cost := range []string{"{2}", "{5}"} {
		spell := factory()
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		p := g.PutOnBattlefield(mage.NewCreature("Payment", cost, 1, 1), a.PlayerID())
		a.choice = p.ID()
		a.ManaPool().Add(core.Black, 1)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		id := g.StackPeek().ID
		change, _ := mage.ReplaceColorWord(core.Red, core.Green)
		if err := g.RevisePrintedSpellText(id, change); err != nil {
			t.Fatal(err)
		}
		if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).ManaPool().Count(core.Black) != 7 || game.GetPlayer(b.PlayerID()).ManaPool().Count(core.Black) != 7 {
			t.Fatal("independent payments were overwritten")
		}
	}
}
