package mage

import (
	"fmt"

	"github.com/google/uuid"
)

// Effect represents a one-shot effect that resolves.
type Effect interface {
	Apply(g *Game, sourceID uuid.UUID, controller uuid.UUID, targets []uuid.UUID) error
	Text() string
}

// gainLifeEffect gains life for the controller.
type gainLifeEffect struct {
	amount int
}

func GainLife(amount int) Effect {
	return &gainLifeEffect{amount: amount}
}

func (e *gainLifeEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return fmt.Errorf("player not found")
	}
	p.GainLife(e.amount)
	g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: controller, Amount: e.amount})
	return nil
}

func (e *gainLifeEffect) Text() string {
	return fmt.Sprintf("gain %d life", e.amount)
}

// addCountersToSourceEffect adds counters to the source permanent.
type addCountersToSourceEffect struct {
	ct     CounterType
	amount int
}

func AddCountersToSource(ct CounterType, amount int) Effect {
	return &addCountersToSourceEffect{ct: ct, amount: amount}
}

func (e *addCountersToSourceEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return nil // source gone, effect does nothing
	}
	p.AddCounter(e.ct, e.amount)
	return nil
}

func (e *addCountersToSourceEffect) Text() string {
	return fmt.Sprintf("put %d %s counter(s) on it", e.amount, e.ct)
}

// addXCountersToSourceEffect adds X counters to the source, reading X from g.CurrentX.
type addXCountersToSourceEffect struct {
	ct CounterType
}

func AddXCountersToSource(ct CounterType) Effect {
	return &addXCountersToSourceEffect{ct: ct}
}

func (e *addXCountersToSourceEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return nil
	}
	if g.CurrentX > 0 {
		p.AddCounter(e.ct, g.CurrentX)
	}
	return nil
}

func (e *addXCountersToSourceEffect) Text() string {
	return fmt.Sprintf("put X %s counters on it", e.ct)
}

// removeCountersFromSourceEffect removes counters from the source permanent.
type removeCountersFromSourceEffect struct {
	ct     CounterType
	amount int
}

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

// dealDamageEffect deals damage to the target.
type dealDamageEffect struct {
	amount int
}

func DealDamage(amount int) Effect {
	return &dealDamageEffect{amount: amount}
}

func (e *dealDamageEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for damage")
	}
	targetID := targets[0]

	// Check if target is a player
	for _, pl := range g.Players {
		if pl.PlayerID() == targetID {
			g.DealDamageToPlayer(pl, e.amount, sourceID)
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

	g.DealDamageToPermanent(perm, e.amount, sourceID)
	return nil
}

func (e *dealDamageEffect) Text() string {
	return fmt.Sprintf("deal %d damage to target", e.amount)
}

// destroyTargetEffect destroys the target permanent.
type destroyTargetEffect struct{}

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

// drawCardsEffect draws cards for the controller.
type drawCardsEffect struct {
	amount int
}

func DrawCards(amount int) Effect {
	return &drawCardsEffect{amount: amount}
}

func (e *drawCardsEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return fmt.Errorf("player not found")
	}
	for i := 0; i < e.amount; i++ {
		p.DrawCard()
	}
	return nil
}

func (e *drawCardsEffect) Text() string {
	return fmt.Sprintf("draw %d card(s)", e.amount)
}

// returnFromGraveyardEffect returns a target creature from graveyard to battlefield.
type returnFromGraveyardEffect struct{}

func ReturnFromGraveyardToBattlefield() Effect {
	return &returnFromGraveyardEffect{}
}

