package mage

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
)

// drawCardsTargetEffect draws cards for a target player (or controller as fallback).
type drawCardsTargetEffect struct {
	props  EffectProperties
	amount ValueSource
}

// DrawCards creates an effect that draws cards for a target player (or controller as fallback).
func DrawCards(amount ValueSource) Effect {
	drawCount := 0
	if fv, ok := amount.(fixedValue); ok {
		drawCount = fv.n
	}
	return DataEffect(&drawCardsTargetEffect{
		props:  EffectProperties{Outcome: OutcomeBenefit, DrawCount: drawCount},
		amount: amount,
	})
}

func (e *drawCardsTargetEffect) EffectText() string {
	if _, ok := e.amount.(xValue); ok {
		return "target player draws X cards"
	}
	return fmt.Sprintf("target player draws %d card(s)", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *drawCardsTargetEffect) EffectProps() EffectProperties { return e.props }

// drawCardsActivePlayerEffect draws cards for the active player (e.g. Howling Mine).
type drawCardsActivePlayerEffect struct {
	props  EffectProperties
	amount ValueSource
}

// DrawCardsActivePlayer creates an effect that draws cards for the active player.
func DrawCardsActivePlayer(amount ValueSource) Effect {
	drawCount := 0
	if fv, ok := amount.(fixedValue); ok {
		drawCount = fv.n
	}
	return DataEffect(&drawCardsActivePlayerEffect{
		props:  EffectProperties{Outcome: OutcomeBenefit, DrawCount: drawCount},
		amount: amount,
	})
}

func (e *drawCardsActivePlayerEffect) EffectText() string {
	return "that player draws an additional card"
}
func (e *drawCardsActivePlayerEffect) EffectProps() EffectProperties { return e.props }

// discardCardsEffect forces a target player to discard cards.
type discardCardsEffect struct {
	amount ValueSource
}

// DiscardCards creates an effect that forces a target player to discard cards.
func DiscardCards(amount ValueSource) Effect {
	return DataEffect(&discardCardsEffect{amount: amount})
}

func (e *discardCardsEffect) EffectText() string {
	if _, ok := e.amount.(xValue); ok {
		return "target player discards X cards"
	}
	return fmt.Sprintf("target player discards %d card(s)", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *discardCardsEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// discardRandomEffect forces an opponent to discard a card at random.
type discardRandomEffect struct {
	amount int
}

// DiscardRandom creates an effect that forces a target player (or opponent) to discard cards at random.
func DiscardRandom(amount int) Effect {
	return DataEffect(&discardRandomEffect{amount: amount})
}

func (e *discardRandomEffect) EffectText() string {
	return fmt.Sprintf("discard %d card(s) at random", e.amount)
}
func (e *discardRandomEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// returnFromGraveyardEffect returns a target creature from graveyard to battlefield.
type returnFromGraveyardEffect struct{}

// ReturnFromGraveyardToBattlefield creates an effect that returns a target creature card
// from the controller's graveyard directly to the battlefield (e.g. Animate Dead, Resurrection).
func ReturnFromGraveyardToBattlefield() Effect {
	return DataEffect(&returnFromGraveyardEffect{})
}

func (e *returnFromGraveyardEffect) EffectText() string {
	return "return target creature card from your graveyard to the battlefield"
}
func (e *returnFromGraveyardEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// returnSourceToHandEffect returns the source card from graveyard to hand.
type returnSourceToHandEffect struct{}

// ReturnSourceToHand creates an effect that returns the source card from the graveyard
// to its owner's hand (e.g. Rancor's triggered ability).
func ReturnSourceToHand() Effect {
	return DataEffect(&returnSourceToHandEffect{})
}

func (e *returnSourceToHandEffect) EffectText() string {
	return "return this card to its owner's hand"
}
func (e *returnSourceToHandEffect) EffectProps() EffectProperties { return EffectProperties{} }

// exileSourceFromGraveyardEffect exiles the source card from the graveyard.
type exileSourceFromGraveyardEffect struct{}

// ExileSourceFromGraveyard creates an effect that exiles the source from the graveyard
// (e.g. Cyclopean Mummy's death trigger).
func ExileSourceFromGraveyard() Effect {
	return DataEffect(&exileSourceFromGraveyardEffect{})
}

func (e *exileSourceFromGraveyardEffect) EffectText() string            { return "exile this card from graveyard" }
func (e *exileSourceFromGraveyardEffect) EffectProps() EffectProperties { return EffectProperties{} }

// millTargetPlayerEffect mills N cards from the target player's library.
type millTargetPlayerEffect struct {
	amount ValueSource
}

// MillTargetPlayer creates an effect that mills N cards from target player's library.
func MillTargetPlayer(amount ValueSource) Effect {
	return DataEffect(&millTargetPlayerEffect{amount: amount})
}

func (e *millTargetPlayerEffect) EffectText() string {
	return "target player mills cards"
}
func (e *millTargetPlayerEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// returnToHandTargetEffect bounces a target permanent to its owner's hand.
type returnToHandTargetEffect struct{}

// ReturnToHandTarget creates an effect that bounces a target permanent to its owner's hand.
func ReturnToHandTarget() Effect {
	return DataEffect(&returnToHandTargetEffect{})
}

func (e *returnToHandTargetEffect) EffectText() string {
	return "return target permanent to its owner's hand"
}
func (e *returnToHandTargetEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, IsBounce: true}
}

// returnFromGraveyardToHandTargetEffect returns a target card from graveyard to hand.
type returnFromGraveyardToHandTargetEffect struct{}

// ReturnFromGraveyardToHandTarget creates an effect that returns a target card
// from the controller's graveyard to their hand (e.g. Raise Dead, Regrowth).
func ReturnFromGraveyardToHandTarget() Effect {
	return DataEffect(&returnFromGraveyardToHandTargetEffect{})
}

func (e *returnFromGraveyardToHandTargetEffect) EffectText() string {
	return "return target card from your graveyard to your hand"
}
func (e *returnFromGraveyardToHandTargetEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// searchLibraryEffect lets the controller search their library and put a card in hand.
type searchLibraryEffect struct{}

// SearchLibraryToHand creates an effect that lets the controller search their library for a card
// and put it into their hand (e.g. Demonic Tutor).
func SearchLibraryToHand() Effect {
	return DataEffect(&searchLibraryEffect{})
}

func (e *searchLibraryEffect) EffectText() string {
	return "search your library for a card and put it into your hand"
}
func (e *searchLibraryEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// searchLibraryToTopEffect lets the controller search their library and put a card on top.
type searchLibraryToTopEffect struct{}

// SearchLibraryToTop creates an effect that lets the controller search their library for a card
// and put it on top of their library (e.g. Worldly Tutor, Vampiric Tutor).
func SearchLibraryToTop() Effect {
	return DataEffect(&searchLibraryToTopEffect{})
}

func (e *searchLibraryToTopEffect) EffectText() string {
	return "search your library for a card and put it on top"
}
func (e *searchLibraryToTopEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// discardHandAndDrawEffect makes each player discard their hand and draw N cards.
type discardHandAndDrawEffect struct {
	drawCount int
}

// DiscardHandAndDraw creates an effect where each player discards their hand then draws n cards (e.g. Timetwister, Wheel of Fortune).
func DiscardHandAndDraw(n int) Effect {
	return DataEffect(&discardHandAndDrawEffect{drawCount: n})
}

func (e *discardHandAndDrawEffect) EffectText() string {
	return fmt.Sprintf("Each player discards their hand, then draws %d cards", e.drawCount)
}
func (e *discardHandAndDrawEffect) EffectProps() EffectProperties { return EffectProperties{} }

// shuffleHandAndGraveyardIntoLibraryAndDrawEffect shuffles each player's hand
// and graveyard into their library, then each player draws N cards.
type shuffleHandAndGraveyardIntoLibraryAndDrawEffect struct {
	drawCount int
}

// ShuffleHandAndGraveyardIntoLibraryAndDraw creates an effect where each player shuffles their
// hand and graveyard into their library, then draws n cards (e.g. Timetwister).
func ShuffleHandAndGraveyardIntoLibraryAndDraw(n int) Effect {
	return DataEffect(&shuffleHandAndGraveyardIntoLibraryAndDrawEffect{drawCount: n})
}

func (e *shuffleHandAndGraveyardIntoLibraryAndDrawEffect) EffectText() string {
	return fmt.Sprintf("Each player shuffles their hand and graveyard into their library, then draws %d cards", e.drawCount)
}
func (e *shuffleHandAndGraveyardIntoLibraryAndDrawEffect) EffectProps() EffectProperties {
	return EffectProperties{}
}

// shuffleLibraryEffect shuffles the controller's library.
type shuffleLibraryEffect struct{}

// ShuffleLibrary creates an effect that shuffles the controller's library.
func ShuffleLibrary() Effect {
	return DataEffect(&shuffleLibraryEffect{})
}

func (e *shuffleLibraryEffect) EffectText() string              { return "shuffle your library" }
func (e *shuffleLibraryEffect) EffectProps() EffectProperties { return EffectProperties{} }

// putFromHandOntoBattlefieldEffect lets the controller put a card from hand onto the battlefield.
type putFromHandOntoBattlefieldEffect struct {
	filter CardFilter
}

// PutFromHandOntoBattlefield creates an effect that lets the controller put a card from hand
// onto the battlefield (e.g. Elvish Piper, Show and Tell). Pass a zero CardFilter to allow any card.
func PutFromHandOntoBattlefield(filter CardFilter) Effect {
	return DataEffect(&putFromHandOntoBattlefieldEffect{filter: filter})
}

func (e *putFromHandOntoBattlefieldEffect) EffectText() string {
	return "put a card from your hand onto the battlefield"
}
func (e *putFromHandOntoBattlefieldEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// searchLibraryToBattlefieldEffect searches library and puts a matching card onto the battlefield.
type searchLibraryToBattlefieldEffect struct {
	filter CardFilter
}

// SearchLibraryToBattlefield creates an effect that searches the controller's
// library for a card matching the filter, puts it onto the battlefield, then
// shuffles (e.g. Untamed Wilds, Rampant Growth).
func SearchLibraryToBattlefield(filter CardFilter) Effect {
	return DataEffect(&searchLibraryToBattlefieldEffect{filter: filter})
}

func (e *searchLibraryToBattlefieldEffect) EffectText() string {
	return "search your library for a card, put it onto the battlefield, then shuffle"
}
func (e *searchLibraryToBattlefieldEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// chooseColorEffect lets the controller choose a color and stores it on the source permanent.
type chooseColorEffect struct {
	reason string
}

// ChooseColor creates an effect that asks the controller to choose a color,
// storing the result on the source permanent's ChosenColor field.
func ChooseColor(reason string) Effect {
	return DataEffect(&chooseColorEffect{reason: reason})
}

func (e *chooseColorEffect) EffectText() string              { return "choose a color" }
func (e *chooseColorEffect) EffectProps() EffectProperties { return EffectProperties{} }

// --- Executor functions ---

func execDrawCardsTarget(ctx *EffectContext, e *drawCardsTargetEffect) error {
	var targetPlayer Player
	if len(ctx.Targets) > 0 {
		targetPlayer = ctx.Game.GetPlayer(ctx.Targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = ctx.Game.GetPlayer(ctx.Controller)
	}
	if targetPlayer == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	for i := 0; i < amount; i++ {
		ctx.Game.PlayerDrawCard(targetPlayer)
	}
	return nil
}

func execDrawCardsActivePlayer(ctx *EffectContext, e *drawCardsActivePlayerEffect) error {
	active := ctx.Game.ActivePlayerObj()
	if active == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, active.PlayerID(), ctx.Targets)
	for i := 0; i < amount; i++ {
		ctx.Game.PlayerDrawCard(active)
	}
	return nil
}

func execDiscardCards(ctx *EffectContext, e *discardCardsEffect) error {
	var targetPlayer Player
	if len(ctx.Targets) > 0 {
		targetPlayer = ctx.Game.GetPlayer(ctx.Targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = ctx.Game.GetOpponent(ctx.Controller)
	}
	if targetPlayer == nil {
		return nil
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	chosen := targetPlayer.ChooseCardsFromHand(amount, "discard", ctx.Game)
	for _, card := range chosen {
		targetPlayer.DiscardCard(card.ID())
	}
	return nil
}

func execDiscardRandom(ctx *EffectContext, e *discardRandomEffect) error {
	var targetPlayer Player
	if len(ctx.Targets) > 0 {
		targetPlayer = ctx.Game.GetPlayer(ctx.Targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = ctx.Game.GetOpponent(ctx.Controller)
	}
	if targetPlayer == nil {
		return nil
	}
	for i := 0; i < e.amount; i++ {
		hand := targetPlayer.Hand()
		if len(hand) == 0 {
			break
		}
		idx := rand.Intn(len(hand))
		targetPlayer.DiscardCard(hand[idx].ID())
	}
	return nil
}

func execReturnFromGraveyardToBattlefield(ctx *EffectContext, _ *returnFromGraveyardEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for reanimate")
	}
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	card, ok := p.RemoveFromGraveyard(ctx.Targets[0])
	if !ok {
		return nil // target gone
	}
	ctx.Game.PutOnBattlefield(card, ctx.Controller)
	return nil
}

func execMillTargetPlayer(ctx *EffectContext, e *millTargetPlayerEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	p := ctx.Game.GetPlayer(ctx.Targets[0])
	if p == nil {
		return nil
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	lib := p.Library()
	for i := 0; i < amount && len(lib) > 0; i++ {
		card := lib[len(lib)-1]
		lib = lib[:len(lib)-1]
		p.AddToGraveyard(card)
	}
	p.SetLibrary(lib)
	return nil
}

func execExileSourceFromGraveyard(ctx *EffectContext, _ *exileSourceFromGraveyardEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return nil
	}
	card, ok := p.RemoveFromGraveyard(ctx.SourceID)
	if ok && card != nil {
		ctx.Game.ExileCard(card, ctx.SourceID)
	}
	return nil
}

func execReturnSourceToHand(ctx *EffectContext, _ *returnSourceToHandEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	card, ok := p.RemoveFromGraveyard(ctx.SourceID)
	if !ok {
		return nil // not in graveyard
	}
	p.AddToHand(card)
	return nil
}

func execReturnToHandTarget(ctx *EffectContext, _ *returnToHandTargetEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for bounce")
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil // target gone, fizzle
	}
	card := perm.Card
	owner := card.Owner()
	if owner == uuid.Nil {
		owner = perm.Controller
	}
	ctx.Game.RemoveFromBattlefield(perm)
	p := ctx.Game.GetPlayer(owner)
	if p != nil {
		p.AddToHand(card)
	}
	return nil
}

func execReturnFromGraveyardToHandTarget(ctx *EffectContext, _ *returnFromGraveyardToHandTargetEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for raise dead")
	}
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	card, ok := p.RemoveFromGraveyard(ctx.Targets[0])
	if !ok {
		return nil // target gone
	}
	p.AddToHand(card)
	return nil
}

func execSearchLibraryToHand(ctx *EffectContext, _ *searchLibraryEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	lib := p.Library()
	if len(lib) == 0 {
		return nil
	}
	card := p.ChooseCardFromLibrary(lib, "search", ctx.Game)
	if card == nil {
		return nil
	}
	// Remove the chosen card from library
	newLib := make([]Card, 0, len(lib)-1)
	for _, c := range lib {
		if c.ID() != card.ID() {
			newLib = append(newLib, c)
		}
	}
	p.SetLibrary(newLib)
	p.ShuffleLibrary()
	p.AddToHand(card)
	return nil
}

func execSearchLibraryToTop(ctx *EffectContext, _ *searchLibraryToTopEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	lib := p.Library()
	if len(lib) == 0 {
		return nil
	}
	card := p.ChooseCardFromLibrary(lib, "search to top", ctx.Game)
	if card == nil {
		return nil
	}
	// Remove the chosen card from library
	newLib := make([]Card, 0, len(lib))
	newLib = append(newLib, card)
	for _, c := range lib {
		if c.ID() != card.ID() {
			newLib = append(newLib, c)
		}
	}
	p.SetLibrary(newLib)
	return nil
}

func execDiscardHandAndDraw(ctx *EffectContext, e *discardHandAndDrawEffect) error {
	for _, p := range ctx.Game.AllPlayers() {
		// Discard entire hand
		hand := p.Hand()
		for _, c := range hand {
			p.DiscardCard(c.ID())
		}
		// Draw N cards
		for i := 0; i < e.drawCount; i++ {
			ctx.Game.PlayerDrawCard(p)
		}
	}
	return nil
}

func execShuffleHandAndGraveyardIntoLibraryAndDraw(ctx *EffectContext, e *shuffleHandAndGraveyardIntoLibraryAndDrawEffect) error {
	for _, p := range ctx.Game.AllPlayers() {
		// Move hand into library (copy slice since RemoveFromHand modifies it)
		hand := append([]Card(nil), p.Hand()...)
		for _, c := range hand {
			p.RemoveFromHand(c.ID())
			p.AddToLibrary(c)
		}
		// Move graveyard into library
		for _, c := range p.Graveyard() {
			p.AddToLibrary(c)
		}
		p.ClearGraveyard()
		p.ShuffleLibrary()
		// Draw N cards
		for i := 0; i < e.drawCount; i++ {
			ctx.Game.PlayerDrawCard(p)
		}
	}
	return nil
}

func execShuffleLibrary(ctx *EffectContext, _ *shuffleLibraryEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	p.ShuffleLibrary()
	return nil
}

func execPutFromHandOntoBattlefield(ctx *EffectContext, e *putFromHandOntoBattlefieldEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	hand := p.Hand()
	var candidates []Card
	for _, c := range hand {
		if e.filter.Match(c) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	chosen := p.ChooseCardFromLibrary(candidates, "put onto battlefield", ctx.Game)
	if chosen == nil {
		return nil
	}
	if _, ok := p.RemoveFromHand(chosen.ID()); ok {
		ctx.Game.PutOnBattlefield(chosen, ctx.Controller)
	}
	return nil
}

func execSearchLibraryToBattlefield(ctx *EffectContext, e *searchLibraryToBattlefieldEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	lib := p.Library()
	var candidates []Card
	for _, c := range lib {
		if e.filter.Match(c) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		p.ShuffleLibrary()
		return nil
	}
	card := p.ChooseCardFromLibrary(candidates, "search to battlefield", ctx.Game)
	if card == nil {
		p.ShuffleLibrary()
		return nil
	}
	// Remove from library
	newLib := make([]Card, 0, len(lib)-1)
	for _, c := range lib {
		if c.ID() != card.ID() {
			newLib = append(newLib, c)
		}
	}
	p.SetLibrary(newLib)
	p.ShuffleLibrary()
	ctx.Game.PutOnBattlefield(card, ctx.Controller)
	return nil
}

func execChooseColor(ctx *EffectContext, e *chooseColorEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	perm := ctx.Game.FindPermanent(ctx.SourceID)
	if perm == nil {
		return nil
	}
	perm.ChosenColor = p.ChooseManaColor(e.reason)
	return nil
}
