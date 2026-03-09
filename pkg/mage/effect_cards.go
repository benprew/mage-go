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
	return &drawCardsTargetEffect{
		props:  EffectProperties{Outcome: OutcomeBenefit, DrawCount: drawCount},
		amount: amount,
	}
}

func (e *drawCardsTargetEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var targetPlayer Player
	if len(targets) > 0 {
		targetPlayer = g.GetPlayer(targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = g.GetPlayer(controller)
	}
	if targetPlayer == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.Resolve(g, sourceID, controller)
	for i := 0; i < amount; i++ {
		g.PlayerDrawCard(targetPlayer)
	}
	return nil
}

func (e *drawCardsTargetEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "target player draws X cards"
	}
	return fmt.Sprintf("target player draws %d card(s)", e.amount.Resolve(nil, uuid.Nil, uuid.Nil))
}
func (e *drawCardsTargetEffect) Properties() EffectProperties { return e.props }

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
	return &drawCardsActivePlayerEffect{
		props:  EffectProperties{Outcome: OutcomeBenefit, DrawCount: drawCount},
		amount: amount,
	}
}

func (e *drawCardsActivePlayerEffect) Apply(g GameMutator, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
	active := g.ActivePlayerObj()
	if active == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.Resolve(g, sourceID, active.PlayerID())
	for i := 0; i < amount; i++ {
		g.PlayerDrawCard(active)
	}
	return nil
}

func (e *drawCardsActivePlayerEffect) Text() string {
	return "that player draws an additional card"
}
func (e *drawCardsActivePlayerEffect) Properties() EffectProperties { return e.props }

// discardCardsEffect forces a target player to discard cards.
type discardCardsEffect struct {
	amount ValueSource
}

// DiscardCards creates an effect that forces a target player to discard cards.
func DiscardCards(amount ValueSource) Effect {
	return &discardCardsEffect{amount: amount}
}

func (e *discardCardsEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var targetPlayer Player
	if len(targets) > 0 {
		targetPlayer = g.GetPlayer(targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = g.GetOpponent(controller)
	}
	if targetPlayer == nil {
		return nil
	}
	amount := e.amount.Resolve(g, sourceID, controller)
	chosen := targetPlayer.ChooseCardsFromHand(amount, "discard", g)
	for _, card := range chosen {
		targetPlayer.RemoveFromHand(card.ID())
		targetPlayer.AddToGraveyard(card)
	}
	return nil
}

func (e *discardCardsEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "target player discards X cards"
	}
	return fmt.Sprintf("target player discards %d card(s)", e.amount.Resolve(nil, uuid.Nil, uuid.Nil))
}
func (e *discardCardsEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// discardRandomEffect forces an opponent to discard a card at random.
type discardRandomEffect struct {
	amount int
}

// DiscardRandom creates an effect that forces a target player (or opponent) to discard cards at random.
func DiscardRandom(amount int) Effect {
	return &discardRandomEffect{amount: amount}
}

func (e *discardRandomEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var targetPlayer Player
	if len(targets) > 0 {
		targetPlayer = g.GetPlayer(targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = g.GetOpponent(controller)
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
		card := hand[idx]
		targetPlayer.RemoveFromHand(card.ID())
		targetPlayer.AddToGraveyard(card)
	}
	return nil
}

func (e *discardRandomEffect) Text() string {
	return fmt.Sprintf("discard %d card(s) at random", e.amount)
}
func (e *discardRandomEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// returnFromGraveyardEffect returns a target creature from graveyard to battlefield.
type returnFromGraveyardEffect struct{}

// ReturnFromGraveyardToBattlefield creates an effect that returns a target creature card
// from the controller's graveyard directly to the battlefield (e.g. Animate Dead, Resurrection).
func ReturnFromGraveyardToBattlefield() Effect {
	return &returnFromGraveyardEffect{}
}

func (e *returnFromGraveyardEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for reanimate")
	}
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	card, ok := p.RemoveFromGraveyard(targets[0])
	if !ok {
		return nil // target gone
	}
	g.PutOnBattlefield(card, controller)
	return nil
}

func (e *returnFromGraveyardEffect) Text() string {
	return "return target creature card from your graveyard to the battlefield"
}
func (e *returnFromGraveyardEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// returnSourceToHandEffect returns the source card from graveyard to hand.
type returnSourceToHandEffect struct{}

// ReturnSourceToHand creates an effect that returns the source card from the graveyard
// to its owner's hand (e.g. Rancor's triggered ability).
func ReturnSourceToHand() Effect {
	return &returnSourceToHandEffect{}
}

func (e *returnSourceToHandEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	card, ok := p.RemoveFromGraveyard(sourceID)
	if !ok {
		return nil // not in graveyard
	}
	p.AddToHand(card)
	return nil
}

func (e *returnSourceToHandEffect) Text() string {
	return "return this card to its owner's hand"
}
func (e *returnSourceToHandEffect) Properties() EffectProperties { return EffectProperties{} }

// returnToHandTargetEffect bounces a target permanent to its owner's hand.
type returnToHandTargetEffect struct{}

// ReturnToHandTarget creates an effect that bounces a target permanent to its owner's hand.
func ReturnToHandTarget() Effect {
	return &returnToHandTargetEffect{}
}

func (e *returnToHandTargetEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for bounce")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil // target gone, fizzle
	}
	card := perm.Card
	owner := card.Owner()
	if owner == uuid.Nil {
		owner = perm.Controller
	}
	g.RemoveFromBattlefield(perm)
	p := g.GetPlayer(owner)
	if p != nil {
		p.AddToHand(card)
	}
	return nil
}

func (e *returnToHandTargetEffect) Text() string {
	return "return target permanent to its owner's hand"
}
func (e *returnToHandTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, IsBounce: true}
}

// returnFromGraveyardToHandTargetEffect returns a target card from graveyard to hand.
type returnFromGraveyardToHandTargetEffect struct{}

// ReturnFromGraveyardToHandTarget creates an effect that returns a target card
// from the controller's graveyard to their hand (e.g. Raise Dead, Regrowth).
func ReturnFromGraveyardToHandTarget() Effect {
	return &returnFromGraveyardToHandTargetEffect{}
}

func (e *returnFromGraveyardToHandTargetEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for raise dead")
	}
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	card, ok := p.RemoveFromGraveyard(targets[0])
	if !ok {
		return nil // target gone
	}
	p.AddToHand(card)
	return nil
}