func (e *returnFromGraveyardEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for reanimate")
	}
	p := g.GetPlayer(controller)
	if p == nil {
		return fmt.Errorf("player not found")
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

func ReturnSourceToHand() Effect {
	return &returnSourceToHandEffect{}
}

func (e *returnSourceToHandEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return fmt.Errorf("player not found")
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

// destroyAllCreaturesEffect destroys all creatures (board wipe).
type destroyAllCreaturesEffect struct{}

func DestroyAllCreatures() Effect {
	return &destroyAllCreaturesEffect{}
}

func (e *destroyAllCreaturesEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var toDestroy []*Permanent
	for _, p := range g.Battlefield {
		if p.HasType(TypeCreature) && !p.HasAbility(Indestructible) {
			toDestroy = append(toDestroy, p)
		}
	}
	for _, p := range toDestroy {
		g.DestroyPermanent(p)
	}
	return nil
}

func (e *destroyAllCreaturesEffect) Text() string {
	return "destroy all creatures"
}

// compositeEffect applies multiple effects in sequence.
type compositeEffect struct {
	effects []Effect
	text    string
}

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

func (e *returnToHandTargetEffect) Text() string { return "return target permanent to its owner's hand" }

// returnFromGraveyardToHandTargetEffect returns a target card from graveyard to hand.
type returnFromGraveyardToHandTargetEffect struct{}

func ReturnFromGraveyardToHandTarget() Effect {
	return &returnFromGraveyardToHandTargetEffect{}
}

func (e *returnFromGraveyardToHandTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for raise dead")
	}
	p := g.GetPlayer(controller)
	if p == nil {
		return fmt.Errorf("player not found")
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

// boostTargetEffect boosts target creature's P/T until end of turn.
type boostTargetEffect struct {
	power     int
	toughness int
}

func BoostTargetUntilEndOfTurn(power, toughness int) Effect {
	return &boostTargetEffect{power: power, toughness: toughness}
}

func (e *boostTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for boost")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	// Apply as an end-of-turn continuous effect
	eff := &temporaryBoostEffect{
		targetID:  perm.ID(),
		power:     e.power,
		toughness: e.toughness,
		sourceID_: sourceID,
	}
	g.Effects.Add(eff)
	g.Effects.Apply(g)
	return nil
}

func (e *boostTargetEffect) Text() string {
	return fmt.Sprintf("target creature gets +%d/+%d until end of turn", e.power, e.toughness)
}

// boostSourceEffect boosts the source permanent's P/T until end of turn (for firebreathing).
type boostSourceEffect struct {
	power     int
	toughness int
}

func BoostSourceUntilEndOfTurn(power, toughness int) Effect {
	return &boostSourceEffect{power: power, toughness: toughness}
}

func (e *boostSourceEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return nil
	}
	eff := &temporaryBoostEffect{
		targetID:  perm.ID(),
		power:     e.power,
		toughness: e.toughness,
		sourceID_: sourceID,
	}
	g.Effects.Add(eff)
	g.Effects.Apply(g)
	return nil
}

func (e *boostSourceEffect) Text() string {
	return fmt.Sprintf("this creature gets +%d/+%d until end of turn", e.power, e.toughness)
}

// markDestroyAtEOTAfterNActivationsEffect tracks pump activations using Charge
// counters. When the count reaches the threshold, sets DestroyAtEndOfTurn.
type markDestroyAtEOTAfterNActivationsEffect struct {
	threshold int
}

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
		perm.DestroyAtEndOfTurn = true
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

func DestroyTargetPermanent() Effect {
	return &destroyTargetPermanentEffect{text: "destroy target permanent"}
}

func DestroyTargetLand() Effect {
	return &destroyTargetPermanentEffect{text: "destroy target land"}
}

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

// destroyAllLandsEffect destroys all lands (Armageddon).
type destroyAllLandsEffect struct{}

func DestroyAllLands() Effect {
	return &destroyAllLandsEffect{}
}

func (e *destroyAllLandsEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var toDestroy []*Permanent
	for _, p := range g.Battlefield {
		if p.HasType(TypeLand) {
			toDestroy = append(toDestroy, p)
		}
	}
	for _, p := range toDestroy {
		g.DestroyPermanent(p)
	}
	return nil
}

func (e *destroyAllLandsEffect) Text() string { return "destroy all lands" }

// destroyAllEnchantmentsEffect destroys all enchantments (Tranquility).
type destroyAllEnchantmentsEffect struct{}

func DestroyAllEnchantments() Effect {
	return &destroyAllEnchantmentsEffect{}
}

func (e *destroyAllEnchantmentsEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var toDestroy []*Permanent
	for _, p := range g.Battlefield {
		if p.HasType(TypeEnchantment) {
			toDestroy = append(toDestroy, p)
		}
	}
	for _, p := range toDestroy {
		g.DestroyPermanent(p)
	}
	return nil
}

func (e *destroyAllEnchantmentsEffect) Text() string { return "destroy all enchantments" }

// destroyAllMatchingEffect destroys all permanents matching a filter.
type destroyAllMatchingEffect struct {
	filter PermanentFilter
	text   string
}

func DestroyAllMatching(filter PermanentFilter, text string) Effect {
	return &destroyAllMatchingEffect{filter: filter, text: text}
}

func (e *destroyAllMatchingEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var toDestroy []*Permanent
	for _, p := range g.Battlefield {
		if e.filter(p, g) && !p.HasAbility(Indestructible) {
			toDestroy = append(toDestroy, p)
		}
	}
	for _, p := range toDestroy {
		g.DestroyPermanent(p)
	}
	return nil
}

func (e *destroyAllMatchingEffect) Text() string { return e.text }

// tapTargetEffect taps a target permanent.
type tapTargetEffect struct{}

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
		// Discard the first card (deterministic for testing)
		card := hand[0]
		targetPlayer.RemoveFromHand(card.ID())
		targetPlayer.AddToGraveyard(card)
	}
	return nil
}

