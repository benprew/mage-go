package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachedEntry(t *testing.T) {
	for _, origin := range []string{"graveyard", "exile"} {
		for _, state := range []string{"legal", "shroud", "hexproof", "protection", "forbidden", "wrong-type", "phased", "phase-cycle", "returned-host", "returned-card", "failed-entry", "nonattachment", "equipment", "missing", "equipment-illegal", "equipment-missing", "nonaura-missing", "creature-aura"} {
			t.Run(origin+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), b.PlayerID())
				if state == "shroud" {
					host.GrantBaseAttr(core.Shroud)
				}
				if state == "hexproof" {
					host.GrantBaseAttr(core.Hexproof)
				}
				if state == "protection" {
					host = g.PutOnBattlefield(NewCreature("Protected", "", 2, 2, WithAbility(ProtectionFromColor(core.Black))), b.PlayerID())
				}
				if state == "forbidden" {
					host.GrantBaseAttr(core.AttrCantBeEnchanted)
				}
				if state == "wrong-type" || state == "equipment-illegal" {
					host = g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
				}
				if state == "phased" {
					g.PhaseOut(host.ID())
				}
				if state == "phase-cycle" {
					g.PhaseIn(g.PhaseOut(host.ID()))
				}
				hostBinding := BindPermanent("host")
				g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{hostBinding: {host.ID(), host.incarnationID}}
				card := NewAura("Guest", "{B}", WithCastTarget(TargetCreature()))
				if state == "nonattachment" || state == "nonaura-missing" {
					card = NewArtifact("Plain", "")
				}
				if state == "equipment" || state == "equipment-illegal" || state == "equipment-missing" {
					card = NewArtifact("Equipment", "", WithSubTypes("Equipment"))
				}
				if state == "returned-host" || state == "returned-card" {
					card.AddAbility(&opponentEntryProbe{check: func(g *Game, _ *Permanent) {
						if state == "returned-host" {
							g.RemoveFromBattlefield(host)
							g.PutOnBattlefield(host.Card, b.PlayerID())
						} else if origin == "graveyard" {
							c, _ := a.RemoveFromGraveyard(card.ID())
							a.AddToGraveyard(c)
						} else {
							c, _ := g.RemoveFromExile(card.ID())
							g.ExileCard(c, uuid.Nil)
						}
					}})
				}
				if state == "failed-entry" {
					card.AddAbility(&graveyardRejectEntry{})
				}
				if state == "creature-aura" {
					card.AddType(core.TypeCreature)
				}
				if state == "missing" || state == "equipment-missing" || state == "nonaura-missing" {
					g.RemoveFromBattlefield(host)
				}
				card.SetOwner(a.PlayerID())
				if origin == "graveyard" {
					a.AddToGraveyard(card)
				} else {
					g.ExileCard(card, uuid.Nil)
				}
				saved := BindCard("guest")
				g.resolution.cardBindings = map[*CardBinding]capturedCard{saved: {id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard}}
				var body = ReturnCardToBattlefield(FromCard(saved), You, EnterAttachedTo(From(hostBinding)))
				if origin == "exile" {
					body = PutExiledCardsOntoBattlefieldUnderOwnersControl(savedTransportVisits{{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile}}, EnterAttachedTo(From(hostBinding)))
				}
				observed := 0
				g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
					if e.SourceID == card.ID() && e.ToZone == core.ZoneBattlefield {
						observed++
						p := g.FindPermanent(card.ID())
						if p == nil {
							t.Error("entry event preceded registration")
						} else if state != "nonattachment" && state != "nonaura-missing" && state != "equipment-illegal" && state != "equipment-missing" && p.AttachedTo != host.ID() {
							t.Error("entry event preceded attachment")
						}
					}
					return false
				}))), a.PlayerID())
				clone := g.Clone()
				if err := body.Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				blocked := state == "protection" || state == "forbidden" || state == "wrong-type" || state == "phased" || state == "returned-host" || state == "returned-card" || state == "failed-entry" || state == "missing" || state == "creature-aura"
				p := g.FindPermanent(card.ID())
				if blocked {
					if p != nil || observed != 0 {
						t.Fatal("invalid entry succeeded")
					}
				} else {
					if p == nil || observed != 1 {
						t.Fatalf("entry failed: permanent=%v observations=%d", p, observed)
					}
					attached := state != "nonattachment" && state != "nonaura-missing" && state != "equipment-illegal" && state != "equipment-missing"
					if (p.AttachedTo == host.ID()) != attached {
						t.Fatal("incorrect entry attachment")
					}
					wantController := a.PlayerID()
					if origin == "graveyard" {
						wantController = b.PlayerID()
					}
					if p.ControllerID() != wantController || p.Card.Owner() != a.PlayerID() {
						t.Fatal("owner or controller changed")
					}
				}
				if origin == "graveyard" {
					if len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
						t.Fatal("clone graveyard changed")
					}
				} else if clone.FindExiledCard(card.ID()) == nil {
					t.Fatal("clone exile changed")
				}
			})
		}
	}
}

