package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentSpellCopyEntersAsToken(t *testing.T) {
	for _, kind := range []string{"creature", "artifact", "enchantment"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			g.SetStep(PrecombatMain)
			g.SetTurn(1)
			var card Card
			switch kind {
			case "creature":
				card = NewCreature("Copy subject", "", 2, 3, WithKeyword(Flying))
			case "artifact":
				card = NewArtifact("Copy subject", "")
			case "enchantment":
				card = NewEnchantment("Copy subject", "")
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			original := g.StackPeek()
			copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
			if copied == nil {
				t.Fatal("no copy")
			}
			clone := g.Clone()
			clone.ResolveTopOfStack()
			token := clone.FindPermanent(copied.SourceID)
			if token == nil || !token.IsToken || !token.Card.IsToken() || token.ControllerID() != b.PlayerID() || token.Card.Owner() != b.PlayerID() {
				t.Fatal("copy did not enter as owned token")
			}
			if copied.Card.IsToken() || original.Card.IsToken() || g.FindPermanent(copied.SourceID) != nil {
				t.Fatal("clone mutated stack definitions")
			}
			if kind == "creature" && (token.CurrentPower(clone) != 2 || token.CurrentToughness(clone) != 3 || !token.HasKeyword(Flying)) {
				t.Fatal("token lost characteristics")
			}
			g.ResolveTopOfStack()
			g.ResolveStack()
			if p := g.FindPermanent(card.ID()); p == nil || p.IsToken {
				t.Fatal("original became token")
			}
			live := g.FindPermanent(copied.SourceID)
			if live == nil || !live.IsToken {
				t.Fatal("live token missing")
			}
			g.BouncePermanentToHand(live)
			if len(b.Hand()) != 0 || g.FindPermanent(copied.SourceID) != nil || len(b.Graveyard()) != 0 {
				t.Fatal("departed token retained card")
			}
			if clone.FindPermanent(copied.SourceID) == nil {
				t.Fatal("token clone changed")
			}
		})
	}
}

func TestPermanentSpellCopyAuraRetarget(t *testing.T) {
	for _, change := range []string{"legal", "return", "phase", "shroud", "counter", "keep-illegal"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			g.SetStep(PrecombatMain)
			g.SetTurn(1)
			first := g.PutOnBattlefield(NewCreature("First", "", 2, 3), a.PlayerID())
			secondCard := NewCreature("Second", "", 4, 5)
			secondCard.SetOwner(b.PlayerID())
			second := g.PutOnBattlefield(secondCard, b.PlayerID())
			aura := NewAura("Bonus", "", WithAbility(Static(While(StateConditionFunc("always", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true }), ModifyPT(AttachedPermanent, Fixed(1), Fixed(2))))))
			aura.SetOwner(a.PlayerID())
			a.AddToHand(aura)
			if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			phase := g.PhaseOut(first.ID())
			copied := g.CopySpellOnStack(aura.ID(), b.PlayerID(), change != "keep-illegal")
			if copied == nil {
				t.Fatal("copy missing")
			}
			if change != "keep-illegal" && (len(copied.Targets) != 1 || copied.Targets[0] != second.ID()) {
				t.Fatal("Aura casting target was not retargeted")
			}
			if g.Stack().FindBySourceID(aura.ID()).Targets[0] != first.ID() {
				t.Fatal("copy changed original")
			}
			switch change {
			case "return":
				g.BouncePermanentToHand(second)
				b.RemoveFromHand(secondCard.ID())
				g.PutOnBattlefield(secondCard, b.PlayerID())
			case "phase":
				g.PhaseOut(second.ID())
			case "shroud":
				g.MutablePermanent(second.ID()).GrantBaseAttr(Shroud)
			case "counter":
				g.CounterStackObject(copied.ID)
			}
			clone := g.Clone()
			if change != "counter" {
				g.ResolveTopOfStack()
				clone.ResolveTopOfStack()
			}
			for _, game := range []*Game{g, clone} {
				token := game.FindPermanent(copied.SourceID)
				if change == "legal" {
					if token == nil || !token.IsToken || token.AttachedTo != second.ID() || game.FindPermanent(second.ID()).CurrentPower(game) != 5 || game.FindPermanent(second.ID()).CurrentToughness(game) != 7 {
						t.Fatal("token Aura attachment or bonus")
					}
				} else if token != nil {
					t.Fatal("illegal copy entered")
				}
			}
			if len(b.Graveyard()) != 0 {
				t.Fatal("countered copy entered graveyard")
			}
			g.PhaseIn(phase)
			g.ResolveStack()
			if p := g.FindPermanent(aura.ID()); p == nil || p.IsToken || p.AttachedTo != first.ID() {
				t.Fatal("original Aura resolution")
			}
		})
	}
}

func TestPermanentSpellCopyEntryContracts(t *testing.T) {
	g, a, b := newSpellCopyGame()
	g.SetStep(PrecombatMain)
	g.SetTurn(1)
	casts := 0
	entered := 0
	donor := g.PutOnBattlefield(NewArtifact("Entry donor", "", WithAbility(NewTriggered(EvtSpellCast, false).SetCondition(func(*GameEvent, GameReader, uuid.UUID, uuid.UUID) bool { casts++; return false })), WithAbility(NewTriggered(EvtZoneChange, false).SetCondition(func(evt *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if evt.ToZone == ZoneBattlefield {
			if p := reader.FindPermanent(evt.SourceID); p != nil && p.IsToken {
				entered++
				if !p.Card.IsToken() || evt.CastColorsSpent != 0 {
					t.Fatal("entry observed wrong token or cast facts")
				}
			}
		}
		return false
	}))), a.PlayerID())
	g.AddETBAdditionalCounters(donor.ID(), P1P1, 1, AnyPermanent, false)
	card := NewCreature("Entry counters", "", 2, 2, WithAbility(EntersWithCounters(P1P1, ColorsSpentToCastEnteringPermanent())))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.StackPeek().CastContext.ColorsSpent = map[Color]int{Red: 1, Blue: 1}
	cp := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if cp == nil || casts != 1 {
		t.Fatal("copy was cast")
	}
	g.ResolveTopOfStack()
	if token := g.FindPermanent(cp.SourceID); token == nil || token.Counters[P1P1] != 1 || entered != 1 {
		t.Fatal("copy reused paid mana or bypassed entry replacement")
	}
	g.ResolveStack()
	if original := g.FindPermanent(card.ID()); original == nil || original.Counters[P1P1] != 3 || original.IsToken {
		t.Fatal("original entry facts changed")
	}

	land := NewCreature("Land creature", "", 2, 2, WithSubTypes("Forest"))
	land.AddType(TypeLand)
	land.SetOwner(a.PlayerID())
	a.AddToHand(land)
	// A land card is not cast, so this test supplies the proposed stack event.
	obj := &StackObject{ID: uuid.New(), Card: land, SourceID: land.ID(), Controller: a.PlayerID()}
	g.pushStack(obj)
	blocked := g.CopySpellOnStack(land.ID(), b.PlayerID(), false)
	if blocked == nil {
		t.Fatal("blocked copy missing")
	}
	g.effects.Rules.SetLandsCantEnter(true)
	g.ResolveTopOfStack()
	if g.FindPermanent(blocked.SourceID) != nil || len(b.Graveyard()) != 0 {
		t.Fatal("prohibited copy entry")
	}
}