func (e *discardRandomEffect) Text() string {
	return fmt.Sprintf("discard %d card(s) at random", e.amount)
}

// discardCardsEffect forces a player to discard N cards.
type discardCardsEffect struct {
	amount int
}

func DiscardCards(amount int) Effect {
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
	for i := 0; i < e.amount; i++ {
		hand := targetPlayer.Hand()
		if len(hand) == 0 {
			break
		}
		card := hand[0]
		targetPlayer.RemoveFromHand(card.ID())
		targetPlayer.AddToGraveyard(card)
	}
	return nil
}

func (e *discardCardsEffect) Text() string {
	return fmt.Sprintf("target player discards %d card(s)", e.amount)
}

// addManaEffect adds mana to the controller's pool.
type addManaEffect struct {
	color  Color
	amount int
}

func AddMana(color Color, amount int) Effect {
	return &addManaEffect{color: color, amount: amount}
}

func (e *addManaEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return fmt.Errorf("player not found")
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
	color  Color // chosen color (default to the first needed)
}

func AddAnyMana(amount int, color Color) Effect {
	return &addAnyManaEffect{amount: amount, color: color}
}

func (e *addAnyManaEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return fmt.Errorf("player not found")
	}
	p.ManaPool().Add(e.color, e.amount)
	return nil
}

func (e *addAnyManaEffect) Text() string {
	return fmt.Sprintf("add %d mana of any one color", e.amount)
}

// drawCardsTargetEffect draws cards for a target player.
type drawCardsTargetEffect struct {
	amount int
}

func DrawCardsTarget(amount int) Effect {
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
		return fmt.Errorf("player not found")
	}
	for i := 0; i < e.amount; i++ {
		targetPlayer.DrawCard()
	}
	return nil
}

func (e *drawCardsTargetEffect) Text() string {
	return fmt.Sprintf("target player draws %d card(s)", e.amount)
}

// exileTargetEffect exiles a target permanent (removes from game).
type exileTargetEffect struct{}

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

// exileAndGainLifeEffect exiles target creature and its controller gains life equal to its power.
type exileAndGainLifeEffect struct{}

func ExileTargetCreatureGainLife() Effect {
	return &exileAndGainLifeEffect{}
}