func TestAttachedEntryInvalid(t *testing.T) {
	for _, opts := range [][]BattlefieldEntryOption{{EnterAttachedTo(nil)}, {EnterAttachedTo(This), EnterAttachedTo(This)}, {nil}, {EnterAttachedTo(From(BindPermanent("unbound")))}} {
		target := DeclareGraveyardCardTarget("guest")
		body := ReturnCardToBattlefield(target, You, opts...)
		card := NewSorcery("Invalid", "", WithAbility(Spell(body, WithTargets(target))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid entry declaration accepted")
		}
		other := NewArtifact("Invalid exile", "", WithAbility(Activate(Costs(), PutExiledCardsOntoBattlefieldUnderOwnersControl(CardsExiledWith(NewExileLink(), This), opts...))))
		if err := ValidateCardDefinition(other.Name(), other); err == nil {
			t.Fatal("invalid exile entry declaration accepted")
		}
	}
}

func TestAttachedEntryCopyReplacements(t *testing.T) {
	for _, origin := range []string{"graveyard", "exile"} {
		for _, mode := range []string{"aura-to-artifact", "artifact-to-aura", "changed-enchant", "changed-color"} {
			t.Run(origin+"/"+mode, func(t *testing.T) {
				a := &entryCopyPlayer{BasePlayer: NewBasePlayer("A"), pick: func(c []*Permanent, _ *Game) *Permanent { return c[0] }}
				b := NewBasePlayer("B")
				g := NewGame(a, b)
				hostCard := NewCreature("Host", "", 2, 2)
				if mode == "changed-color" {
					hostCard.AddAbility(ProtectionFromColor(core.Black))
				}
				host := g.PutOnBattlefield(hostCard, b.PlayerID())
				donorPlan, err := NewCardDefinitionPlan(func() Card {
					if mode == "aura-to-artifact" {
						return NewArtifact("Donor", "")
					}
					target := TargetCreature()
					if mode == "changed-enchant" {
						return NewAura("Donor", "{W}", WithCastTarget(TargetLand()))
					}
					return NewAura("Donor", "{W}", WithCastTarget(target))
				})
				if err != nil {
					t.Fatal(err)
				}
				donor, err := donorPlan.Build()
				if err != nil {
					t.Fatal(err)
				}
				g.PutOnBattlefield(donor, b.PlayerID())
				guestPlan, err := NewCardDefinitionPlan(func() Card {
					if mode == "artifact-to-aura" {
						return NewArtifact("Guest", "", WithAbility(MayCopyAsEnters(Named("Donor"))))
					}
					return NewAura("Guest", "{B}", WithCastTarget(TargetCreature()), WithAbility(MayCopyAsEnters(Named("Donor"))))
				})
				if err != nil {
					t.Fatal(err)
				}
				guest, err := guestPlan.Build()
				if err != nil {
					t.Fatal(err)
				}
				guest.SetOwner(a.PlayerID())
				hostBinding := BindPermanent("host")
				g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{hostBinding: {host.ID(), host.incarnationID}}
				var body ResolvingEffect
				if origin == "graveyard" {
					a.AddToGraveyard(guest)
					saved := BindCard("guest")
					g.resolution.cardBindings = map[*CardBinding]capturedCard{saved: {id: guest.ID(), visit: g.graveyardVisit(guest.ID()), zone: core.ZoneGraveyard}}
					body = ReturnCardToBattlefield(FromCard(saved), You, EnterAttachedTo(From(hostBinding)))
				} else {
					g.ExileCard(guest, uuid.Nil)
					body = PutExiledCardsOntoBattlefieldUnderOwnersControl(savedTransportVisits{{id: guest.ID(), visit: g.FindExiledCard(guest.ID()).Visit, zone: core.ZoneExile}}, EnterAttachedTo(From(hostBinding)))
				}
				if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				p := g.FindPermanent(guest.ID())
				if mode == "changed-enchant" {
					if p != nil {
						t.Fatal("copied enchant restriction ignored")
					}
					return
				}
				if p == nil || p.Name() != "Donor" || (p.AttachedTo == host.ID()) != (mode != "aura-to-artifact") {
					t.Fatal("entry used original characteristics")
				}
			})
		}
	}
}

func TestAttachedEntryBatch(t *testing.T) {
	for _, mode := range []string{"normal", "host-changed-during-entry"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), b.PlayerID())
			saved := make(savedTransportVisits, 0, 2)
			ids := make([]uuid.UUID, 0, 2)
			for i := range 2 {
				card := NewAura("Aura", "", WithCastTarget(TargetCreature()))
				if i == 1 && mode != "normal" {
					card.AddAbility(&opponentEntryProbe{check: func(g *Game, _ *Permanent) {
						g.RemoveFromBattlefield(host)
						g.PutOnBattlefield(host.Card, b.PlayerID())
					}})
				}
				card.SetOwner(a.PlayerID())
				g.ExileCard(card, uuid.Nil)
				saved = append(saved, capturedCard{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile})
				ids = append(ids, card.ID())
			}
			observed := 0
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				if e.ToZone == core.ZoneBattlefield && (e.SourceID == ids[0] || e.SourceID == ids[1]) {
					observed++
					for _, id := range ids {
						p := g.FindPermanent(id)
						if p == nil || p.AttachedTo != host.ID() || !p.Tapped || g.FindExiledCard(id) != nil {
							t.Error("batch entry observed incomplete attachments")
						}
					}
				}
				return false
			}))), a.PlayerID())
			if err := ApplyEffect(g, PutExiledCardsOntoBattlefieldUnderOwnersControl(saved, EnterAttachedTo(This), EnterTapped()), host.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := 2
			if mode != "normal" {
				want = 0
			}
			if observed != want {
				t.Fatalf("observations=%d want=%d", observed, want)
			}
			if mode != "normal" {
				for _, id := range ids {
					if g.FindExiledCard(id) == nil {
						t.Fatal("stale host consumed Aura")
					}
				}
			}
		})
	}
}

func TestAttachedEntryWaiting(t *testing.T) {
	for _, depart := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), a.PlayerID())
		card := NewAura("Aura", "", WithCastTarget(TargetCreature()))
		card.SetOwner(a.PlayerID())
		g.ExileCard(card, uuid.Nil)
		saved := savedTransportVisits{{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile}}
		body := AtNext(EachEndStep, PutExiledCardsOntoBattlefieldUnderOwnersControl(saved, EnterAttachedTo(This)))
		if err := ApplyEffect(g, body, host.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		for range 2 {
			if len(g.WaitingDelayedPermanentSources()) != 1 || g.FindExiledCard(card.ID()) == nil {
				t.Fatal("waiting query lost host or moved card")
			}
		}
		clone := g.Clone()
		if depart {
			g.RemoveFromBattlefield(host)
			g.PutOnBattlefield(host.Card, a.PlayerID())
		}
		for _, game := range []*Game{g, clone} {
			game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			game.PutTriggersOnStack()
			game.ResolveStack()
		}
		if clone.FindPermanent(card.ID()) == nil {
			t.Fatal("clone lost saved host")
		}
		if (g.FindPermanent(card.ID()) != nil) == depart {
			t.Fatal("entry followed wrong host visit")
		}
	}
}
