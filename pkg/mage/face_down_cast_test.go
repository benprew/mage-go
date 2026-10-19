package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestFaceDownCast(t *testing.T) {
	for _, outcome := range []string{"resolve", "counter", "bounce", "exile", "clone", "copy", "phase", "turn-up", "end"} {
		t.Run(outcome, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Secret", "{G}{G}", 5, 6, WithKeyword(core.Flying), WithAbility(EntersWithNCounters(core.P1P1, 4)))
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			obj, err := g.castHandCardFaceDownWithoutPaying(a.PlayerID(), card.ID())
			if err != nil {
				t.Fatal(err)
			}
			if obj.Card.Name() != "" || len(obj.Card.Abilities()) != 0 || stackSpellManaValue(obj) != 0 || obj.Card.Power() != 2 || !obj.IsFaceDown() || len(a.Hand()) != 0 {
				t.Fatal("wrong hidden spell", obj)
			}
			if got := g.InspectFaceDownObject(b.PlayerID(), obj.ID); got != nil {
				t.Fatal("opponent inspected spell")
			}
			if got := g.InspectFaceDownObject(a.PlayerID(), obj.ID); got == nil || got.Name() != "Secret" {
				t.Fatal("controller cannot inspect spell")
			}
			if outcome == "counter" {
				g.CounterStackObject(obj.ID)
				if len(a.Graveyard()) != 1 || a.Graveyard()[0].Name() != "Secret" || len(g.PublicReveals()) != 1 {
					t.Fatal("counter did not reveal original")
				}
				return
			}
			if outcome == "end" {
				b.SetLost()
				if len(g.PublicReveals()) != 1 {
					t.Fatal("game end did not reveal spell")
				}
				return
			}
			if outcome == "clone" {
				clone := g.Clone()
				clone.CounterStackObject(obj.ID)
				if g.StackSize() != 1 || len(g.PublicReveals()) != 0 {
					t.Fatal("clone changed original")
				}
			}
			if outcome == "copy" {
				copiedSpell := g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				if copiedSpell.IsFaceDown() {
					t.Fatal("copy inherited status")
				}
				g.ResolveTopOfStack()
				p := g.FindPermanent(copiedSpell.Card.ID())
				if p == nil || !p.IsToken || p.FaceDown || p.Name() != "" {
					t.Fatal("copy entry")
				}
				g.turnFaceUp(p)
				if p.Name() != "" {
					t.Fatal("copy inherited private identity")
				}
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || !p.FaceDown || p.Name() != "" || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 2 || p.Counters[core.P1P1] != 0 || p.HasAttr(core.Flying) {
				t.Fatal("wrong face-down entry", p)
			}
			if got := g.InspectFaceDownObject(a.PlayerID(), p.ID()); got == nil || got.Name() != "Secret" {
				t.Fatal("missing permanent identity")
			}
			if outcome == "phase" {
				g.PhaseOut(p.ID())
				if g.InspectFaceDownObject(a.PlayerID(), p.ID()) == nil {
					t.Fatal("phased inspection")
				}
				g.PhaseIn([]uuid.UUID{p.ID()})
				if !g.FindPermanent(p.ID()).FaceDown {
					t.Fatal("phasing changed status")
				}
			}
			if outcome == "turn-up" {
				g.turnFaceUp(p)
				p = g.FindPermanent(p.ID())
				if p.Name() != "Secret" || p.CurrentPower(g) != 5 || !p.HasAttr(core.Flying) || p.Counters[core.P1P1] != 0 {
					t.Fatal("face-up restoration", p)
				}
			}
			if outcome == "bounce" {
				g.BouncePermanentToHand(p)
				if len(a.Hand()) != 1 || a.Hand()[0].Name() != "Secret" || len(g.PublicReveals()) != 1 {
					t.Fatal("bounce did not reveal original")
				}
				returnedCard, _ := a.RemoveFromHand(card.ID())
				returned := g.PutOnBattlefield(returnedCard, a.PlayerID())
				if returned.FaceDown || returned.Name() != "Secret" || returned.Counters[core.P1P1] != 4 {
					t.Fatal("return retained hidden status")
				}
			}
			if outcome == "exile" {
				g.ExilePermanent(p)
				if g.FindExiledCard(card.ID()).Card.Name() != "Secret" || len(g.PublicReveals()) != 1 {
					t.Fatal("exile did not reveal")
				}
			}
		})
	}
}

