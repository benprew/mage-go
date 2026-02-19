package mage

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
)

// Effect represents a one-shot effect that resolves.
type Effect interface {
	Apply(g *Game, sourceID uuid.UUID, controller uuid.UUID, targets []uuid.UUID) error
	Text() string
}

// funcEffect wraps an anonymous function as an Effect. Use FuncEffect to create
// one-off effects inline in card definitions without needing a dedicated struct.
type funcEffect struct {
	text string
	fn   func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error
}

// FuncEffect creates an Effect from an anonymous function. This is ideal for
// card-specific effects that are used by only one card and don't warrant a
// dedicated type. The text parameter is used for Text() (rules text display).
func FuncEffect(text string, fn func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error) Effect {
	return &funcEffect{text: text, fn: fn}
}

func (e *funcEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	return e.fn(g, sourceID, controller, targets)
}

func (e *funcEffect) Text() string { return e.text }

// gainLifeEffect gains life for the controller.
type gainLifeEffect struct {
	amount int
}

// GainLife creates an effect that gains life for the controller.
func GainLife(amount int) Effect {
	return &gainLifeEffect{amount: amount}
}

func (e *gainLifeEffect) Apply(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	g.PlayerGainLife(p, e.amount)
	if !g.Effects.IsLichActive(controller) {
		g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: controller, Amount: e.amount})
	}
	return nil
}

func (e *gainLifeEffect) Text() string {
	return fmt.Sprintf("gain %d life", e.amount)
}

// addCountersEffect adds counters to the source or a target permanent.
type addCountersEffect struct {
	ct     CounterType
	amount ValueSource
	target PermanentSelector
}

// AddCounters creates an effect that adds counters to the selected permanent.
func AddCounters(ct CounterType, amount ValueSource, target PermanentSelector) Effect {
	return &addCountersEffect{ct: ct, amount: amount, target: target}
}

func (e *addCountersEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var perm *Permanent
	if e.target == SelectSource {
		perm = g.FindPermanent(sourceID)
	} else {
		if len(targets) == 0 {
			return fmt.Errorf("no target for counters")
		}
		perm = g.FindPermanent(targets[0])
	}
	if perm == nil {
		return nil
	}
	amount := e.amount.Resolve(g, sourceID, controller)
	if amount > 0 {
		perm.AddCounter(e.ct, amount)
	}
	return nil
}

func (e *addCountersEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return fmt.Sprintf("put X %s counters on it", e.ct)
	}
	n := e.amount.Resolve(nil, uuid.Nil, uuid.Nil)
	if e.target == SelectSource {
		return fmt.Sprintf("put %d %s counter(s) on it", n, e.ct)
	}
	return fmt.Sprintf("put %d %s counter(s) on target", n, e.ct)
}

// cloneTargetEffect copies target permanent's characteristics onto the source
// card. If additionalTypes are provided, they are added after cloning (e.g.
// Copy Artifact adds TypeEnchantment). Target filtering is handled by the
// Target passed to NewTargetedSpell, not by this effect.
type cloneTargetEffect struct {
	additionalTypes []CardType
}

// CloneTarget creates an effect that copies a target permanent's characteristics onto the source.
// Additional types (e.g. TypeEnchantment for Copy Artifact) are added after cloning.
func CloneTarget(additionalTypes ...CardType) Effect {
	return &cloneTargetEffect{additionalTypes: additionalTypes}
}

func (e *cloneTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 || g.ResolvingCard == nil {
		return nil
	}
	target := g.FindPermanent(targets[0])
	if target == nil {
		return nil
	}
	g.ResolvingCard.CloneFrom(target.Card)
	for _, t := range e.additionalTypes {
		g.ResolvingCard.AddType(t)
	}
	return nil
}

func (e *cloneTargetEffect) Text() string {
	return "enters the battlefield as a copy of target permanent"
}

// removeCountersFromSourceEffect removes counters from the source permanent.
type removeCountersFromSourceEffect struct {
	ct     CounterType
	amount int
}

// RemoveCountersFromSource creates an effect that removes counters from the source permanent.
func RemoveCountersFromSource(ct CounterType, amount int) Effect {
	return &removeCountersFromSourceEffect{ct: ct, amount: amount}
}

func (e *removeCountersFromSourceEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return nil
	}
	p.RemoveCounter(e.ct, e.amount)
	return nil
}

func (e *removeCountersFromSourceEffect) Text() string {
	return fmt.Sprintf("remove %d %s counter(s) from it", e.amount, e.ct)
}

// dealDamageEffect deals damage to a target (creature, player, or planeswalker).
type dealDamageEffect struct {
	amount ValueSource
}

// DealDamage creates an effect that deals damage to a target.
func DealDamage(amount ValueSource) Effect {
	return &dealDamageEffect{amount: amount}
}

func (e *dealDamageEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for damage")
	}
	amount := e.amount.Resolve(g, sourceID, controller)
	if amount <= 0 {
		return nil
	}
	targetID := targets[0]

	// Check if target is a player
	for _, pl := range g.Players {
		if pl.PlayerID() == targetID {
			g.DealDamageToPlayer(pl, amount, sourceID)
			return nil
		}
	}

	// Otherwise target is a permanent
	perm := g.FindPermanent(targetID)
	if perm == nil {
		return nil // target gone, fizzle
	}

	// Check protection
	sourceCard := g.FindCardAnywhere(sourceID)
	if sourceCard != nil && perm.HasProtectionFrom(sourceCard) {
		return nil // damage prevented by protection
	}

	g.DealDamageToPermanent(perm, amount, sourceID)
	return nil
}

func (e *dealDamageEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "deal X damage to target"
	}
	return fmt.Sprintf("deal %d damage to target", e.amount.Resolve(nil, uuid.Nil, uuid.Nil))
}

// destroyTargetEffect destroys the target permanent.
type destroyTargetEffect struct{}

// DestroyTarget creates an effect that destroys the first target permanent.
func DestroyTarget() Effect {
	return &destroyTargetEffect{}
}

func (e *destroyTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for destroy")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil // target gone, fizzle
	}
	if perm.HasAbility(Indestructible) {
		return nil
	}
	g.DestroyPermanent(perm)
	return nil
}

func (e *destroyTargetEffect) Text() string { return "destroy target" }

// returnFromGraveyardEffect returns a target creature from graveyard to battlefield.
type returnFromGraveyardEffect struct{}

// ReturnFromGraveyardToBattlefield creates an effect that returns a target creature card
// from the controller's graveyard directly to the battlefield (e.g. Animate Dead, Resurrection).
func ReturnFromGraveyardToBattlefield() Effect {
	return &returnFromGraveyardEffect{}
}

