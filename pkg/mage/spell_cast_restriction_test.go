package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastRestrictionLifecycle(t *testing.T) {
	for _, active := range []int{0, 1} {
		for _, state := range []string{"normal", "tapped", "face-down", "control", "phased", "cycle", "removed", "restored", "departed", "returned", "clone"} {
			t.Run(string(rune('A'+active))+"/"+state, func(t *testing.T) {
				g, a, b := newSpellCopyGame()
				g.SetStep(core.PrecombatMain)
				g.SetActivePlayerIndex(active)
				filter := NewCardFilter("blocked name", func(c Card) bool { return c.Name() == "Blocked" })
				card := NewArtifact("Restriction", "", WithAbility(Static(PreventSpellCasts(filter))))
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				blocked := true
				switch state {
				case "face-down":
					g.MutablePermanent(source.ID()).FaceDown = true
					g.ApplyContinuousEffects()
					blocked = false
				case "tapped":
					g.TapPermanent(source)
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "phased":
					g.PhaseOut(source.ID())
					blocked = false
				case "cycle":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "removed", "restored":
					if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					g.ApplyContinuousEffects()
					blocked = false
					if state == "restored" {
						g.GetEffects().RemoveEndOfTurn()
						g.ApplyContinuousEffects()
						blocked = true
					}
				case "departed", "returned":
					g.BouncePermanentToHand(source)
					blocked = false
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, a.PlayerID())
						blocked = true
					}
				case "clone":
					clone := g.Clone()
					clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
					if clone.IsSpellCastProhibited(a.PlayerID(), NewInstant("Blocked", "")) {
						t.Fatal("clone retained departed restriction")
					}
				}
				player := g.ActivePlayerObj()
				spell := NewInstant("Blocked", "")
				player.AddToHand(spell)
				handCount := len(player.Hand())
				for range 2 {
					if g.IsSpellCastProhibited(player.PlayerID(), spell) != blocked {
						t.Fatal("wrong restriction query")
					}
					if (len(g.GetCastableSpells(player.PlayerID())) == 0) != blocked {
						t.Fatal("wrong castable spell list")
					}
				}
				if g.GetLandsPlayedThisTurn() != 0 || len(player.Hand()) != handCount {
					t.Fatal("query changed state")
				}
				err := g.CastSpellByID(player.PlayerID(), spell.ID(), nil, 0)
				if (err != nil) != blocked {
					t.Fatalf("cast error %v, blocked %v", err, blocked)
				}
				if blocked && (len(player.Hand()) != handCount || g.GetLandsPlayedThisTurn() != 0 || g.FindPermanent(spell.ID()) != nil) {
					t.Fatal("rejected cast changed state")
				}
				other := NewInstant("Allowed", "")
				player.AddToHand(other)
				if err := g.CastSpellByID(player.PlayerID(), other.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				direct := g.PutOnBattlefield(NewArtifact("Blocked", ""), player.PlayerID())
				if direct == nil {
					t.Fatal("casting rule prevented direct entry")
				}
			})
		}
	}
}

func TestSpellCastRestrictionPathsAndRollback(t *testing.T) {
	for _, path := range []string{"hand", "free", "alternate", "graveyard", "copy cast"} {
		t.Run(path, func(t *testing.T) {
			g, a, _ := newSpellCopyGame()
			g.SetStep(core.PrecombatMain)
			card := NewInstant("Blocked", "{U}", WithAbility(Spell(GainLifeFor(You, Fixed(2)))))
			card.SetOwner(a.PlayerID())
			first, last := NewInstant("First", ""), NewInstant("Last", "")
			a.AddToHand(first)
			a.AddToHand(card)
			a.AddToHand(last)
			a.ManaPool().Add(core.Blue, 1)
			g.PutOnBattlefield(NewArtifact("Restriction", "", WithAbility(Static(PreventSpellCasts(NewCardFilter("blocked", func(c Card) bool { return c.Name() == "Blocked" }))))), a.PlayerID())
			zone := core.ZoneHand
			if path == "graveyard" {
				a.RemoveFromHand(card.ID())
				a.AddToGraveyard(card)
				zone = core.ZoneGraveyard
			}
			hand := append([]Card(nil), a.Hand()...)
			var err error
			switch path {
			case "hand":
				err = g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
			case "free", "graveyard":
				err = g.CastCardFromZoneWithoutPaying(a.PlayerID(), card.ID(), zone, nil, 0)
			case "alternate":
				err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), card.ID(), zone, card.ManaCost(), nil, 0)
			case "copy cast":
				_, err = g.pushCastSpellObject(castStackObjectOptions{Card: card.Copy(), Controller: a.PlayerID(), IsCopy: true})
			}
			if err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 || a.Life() != 20 || len(a.Hand()) != len(hand) {
				t.Fatal("prohibited cast changed state", err)
			}
			for i, before := range hand {
				if a.Hand()[i].ID() != before.ID() {
					t.Fatal("prohibited cast changed hand order")
				}
			}
			if path == "graveyard" && (len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != card.ID()) {
				t.Fatal("prohibited cast changed graveyard")
			}
		})
	}
}

func TestSpellCastRestrictionStackCopyAndOverlap(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	card := NewInstant("Blocked", "", WithAbility(Spell(GainLifeFor(You, Fixed(2)))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	factory := func() Card {
		return NewArtifact("Restriction", "", WithAbility(Static(PreventSpellCasts(NewCardFilter("all", func(Card) bool { return true })))))
	}
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), a.PlayerID())
	clone := g.Clone()
	g.RemoveFromBattlefield(first)
	if !g.IsSpellCastProhibited(a.PlayerID(), card) || g.CopySpellOnStack(card.ID(), a.PlayerID(), false) == nil {
		t.Fatal("overlap or stack copying failed")
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("restriction stopped pending resolution")
	}
	g.RemoveFromBattlefield(second)
	if g.IsSpellCastProhibited(a.PlayerID(), card) || !clone.IsSpellCastProhibited(a.PlayerID(), card) {
		t.Fatal("cloned source lifetimes are shared")
	}
	if g.IsSpellCastProhibited(a.PlayerID(), nil) || clone.IsSpellCastProhibited(a.PlayerID(), NewLand("Blocked")) {
		t.Fatal("nonspell query was prohibited")
	}
	card.SetID(uuid.New())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
}

func TestSpellCastRestrictionInvalid(t *testing.T) {
	card := NewArtifact("Bad", "", WithAbility(Static(PreventSpellCasts(CardFilter{}))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("empty filter accepted")
	}
}