func TestFaceDownCastLegality(t *testing.T) {
	for _, scenario := range []string{"allowed", "named-rule", "creature-rule", "tax", "paid-tax", "foreign", "land", "absent"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Secret", "{G}", 4, 4)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if scenario == "land" {
				a.RemoveFromHand(card.ID())
				card = NewLand("Land")
				a.AddToHand(card)
			}
			if scenario == "named-rule" || scenario == "creature-rule" {
				filter := NewCardFilter("blocked", func(c Card) bool {
					if scenario == "named-rule" {
						return c.Name() == "Secret"
					}
					return c.HasType(core.TypeCreature)
				})
				g.PutOnBattlefield(NewArtifact("Rule", "", WithAbility(Static(PreventSpellCasts(filter)))), b.PlayerID())
			}
			if scenario == "tax" || scenario == "paid-tax" {
				g.PutOnBattlefield(NewArtifact("Tax", "", WithAbility(StaticAbility(IncreaseSpellCostStatic(func(*Game, uuid.UUID, Card, uuid.UUID) bool { return true }, core.ManaCost{Generic: 1}, false)))), b.PlayerID())
			}
			if scenario == "paid-tax" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			caster := a.PlayerID()
			if scenario == "foreign" {
				caster = b.PlayerID()
			}
			if scenario == "absent" {
				a.RemoveFromHand(card.ID())
			}
			_, err := g.castHandCardFaceDownWithoutPaying(caster, card.ID())
			want := scenario == "allowed" || scenario == "named-rule" || scenario == "paid-tax" || scenario == "land"
			if (err == nil) != want {
				t.Fatalf("error=%v want cast=%v", err, want)
			}
			if !want && scenario != "absent" && (len(a.Hand()) != 1 || g.StackSize() != 0) {
				t.Fatal("failed cast changed hand or stack")
			}
		})
	}
}

func TestFaceDownCastLayers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Hidden", "{G}", 4, 4)
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if _, err := g.castHandCardFaceDownWithoutPaying(a.PlayerID(), card.ID()); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p.CanDeclareAsAttacker(g) {
		t.Fatal("face-down creature lost summoning sickness")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, GrantKeywordModification(This, core.Flying)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	g.ApplyContinuousEffects()
	p = g.FindPermanent(card.ID())
	if !p.HasAttr(core.Flying) {
		t.Fatal("face-down creature lost a granted ability")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.FindPermanent(card.ID()).HasAttr(core.Flying) {
		t.Fatal("expired grant retained")
	}
}

func TestFaceDownCastDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{CastHandCardFaceDownWithoutPaying(nil, FromCard(BindCard("missing"))), CastHandCardFaceDownWithoutPaying(You, nil), CastHandCardFaceDownWithoutPaying(You, FromCard(BindCard("unbound")))} {
		if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", WithAbility(Spell(effect)))); err == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
}

func TestFaceDownCastPermanentCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	plan := entryCopyPlan(t, "Physical", func() []CardOption { return nil })
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if _, err := g.castHandCardFaceDownWithoutPaying(a.PlayerID(), card.ID()); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	donorPlan := entryCopyPlan(t, "Donor", func() []CardOption { return []CardOption{WithKeyword(core.Flying)} })
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	q := g.PutOnBattlefield(donor, b.PlayerID())
	if err := g.BecomeCopyOfPermanent(p, q, false); err != nil {
		t.Fatal(err)
	}
	p = g.FindPermanent(p.ID())
	if !p.FaceDown || p.Name() != "" || p.HasAttr(core.Flying) {
		t.Fatal("copy exposed face-down values")
	}
	g.turnFaceUp(p)
	p = g.FindPermanent(p.ID())
	if p.Name() != "Donor" || !p.HasAttr(core.Flying) {
		t.Fatal("face up lost copy values")
	}
	g.BouncePermanentToHand(p)
	if a.Hand()[0].Name() != "Physical" {
		t.Fatal("copy lost physical departure identity")
	}
}

type faceDownTestSpellRef struct{ id uuid.UUID }

func (faceDownTestSpellRef) stackObjectRef() {}
func (r faceDownTestSpellRef) resolveStackObject(ctx *EffectContext) (*StackObject, bool) {
	obj := ctx.Game.stack.FindByID(r.id)
	return obj, obj != nil
}
func (faceDownTestSpellRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "test-spell-ref"}
}

