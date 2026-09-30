package mage

import (
	"maps"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func placeAlternateDivisionCard(g *Game, owner Player, card Card, zone Zone) {
	card.SetOwner(owner.PlayerID())
	switch zone {
	case ZoneHand:
		owner.AddToHand(card)
	case ZoneGraveyard:
		owner.AddToGraveyard(card)
	case ZoneLibrary:
		owner.SetLibrary([]Card{card})
	case ZoneExile:
		g.ExileCard(card, uuid.Nil)
	}
}

func TestAlternateDamageDivisionBeforePayment(t *testing.T) {
	for _, zone := range []Zone{ZoneHand, ZoneGraveyard, ZoneLibrary, ZoneExile} {
		for _, route := range []string{"free", "supplied", "registered", "flashback", "permission", "foreign free"} {
			if (route == "flashback" && zone != ZoneGraveyard) || ((route == "permission" || route == "foreign free") && zone != ZoneExile) {
				continue
			}
			t.Run(zone.String()+"/"+route, func(t *testing.T) {
				g, a, b := newActivationDivisionGame()
				a.SetLife(4)
				land := g.PutOnBattlefield(NewLand("Mana source", WithManaAbility(Red)), a.PlayerID())
				card := NewInstant("Alternate division", "{R}", NewMultiTargetSpell([]Target{TargetUpToNCreaturesOrPlayers(2)}, DealDividedDamage(activationDivisionLife{})), WithAdditionalCost(LifePayCost(2)))
				if route == "registered" {
					WithAlternateCost(zone, ParseManaCost("{R}"))(card)
				}
				if route == "flashback" {
					WithFlashback(ParseManaCost("{R}"))(card)
				}
				owner := Player(a)
				if route == "permission" || route == "foreign free" {
					owner = b
				}
				placeAlternateDivisionCard(g, owner, card, zone)
				response := map[uuid.UUID]int{b.PlayerID(): 4}
				a.choose = func(_ []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
					if total != 4 || a.Life() != 4 || land.Tapped || g.findCardInZoneOpt(a.PlayerID(), card.ID(), zone, true) == nil {
						t.Errorf("allocation after payment: total=%d life=%d tapped=%v", total, a.Life(), land.Tapped)
					}
					return response
				}
				targets := []uuid.UUID{b.PlayerID()}
				var err error
				switch route {
				case "free":
					err = g.CastCardFromZoneWithoutPaying(a.PlayerID(), card.ID(), zone, targets, 0)
				case "supplied":
					err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), card.ID(), zone, ParseManaCost("{R}"), targets, 0)
				case "registered", "flashback":
					err = g.CastCardWithAlternateCost(a.PlayerID(), card.ID(), 0, targets, 0)
				case "permission":
					g.GrantCastFromExile(a.PlayerID(), card.ID(), true)
					err = g.CastExiledCardWithPermission(a.PlayerID(), card.ID(), targets, 0)
				case "foreign free":
					err = g.CastCardFromExileWithoutPaying(a.PlayerID(), card.ID(), targets, 0)
				}
				if err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				if a.divisions != 1 || !maps.Equal(obj.DamageDistribution, response) || a.Life() != 2 || obj.Controller != a.PlayerID() || obj.Card.Owner() != owner.PlayerID() || obj.CastZone != zone {
					t.Fatal("cast did not preserve announced allocation or identities")
				}
				response[b.PlayerID()] = 99
				clone := g.Clone()
				copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
				copied.DamageDistribution[b.PlayerID()] = 7
				if obj.DamageDistribution[b.PlayerID()] != 4 || clone.StackObjects()[0].DamageDistribution[b.PlayerID()] != 4 {
					t.Fatal("allocation maps share state")
				}
				g.ResolveStack()
				if b.Life() != 16 {
					t.Fatal("payment changed damage")
				}
				if route == "flashback" && g.findCardInZone(a.PlayerID(), card.ID(), ZoneExile) == nil {
					t.Fatal("flashback destination changed")
				}
			})
		}
	}
}