func (e *exileAndGainLifeEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	power := perm.CurrentPower(g)
	permController := perm.Controller
	g.ExilePermanent(perm)
	// Controller of the exiled creature gains life equal to its power
	p := g.GetPlayer(permController)
	if p != nil && power > 0 {
		p.GainLife(power)
		g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: permController, Amount: power})
	}
	return nil
}

func (e *exileAndGainLifeEffect) Text() string {
	return "exile target creature. Its controller gains life equal to its power"
}

// gainLifeTargetEffect gains life for the target player.
type gainLifeTargetEffect struct {
	amount int
}

func GainLifeTarget(amount int) Effect {
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
		return fmt.Errorf("player not found")
	}
	targetPlayer.GainLife(e.amount)
	g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: targetPlayer.PlayerID(), Amount: e.amount})
	return nil
}

func (e *gainLifeTargetEffect) Text() string {
	return fmt.Sprintf("target player gains %d life", e.amount)
}

// loseLifeEffect causes the controller to lose life.
type loseLifeEffect struct {
	amount int
}

func LoseLife(amount int) Effect {
	return &loseLifeEffect{amount: amount}
}

func (e *loseLifeEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return fmt.Errorf("player not found")
	}
	p.LoseLife(e.amount)
	g.FireEvent(GameEvent{Type: EvtLifeLost, PlayerID: controller, Amount: e.amount})
	return nil
}

func (e *loseLifeEffect) Text() string {
	return fmt.Sprintf("you lose %d life", e.amount)
}

// dealDamageToAllCreaturesEffect deals damage to all creatures.
type dealDamageToAllCreaturesEffect struct {
	amount int
	filter PermanentFilter // optional filter (e.g., without flying)
}

func DealDamageToAllCreatures(amount int, filter PermanentFilter) Effect {
	return &dealDamageToAllCreaturesEffect{amount: amount, filter: filter}
}

func (e *dealDamageToAllCreaturesEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, p := range g.Battlefield {
		if !p.HasType(TypeCreature) {
			continue
		}
		if e.filter != nil && !e.filter(p, g) {
			continue
		}
		g.DealDamageToPermanent(p, e.amount, sourceID)
	}
	return nil
}

func (e *dealDamageToAllCreaturesEffect) Text() string {
	return fmt.Sprintf("deal %d damage to each creature", e.amount)
}

// dealDamageToEachPlayerEffect deals damage to each player.
type dealDamageToEachPlayerEffect struct {
	amount int
}

func DealDamageToEachPlayer(amount int) Effect {
	return &dealDamageToEachPlayerEffect{amount: amount}
}

func (e *dealDamageToEachPlayerEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, p := range g.Players {
		g.DealDamageToPlayer(p, e.amount, sourceID)
	}
	return nil
}

func (e *dealDamageToEachPlayerEffect) Text() string {
	return fmt.Sprintf("deal %d damage to each player", e.amount)
}

// sacrificeSourceEffect sacrifices the source permanent.
type sacrificeSourceEffect struct{}

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

func SearchLibraryToHand() Effect {
	return &searchLibraryEffect{}
}

func (e *searchLibraryEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return fmt.Errorf("player not found")
	}
	lib := p.Library()
	if len(lib) == 0 {
		return nil
	}
	// In testing, just take the first card from library
	card := lib[0]
	newLib := make([]Card, len(lib)-1)
	copy(newLib, lib[1:])
	p.SetLibrary(newLib)
	p.AddToHand(card)
	return nil
}

func (e *searchLibraryEffect) Text() string {
	return "search your library for a card and put it into your hand"
}

// counterSpellEffect counters a target spell on the stack.
type counterSpellEffect struct{}

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

// dealDamageToSourceControllerEffect deals damage to the controller of the source.
type dealDamageToSourceControllerEffect struct {
	amount int
}

func DealDamageToSourceController(amount int) Effect {
	return &dealDamageToSourceControllerEffect{amount: amount}
}

func (e *dealDamageToSourceControllerEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	g.DealDamageToPlayer(p, e.amount, sourceID)
	return nil
}

func (e *dealDamageToSourceControllerEffect) Text() string {
	return fmt.Sprintf("deal %d damage to you", e.amount)
}

// dealDamageToEachOpponentEffect deals damage to each opponent.
type dealDamageToEachOpponentEffect struct {
	amount int
}

func DealDamageToEachOpponent(amount int) Effect {
	return &dealDamageToEachOpponentEffect{amount: amount}
}

func (e *dealDamageToEachOpponentEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, p := range g.Players {
		if p.PlayerID() != controller {
			g.DealDamageToPlayer(p, e.amount, sourceID)
		}
	}
	return nil
}

func (e *dealDamageToEachOpponentEffect) Text() string {
	return fmt.Sprintf("deal %d damage to each opponent", e.amount)
}

// addCountersToTargetEffect adds counters to a target permanent.
type addCountersToTargetEffect struct {
	ct     CounterType
	amount int
}

func AddCountersToTarget(ct CounterType, amount int) Effect {
	return &addCountersToTargetEffect{ct: ct, amount: amount}
}

func (e *addCountersToTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for counters")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	perm.AddCounter(e.ct, e.amount)
	return nil
}

func (e *addCountersToTargetEffect) Text() string {
	return fmt.Sprintf("put %d %s counter(s) on target", e.amount, e.ct)
}

// dealXDamageEffect deals X damage to the target (reads X from game.CurrentX).
type dealXDamageEffect struct{}

func DealXDamage() Effect {
	return &dealXDamageEffect{}
}

func (e *dealXDamageEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for X damage")
	}
	x := g.CurrentX
	if x <= 0 {
		return nil
	}
	targetID := targets[0]
	for _, pl := range g.Players {
		if pl.PlayerID() == targetID {
			g.DealDamageToPlayer(pl, x, sourceID)
			return nil
		}
	}
	perm := g.FindPermanent(targetID)
	if perm == nil {
		return nil
	}
	g.DealDamageToPermanent(perm, x, sourceID)
	return nil
}

func (e *dealXDamageEffect) Text() string { return "deal X damage to target" }

// drawXCardsEffect draws X cards for the target.
type drawXCardsEffect struct{}

func DrawXCards() Effect {
	return &drawXCardsEffect{}
}

func (e *drawXCardsEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var targetPlayer Player
	if len(targets) > 0 {
		targetPlayer = g.GetPlayer(targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = g.GetPlayer(controller)
	}
	if targetPlayer == nil {
		return fmt.Errorf("player not found")
	}
	for i := 0; i < g.CurrentX; i++ {
		targetPlayer.DrawCard()
	}
	return nil
}

func (e *drawXCardsEffect) Text() string { return "target player draws X cards" }

// discardXCardsEffect forces target player to discard X cards.
type discardXCardsEffect struct{}

func DiscardXCards() Effect {
	return &discardXCardsEffect{}
}

func (e *discardXCardsEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
	for i := 0; i < g.CurrentX; i++ {
		hand := targetPlayer.Hand()
		if len(hand) == 0 {
			break
		}
		card := hand[0]
		targetPlayer.RemoveFromHand(card.ID())
		targetPlayer.AddToGraveyard(card)
	}
	return nil
}

func (e *discardXCardsEffect) Text() string { return "target player discards X cards" }

// gainXLifeEffect gains X life for the controller.
type gainXLifeEffect struct{}

func GainXLife() Effect {
	return &gainXLifeEffect{}
}

func (e *gainXLifeEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var targetPlayer Player
	if len(targets) > 0 {
		targetPlayer = g.GetPlayer(targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = g.GetPlayer(controller)
	}
	if targetPlayer == nil {
		return fmt.Errorf("player not found")
	}
	targetPlayer.GainLife(g.CurrentX)
	g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: targetPlayer.PlayerID(), Amount: g.CurrentX})
	return nil
}

