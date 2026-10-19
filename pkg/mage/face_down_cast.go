package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CastHandCardFaceDownWithoutPaying casts a present hand visit as a 2/2 creature.
// The printed identity stays private. External casting costs still apply.
func CastHandCardFaceDownWithoutPaying(player PlayerRef, card CardRef) ResolvingEffect {
	return &faceDownCastEffect{player: player, card: card}
}

type faceDownCastEffect struct {
	player        PlayerRef
	card          CardRef
	result        *DamageSourceBinding
	captureResult bool
}

func (*faceDownCastEffect) Text() string                 { return "cast the chosen card face down" }
func (*faceDownCastEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *faceDownCastEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "face-down-cast", children: []definitionEdge{{"player", e.player}, {"card", e.card}}}
	if e.captureResult {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}
func (e *faceDownCastEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *faceDownCastEffect) Apply(ctx *EffectContext) error {
	e.clearResult(ctx)
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	card, ok := e.card.resolveCard(ctx)
	if !ok {
		return nil
	}
	// A failed cast must not stop the next instruction.
	obj, err := ctx.Game.castHandCardFaceDownWithoutPaying(player, card.ID())
	if err == nil && obj != nil && e.captureResult {
		if ctx.Game.resolution.damageSourceBindings == nil {
			ctx.Game.resolution.damageSourceBindings = make(map[*DamageSourceBinding]DamageSource)
		}
		ctx.Game.resolution.damageSourceBindings[e.result] = damageSourceFromSpellObject(obj)
	}
	return nil
}

func (g *Game) castHandCardFaceDownWithoutPaying(player, id uuid.UUID) (*StackObject, error) {
	payment, err := g.prepareFaceDownHandCast(player, id)
	if err != nil {
		return nil, err
	}
	if err := payment.Commit(); err != nil {
		return nil, err
	}
	return g.pushCastSpellObject(castStackObjectOptions{Card: payment.card, Controller: player, CastZone: core.ZoneHand, SnapshotCast: true})
}

func (g *Game) canCastHandCardFaceDownWithoutPaying(player, id uuid.UUID) bool {
	_, err := g.prepareFaceDownHandCast(player, id)
	return err == nil
}

func (g *Game) prepareFaceDownHandCast(player, id uuid.UUID) (*spellPaymentTransaction, error) {
	original := g.findCardInZone(player, id, core.ZoneHand)
	if original == nil || original.IsToken() {
		return nil, fmt.Errorf("face-down casting requires a nontoken card in the caster's hand")
	}
	hidden := newFaceDownCard(original)
	if err := g.checkSpellTiming(hidden, player); err != nil {
		return nil, err
	}
	return g.prepareSpellPaymentTransaction(spellPaymentSpec{Card: hidden, Controller: player, Zone: core.ZoneHand})
}

func newFaceDownCard(original Card) *BaseCard {
	hidden := NewCreature("", "", 2, 2)
	hidden.SetID(original.ID())
	hidden.SetOwner(original.Owner())
	hidden.faceDown = true
	hidden.faceDownOriginal = original
	hidden.isToken = original.IsToken()
	hidden.copiableValues = &CopiableValues{faceDown: true, printed: printedCopiableCharacteristics(hidden)}
	hidden.departureCard = cardForBattlefieldDeparture(original)
	if base, ok := original.(*BaseCard); ok {
		hidden.printedDefinitionPlan = base.printedDefinitionPlan
	}
	return hidden
}

// IsFaceDown reports the spell's face-down status without its private identity.
func (obj *StackObject) IsFaceDown() bool {
	return obj != nil && !obj.IsAbility && isFaceDownCard(obj.Card)
}
func isFaceDownCard(card Card) bool {
	if token, ok := card.(permanentSpellCopyCard); ok {
		card = token.Card
	}
	base, ok := card.(*BaseCard)
	return ok && base != nil && base.faceDown
}
func faceDownOriginal(card Card) Card {
	if token, ok := card.(permanentSpellCopyCard); ok {
		card = token.Card
	}
	if base, ok := card.(*BaseCard); ok && base.faceDown {
		return base.faceDownOriginal
	}
	return nil
}

// InspectFaceDownObject returns a private copy for the object's controller.
// Phasing keeps this permission. Spell readers use the exact stack object ID.
func (g *Game) InspectFaceDownObject(viewer, id uuid.UUID) Card {
	var hidden Card
	if obj := g.stack.FindByID(id); obj != nil && obj.Controller == viewer && obj.IsFaceDown() {
		hidden = obj.Card
	}
	if p := g.FindPermanentIncludingPhased(id); p != nil && p.ControllerID() == viewer && p.FaceDown {
		hidden = p.Card
	}
	if faceDownOriginal(hidden) == nil {
		return nil
	}
	original := cardForBattlefieldDeparture(hidden)
	result := original.Copy()
	result.SetID(original.ID())
	result.SetOwner(original.Owner())
	return result
}

func (g *Game) revealFaceDownCard(card Card) Card {
	if original := faceDownOriginal(card); original != nil {
		original = cardForBattlefieldDeparture(card)
		g.recordPublicReveal(original)
		return original
	}
	return card
}