func TestAlternateDamageDivisionRejectedBeforePayment(t *testing.T) {
	for _, state := range []string{"negative", "zero", "insufficient", "absent", "phased", "duplicate", "wrong type"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			a.ManaPool().Add(Red, 1)
			total := 3
			switch state {
			case "negative":
				total = -1
			case "zero":
				total = 0
			case "insufficient":
				total = 1
			}
			card := NewInstant("Rejected division", "{R}", NewMultiTargetSpell([]Target{TargetUpToNCreaturesOrPlayers(2)}, DealDividedDamage(Fixed(total))), WithAdditionalCost(LifePayCost(2)))
			placeAlternateDivisionCard(g, b, card, ZoneExile)
			g.GrantCastFromExile(a.PlayerID(), card.ID(), true)
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
			targets := []uuid.UUID{a.PlayerID(), b.PlayerID()}
			switch state {
			case "absent":
				targets = []uuid.UUID{uuid.New()}
			case "phased":
				g.PhaseOut(victim.ID())
				targets = []uuid.UUID{victim.ID()}
			case "duplicate":
				targets = []uuid.UUID{victim.ID(), victim.ID()}
			case "wrong type":
				targets = []uuid.UUID{g.PutOnBattlefield(NewArtifact("Other", ""), b.PlayerID()).ID()}
			}
			events := 0
			for _, event := range []EventType{EvtLifeLost, EvtSpellCast, EvtBecomesTarget} {
				g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(event, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, _ GameReader, _, _ uuid.UUID) bool { events++; return false }))), a.PlayerID())
			}
			a.choose = func(ids []uuid.UUID, amount int, _ *Game) map[uuid.UUID]int {
				return sanitizeDamageDistribution(nil, ids, amount)
			}
			if err := g.CastExiledCardWithPermission(a.PlayerID(), card.ID(), targets, 0); err == nil {
				t.Fatal("accepted invalid announcement")
			}
			if a.divisions != 0 || a.Life() != 20 || a.ManaPool().Count(Red) != 1 || g.StackSize() != 0 || events != 0 || g.findCardInZone(b.PlayerID(), card.ID(), ZoneExile) == nil || g.CastFromExilePermissionFor(a.PlayerID(), card.ID()) == nil {
				t.Fatal("rejected cast prompted, paid, or lost permission")
			}
		})
	}
}

func TestAlternateDamageDivisionFailedPayment(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	a.ManaPool().Add(Red, 1)
	card := NewInstant("Failed division", "{R}", NewMultiTargetSpell([]Target{TargetUpToNCreaturesOrPlayers(2)}, DealDividedDamage(Fixed(3))), WithAdditionalCost(LifePayCost(11)), WithAdditionalCost(LifePayCost(11)))
	placeAlternateDivisionCard(g, a, card, ZoneGraveyard)
	a.choose = func(_ []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{b.PlayerID(): total}
	}
	if err := g.CastCardFromZoneWithAlternateCost(a.PlayerID(), card.ID(), ZoneGraveyard, ParseManaCost("{R}"), []uuid.UUID{b.PlayerID()}, 0); err == nil {
		t.Fatal("paid impossible total cost")
	}
	if a.divisions != 1 || a.Life() != 20 || a.ManaPool().Count(Red) != 1 || g.StackSize() != 0 || g.findCardInZone(a.PlayerID(), card.ID(), ZoneGraveyard) == nil {
		t.Fatal("failed payment skipped announcement or changed state")
	}
}

func TestAlternateDamageDivisionWithoutTargets(t *testing.T) {
	for _, total := range []int{-1, 0, 3} {
		g, a, _ := newActivationDivisionGame()
		card := NewInstant("Empty division", "", NewMultiTargetSpell([]Target{TargetUpToNCreaturesOrPlayers(2)}, DealDividedDamage(Fixed(total))))
		placeAlternateDivisionCard(g, a, card, ZoneLibrary)
		if err := g.CastCardFromZoneWithoutPaying(a.PlayerID(), card.ID(), ZoneLibrary, nil, 0); err != nil {
			t.Fatal(err)
		}
		if a.divisions != 0 || len(g.StackPeek().DamageDistribution) != 0 {
			t.Fatal("empty allocation prompted")
		}
	}
}

func TestAlternateDamageDivisionSelectedMode(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	a.ManaPool().Add(Red, 1)
	card := NewInstant("Modal alternate division", "", WithAbility(NewModalSpell([]Mode{
		{Label: "Divide", Targets: []Target{TargetUpToNCreaturesOrPlayers(1)}, Effects: []Effect{DealDividedDamage(Fixed(3))}},
		{Label: "Other", Effects: []Effect{DealDividedDamage(Fixed(-1))}},
	})))
	placeAlternateDivisionCard(g, a, card, ZoneGraveyard)
	a.choose = func(_ []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
		if total != 3 || a.ManaPool().Count(Red) != 1 {
			t.Error("mode allocation after payment or from inactive mode")
		}
		return map[uuid.UUID]int{b.PlayerID(): total}
	}
	if err := g.CastCardFromZoneWithAlternateCost(a.PlayerID(), card.ID(), ZoneGraveyard, ParseManaCost("{R}"), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	if a.divisions != 1 || g.StackPeek().ModeChoice != 0 || g.StackPeek().DamageDistribution[b.PlayerID()] != 3 {
		t.Fatal("wrong mode allocation")
	}
}
