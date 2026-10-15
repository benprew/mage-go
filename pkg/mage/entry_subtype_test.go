package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEntryWithoutSubtype(t *testing.T) {
	for _, origin := range []string{"graveyard", "exile"} {
		for _, state := range []string{"normal", "cancel", "stale", "phase", "return", "copy-entry", "filtered"} {
			t.Run(origin+"/"+state, func(t *testing.T) {
				a, b := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}, &optionalEntryPlayer{BasePlayer: NewBasePlayer("B"), accept: true}
				g := NewGame(a, b)
				plan, err := NewCardDefinitionPlan(func() Card {
					card := NewAura("Guest", "", WithCastTarget(TargetCreature()), WithSubTypes("Curse"))
					if state == "copy-entry" {
						card.AddAbility(MayCopyAsEnters(IsEnchantment))
					}
					return card
				})
				if err != nil {
					t.Fatal(err)
				}
				card, err := plan.Build()
				if err != nil {
					t.Fatal(err)
				}
				card.SetOwner(b.PlayerID())
				if state == "copy-entry" {
					modelPlan, err := NewCardDefinitionPlan(func() Card { return NewAura("Model", "", WithCastTarget(TargetCreature()), WithSubTypes("Curse")) })
					if err != nil {
						t.Fatal(err)
					}
					model, err := modelPlan.Build()
					if err != nil {
						t.Fatal(err)
					}
					g.PutOnBattlefield(model, a.PlayerID())
				}

				if state == "filtered" {
					g.PutOnBattlefield(NewArtifact("Entry rule", "", WithAbility(EntersTappedFor(EachPlayer, And(IsEnchantment, Not(HasSubType("Aura")))))), a.PlayerID())
				}
				var body ResolvingEffect
				if origin == "graveyard" {
					b.AddToGraveyard(card)
					saved := BindCard("guest")
					g.resolution.cardBindings = map[*CardBinding]capturedCard{saved: {id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard}}
					body = ReturnCardToBattlefield(FromCard(saved), You, EnterWithoutSubtype(SubtypeEnchantment, "Aura"))
				} else {
					g.ExileCard(card, uuid.Nil)
					body = PutExiledCardsOntoBattlefieldUnderOwnersControl(savedTransportVisits{{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile}}, EnterWithoutSubtype(SubtypeEnchantment, "Aura"))
				}
				seen := 0
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(action Action) bool {
					entry, ok := action.(*battlefieldEntryAction)
					return ok && entry.permanent.ID() == card.ID()
				}, replaceFunc: func(action Action, g *Game) Action {
					entry := action.(*battlefieldEntryAction)
					seen++
					if entry.context.entry.withoutSubtype != nil && (entry.permanent.HasSubType("Aura") || !entry.permanent.HasSubType("Curse")) {
						t.Error("entry replacement read wrong subtypes")
					}
					if state == "cancel" {
						return nil
					}
					if state == "stale" {
						if origin == "graveyard" {
							owner := g.GetPlayer(b.PlayerID())
							c, _ := owner.RemoveFromGraveyard(card.ID())
							owner.AddToGraveyard(c)
						} else {
							c, _ := g.RemoveFromExile(card.ID())
							g.ExileCard(c, uuid.Nil)
						}
					}
					return action
				}})
				clone := g.Clone()
				for _, game := range []*Game{g, clone} {
					before := len(game.effects.effects)
					if err := body.Apply(&EffectContext{Game: game, Controller: a.PlayerID()}); err != nil {
						t.Fatal(err)
					}
					p := game.FindPermanent(card.ID())
					if state == "cancel" || state == "stale" {
						if len(game.effects.effects) != before {
							t.Fatal("failed entry retained subtype effect")
						}
						if p != nil {
							t.Fatal("failed entry moved card")
						}
						continue
					}
					if p == nil || p.HasSubType("Aura") || !p.HasSubType("Curse") || !p.HasType(core.TypeEnchantment) || !p.Card.HasSubType("Aura") {
						t.Fatal("entry changed printed definition or lost subtype effect")
					}
					if state == "filtered" && !p.Tapped {
						t.Fatal("entry projection did not apply subtype removal")
					}
					if state == "copy-entry" && game == g && p.Name() != "Model" {
						t.Fatal("entry did not copy model")
					}
					values, err := game.CaptureCopiableValues(p)
					if err != nil {
						t.Fatal(err)
					}
					rebuilt, err := values.Build()
					if err != nil || !rebuilt.HasSubType("Aura") {
						t.Fatal("entry removal changed copiable values")
					}
					if state == "phase" {
						game.PhaseIn(game.PhaseOut(p.ID()))
					}
					if state == "return" {
						game.BouncePermanentToHand(p)
						owner := game.GetPlayer(b.PlayerID())
						c, _ := owner.RemoveFromHand(card.ID())
						game.PutOnBattlefield(c, a.PlayerID())
					}
					game.ApplyContinuousEffects()
					p = game.FindPermanent(card.ID())
					if p.HasSubType("Aura") != (state == "return") {
						t.Fatal("entry subtype effect followed wrong visit")
					}
				}
				if seen < 2 {
					t.Fatal("entry replacement did not run")
				}
			})
		}
	}
}