func (e *gainXLifeEffect) Text() string { return "target player gains X life" }

// dealXDamageToAllCreaturesEffect deals X damage to all matching creatures.
type dealXDamageToAllCreaturesEffect struct {
	filter PermanentFilter
}

func DealXDamageToAllCreatures(filter PermanentFilter) Effect {
	return &dealXDamageToAllCreaturesEffect{filter: filter}
}

func (e *dealXDamageToAllCreaturesEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	x := g.CurrentX
	if x <= 0 {
		return nil
	}
	var creatures []*Permanent
	for _, p := range g.Battlefield {
		if !p.HasType(TypeCreature) {
			continue
		}
		if e.filter != nil && !e.filter(p, g) {
			continue
		}
		creatures = append(creatures, p)
	}
	for _, p := range creatures {
		g.DealDamageToPermanent(p, x, sourceID)
	}
	return nil
}

func (e *dealXDamageToAllCreaturesEffect) Text() string {
	return "deal X damage to each creature"
}

// dealXDamageToEachPlayerEffect deals X damage to each player.
type dealXDamageToEachPlayerEffect struct{}

func DealXDamageToEachPlayer() Effect {
	return &dealXDamageToEachPlayerEffect{}
}

func (e *dealXDamageToEachPlayerEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	x := g.CurrentX
	if x <= 0 {
		return nil
	}
	for _, p := range g.Players {
		g.DealDamageToPlayer(p, x, sourceID)
	}
	return nil
}

func (e *dealXDamageToEachPlayerEffect) Text() string { return "deal X damage to each player" }

// drainXLifeEffect deals X damage to target and gains X life.
type drainXLifeEffect struct{}

func DrainXLife() Effect {
	return &drainXLifeEffect{}
}

func (e *drainXLifeEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target")
	}
	x := g.CurrentX
	if x <= 0 {
		return nil
	}
	targetID := targets[0]
	for _, pl := range g.Players {
		if pl.PlayerID() == targetID {
			g.DealDamageToPlayer(pl, x, sourceID)
			p := g.GetPlayer(controller)
			if p != nil {
				p.GainLife(x)
			}
			return nil
		}
	}
	perm := g.FindPermanent(targetID)
	if perm == nil {
		return nil
	}
	g.DealDamageToPermanent(perm, x, sourceID)
	p := g.GetPlayer(controller)
	if p != nil {
		p.GainLife(x)
	}
	return nil
}

func (e *drainXLifeEffect) Text() string { return "deal X damage to target and gain X life" }

// dealDamageToAttachedControllerEffect deals damage to the controller of the
// permanent that the source aura is attached to.
type dealDamageToAttachedControllerEffect struct {
	amount int
}

func DealDamageToAttachedController(amount int) Effect {
	return &dealDamageToAttachedControllerEffect{amount: amount}
}

func (e *dealDamageToAttachedControllerEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	src := g.FindPermanent(sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	host := g.FindPermanent(src.AttachedTo)
	if host == nil {
		return nil
	}
	p := g.GetPlayer(host.Controller)
	if p == nil {
		return nil
	}
	g.DealDamageToPlayer(p, e.amount, sourceID)
	return nil
}

func (e *dealDamageToAttachedControllerEffect) Text() string {
	return fmt.Sprintf("deal %d damage to enchanted permanent's controller", e.amount)
}

// dealDamageToEventControllerEffect deals damage to the controller of the
// permanent that triggered the event (used with land ETB triggers).
type dealDamageToEventControllerEffect struct {
	amount int
}

func DealDamageToEventController(amount int) Effect {
	return &dealDamageToEventControllerEffect{amount: amount}
}

func (e *dealDamageToEventControllerEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	// The event's PlayerID (the entering permanent's controller) is passed through
	// as a target on the StackObject by PutTriggersOnStack.
	if len(targets) > 0 {
		p := g.GetPlayer(targets[0])
		if p != nil {
			g.DealDamageToPlayer(p, e.amount, sourceID)
		}
	}
	return nil
}

func (e *dealDamageToEventControllerEffect) Text() string {
	return fmt.Sprintf("deal %d damage to that permanent's controller", e.amount)
}

// grantKeywordTargetUntilEndOfTurnEffect grants a keyword to a target creature until end of turn.
type grantKeywordTargetUntilEndOfTurnEffect struct {
	keyword Keyword
}

func GrantKeywordTargetUntilEndOfTurn(kw Keyword) Effect {
	return &grantKeywordTargetUntilEndOfTurnEffect{keyword: kw}
}

func (e *grantKeywordTargetUntilEndOfTurnEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for keyword grant")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	eff := &temporaryKeywordEffect{
		targetID:  perm.ID(),
		keyword:   e.keyword,
		sourceID_: sourceID,
	}
	g.Effects.Add(eff)
	g.Effects.Apply(g)
	return nil
}