func (e *returnFromGraveyardEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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

// returnSourceToHandEffect returns the source card from graveyard to hand.
type returnSourceToHandEffect struct{}

// ReturnSourceToHand creates an effect that returns the source card from the graveyard
// to its owner's hand (e.g. Rancor's triggered ability).
func ReturnSourceToHand() Effect {
	return &returnSourceToHandEffect{}
}

func (e *returnSourceToHandEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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

// DestroyAllCreatures destroys all creatures (board wipe).
// This is a convenience alias for DestroyAllMatching with the IsCreature filter.
func DestroyAllCreatures() Effect {
	return DestroyAllMatching(IsCreature, "destroy all creatures")
}

// compositeEffect applies multiple effects in sequence.
type compositeEffect struct {
	effects []Effect
	text    string
}

// CompositeEffects creates an effect that applies multiple effects in sequence.
func CompositeEffects(text string, effects ...Effect) Effect {
	return &compositeEffect{effects: effects, text: text}
}

func (e *compositeEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, eff := range e.effects {
		if err := eff.Apply(g, sourceID, controller, targets); err != nil {
			return err
		}
	}
	return nil
}

func (e *compositeEffect) Text() string { return e.text }

// attachToTargetEffect attaches the source (aura/equipment) to the target.
type attachToTargetEffect struct{}

// AttachToTarget creates an effect that attaches the source (aura or equipment) to the target permanent.
func AttachToTarget() Effect {
	return &attachToTargetEffect{}
}

func (e *attachToTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for attach")
	}
	g.Attach(sourceID, targets[0])
	return nil
}

func (e *attachToTargetEffect) Text() string { return "attach to target" }

// returnToHandTargetEffect bounces a target permanent to its owner's hand.
type returnToHandTargetEffect struct{}

// ReturnToHandTarget creates an effect that bounces a target permanent to its owner's hand.
func ReturnToHandTarget() Effect {
	return &returnToHandTargetEffect{}
}

func (e *returnToHandTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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

// returnFromGraveyardToHandTargetEffect returns a target card from graveyard to hand.
type returnFromGraveyardToHandTargetEffect struct{}

// ReturnFromGraveyardToHandTarget creates an effect that returns a target card
// from the controller's graveyard to their hand (e.g. Raise Dead, Regrowth).
func ReturnFromGraveyardToHandTarget() Effect {
	return &returnFromGraveyardToHandTargetEffect{}
}

func (e *returnFromGraveyardToHandTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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

// boostMatchingUntilEndOfTurnEffect boosts the P/T of all creatures matching a predicate until end of turn
type boostMatchingUntilEndOfTurnEffect struct {
	power     ValueSource
	toughness ValueSource
	predicate PermanentFilter
}

// BoostMatchingUntilEndOfTurn creates an effect that gives +P/+T until end of turn to all
// creatures the controller owns that match the predicate (e.g. Crusade, Bad Moon).
func BoostMatchingUntilEndOfTurn(power, toughness ValueSource, predicate PermanentFilter) Effect {
	return &boostMatchingUntilEndOfTurnEffect{power: power, toughness: toughness, predicate: predicate}
}

func (e *boostMatchingUntilEndOfTurnEffect) Apply(g *Game, sourceID uuid.UUID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, perm := range g.FilterBattlefield(And(ControlledBy(controller), IsCreature, e.predicate)) {
		p := e.power.Resolve(g, sourceID, controller)
		t := e.toughness.Resolve(g, sourceID, controller)
		eff := TemporaryBoost(perm.ID(), p, t)
		eff.SetSourceID(sourceID)
		g.Effects.Add(eff)
	}
	g.Effects.Apply(g)
	return nil
}

func (e *boostMatchingUntilEndOfTurnEffect) Text() string {
	return "XXX populate filter predicate text"
}

// boostUntilEndOfTurnEffect boosts a creature's P/T until end of turn.
type boostUntilEndOfTurnEffect struct {
	power     ValueSource
	toughness ValueSource
	target    PermanentSelector
}

// BoostUntilEndOfTurn creates an effect that boosts the selected creature's P/T until end of turn.
func BoostUntilEndOfTurn(power, toughness ValueSource, target PermanentSelector) Effect {
	return &boostUntilEndOfTurnEffect{power: power, toughness: toughness, target: target}
}

func (e *boostUntilEndOfTurnEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var perm *Permanent
	if e.target == SelectSource {
		perm = g.FindPermanent(sourceID)
	} else {
		if len(targets) == 0 {
			return fmt.Errorf("no target for boost")
		}
		perm = g.FindPermanent(targets[0])
	}
	if perm == nil {
		return nil
	}
	p := e.power.Resolve(g, sourceID, controller)
	t := e.toughness.Resolve(g, sourceID, controller)
	eff := TemporaryBoost(perm.ID(), p, t)
	eff.SetSourceID(sourceID)
	g.Effects.Add(eff)
	g.Effects.Apply(g)
	return nil
}

func (e *boostUntilEndOfTurnEffect) Text() string {
	_, pIsX := e.power.(xValue)
	_, tIsX := e.toughness.(xValue)
	if pIsX || tIsX {
		return "Target creature gets +X/+0 until end of turn"
	}
	p := e.power.Resolve(nil, uuid.Nil, uuid.Nil)
	t := e.toughness.Resolve(nil, uuid.Nil, uuid.Nil)
	if e.target == SelectSource {
		return fmt.Sprintf("this creature gets +%d/+%d until end of turn", p, t)
	}
	return fmt.Sprintf("target creature gets +%d/+%d until end of turn", p, t)
}

// markDestroyAtEOTAfterNActivationsEffect tracks pump activations using Charge
// counters. When the count reaches the threshold, sets DestroyAtEndOfTurn.
type markDestroyAtEOTAfterNActivationsEffect struct {
	threshold int
}

// MarkDestroyAtEOTAfterNActivations creates an effect that tracks activations via Charge counters.
// When the count reaches the threshold, the source is destroyed at end of turn (e.g. Basalt Monolith variant).
func MarkDestroyAtEOTAfterNActivations(threshold int) Effect {
	return &markDestroyAtEOTAfterNActivationsEffect{threshold: threshold}
}

func (e *markDestroyAtEOTAfterNActivationsEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return nil
	}
	perm.AddCounter(Charge, 1)
	if perm.Counters[Charge] >= e.threshold {
		g.RegisterDelayedTrigger(&DelayedTrigger{
			EventType:  EvtEndStep,
			TargetID:   perm.ID(),
			Effects:    []Effect{DestroyTarget()},
			SourceID:   sourceID,
			Controller: controller,
		})
	}
	return nil
}

func (e *markDestroyAtEOTAfterNActivationsEffect) Text() string {
	return fmt.Sprintf("if activated %d+ times, destroy at end of turn", e.threshold)
}

// destroyTargetPermanentEffect destroys a target permanent matching a filter.
type destroyTargetPermanentEffect struct {
	text string
}

// DestroyTargetPermanent creates an effect that destroys a target permanent.
func DestroyTargetPermanent() Effect {
	return &destroyTargetPermanentEffect{text: "destroy target permanent"}
}

// DestroyTargetLand creates an effect that destroys a target land (e.g. Stone Rain, Sinkhole).
func DestroyTargetLand() Effect {
	return &destroyTargetPermanentEffect{text: "destroy target land"}
}

// DestroyTargetArtifact creates an effect that destroys a target artifact (e.g. Shatter).
func DestroyTargetArtifact() Effect {
	return &destroyTargetPermanentEffect{text: "destroy target artifact"}
}

func (e *destroyTargetPermanentEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for destroy")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	if perm.HasAbility(Indestructible) {
		return nil
	}
	g.DestroyPermanent(perm)
	return nil
}

func (e *destroyTargetPermanentEffect) Text() string { return e.text }

// DestroyAllLands destroys all lands (Armageddon).
// This is a convenience alias for DestroyAllMatching with the IsLand filter.
func DestroyAllLands() Effect {
	return DestroyAllMatching(IsLand, "destroy all lands")
}

// DestroyAllEnchantments destroys all enchantments (Tranquility).
// This is a convenience alias for DestroyAllMatching with the IsEnchantment filter.
func DestroyAllEnchantments() Effect {
	return DestroyAllMatching(IsEnchantment, "destroy all enchantments")
}

// destroyAllMatchingEffect destroys all permanents matching a filter.
type destroyAllMatchingEffect struct {
	filter PermanentFilter
	text   string
}

// DestroyAllMatching creates an effect that destroys all permanents matching the filter.
// The text parameter is used for rules text display.
func DestroyAllMatching(filter PermanentFilter, text string) Effect {
	return &destroyAllMatchingEffect{filter: filter, text: text}
}

func (e *destroyAllMatchingEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	toDestroy := g.FilterBattlefield(And(e.filter, Not(HasKeywordFilter(Indestructible))))
	for _, p := range toDestroy {
		g.DestroyPermanent(p)
	}
	return nil
}

func (e *destroyAllMatchingEffect) Text() string { return e.text }

// tapTargetEffect taps a target permanent.
type tapTargetEffect struct{}

// TapTarget creates an effect that taps a target permanent.
func TapTarget() Effect {
	return &tapTargetEffect{}
}

func (e *tapTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for tap")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	perm.Tapped = true
	return nil
}

func (e *tapTargetEffect) Text() string { return "tap target permanent" }

// untapTargetEffect untaps a target permanent.
type untapTargetEffect struct{}

// UntapTarget creates an effect that untaps a target permanent.
func UntapTarget() Effect {
	return &untapTargetEffect{}
}

func (e *untapTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for untap")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	perm.Tapped = false
	return nil
}

func (e *untapTargetEffect) Text() string { return "untap target permanent" }

// discardRandomEffect forces an opponent to discard a card at random.
type discardRandomEffect struct {
	amount int
}

// DiscardRandom creates an effect that forces a target player (or opponent) to discard cards at random.
func DiscardRandom(amount int) Effect {
	return &discardRandomEffect{amount: amount}
}

func (e *discardRandomEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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

// discardCardsEffect forces a target player to discard cards.
type discardCardsEffect struct {
	amount ValueSource
}

// DiscardCards creates an effect that forces a target player to discard cards.
func DiscardCards(amount ValueSource) Effect {
	return &discardCardsEffect{amount: amount}
}

func (e *discardCardsEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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

// addManaEffect adds mana to the controller's pool.
type addManaEffect struct {
	color  Color
	amount int
}

// AddMana creates an effect that adds mana of the given color to the controller's pool.
func AddMana(color Color, amount int) Effect {
	return &addManaEffect{color: color, amount: amount}
}

func (e *addManaEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	p.ManaPool().Add(e.color, e.amount)
	return nil
}

func (e *addManaEffect) Text() string {
	return fmt.Sprintf("add %d %s mana", e.amount, e.color)
}

// addAnyManaEffect adds mana of any one color to the controller's pool.
type addAnyManaEffect struct {
	amount int
}

// AddAnyMana creates an effect that adds mana of any one color (player chooses) to the controller's pool.
func AddAnyMana(amount int, _ Color) Effect {
	return &addAnyManaEffect{amount: amount}
}

func (e *addAnyManaEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	color := p.ChooseManaColor("add mana")
	p.ManaPool().Add(color, e.amount)
	return nil
}

func (e *addAnyManaEffect) Text() string {
	return fmt.Sprintf("add %d mana of any one color", e.amount)
}

// drawCardsTargetEffect draws cards for a target player (or controller as fallback).
type drawCardsTargetEffect struct {
	amount ValueSource
}

// DrawCards creates an effect that draws cards for a target player (or controller as fallback).
func DrawCards(amount ValueSource) Effect {
	return &drawCardsTargetEffect{amount: amount}
}

func (e *drawCardsTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
		targetPlayer.DrawCard()
	}
	return nil
}

func (e *drawCardsTargetEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "target player draws X cards"
	}
	return fmt.Sprintf("target player draws %d card(s)", e.amount.Resolve(nil, uuid.Nil, uuid.Nil))
}

// drawCardsActivePlayerEffect draws cards for the active player (e.g. Howling Mine).
type drawCardsActivePlayerEffect struct {
	amount ValueSource
}

// DrawCardsActivePlayer creates an effect that draws cards for the active player.
func DrawCardsActivePlayer(amount ValueSource) Effect {
	return &drawCardsActivePlayerEffect{amount: amount}
}

func (e *drawCardsActivePlayerEffect) Apply(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
	active := g.ActivePlayerObj()
	if active == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.Resolve(g, sourceID, active.PlayerID())
	for i := 0; i < amount; i++ {
		active.DrawCard()
	}
	return nil
}

func (e *drawCardsActivePlayerEffect) Text() string {
	return "that player draws an additional card"
}

// exileTargetEffect exiles a target permanent (removes from game).
type exileTargetEffect struct{}

// ExileTarget creates an effect that exiles a target permanent (e.g. Swords to Plowshares).
func ExileTarget() Effect {
	return &exileTargetEffect{}
}

func (e *exileTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for exile")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	g.ExilePermanent(perm)
	return nil
}

func (e *exileTargetEffect) Text() string { return "exile target permanent" }

// gainLifeTargetEffect gains life for a target player (or controller as fallback).
type gainLifeTargetEffect struct {
	amount ValueSource
}

// GainLifeTarget creates an effect that gains life for a target player (or controller as fallback).
func GainLifeTarget(amount ValueSource) Effect {
	return &gainLifeTargetEffect{amount: amount}
}

func (e *gainLifeTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
	g.PlayerGainLife(targetPlayer, amount)
	if !g.Effects.IsLichActive(targetPlayer.PlayerID()) {
		g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: targetPlayer.PlayerID(), Amount: amount})
	}
	return nil
}