func TestFaceDownCastStackDeparture(t *testing.T) {
	for _, route := range []string{"exile-spell", "ante", "counter-exile"} {
		t.Run(route, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g, err := NewGameWithAnte(a, b, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			card := NewCreature("Physical", "{G}", 4, 4)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			obj, err := g.castHandCardFaceDownWithoutPaying(a.PlayerID(), card.ID())
			if err != nil {
				t.Fatal(err)
			}
			switch route {
			case "ante":
				if err := g.MoveToAnte(a.PlayerID(), card.ID()); err != nil {
					t.Fatal(err)
				}
				cards, err := g.AnteCards()
				if err != nil || len(cards) != 1 || cards[0].Name() != "Physical" {
					t.Fatal("ante lost physical card")
				}
			case "exile-spell":
				if err := ExileSpell(faceDownTestSpellRef{obj.ID}).Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			case "counter-exile":
				obj.ExileOnLeaveStack = true
				g.CounterStackObject(obj.ID)
			}
			if route != "ante" && (g.FindExiledCard(card.ID()) == nil || g.FindExiledCard(card.ID()).Card.Name() != "Physical") {
				t.Fatal("exile lost physical card")
			}
			if len(g.PublicReveals()) != 1 || g.StackSize() != 0 {
				t.Fatal("stack departure did not reveal")
			}
		})
	}
}

func TestFaceDownCastOwnership(t *testing.T) {
	for _, stage := range []string{"spell", "permanent"} {
		t.Run(stage, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g, err := NewGameWithAnte(a, b, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			card := NewCreature("Physical", "{G}", 4, 4)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			obj, err := g.castHandCardFaceDownWithoutPaying(a.PlayerID(), card.ID())
			if err != nil {
				t.Fatal(err)
			}
			if stage == "permanent" {
				g.ResolveStack()
			}
			clone := g.Clone()
			if err := clone.ChangeOwner(card.ID(), b.PlayerID()); err != nil {
				t.Fatal(err)
			}
			if stage == "spell" {
				clone.CounterStackObject(obj.ID)
			} else {
				clone.BouncePermanentToHand(clone.FindPermanent(card.ID()))
			}
			player := clone.GetPlayer(b.PlayerID())
			var moved Card
			if stage == "spell" {
				if len(player.Graveyard()) > 0 {
					moved = player.Graveyard()[0]
				}
			} else {
				if len(player.Hand()) > 0 {
					moved = player.Hand()[0]
				}
			}
			if moved == nil || moved.ID() != card.ID() || moved.Name() != "Physical" || moved.Owner() != b.PlayerID() {
				t.Fatal("ownership change lost physical identity", moved)
			}
			if len(clone.PublicReveals()) != 1 || clone.PublicReveals()[0].Owner != b.PlayerID() || len(g.PublicReveals()) != 0 {
				t.Fatal("reveal lost current owner or changed parent")
			}
		})
	}
}

func TestFaceDownCastFailureContinues(t *testing.T) {
	a := &optionalHandChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Chosen", "{G}", 4, 4)
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	g.PutOnBattlefield(NewArtifact("Restriction", "", WithAbility(Static(PreventSpellCasts(NewCardFilter("creatures", func(c Card) bool { return c.HasType(core.TypeCreature) }))))), b.PlayerID())
	result := BindCard("chosen")
	body := ChooseHandCardOrElse(You, CardFilter{}, result, Sequence(CastHandCardFaceDownWithoutPaying(You, FromCard(result)), GainLifeFor(You, Fixed(1))), nil)
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed casting stopped independent actions or changed the hand")
	}
}

func TestFaceDownCastPermissionCharacteristics(t *testing.T) {
	for _, card := range []Card{NewLand("Land"), NewArtifact("Artifact", "{9}"), NewInstant("Instant", "{R}")} {
		t.Run(card.Name(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			obj, err := g.castHandCardFaceDownWithoutPaying(a.PlayerID(), card.ID())
			if err != nil {
				t.Fatal(err)
			}
			if !obj.Card.HasType(core.TypeCreature) || obj.Card.HasType(core.TypeLand) || obj.Card.HasType(core.TypeInstant) || obj.Card.HasType(core.TypeArtifact) {
				t.Fatal("printed types entered the cast")
			}
			g.CounterStackObject(obj.ID)
			if len(a.Graveyard()) != 1 || a.Graveyard()[0].Name() != card.Name() {
				t.Fatal("departure lost printed identity")
			}
		})
	}
}