func (e *grantKeywordTargetUntilEndOfTurnEffect) Text() string {
	return fmt.Sprintf("target creature gains %s until end of turn", e.keyword)
}

// grantKeywordSourceUntilEndOfTurnEffect grants a keyword to the source permanent until end of turn.
type grantKeywordSourceUntilEndOfTurnEffect struct {
	keyword Keyword
}

func GrantKeywordSourceUntilEndOfTurn(kw Keyword) Effect {
	return &grantKeywordSourceUntilEndOfTurnEffect{keyword: kw}
}

func (e *grantKeywordSourceUntilEndOfTurnEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return nil
	}
	eff := &temporaryKeywordEffect{
		targetID:  perm.ID(),
		keyword:   e.keyword,
		sourceID_: sourceID,
	}
	g.Effects.Add(eff)
	g.Effects.Apply(g)
	return nil
}

func (e *grantKeywordSourceUntilEndOfTurnEffect) Text() string {
	return fmt.Sprintf("~ gains %s until end of turn", e.keyword)
}

// regenerateSourceEffect sets a regeneration shield on the source.
type regenerateSourceEffect struct{}

func RegenerateSource() Effect {
	return &regenerateSourceEffect{}
}

func (e *regenerateSourceEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return nil
	}
	perm.RegenerationShield = true
	return nil
}

func (e *regenerateSourceEffect) Text() string { return "Regenerate ~" }

// preventDamageToTargetEffect sets a damage prevention shield on a target.
type preventDamageToTargetEffect struct {
	amount int
}

func PreventDamageToTarget(amount int) Effect {
	return &preventDamageToTargetEffect{amount: amount}
}

func (e *preventDamageToTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm != nil {
		perm.DamagePreventionShield += e.amount
		return nil
	}
	// Could also prevent damage to player - not implemented yet
	return nil
}

func (e *preventDamageToTargetEffect) Text() string {
	return fmt.Sprintf("Prevent the next %d damage to target", e.amount)
}

// preventXDamageToTargetEffect prevents X damage to a target.
type preventXDamageToTargetEffect struct{}

func PreventXDamageToTarget() Effect {
	return &preventXDamageToTargetEffect{}
}

func (e *preventXDamageToTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm != nil {
		perm.DamagePreventionShield += g.CurrentX
		return nil
	}
	return nil
}

func (e *preventXDamageToTargetEffect) Text() string {
	return "Prevent the next X damage to target"
}

// sacrificeOrDamageEffect sacrifices a creature you control, or deals damage
// to the source's controller if no creature is available.
type sacrificeOrDamageEffect struct {
	damage int
}

func SacrificeCreatureOrDamage(damage int) Effect {
	return &sacrificeOrDamageEffect{damage: damage}
}

