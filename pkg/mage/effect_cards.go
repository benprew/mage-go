package mage

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
)

// drawCardsTargetEffect draws cards for a player. By default the player is read
// from ctx.Targets[0] (with controller as fallback). Use Targeting to override
// with a PlayerSelector.
type drawCardsTargetEffect struct {
	props  EffectProperties
	amount ValueSource
	sel    PlayerSelector
}

// DrawCards creates an effect that draws cards for a target player (or controller
// as fallback). Chain .Targeting(sel) to pick the player(s) via a PlayerSelector.
func DrawCards(amount ValueSource) TargetedEffect {
	drawCount := 0
	if fv, ok := amount.(fixedValue); ok {
		drawCount = fv.n
	}
	return &drawCardsTargetEffect{
		props:  EffectProperties{Outcome: OutcomeBenefit, DrawCount: drawCount},
		amount: amount,
	}
}

// Targeting overrides the default target (ctx.Targets[0]) with a PlayerSelector.
func (e *drawCardsTargetEffect) Targeting(sel PlayerSelector) TargetedEffect {
	e.sel = sel
	return e
}

func (e *drawCardsTargetEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "target player draws X cards"
	}
	return fmt.Sprintf("target player draws %d card(s)", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
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

func (e *drawCardsActivePlayerEffect) Text() string {
	return "that player draws an additional card"
}
func (e *drawCardsActivePlayerEffect) Properties() EffectProperties { return e.props }

// discardCardsEffect forces a player to discard cards. By default the player
// is read from ctx.Targets[0] (the spell's chosen target). Use Targeting to
// override with a PlayerSelector — for example, SelectEachOpponent() for
// "defending player discards" triggers.
type discardCardsEffect struct {
	amount ValueSource
	sel    PlayerSelector
}

// DiscardCards creates an effect that forces a player to discard cards.
// Without Targeting, it acts on ctx.Targets[0]; chain .Targeting(sel) to pick
// the player(s) via a PlayerSelector instead.
func DiscardCards(amount ValueSource) TargetedEffect {
	return &discardCardsEffect{amount: amount}
}

// Targeting overrides the default target (ctx.Targets[0]) with a PlayerSelector.
func (e *discardCardsEffect) Targeting(sel PlayerSelector) TargetedEffect {
	e.sel = sel
	return e
}

func (e *discardCardsEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "target player discards X cards"
	}
	return fmt.Sprintf("target player discards %d card(s)", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *discardCardsEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// discardRandomEffect forces an opponent to discard a card at random.
type discardRandomEffect struct {
	amount int
	sel    PlayerSelector
}

// DiscardRandom creates an effect that forces a target player (or opponent) to discard cards at random.
func DiscardRandom(amount int) TargetedEffect {
	return &discardRandomEffect{amount: amount}
}

func (e *discardRandomEffect) Text() string {
	return fmt.Sprintf("discard %d card(s) at random", e.amount)
}
func (e *discardRandomEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (e *discardRandomEffect) Targeting(sel PlayerSelector) TargetedEffect {
	e.sel = sel
	return e
}

// returnFromGraveyardEffect returns a target creature from graveyard to battlefield.
type returnFromGraveyardEffect struct{}

// ReturnFromGraveyardToBattlefield creates an effect that returns a target creature card
// from the controller's graveyard directly to the battlefield (e.g. Animate Dead, Resurrection).
func ReturnFromGraveyardToBattlefield() Effect {
	return &returnFromGraveyardEffect{}
}

func (e *returnFromGraveyardEffect) Text() string {
	return "return target creature card from your graveyard to the battlefield"
}
func (e *returnFromGraveyardEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// returnSourceFromGraveyardToBattlefieldEffect returns the source card from its
// owner's graveyard directly to the battlefield. Used by graveyard-functional
// triggered abilities like Nether Shadow's "put it onto the battlefield".
type returnSourceFromGraveyardToBattlefieldEffect struct{}

// ReturnSourceFromGraveyardToBattlefield creates an effect that moves the
// source card from its controller's graveyard to the battlefield. If the
// source is no longer in the graveyard at resolution, the effect fizzles.
func ReturnSourceFromGraveyardToBattlefield() Effect {
	return &returnSourceFromGraveyardToBattlefieldEffect{}
}

func (e *returnSourceFromGraveyardToBattlefieldEffect) Text() string {
	return "return ~ from your graveyard to the battlefield"
}
func (e *returnSourceFromGraveyardToBattlefieldEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// returnSourceToHandEffect returns the source card from graveyard to hand.
type returnSourceToHandEffect struct{}

// ReturnSourceToHand creates an effect that returns the source card from the graveyard
// to its owner's hand (e.g. Rancor's triggered ability).
func ReturnSourceToHand() Effect {
	return &returnSourceToHandEffect{}
}

func (e *returnSourceToHandEffect) Text() string {
	return "return this card to its owner's hand"
}
func (e *returnSourceToHandEffect) Properties() EffectProperties { return EffectProperties{} }

// exileSourceFromGraveyardEffect exiles the source card from the graveyard.
type exileSourceFromGraveyardEffect struct{}

// ExileSourceFromGraveyard creates an effect that exiles the source from the graveyard
// (e.g. Cyclopean Mummy's death trigger).
func ExileSourceFromGraveyard() Effect {
	return &exileSourceFromGraveyardEffect{}
}

func (e *exileSourceFromGraveyardEffect) Text() string                 { return "exile this card from graveyard" }
func (e *exileSourceFromGraveyardEffect) Properties() EffectProperties { return EffectProperties{} }

// millTargetPlayerEffect mills N cards from a player's library. By default the
// player is read from ctx.Targets[0]. Use Targeting to override with a
// PlayerSelector.
type millTargetPlayerEffect struct {
	amount ValueSource
	sel    PlayerSelector
}

// MillTargetPlayer creates an effect that mills N cards from target player's library.
// Chain .Targeting(sel) to pick the player(s) via a PlayerSelector.
func MillTargetPlayer(amount ValueSource) *millTargetPlayerEffect {
	return &millTargetPlayerEffect{amount: amount}
}

// Targeting overrides the default target (ctx.Targets[0]) with a PlayerSelector.
func (e *millTargetPlayerEffect) Targeting(sel PlayerSelector) *millTargetPlayerEffect {
	e.sel = sel
	return e
}

func (e *millTargetPlayerEffect) Text() string {
	return "target player mills cards"
}
func (e *millTargetPlayerEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// scryEffect implements Scry N (CR 701.18) for the controller of the effect.
type scryEffect struct {
	amount ValueSource
}

// Scry creates an effect that scries N for the controller (CR 701.18):
// "Look at the top N cards of your library, then put any number of them on the
// bottom of your library and the rest on top in any order."
func Scry(amount ValueSource) Effect {
	return DataEffect(&scryEffect{amount: amount})
}

func (e *scryEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "scry X"
	}
	return fmt.Sprintf("scry %d", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *scryEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// returnToHandTargetEffect bounces a target permanent to its owner's hand.
type returnToHandTargetEffect struct{}

// ReturnToHandTarget creates an effect that bounces a target permanent to its owner's hand.
func ReturnToHandTarget() Effect {
	return &returnToHandTargetEffect{}
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

func (e *discardHandAndDrawEffect) Text() string {
	return fmt.Sprintf("Each player discards their hand, then draws %d cards", e.drawCount)
}
func (e *discardHandAndDrawEffect) Properties() EffectProperties { return EffectProperties{} }

// shuffleHandAndGraveyardIntoLibraryAndDrawEffect shuffles each player's hand
// and graveyard into their library, then each player draws N cards.
type shuffleHandAndGraveyardIntoLibraryAndDrawEffect struct {
	drawCount int
}

// ShuffleHandAndGraveyardIntoLibraryAndDraw creates an effect where each player shuffles their
// hand and graveyard into their library, then draws n cards (e.g. Timetwister).
func ShuffleHandAndGraveyardIntoLibraryAndDraw(n int) Effect {
	return &shuffleHandAndGraveyardIntoLibraryAndDrawEffect{drawCount: n}
}

func (e *shuffleHandAndGraveyardIntoLibraryAndDrawEffect) Text() string {
	return fmt.Sprintf("Each player shuffles their hand and graveyard into their library, then draws %d cards", e.drawCount)
}
func (e *shuffleHandAndGraveyardIntoLibraryAndDrawEffect) Properties() EffectProperties {
	return EffectProperties{}
}

// shuffleLibraryEffect shuffles the controller's library.
type shuffleLibraryEffect struct{}

// ShuffleLibrary creates an effect that shuffles the controller's library.
func ShuffleLibrary() Effect {
	return &shuffleLibraryEffect{}
}

func (e *shuffleLibraryEffect) Text() string                 { return "shuffle your library" }
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

func (e *chooseColorEffect) Text() string                 { return "choose a color" }
func (e *chooseColorEffect) Properties() EffectProperties { return EffectProperties{} }

// --- Executor functions ---

func execDrawCardsTarget(ctx *EffectContext, e *drawCardsTargetEffect) error {
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)

	var players []Player
	if e.sel != nil {
		for _, pid := range e.sel.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets) {
			if p := ctx.Game.GetPlayer(pid); p != nil {
				players = append(players, p)
			}
		}
	} else {
		// Default: targets[0] if it resolves to a player; otherwise controller.
		// Triggers often pass a permanent/spell ID in Targets, so we fall back.
		var p Player
		if len(ctx.Targets) > 0 {
			p = ctx.Game.GetPlayer(ctx.Targets[0])
		}
		if p == nil {
			p = ctx.Game.GetPlayer(ctx.Controller)
		}
		if p == nil {
			return ErrPlayerNotFound
		}
		players = []Player{p}
	}

	for _, p := range players {
		for range amount {
			ctx.Game.PlayerDrawCard(p)
		}
	}
	return nil
}

func execDrawCardsActivePlayer(ctx *EffectContext, e *drawCardsActivePlayerEffect) error {
	active := ctx.Game.ActivePlayerObj()
	if active == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, active.PlayerID(), ctx.Targets)
	for range amount {
		ctx.Game.PlayerDrawCard(active)
	}
	return nil
}

func execDiscardCards(ctx *EffectContext, e *discardCardsEffect) error {
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)

	var playerIDs []uuid.UUID
	switch {
	case e.sel != nil:
		playerIDs = e.sel.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	case len(ctx.Targets) > 0:
		playerIDs = []uuid.UUID{ctx.Targets[0]}
	default:
		if opp := ctx.Game.GetOpponent(ctx.Controller); opp != nil {
			playerIDs = []uuid.UUID{opp.PlayerID()}
		}
	}

	for _, pid := range playerIDs {
		p := ctx.Game.GetPlayer(pid)
		if p == nil {
			continue
		}
		chosen := p.ChooseCardsFromHand(amount, "discard", ctx.Game)
		for _, card := range chosen {
			ctx.Game.PlayerDiscard(p, card.ID())
		}
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
		ctx.Game.PlayerDiscard(targetPlayer, hand[idx].ID())
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
	baseAmount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)

	var playerIDs []uuid.UUID
	switch {
	case e.sel != nil:
		playerIDs = e.sel.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	case len(ctx.Targets) > 0:
		playerIDs = []uuid.UUID{ctx.Targets[0]}
	}

	for _, pid := range playerIDs {
		p := ctx.Game.GetPlayer(pid)
		if p == nil {
			continue
		}
		amount := ctx.Game.ApplyMillModifiers(p.PlayerID(), baseAmount)
		lib := p.Library()
		for i := 0; i < amount && len(lib) > 0; i++ {
			card := lib[len(lib)-1]
			lib = lib[:len(lib)-1]
			p.AddToGraveyard(card)
		}
		p.SetLibrary(lib)
	}
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

func execReturnSourceFromGraveyardToBattlefield(ctx *EffectContext, _ *returnSourceFromGraveyardToBattlefieldEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	card, ok := p.RemoveFromGraveyard(ctx.SourceID)
	if !ok {
		return nil // source no longer in graveyard — fizzle
	}
	ctx.Game.PutOnBattlefield(card, ctx.Controller)
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
	ctx.Game.BouncePermanentToHand(perm)
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
	for _, tid := range ctx.Targets {
		card, ok := p.RemoveFromGraveyard(tid)
		if !ok {
			continue
		}
		p.AddToHand(card)
	}
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
			ctx.Game.PlayerDiscard(p, c.ID())
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

func execScry(ctx *EffectContext, e *scryEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	n := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if n <= 0 {
		return nil
	}
	ctx.Game.PerformScry(p, n)
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