func (e *gainLifeTargetEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "target player gains X life"
	}
	return fmt.Sprintf("target player gains %d life", e.amount.Resolve(nil, uuid.Nil, uuid.Nil))
}

// loseLifeEffect causes the controller to lose life.
type loseLifeEffect struct {
	amount int
}

// LoseLife creates an effect that causes the controller to lose life.
func LoseLife(amount int) Effect {
	return &loseLifeEffect{amount: amount}
}

func (e *loseLifeEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	p.LoseLife(e.amount)
	g.FireEvent(GameEvent{Type: EvtLifeLost, PlayerID: controller, Amount: e.amount})
	return nil
}

func (e *loseLifeEffect) Text() string {
	return fmt.Sprintf("you lose %d life", e.amount)
}

// dealDamageToAllCreaturesEffect deals damage to all creatures matching an optional filter.
type dealDamageToAllCreaturesEffect struct {
	amount ValueSource
	filter PermanentFilter
}

// DealDamageToAllCreatures creates an effect that deals damage to all matching creatures.
func DealDamageToAllCreatures(amount ValueSource, filter PermanentFilter) Effect {
	return &dealDamageToAllCreaturesEffect{amount: amount, filter: filter}
}

func (e *dealDamageToAllCreaturesEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	amount := e.amount.Resolve(g, sourceID, controller)
	if amount <= 0 {
		return nil
	}
	f := IsCreature
	if e.filter != nil {
		f = And(IsCreature, e.filter)
	}
	for _, p := range g.FilterBattlefield(f) {
		g.DealDamageToPermanent(p, amount, sourceID)
	}
	return nil
}

func (e *dealDamageToAllCreaturesEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "deal X damage to each creature"
	}
	return fmt.Sprintf("deal %d damage to each creature", e.amount.Resolve(nil, uuid.Nil, uuid.Nil))
}

// dealDamageToPlayersEffect deals damage to players selected by a PlayerSelector.
type dealDamageToPlayersEffect struct {
	amount   ValueSource
	selector PlayerSelector
}

// DealDamageToPlayers creates an effect that deals damage to players selected by the selector.
func DealDamageToPlayers(amount ValueSource, selector PlayerSelector) Effect {
	return &dealDamageToPlayersEffect{amount: amount, selector: selector}
}

func (e *dealDamageToPlayersEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	amount := e.amount.Resolve(g, sourceID, controller)
	if amount <= 0 {
		return nil
	}
	playerIDs := e.selector.Select(g, sourceID, controller, targets)
	for _, pid := range playerIDs {
		p := g.GetPlayer(pid)
		if p != nil {
			g.DealDamageToPlayer(p, amount, sourceID)
		}
	}
	return nil
}

func (e *dealDamageToPlayersEffect) Text() string {
	return fmt.Sprintf("deal %s damage to %s", e.amount.Text(), e.selector.Text())
}

// sacrificeSourceEffect sacrifices the source permanent.
type sacrificeSourceEffect struct{}

// SacrificeSource creates an effect that sacrifices the source permanent.
func SacrificeSource() Effect {
	return &sacrificeSourceEffect{}
}

func (e *sacrificeSourceEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return nil
	}
	g.Sacrifice(perm)
	return nil
}

func (e *sacrificeSourceEffect) Text() string { return "sacrifice this permanent" }

// searchLibraryEffect lets the controller search their library and put a card in hand.
type searchLibraryEffect struct{}

// SearchLibraryToHand creates an effect that lets the controller search their library for a card
// and put it into their hand (e.g. Demonic Tutor).
func SearchLibraryToHand() Effect {
	return &searchLibraryEffect{}
}

func (e *searchLibraryEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
	p.AddToHand(card)
	return nil
}

func (e *searchLibraryEffect) Text() string {
	return "search your library for a card and put it into your hand"
}

// counterSpellEffect counters a target spell on the stack.
type counterSpellEffect struct{}

// CounterSpell creates an effect that counters a target spell on the stack.
func CounterSpell() Effect {
	return &counterSpellEffect{}
}

func (e *counterSpellEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target spell to counter")
	}
	g.CounterSpellOnStack(targets[0])
	return nil
}

func (e *counterSpellEffect) Text() string { return "counter target spell" }

// preventAllCombatDamageEffect prevents all combat damage this turn (Fog).
type preventAllCombatDamageEffect struct{}

// PreventAllCombatDamage creates an effect that prevents all combat damage this turn (e.g. Fog).
func PreventAllCombatDamage() Effect {
	return &preventAllCombatDamageEffect{}
}

func (e *preventAllCombatDamageEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	g.PreventCombatDamage = true
	return nil
}

func (e *preventAllCombatDamageEffect) Text() string {
	return "prevent all combat damage that would be dealt this turn"
}

// extraTurnEffect gives the controller an extra turn.
type extraTurnEffect struct{}

// ExtraTurn creates an effect that gives the controller an extra turn (e.g. Time Walk).
func ExtraTurn() Effect {
	return &extraTurnEffect{}
}

func (e *extraTurnEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	g.ExtraTurns = append(g.ExtraTurns, controller)
	return nil
}

func (e *extraTurnEffect) Text() string { return "take an extra turn after this one" }

// controlChangeTargetEffect gains control of a target permanent.
type controlChangeTargetEffect struct{}

// ControlChangeTarget creates an effect that gives the controller permanent control of a target (e.g. Control Magic).
func ControlChangeTarget() Effect {
	return &controlChangeTargetEffect{}
}

func (e *controlChangeTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for control change")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	perm.Controller = controller
	return nil
}

func (e *controlChangeTargetEffect) Text() string { return "gain control of target permanent" }

// grantKeywordUntilEndOfTurnEffect grants a keyword to the source or a target
// creature until end of turn.
type grantKeywordUntilEndOfTurnEffect struct {
	keyword Keyword
	target  PermanentSelector
}

// GrantKeywordUntilEndOfTurn creates an effect that grants a keyword to the selected creature until end of turn.
func GrantKeywordUntilEndOfTurn(kw Keyword, target PermanentSelector) Effect {
	return &grantKeywordUntilEndOfTurnEffect{keyword: kw, target: target}
}

func (e *grantKeywordUntilEndOfTurnEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var perm *Permanent
	if e.target == SelectSource {
		perm = g.FindPermanent(sourceID)
	} else {
		if len(targets) == 0 {
			return fmt.Errorf("no target for keyword grant")
		}
		perm = g.FindPermanent(targets[0])
	}
	if perm == nil {
		return nil
	}
	eff := TemporaryKeyword(perm.ID(), e.keyword)
	eff.SetSourceID(sourceID)
	g.Effects.Add(eff)
	g.Effects.Apply(g)
	return nil
}

func (e *grantKeywordUntilEndOfTurnEffect) Text() string {
	if e.target == SelectSource {
		return fmt.Sprintf("~ gains %s until end of turn", e.keyword)
	}
	return fmt.Sprintf("target creature gains %s until end of turn", e.keyword)
}

// regenerateSourceEffect sets a regeneration shield on the source.
type regenerateSourceEffect struct{}

// RegenerateSource creates an effect that sets a regeneration shield on the source permanent.
func RegenerateSource() Effect {
	return &regenerateSourceEffect{}
}

func (e *regenerateSourceEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return nil
	}
	g.Effects.AddRegenerationShield(sourceID)
	return nil
}

func (e *regenerateSourceEffect) Text() string { return "Regenerate ~" }

// regenerateTargetEffect sets a regeneration shield on the target.
type regenerateTargetEffect struct{}

// RegenerateTarget creates an effect that sets a regeneration shield on a target creature.
func RegenerateTarget() Effect {
	return &regenerateTargetEffect{}
}

func (e *regenerateTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	g.Effects.AddRegenerationShield(perm.ID())
	return nil
}

func (e *regenerateTargetEffect) Text() string { return "Regenerate target creature" }

// preventDamageToTargetEffect sets a damage prevention shield on a target.
type preventDamageToTargetEffect struct {
	amount ValueSource
}

// PreventDamageToTarget creates an effect that prevents damage to a target.
func PreventDamageToTarget(amount ValueSource) Effect {
	return &preventDamageToTargetEffect{amount: amount}
}

func (e *preventDamageToTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	amount := e.amount.Resolve(g, sourceID, controller)
	perm := g.FindPermanent(targets[0])
	if perm != nil {
		g.Effects.AddPreventionShield(perm.ID(), amount)
		return nil
	}
	// Could also prevent damage to player - not implemented yet
	return nil
}

func (e *preventDamageToTargetEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "Prevent the next X damage to target"
	}
	return fmt.Sprintf("Prevent the next %d damage to target", e.amount.Resolve(nil, uuid.Nil, uuid.Nil))
}

// sacrificeOrDamageEffect sacrifices a creature you control, or deals damage
// to the source's controller if no creature is available.
type sacrificeOrDamageEffect struct {
	damage int
}

// SacrificeCreatureOrDamage creates an effect where the controller sacrifices a creature,
// or takes damage if no creature is available (e.g. Lord of the Pit upkeep).
func SacrificeCreatureOrDamage(damage int) Effect {
	return &sacrificeOrDamageEffect{damage: damage}
}

func (e *sacrificeOrDamageEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	// Collect creatures that can be sacrificed (not the source itself)
	candidates := g.FilterBattlefield(And(ControlledBy(controller), IsCreature, NotID(sourceID)))
	if len(candidates) > 0 {
		player := g.GetPlayer(controller)
		chosen := player.ChoosePermanent(candidates, "sacrifice", g)
		if chosen != nil {
			g.Sacrifice(chosen)
			return nil
		}
	}
	// No creature available - deal damage to controller
	player := g.GetPlayer(controller)
	if player != nil {
		g.DealDamageToPlayer(player, e.damage, sourceID)
	}
	return nil
}

func (e *sacrificeOrDamageEffect) Text() string {
	return fmt.Sprintf("Sacrifice a creature or take %d damage", e.damage)
}

// doubleSourcePowerEffect doubles the source creature's power until end of turn.
type doubleSourcePowerEffect struct{}

// DoubleTargetPower creates an effect that doubles a target creature's power until end of turn (e.g. Berserk).
func DoubleTargetPower() Effect {
	return &doubleSourcePowerEffect{}
}

func (e *doubleSourcePowerEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	currentPower := perm.CurrentPower(g)
	eff := TemporaryBoost(perm.ID(), currentPower, 0)
	eff.SetSourceID(sourceID)
	g.Effects.Add(eff)
	g.Effects.Apply(g)
	return nil
}

func (e *doubleSourcePowerEffect) Text() string {
	return "Target creature's power is doubled until end of turn"
}

// destroyTargetAtEndOfTurnEffect marks a creature for destruction at end of turn.
type destroyTargetAtEndOfTurnEffect struct{}

// DestroyTargetAtEndOfTurn creates an effect that registers a delayed trigger to destroy
// the target creature at the next end step.
func DestroyTargetAtEndOfTurn() Effect {
	return &destroyTargetAtEndOfTurnEffect{}
}

func (e *destroyTargetAtEndOfTurnEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	g.RegisterDelayedTrigger(&DelayedTrigger{
		EventType:  EvtEndStep,
		TargetID:   perm.ID(),
		Effects:    []Effect{DestroyTarget()},
		SourceID:   sourceID,
		Controller: controller,
	})
	return nil
}

func (e *destroyTargetAtEndOfTurnEffect) Text() string {
	return "Destroy target creature at end of turn"
}

// discardHandAndDrawEffect makes each player discard their hand and draw N cards.
type discardHandAndDrawEffect struct {
	drawCount int
}

// DiscardHandAndDraw creates an effect where each player discards their hand then draws n cards (e.g. Timetwister, Wheel of Fortune).
func DiscardHandAndDraw(n int) Effect {
	return &discardHandAndDrawEffect{drawCount: n}
}

func (e *discardHandAndDrawEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, p := range g.Players {
		// Discard entire hand
		hand := p.Hand()
		for _, c := range hand {
			p.RemoveFromHand(c.ID())
			p.AddToGraveyard(c)
		}
		// Draw N cards
		for i := 0; i < e.drawCount; i++ {
			p.DrawCard()
		}
	}
	return nil
}

func (e *discardHandAndDrawEffect) Text() string {
	return fmt.Sprintf("Each player discards their hand, then draws %d cards", e.drawCount)
}

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

func (e *shuffleGraveyardIntoLibraryAndDrawEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, p := range g.Players {
		// Move graveyard into library
		for _, c := range p.Graveyard() {
			p.AddToLibrary(c)
		}
		p.ClearGraveyard()
		// Draw N cards
		for i := 0; i < e.drawCount; i++ {
			p.DrawCard()
		}
	}
	return nil
}