func (e *sacrificeOrDamageEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	// Try to find a creature to sacrifice (not the source itself)
	for _, p := range g.Battlefield {
		if p.Controller == controller && p.HasType(TypeCreature) && p.ID() != sourceID {
			g.Sacrifice(p)
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
	eff := &temporaryBoostEffect{
		targetID:  perm.ID(),
		power:     currentPower,
		toughness: 0,
		sourceID_: sourceID,
	}
	g.Effects.Add(eff)
	g.Effects.Apply(g)
	return nil
}

func (e *doubleSourcePowerEffect) Text() string {
	return "Target creature's power is doubled until end of turn"
}

// destroyTargetAtEndOfTurnEffect marks a creature for destruction at end of turn.
type destroyTargetAtEndOfTurnEffect struct{}

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
	perm.DestroyAtEndOfTurn = true
	return nil
}

func (e *destroyTargetAtEndOfTurnEffect) Text() string {
	return "Destroy target creature at end of turn"
}

// boostTargetXEffect gives +X/+0 to a target creature until end of turn.
type boostTargetXEffect struct {
	power     bool // if true, boost power by X
	toughness bool // if true, boost toughness by X
}

func BoostTargetXUntilEndOfTurn(power, toughness bool) Effect {
	return &boostTargetXEffect{power: power, toughness: toughness}
}

func (e *boostTargetXEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	p, t := 0, 0
	if e.power {
		p = g.CurrentX
	}
	if e.toughness {
		t = g.CurrentX
	}
	eff := &temporaryBoostEffect{
		targetID:  perm.ID(),
		power:     p,
		toughness: t,
		sourceID_: sourceID,
	}
	g.Effects.Add(eff)
	g.Effects.Apply(g)
	return nil
}

func (e *boostTargetXEffect) Text() string {
	return "Target creature gets +X/+0 until end of turn"
}

// discardHandAndDrawEffect makes each player discard their hand and draw N cards.
type discardHandAndDrawEffect struct {
	drawCount int
}

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
	perm.Unblockable = true
	return nil
}

func (e *makeUnblockableUntilEndOfTurnEffect) Text() string {
	return "Target creature can't be blocked this turn"
}

// tapAttachedCreatureEffect taps the creature attached to the source aura.
type tapAttachedCreatureEffect struct{}

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

// dealDamageToActivePlayerEffect deals damage to the active player (the one whose upkeep it is).
type dealDamageToActivePlayerEffect struct {
	amount int
}

func DealDamageToActivePlayer(amount int) Effect {
	return &dealDamageToActivePlayerEffect{amount: amount}
}

func (e *dealDamageToActivePlayerEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	active := g.ActivePlayerObj()
	active.LoseLife(e.amount)
	return nil
}

func (e *dealDamageToActivePlayerEffect) Text() string {
	return fmt.Sprintf("Deal %d damage to active player", e.amount)
}

// dealDamagePerSwampEffect deals damage to the active player equal to the number of Swamps they control.
type dealDamagePerSwampEffect struct{}

func DealDamagePerSwamp() Effect {
	return &dealDamagePerSwampEffect{}
}

func (e *dealDamagePerSwampEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	active := g.ActivePlayerObj()
	activeID := active.PlayerID()
	swampCount := 0
	for _, p := range g.Battlefield {
		if p.Controller == activeID && p.HasSubType("Swamp") {
			swampCount++
		}
	}
	if swampCount > 0 {
		active.LoseLife(swampCount)
	}
	return nil
}

func (e *dealDamagePerSwampEffect) Text() string {
	return "Deal damage to active player equal to Swamps they control"
}

// blackViseEffectImpl deals damage to the active player based on hand size > 4.
type blackViseEffectImpl struct{}

func BlackViseEffect() Effect {
	return &blackViseEffectImpl{}
}

func (e *blackViseEffectImpl) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	active := g.ActivePlayerObj()
	handSize := len(active.Hand())
	if handSize > 4 {
		damage := handSize - 4
		active.LoseLife(damage)
	}
	return nil
}

func (e *blackViseEffectImpl) Text() string {
	return "Deal damage to active player equal to cards in hand minus 4"
}
