package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type handEntryPlayer struct {
	*BasePlayer
	accept          bool
	pick            Card
	offers, choices int
	change          func()
}

func (p *handEntryPlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }
func (p *handEntryPlayer) ChooseCardFromHand(cards []Card, _ string, _ GameReader) Card {
	p.choices++
	if p.change != nil {
		p.change()
	}
	if p.pick != nil {
		return p.pick
	}
	if len(cards) > 0 {
		return cards[0]
	}
	return nil
}

func TestOptionalHandEntryChoices(t *testing.T) {
	for _, scenario := range []string{"accept", "decline", "empty", "foreign", "returned", "removed", "prohibited", "tapped"} {
		t.Run(scenario, func(t *testing.T) {
			a := &handEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: scenario != "decline"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			land := NewLand("Land")
			land.SetOwner(a.PlayerID())
			if scenario == "tapped" {
				land = NewLand("Land", WithKeyword(core.EntersTapped))
				land.SetOwner(a.PlayerID())
			}
			if scenario != "empty" {
				a.AddToHand(land)
			}
			bad := NewInstant("Bad", "")
			bad.SetOwner(a.PlayerID())
			a.AddToHand(bad)
			if scenario == "foreign" {
				a.pick = bad
			}
			if scenario == "returned" {
				a.change = func() { a.RemoveFromHand(land.ID()); a.AddToHand(land) }
			}
			if scenario == "removed" {
				a.change = func() { a.RemoveFromHand(land.ID()); a.AddToGraveyard(land) }
			}
			visit := a.handVisit(land.ID())
			if scenario == "prohibited" {
				g.PutOnBattlefield(NewEnchantment("Prohibition", "", WithStaticAbility(WormsOfTheEarthEffect())), b.PlayerID())
			}
			effect := MayPutFromHandOntoBattlefield(You, IsPermanentCard)
			card := NewSorcery("Declaration", "", WithAbility(Spell(effect)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil)
			p := g.FindPermanent(land.ID())
			entered := scenario == "accept" || scenario == "tapped"
			if (p != nil) != entered {
				t.Fatal("wrong entry", p)
			}
			if g.FindPermanent(bad.ID()) != nil || len(b.Hand()) != 0 || len(g.PublicReveals()) != 0 {
				t.Fatal("invalid card or hidden information")
			}
			if scenario == "empty" && (a.offers != 0 || a.choices != 0) {
				t.Fatal("empty choice")
			}
			if scenario == "decline" && a.choices != 0 {
				t.Fatal("choice after decline")
			}
			if scenario == "prohibited" && (len(a.Hand()) != 2 || a.handVisit(land.ID()) != visit) {
				t.Fatal("blocked entry changed hand visit")
			}
			if entered && (p.ControllerID() != a.PlayerID() || p.Card.Owner() != a.PlayerID() || p.Tapped != (scenario == "tapped")) {
				t.Fatal("normal entry state")
			}
		})
	}
}

func TestOptionalHandEntryDeclarationAndClone(t *testing.T) {
	bad := NewSorcery("Bad", "", WithAbility(Spell(MayPutFromHandOntoBattlefield(nil, IsPermanentCard))))
	if ValidateCardDefinition(bad.Name(), bad) == nil {
		t.Fatal("nil player accepted")
	}
	a := &handEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land")
	land.SetOwner(a.PlayerID())
	a.AddToHand(land)
	clone := g.Clone()
	ApplyEffect(g, MayPutFromHandOntoBattlefield(You, IsPermanentCard), land.ID(), a.PlayerID(), nil)
	if clone.FindPermanent(land.ID()) != nil || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
		t.Fatal("clone shared hand movement")
	}
}

func TestOptionalHandEntryAura(t *testing.T) {
	for _, scenario := range []string{"legal", "shroud", "no-host", "phased", "protected"} {
		t.Run(scenario, func(t *testing.T) {
			a := &handEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			aura := NewBoostAura("Aura", "{W}", 1, 1)
			aura.SetOwner(a.PlayerID())
			a.AddToHand(aura)
			var host *Permanent
			if scenario != "no-host" {
				options := []CardOption{}
				if scenario == "protected" {
					options = append(options, WithAbility(ProtectionFromColor(core.White)))
				}
				host = g.PutOnBattlefield(NewCreature("Host", "", 2, 2, options...), b.PlayerID())
			}
			switch scenario {
			case "shroud":
				host.GrantBaseAttr(core.Shroud)
			case "phased":
				g.PhaseOut(host.ID())
			}
			ApplyEffect(g, MayPutFromHandOntoBattlefield(You, IsPermanentCard), aura.ID(), a.PlayerID(), nil)
			entered := scenario == "legal" || scenario == "shroud"
			p := g.FindPermanent(aura.ID())
			if (p != nil) != entered {
				t.Fatal("wrong Aura entry")
			}
			if entered && (p.AttachedTo != host.ID() || g.FindPermanent(host.ID()).CurrentPower(g) != 3) {
				t.Fatal("Aura host or static effect")
			}
			if !entered && len(a.Hand()) != 1 {
				t.Fatal("Aura left hand without legal host")
			}
		})
	}
}

type invalidHandEntryHostPlayer struct{ *handEntryPlayer }

func (*invalidHandEntryHostPlayer) ChoosePermanent([]*Permanent, string, GameReader) *Permanent {
	return nil
}

func TestOptionalHandEntryAuraRequiredHost(t *testing.T) {
	a := &invalidHandEntryHostPlayer{&handEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}}
	g := NewGame(a, NewBasePlayer("B"))
	aura := NewBoostAura("Aura", "{W}", 1, 1)
	aura.SetOwner(a.PlayerID())
	a.AddToHand(aura)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), a.PlayerID())
	ApplyEffect(g, MayPutFromHandOntoBattlefield(You, IsPermanentCard), aura.ID(), a.PlayerID(), nil)
	if p := g.FindPermanent(aura.ID()); p == nil || p.AttachedTo != host.ID() {
		t.Fatal("required Aura host choice did not use legal fallback")
	}
}