func (e *shuffleGraveyardIntoLibraryAndDrawEffect) Text() string {
	return fmt.Sprintf("Each player shuffles graveyard into library, then draws %d cards", e.drawCount)
}

// counterSpellIfColorEffect counters a target spell only if it matches a specific color.
type counterSpellIfColorEffect struct {
	color Color
}

// CounterSpellIfColor creates an effect that counters a target spell only if it matches the given color
// (e.g. Blue Elemental Blast, Red Elemental Blast).
func CounterSpellIfColor(c Color) Effect {
	return &counterSpellIfColorEffect{color: c}
}

func (e *counterSpellIfColorEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	// Find the spell on the stack
	obj := g.Stack.FindBySourceID(targets[0])
	if obj == nil || obj.Card == nil {
		return nil
	}
	// Check if the spell has the required color
	for _, c := range obj.Card.ManaCost().Colors() {
		if c == e.color {
			g.CounterSpellOnStack(targets[0])
			return nil
		}
	}
	// Color doesn't match, spell is NOT countered
	return nil
}

func (e *counterSpellIfColorEffect) Text() string {
	return fmt.Sprintf("Counter target %s spell", e.color)
}

// counterSpellIfXMeetsOrExceedsCMCEffect counters a spell only if X >= its CMC.
type counterSpellIfXMeetsOrExceedsCMCEffect struct{}

// CounterSpellIfXMeetsCMC creates an effect that counters a target spell only if X >= its mana value
// (e.g. Spell Blast).
func CounterSpellIfXMeetsCMC() Effect {
	return &counterSpellIfXMeetsOrExceedsCMCEffect{}
}

func (e *counterSpellIfXMeetsOrExceedsCMCEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	obj := g.Stack.FindBySourceID(targets[0])
	if obj == nil || obj.Card == nil {
		return nil
	}
	cmc := obj.Card.ManaCost().CMC()
	if g.CurrentX >= cmc {
		g.CounterSpellOnStack(targets[0])
	}
	return nil
}

func (e *counterSpellIfXMeetsOrExceedsCMCEffect) Text() string {
	return "Counter target spell if X >= its mana value"
}

// powerSinkEffect counters a spell unless its controller pays X mana.
type powerSinkEffect struct{}

// PowerSinkEffect creates an effect that counters a spell unless its controller pays X mana,
// draining their pool either way (Power Sink).
func PowerSinkEffect() Effect {
	return &powerSinkEffect{}
}

func (e *powerSinkEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	obj := g.Stack.FindBySourceID(targets[0])
	if obj == nil {
		return nil
	}
	// Check if the spell's controller can pay X mana
	spellController := g.GetPlayer(obj.Controller)
	if spellController == nil {
		return nil
	}
	// If they have enough mana in pool, they pay and spell resolves
	totalMana := spellController.ManaPool().TotalMana()
	if totalMana >= g.CurrentX {
		// Opponent can pay - drain X mana but don't counter
		spellController.ManaPool().DrainGeneric(g.CurrentX)
		return nil
	}
	// Can't pay - counter the spell and drain all mana
	spellController.ManaPool().Clear()
	g.CounterSpellOnStack(targets[0])
	return nil
}

func (e *powerSinkEffect) Text() string {
	return "Counter target spell unless its controller pays {X}"
}

// tapOrUntapTargetEffect lets you tap or untap a target permanent.
type tapOrUntapTargetEffect struct{}

// TapOrUntapTarget creates an effect that toggles a target permanent's tapped state (e.g. Twiddle).
func TapOrUntapTarget() Effect {
	return &tapOrUntapTargetEffect{}
}

func (e *tapOrUntapTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	// Toggle: if tapped, untap; if untapped, tap
	perm.Tapped = !perm.Tapped
	return nil
}

func (e *tapOrUntapTargetEffect) Text() string {
	return "Tap or untap target permanent"
}

// makeUnblockableUntilEndOfTurnEffect makes a target creature unblockable until end of turn.
type makeUnblockableUntilEndOfTurnEffect struct{}

// MakeUnblockableUntilEndOfTurn creates an effect that makes a target creature unblockable until end of turn.
func MakeUnblockableUntilEndOfTurn() Effect {
	return &makeUnblockableUntilEndOfTurnEffect{}
}

func (e *makeUnblockableUntilEndOfTurnEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	eff := TemporaryKeyword(perm.ID(), UnblockableKW)
	eff.SetSourceID(sourceID)
	g.Effects.Add(eff)
	g.Effects.Apply(g)
	return nil
}

func (e *makeUnblockableUntilEndOfTurnEffect) Text() string {
	return "Target creature can't be blocked this turn"
}

// tapAttachedCreatureEffect taps the creature attached to the source aura.
type tapAttachedCreatureEffect struct{}

// TapAttachedCreature creates an effect that taps the creature the source aura is attached to.
func TapAttachedCreature() Effect {
	return &tapAttachedCreatureEffect{}
}