func TestEntryWithoutSubtypeInvalid(t *testing.T) {
	for _, options := range [][]BattlefieldEntryOption{
		{EnterWithoutSubtype(SubtypeEnchantment, "Forest")},
		{EnterWithoutSubtype(SubtypeEnchantment, "")},
		{EnterWithoutSubtype(SubtypeEnchantment, "Aura"), EnterWithoutSubtype(SubtypeEnchantment, "Aura")},
		{EnterAsNonAuraEnchantment(), EnterAsNonAuraEnchantment()},
		{EnterWithoutSubtype(SubtypeEnchantment, "Aura"), EnterAsNonAuraEnchantment()},
	} {
		target := DeclareGraveyardCardTarget("guest")
		card := NewSorcery("Invalid", "", WithAbility(Spell(ReturnCardToBattlefield(target, You, options...), WithTargets(target))))
		if err := ValidateCardDefinition("Invalid", card); err == nil {
			t.Fatal("invalid subtype entry accepted")
		}
	}
}

func TestEntryWithoutSubtypeLayers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewAura("Guest", "", WithSubTypes("Curse"))
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("guest")
	_, obj := graveyardTestSpell(a.PlayerID(), target, ReturnCardToBattlefield(target, You, EnterWithoutSubtype(SubtypeEnchantment, "Aura")))
	obj.Targets = []uuid.UUID{card.ID()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.HasSubType("Aura") {
		t.Fatal("non-Aura entry failed")
	}
	clone := g.Clone()
	g.AddContinuousEffect(AddSubtypes(p.ID(), SubtypeEnchantment, core.Indefinite, "Aura"))
	if !g.FindPermanent(card.ID()).HasSubType("Aura") {
		t.Fatal("later subtype addition lost its timestamp")
	}
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(card.ID()).HasSubType("Aura") {
		t.Fatal("clone lost entry removal or shared later layers")
	}
	g.AddContinuousEffect(SetSubtypes(p.ID(), SubtypeEnchantment, core.Indefinite, "Curse"))
	if g.FindPermanent(card.ID()).HasSubType("Aura") || !g.FindPermanent(card.ID()).HasSubType("Curse") {
		t.Fatal("later subtype set lost its timestamp")
	}
}

func TestEntryAsNonAuraEnchantmentCardTypes(t *testing.T) {
	for _, kind := range []core.CardType{core.TypeCreature, core.TypeArtifact, core.TypeEnchantment, core.TypeLand, core.TypePlaneswalker} {
		t.Run(kind.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Physical card", "", WithSubTypes("Equipment"))
			card.AddType(kind)
			card.SetOwner(b.PlayerID())
			b.AddToGraveyard(card)
			if kind == core.TypeLand {
				g.effects.Rules.SetLandsCantEnter(true)
			}
			target := DeclareGraveyardCardTarget("guest", InAnyGraveyard())
			_, obj := graveyardTestSpell(a.PlayerID(), target, ReturnCardToBattlefield(target, You, EnterAsNonAuraEnchantment()))
			obj.Targets = []uuid.UUID{card.ID()}
			g.pushStack(obj)
			g.ResolveTopOfStack()
			p := g.FindPermanent(card.ID())
			if p == nil {
				t.Fatal("entry failed")
			}
			for typ := core.TypeCreature; typ <= core.TypePlaneswalker; typ++ {
				if p.HasType(typ) != (typ == core.TypeEnchantment) {
					t.Fatalf("entry kept type %v", typ)
				}
			}
			if !p.Card.HasType(core.TypeArtifact) || p.HasSubType("Equipment") || p.HasSubType("Aura") {
				t.Fatal("entry changed physical card or kept wrong subtype")
			}
			clone := g.Clone()
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(card.ID()).HasType(core.TypeArtifact) {
				t.Fatal("clone lost type change")
			}
			g.BouncePermanentToHand(p)
			c, _ := b.RemoveFromHand(card.ID())
			p = g.PutOnBattlefield(c, a.PlayerID())
			if !p.HasType(core.TypeArtifact) {
				t.Fatal("new visit inherited type change")
			}
		})
	}
}