func (e *returnFromGraveyardToHandTargetEffect) Text() string {
	return "return target card from your graveyard to your hand"
}
func (e *returnFromGraveyardToHandTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// searchLibraryEffect lets the controller search their library and put a card in hand.
type searchLibraryEffect struct{}

// SearchLibraryToHand creates an effect that lets the controller search their library for a card
// and put it into their hand (e.g. Demonic Tutor).
func SearchLibraryToHand() Effect {
	return &searchLibraryEffect{}
}

func (e *searchLibraryEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	lib := p.Library()
	if len(lib) == 0 {
		return nil
	}
	card := p.ChooseCardFromLibrary(lib, "search", g)
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

func (e *searchLibraryEffect) Text() string {
	return "search your library for a card and put it into your hand"
}
func (e *searchLibraryEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// searchLibraryToTopEffect lets the controller search their library and put a card on top.
type searchLibraryToTopEffect struct{}

// SearchLibraryToTop creates an effect that lets the controller search their library for a card
// and put it on top of their library (e.g. Worldly Tutor, Vampiric Tutor).
func SearchLibraryToTop() Effect {
	return &searchLibraryToTopEffect{}
}

func (e *searchLibraryToTopEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	lib := p.Library()
	if len(lib) == 0 {
		return nil
	}
	card := p.ChooseCardFromLibrary(lib, "search to top", g)
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

func (e *searchLibraryToTopEffect) Text() string {
	return "search your library for a card and put it on top"
}
func (e *searchLibraryToTopEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// discardHandAndDrawEffect makes each player discard their hand and draw N cards.
type discardHandAndDrawEffect struct {
	drawCount int
}

// DiscardHandAndDraw creates an effect where each player discards their hand then draws n cards (e.g. Timetwister, Wheel of Fortune).
func DiscardHandAndDraw(n int) Effect {
	return &discardHandAndDrawEffect{drawCount: n}
}

func (e *discardHandAndDrawEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, p := range g.AllPlayers() {
		// Discard entire hand
		hand := p.Hand()
		for _, c := range hand {
			p.RemoveFromHand(c.ID())
			p.AddToGraveyard(c)
		}
		// Draw N cards
		for i := 0; i < e.drawCount; i++ {
			g.PlayerDrawCard(p)
		}
	}
	return nil
}

func (e *discardHandAndDrawEffect) Text() string {
	return fmt.Sprintf("Each player discards their hand, then draws %d cards", e.drawCount)
}
func (e *discardHandAndDrawEffect) Properties() EffectProperties { return EffectProperties{} }

// shuffleGraveyardIntoLibraryAndDrawEffect shuffles each player's graveyard
// into their library, then each player draws N cards.
type shuffleGraveyardIntoLibraryAndDrawEffect struct {
	drawCount int
}

// ShuffleGraveyardIntoLibraryAndDraw creates an effect where each player shuffles their graveyard
// into their library, then draws n cards (e.g. Feldon's Cane variant).
func ShuffleGraveyardIntoLibraryAndDraw(n int) Effect {
	return &shuffleGraveyardIntoLibraryAndDrawEffect{drawCount: n}
}

func (e *shuffleGraveyardIntoLibraryAndDrawEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, p := range g.AllPlayers() {
		// Move graveyard into library
		for _, c := range p.Graveyard() {
			p.AddToLibrary(c)
		}
		p.ClearGraveyard()
		p.ShuffleLibrary()
		// Draw N cards
		for i := 0; i < e.drawCount; i++ {
			g.PlayerDrawCard(p)
		}
	}
	return nil
}

func (e *shuffleGraveyardIntoLibraryAndDrawEffect) Text() string {
	return fmt.Sprintf("Each player shuffles graveyard into library, then draws %d cards", e.drawCount)
}
func (e *shuffleGraveyardIntoLibraryAndDrawEffect) Properties() EffectProperties {
	return EffectProperties{}
}

// shuffleLibraryEffect shuffles the controller's library.
type shuffleLibraryEffect struct{}

// ShuffleLibrary creates an effect that shuffles the controller's library.
func ShuffleLibrary() Effect {
	return &shuffleLibraryEffect{}
}

func (e *shuffleLibraryEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	p.ShuffleLibrary()
	return nil
}

func (e *shuffleLibraryEffect) Text() string              { return "shuffle your library" }
func (e *shuffleLibraryEffect) Properties() EffectProperties { return EffectProperties{} }

// putFromHandOntoBattlefieldEffect lets the controller put a card from hand onto the battlefield.
type putFromHandOntoBattlefieldEffect struct {
	filter CardFilter
}

// PutFromHandOntoBattlefield creates an effect that lets the controller put a card from hand
// onto the battlefield (e.g. Elvish Piper, Show and Tell). Pass a zero CardFilter to allow any card.
func PutFromHandOntoBattlefield(filter CardFilter) Effect {
	return &putFromHandOntoBattlefieldEffect{filter: filter}
}

func (e *putFromHandOntoBattlefieldEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
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
	chosen := p.ChooseCardFromLibrary(candidates, "put onto battlefield", g)
	if chosen == nil {
		return nil
	}
	if _, ok := p.RemoveFromHand(chosen.ID()); ok {
		g.PutOnBattlefield(chosen, controller)
	}
	return nil
}

func (e *putFromHandOntoBattlefieldEffect) Text() string {
	return "put a card from your hand onto the battlefield"
}
func (e *putFromHandOntoBattlefieldEffect) Properties() EffectProperties {
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
	return &searchLibraryToBattlefieldEffect{filter: filter}
}

func (e *searchLibraryToBattlefieldEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
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
	card := p.ChooseCardFromLibrary(candidates, "search to battlefield", g)
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
	g.PutOnBattlefield(card, controller)
	return nil
}

func (e *searchLibraryToBattlefieldEffect) Text() string {
	return "search your library for a card, put it onto the battlefield, then shuffle"
}
func (e *searchLibraryToBattlefieldEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// chooseColorEffect lets the controller choose a color and stores it on the source permanent.
type chooseColorEffect struct {
	reason string
}

// ChooseColor creates an effect that asks the controller to choose a color,
// storing the result on the source permanent's ChosenColor field.
func ChooseColor(reason string) Effect {
	return &chooseColorEffect{reason: reason}
}

func (e *chooseColorEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return nil
	}
	perm.ChosenColor = p.ChooseManaColor(e.reason)
	return nil
}

func (e *chooseColorEffect) Text() string              { return "choose a color" }
func (e *chooseColorEffect) Properties() EffectProperties { return EffectProperties{} }