func (e *tapAttachedCreatureEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	src := g.FindPermanent(sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target != nil {
		target.Tapped = true
	}
	return nil
}

func (e *tapAttachedCreatureEffect) Text() string { return "Tap enchanted creature" }

// untapSourceEffect untaps the source permanent.
type untapSourceEffect struct{}

// UntapSource creates an effect that untaps the source permanent.
func UntapSource() Effect {
	return &untapSourceEffect{}
}

func (e *untapSourceEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm != nil {
		perm.Tapped = false
	}
	return nil
}

func (e *untapSourceEffect) Text() string { return "Untap this permanent" }

// dealDamagePerSwampEffect deals damage to the active player equal to the number of Swamps they control.
type dealDamagePerSwampEffect struct{}

// DealDamagePerSwamp creates an effect that deals damage to the active player equal to the
// number of Swamps they control (e.g. Karma).
func DealDamagePerSwamp() Effect {
	return &dealDamagePerSwampEffect{}
}

func (e *dealDamagePerSwampEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	active := g.ActivePlayerObj()
	activeID := active.PlayerID()
	swampCount := g.CountBattlefield(And(ControlledBy(activeID), HasSubType("Swamp")))
	if swampCount > 0 {
		active.LoseLife(swampCount)
	}
	return nil
}

func (e *dealDamagePerSwampEffect) Text() string {
	return "Deal damage to active player equal to Swamps they control"
}

// blackViseEffect deals damage to the active player based on hand size > 4.
type blackViseEffect struct{}

// BlackViseEffect creates an effect that deals damage to the active player equal to
// cards in hand minus 4 (Black Vise).
func BlackViseEffect() Effect {
	return &blackViseEffect{}
}

func (e *blackViseEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	active := g.ActivePlayerObj()
	handSize := len(active.Hand())
	if handSize > 4 {
		damage := handSize - 4
		active.LoseLife(damage)
	}
	return nil
}

func (e *blackViseEffect) Text() string {
	return "Deal damage to active player equal to cards in hand minus 4"
}

// createTokenEffect creates a token creature on the battlefield.
type createTokenEffect struct {
	name      string
	power     int
	toughness int
	types     []CardType
	subTypes  []string
	keywords  []Keyword
}

// CreateToken creates an effect that puts a token creature onto the battlefield.
func CreateToken(name string, power, toughness int, types []CardType, subTypes []string, keywords ...Keyword) Effect {
	return &createTokenEffect{
		name:      name,
		power:     power,
		toughness: toughness,
		types:     types,
		subTypes:  subTypes,
		keywords:  keywords,
	}
}

func (e *createTokenEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	token := NewToken(e.name, e.power, e.toughness, e.types, e.subTypes, e.keywords...)
	token.SetOwner(controller)
	g.PutOnBattlefield(token, controller)
	return nil
}

func (e *createTokenEffect) Text() string {
	return fmt.Sprintf("create a %d/%d %s token", e.power, e.toughness, e.name)
}

// forcefieldEffect activates a Forcefield shield on the controller for this turn.
type forcefieldEffect struct{}

// ForcefieldEffect creates an effect that reduces all unblocked combat damage to the controller
// to 1 for this turn (Forcefield).
func ForcefieldEffect() Effect {
	return &forcefieldEffect{}
}

func (e *forcefieldEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	g.Effects.AddForcefieldShield(controller)
	return nil
}

func (e *forcefieldEffect) Text() string {
	return "prevent all but 1 combat damage from each unblocked creature this turn"
}

// --- ValueSource, PlayerSelector, PermanentSelector ---

// ValueSource resolves a dynamic integer value for an effect.
type ValueSource interface {
	Resolve(g *Game, sourceID, controller uuid.UUID) int
	Text() string
}

// PlayerSelector picks one or more players for an effect.
type PlayerSelector interface {
	Select(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) []uuid.UUID
	Text() string
}

// PermanentSelector chooses between the target permanent and the source permanent.
type PermanentSelector int

const (
	SelectTarget PermanentSelector = iota
	SelectSource
)

// fixedValue is a ValueSource that always returns a constant.
type fixedValue struct{ n int }

// Fixed creates a ValueSource that always returns the constant n.
func Fixed(n int) ValueSource                            { return fixedValue{n: n} }
func (v fixedValue) Resolve(_ *Game, _, _ uuid.UUID) int { return v.n }
func (v fixedValue) Text() string                        { return fmt.Sprintf("%d", v.n) }

// xValue is a ValueSource that reads g.CurrentX.
type xValue struct{}

// XValue creates a ValueSource that reads the X value from the current spell/ability (g.CurrentX).
func XValue() ValueSource                            { return xValue{} }
func (v xValue) Resolve(g *Game, _, _ uuid.UUID) int { return g.CurrentX }
func (v xValue) Text() string                        { return "X" }

// selectController returns the effect's controller.
type selectController struct{}

// SelectController creates a PlayerSelector that returns the effect's controller.
func SelectController() PlayerSelector { return selectController{} }
func (s selectController) Select(_ *Game, _, controller uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	return []uuid.UUID{controller}
}
func (s selectController) Text() string { return "controller" }

// selectActivePlayer returns the active player (whose turn it is).
type selectActivePlayer struct{}

// SelectActivePlayer creates a PlayerSelector that returns the active player (whose turn it is).
func SelectActivePlayer() PlayerSelector { return selectActivePlayer{} }
func (s selectActivePlayer) Select(g *Game, _, _ uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	return []uuid.UUID{g.ActivePlayerObj().PlayerID()}
}
func (s selectActivePlayer) Text() string { return "active player" }

// selectEachPlayer returns all players.
type selectEachPlayer struct{}

// SelectEachPlayer creates a PlayerSelector that returns all players in the game.
func SelectEachPlayer() PlayerSelector { return selectEachPlayer{} }
func (s selectEachPlayer) Select(g *Game, _, _ uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	ids := make([]uuid.UUID, len(g.Players))
	for i, p := range g.Players {
		ids[i] = p.PlayerID()
	}
	return ids
}
func (s selectEachPlayer) Text() string { return "each player" }

// selectEachOpponent returns all players other than the controller.
type selectEachOpponent struct{}

// SelectEachOpponent creates a PlayerSelector that returns all opponents of the controller.
func SelectEachOpponent() PlayerSelector { return selectEachOpponent{} }
func (s selectEachOpponent) Select(g *Game, _, controller uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	var ids []uuid.UUID
	for _, p := range g.Players {
		if p.PlayerID() != controller {
			ids = append(ids, p.PlayerID())
		}
	}
	return ids
}
func (s selectEachOpponent) Text() string { return "each opponent" }

// selectAttachedController follows source → AttachedTo → Controller.
type selectAttachedController struct{}

// SelectAttachedController creates a PlayerSelector that returns the controller of the permanent
// the source is attached to (for aura-based effects like Psychic Venom).
func SelectAttachedController() PlayerSelector { return selectAttachedController{} }
func (s selectAttachedController) Select(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	src := g.FindPermanent(sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target == nil {
		return nil
	}
	return []uuid.UUID{target.Controller}
}
func (s selectAttachedController) Text() string { return "enchanted creature's controller" }

// selectEventController reads targets[0] as a player ID (for event-based triggers).
type selectEventController struct{}

// SelectEventController creates a PlayerSelector that reads targets[0] as a player ID,
// used for event-based triggers that pass the relevant player through the target list.
func SelectEventController() PlayerSelector { return selectEventController{} }
func (s selectEventController) Select(_ *Game, _, _ uuid.UUID, targets []uuid.UUID) []uuid.UUID {
	if len(targets) == 0 {
		return nil
	}
	return []uuid.UUID{targets[0]}
}
func (s selectEventController) Text() string { return "that player" }

// tapAllLandsEffect taps all lands target player controls.
type tapAllLandsEffect struct{}

// TapAllLands creates an effect that taps all lands a target player controls (e.g. Mana Short).
func TapAllLands() Effect { return &tapAllLandsEffect{} }

func (e *tapAllLandsEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	playerID := targets[0]
	for _, p := range g.Battlefield {
		if p.Controller == playerID && p.HasType(TypeLand) {
			p.Tapped = true
		}
	}
	return nil
}
func (e *tapAllLandsEffect) Text() string { return "Tap all lands target player controls" }

// balanceEffect equalizes lands, creatures, and hand sizes.
type balanceEffect struct{}

// BalanceEffect creates an effect that equalizes lands, creatures, and hand sizes across all
// players by having each player sacrifice/discard down to the minimum (Balance).
func BalanceEffect() Effect { return &balanceEffect{} }

func (e *balanceEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	// Count lands for each player
	landCounts := make(map[uuid.UUID]int)
	creatureCounts := make(map[uuid.UUID]int)
	for _, p := range g.Battlefield {
		if p.HasType(TypeLand) {
			landCounts[p.Controller]++
		}
		if p.HasType(TypeCreature) {
			creatureCounts[p.Controller]++
		}
	}

	// Find minimums
	minLands := -1
	minCreatures := -1
	minHand := -1
	for _, p := range g.Players {
		pid := p.PlayerID()
		if minLands < 0 || landCounts[pid] < minLands {
			minLands = landCounts[pid]
		}
		if minCreatures < 0 || creatureCounts[pid] < minCreatures {
			minCreatures = creatureCounts[pid]
		}
		if minHand < 0 || len(p.Hand()) < minHand {
			minHand = len(p.Hand())
		}
	}

	// Sacrifice lands down to minimum
	for _, p := range g.Players {
		pid := p.PlayerID()
		toSac := landCounts[pid] - minLands
		for toSac > 0 {
			for _, perm := range g.Battlefield {
				if perm.Controller == pid && perm.HasType(TypeLand) {
					g.Sacrifice(perm)
					toSac--
					break
				}
			}
		}
	}

	// Sacrifice creatures down to minimum
	for _, p := range g.Players {
		pid := p.PlayerID()
		toSac := creatureCounts[pid] - minCreatures
		for toSac > 0 {
			for _, perm := range g.Battlefield {
				if perm.Controller == pid && perm.HasType(TypeCreature) {
					g.Sacrifice(perm)
					toSac--
					break
				}
			}
		}
	}

	// Discard down to minimum hand size
	for _, p := range g.Players {
		for len(p.Hand()) > minHand {
			hand := p.Hand()
			if len(hand) == 0 {
				break
			}
			chosen := p.ChooseCardsFromHand(1, "discard for Balance", g)
			if len(chosen) > 0 {
				p.RemoveFromHand(chosen[0].ID())
				p.AddToGraveyard(chosen[0])
			} else {
				break
			}
		}
	}

	return nil
}
func (e *balanceEffect) Text() string {
	return "Each player sacrifices to match fewest lands, creatures; discards to match smallest hand"
}

// destroyRandomNontokenPermanent destroys a random nontoken permanent an
// opponent controls, then destroys the source.
type chaosOrbEffect struct{}

// ChaosOrbEffect creates an effect that destroys a random nontoken permanent an opponent
// controls, then destroys the source (Chaos Orb).
func ChaosOrbEffect() Effect { return &chaosOrbEffect{} }

func (e *chaosOrbEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var candidates []*Permanent
	for _, p := range g.Battlefield {
		if p.Controller != controller && !p.Card.(*BaseCard).IsToken() {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) > 0 {
		chosen := candidates[rand.Intn(len(candidates))]
		g.DestroyPermanent(chosen)
	}
	// Destroy self
	src := g.FindPermanent(sourceID)
	if src != nil {
		g.DestroyPermanent(src)
	}
	return nil
}
func (e *chaosOrbEffect) Text() string { return "Destroy a random nontoken permanent, then destroy ~" }

// removeFromCombatEffect removes a target creature from combat.
type removeFromCombatEffect struct{}

// RemoveFromCombat creates an effect that removes a target creature from combat.
func RemoveFromCombat() Effect { return &removeFromCombatEffect{} }

func (e *removeFromCombatEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	g.Combat.RemoveFromCombat(perm.ID())
	return nil
}
func (e *removeFromCombatEffect) Text() string { return "Remove target creature from combat" }

// replaceKeywordEffect replaces one keyword with another on a target permanent.
type replaceKeywordEffect struct {
	from Keyword
	to   Keyword
}

// ReplaceKeywordEffect creates an effect that replaces one keyword with another on a target permanent
// as a continuous effect (e.g. replacing Flying with a different evasion).
func ReplaceKeywordEffect(from, to Keyword) Effect {
	return &replaceKeywordEffect{from: from, to: to}
}

func (e *replaceKeywordEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	// Remove the old keyword and add the new one as a continuous effect
	eff := KeywordReplacement(perm.ID(), e.from, e.to)
	eff.SetSourceID(sourceID)
	g.Effects.Add(eff)
	g.Effects.Apply(g)
	return nil
}

func (e *replaceKeywordEffect) Text() string {
	return fmt.Sprintf("Replace %s with %s", e.from, e.to)
}

// changeColorEffect changes a target permanent's color.
type changeColorEffect struct {
	color Color
}

// ChangeColorEffect creates an effect that changes a target permanent's color.
func ChangeColorEffect(color Color) Effect {
	return &changeColorEffect{color: color}
}

func (e *changeColorEffect) Apply(g *Game, sourceID, _ uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	// Register as a continuous effect so the color change persists
	ce := ColorOverride(perm.ID(), e.color)
	ce.SetSourceID(sourceID)
	g.Effects.Add(ce)
	return nil
}

func (e *changeColorEffect) Text() string {
	return fmt.Sprintf("Target permanent becomes %s", e.color)
}

// copySpellOnStackEffect copies the top spell on the stack. Used by Fork.
type copySpellOnStackEffect struct{}

// CopySpellOnStack creates an effect that copies the target spell on the stack.
func CopySpellOnStack() Effect {
	return &copySpellOnStackEffect{}
}

func (e *copySpellOnStackEffect) Apply(g *Game, _, ctrl uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	// Find the target spell on the stack
	original := g.Stack.FindBySourceID(targets[0])
	if original == nil {
		return nil
	}
	// Create a copy of the stack object
	cp := &StackObject{
		ID:         uuid.New(),
		Card:       original.Card,
		Controller: ctrl,
		SourceID:   original.SourceID,
		Effects:    make([]Effect, len(original.Effects)),
		Targets:    make([]uuid.UUID, len(original.Targets)),
		IsAbility:  original.IsAbility,
		XValue:     original.XValue,
	}
	copy(cp.Effects, original.Effects)
	copy(cp.Targets, original.Targets)
	g.Stack.Push(cp)
	return nil
}

func (e *copySpellOnStackEffect) Text() string {
	return "copy target instant or sorcery spell"
}
