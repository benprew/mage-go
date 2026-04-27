package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// counterSpellEffect counters a target spell on the stack.
type counterSpellEffect struct{}

// CounterSpell creates an effect that counters a target spell on the stack.
func CounterSpell() Effect {
	return &counterSpellEffect{}
}

func (e *counterSpellEffect) Text() string { return "counter target spell" }
func (e *counterSpellEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
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

func (e *counterSpellIfColorEffect) Text() string {
	return fmt.Sprintf("Counter target %s spell", e.color)
}
func (e *counterSpellIfColorEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// counterSpellIfXMeetsOrExceedsCMCEffect counters a spell only if X >= its CMC.
type counterSpellIfXMeetsOrExceedsCMCEffect struct{}

// CounterSpellIfXMeetsCMC creates an effect that counters a target spell only if X >= its mana value
// (e.g. Spell Blast).
func CounterSpellIfXMeetsCMC() Effect {
	return &counterSpellIfXMeetsOrExceedsCMCEffect{}
}

func (e *counterSpellIfXMeetsOrExceedsCMCEffect) Text() string {
	return "Counter target spell if X >= its mana value"
}
func (e *counterSpellIfXMeetsOrExceedsCMCEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// powerSinkEffect counters a spell unless its controller pays X mana.
type powerSinkEffect struct{}

// PowerSinkEffect creates an effect that counters a spell unless its controller pays X mana,
// draining their pool either way (Power Sink).
func PowerSinkEffect() Effect {
	return &powerSinkEffect{}
}

func (e *powerSinkEffect) Text() string {
	return "Counter target spell unless its controller pays {X}"
}
func (e *powerSinkEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
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

func (e *addManaEffect) Text() string {
	return fmt.Sprintf("add %d %s mana", e.amount, e.color)
}
func (e *addManaEffect) Properties() EffectProperties { return EffectProperties{} }

// addAnyManaEffect adds mana of any one color to the controller's pool.
type addAnyManaEffect struct {
	amount int
}

// AddAnyMana creates an effect that adds mana of any one color (player chooses) to the controller's pool.
func AddAnyMana(amount int, _ Color) Effect {
	return &addAnyManaEffect{amount: amount}
}

func (e *addAnyManaEffect) Text() string {
	return fmt.Sprintf("add %d mana of any one color", e.amount)
}
func (e *addAnyManaEffect) Properties() EffectProperties { return EffectProperties{} }

// createTokenEffect creates a token creature on the battlefield.
type createTokenEffect struct {
	name      string
	power     int
	toughness int
	types     []CardType
	subTypes  []string
	keywords  []Keyword
	colors    []Color
	count     int // 0 means 1
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

// CreateTokens creates an effect that puts N token creatures onto the battlefield.
func CreateTokens(count int, name string, power, toughness int, types []CardType, subTypes []string, keywords ...Keyword) Effect {
	return &createTokenEffect{
		name:      name,
		power:     power,
		toughness: toughness,
		types:     types,
		subTypes:  subTypes,
		keywords:  keywords,
		count:     count,
	}
}

// CreateColoredToken creates an effect that puts a colored token creature onto the battlefield.
func CreateColoredToken(name string, power, toughness int, colors []Color, types []CardType, subTypes []string, keywords ...Keyword) Effect {
	return &createTokenEffect{
		name:      name,
		power:     power,
		toughness: toughness,
		types:     types,
		subTypes:  subTypes,
		keywords:  keywords,
		colors:    colors,
	}
}

func (e *createTokenEffect) Text() string {
	return fmt.Sprintf("create a %d/%d %s token", e.power, e.toughness, e.name)
}
func (e *createTokenEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, TokenPower: e.power, TokenToughness: e.toughness}
}

// createTokenAttackingEffect creates a token creature that enters the
// battlefield attacking the defending player (CR 508.4). AttacksTrigger does
// not fire for such tokens (CR 508.3a).
type createTokenAttackingEffect struct {
	name      string
	power     int
	toughness int
	types     []CardType
	subTypes  []string
	keywords  []Keyword
}

// CreateTokenAttacking creates an effect that puts a creature token onto the
// battlefield attacking the defending player (the controller's opponent in a
// 2-player game). Per CR 508.4, such a creature is "attacking" but never
// "attacked"; AttacksTrigger abilities do not fire.
func CreateTokenAttacking(name string, power, toughness int, types []CardType, subTypes []string, keywords ...Keyword) Effect {
	return &createTokenAttackingEffect{
		name:      name,
		power:     power,
		toughness: toughness,
		types:     types,
		subTypes:  subTypes,
		keywords:  keywords,
	}
}

func execCreateTokenAttacking(ctx *EffectContext, e *createTokenAttackingEffect) error {
	token := NewToken(e.name, e.power, e.toughness, e.types, e.subTypes, e.keywords...)
	token.SetOwner(ctx.Controller)
	var defenderID uuid.UUID
	for _, p := range ctx.Game.AllPlayers() {
		if p.PlayerID() != ctx.Controller {
			defenderID = p.PlayerID()
			break
		}
	}
	perm := ctx.Game.PutOnBattlefieldAttacking(token, ctx.Controller, defenderID)
	perm.IsToken = true
	return nil
}

func (e *createTokenAttackingEffect) Text() string {
	return fmt.Sprintf("create a %d/%d %s token attacking defending player", e.power, e.toughness, e.name)
}
func (e *createTokenAttackingEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, TokenPower: e.power, TokenToughness: e.toughness}
}

// createTokenBlockingEffect creates a token creature that enters the
// battlefield blocking a target attacking creature (CR 509.4). The single
// target is the attacker the token will block.
type createTokenBlockingEffect struct {
	name      string
	power     int
	toughness int
	types     []CardType
	subTypes  []string
	keywords  []Keyword
}

// CreateTokenBlocking creates an effect that puts a creature token onto the
// battlefield blocking a target attacking creature. Per CR 509.4 the token is
// "blocking" but never "blocked"; BlocksTrigger abilities do not fire.
func CreateTokenBlocking(name string, power, toughness int, types []CardType, subTypes []string, keywords ...Keyword) Effect {
	return &createTokenBlockingEffect{
		name:      name,
		power:     power,
		toughness: toughness,
		types:     types,
		subTypes:  subTypes,
		keywords:  keywords,
	}
}

func execCreateTokenBlocking(ctx *EffectContext, e *createTokenBlockingEffect) error {
	token := NewToken(e.name, e.power, e.toughness, e.types, e.subTypes, e.keywords...)
	token.SetOwner(ctx.Controller)
	var attackerID uuid.UUID
	if len(ctx.Targets) > 0 {
		attackerID = ctx.Targets[0]
	}
	perm := ctx.Game.PutOnBattlefieldBlocking(token, ctx.Controller, attackerID)
	perm.IsToken = true
	return nil
}

func (e *createTokenBlockingEffect) Text() string {
	return fmt.Sprintf("create a %d/%d %s token blocking target attacker", e.power, e.toughness, e.name)
}
func (e *createTokenBlockingEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, TokenPower: e.power, TokenToughness: e.toughness}
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

func (e *cloneTargetEffect) Text() string {
	return "enters the battlefield as a copy of target permanent"
}
func (e *cloneTargetEffect) Properties() EffectProperties { return EffectProperties{} }

// copySpellOnStackEffect copies the top spell on the stack. Used by Fork.
type copySpellOnStackEffect struct{}

// CopySpellOnStack creates an effect that copies the target spell on the stack.
func CopySpellOnStack() Effect {
	return &copySpellOnStackEffect{}
}

func (e *copySpellOnStackEffect) Text() string {
	return "copy target instant or sorcery spell"
}
func (e *copySpellOnStackEffect) Properties() EffectProperties { return EffectProperties{} }

// attachToTargetEffect attaches the source (aura/equipment) to the target.
type attachToTargetEffect struct{}

// AttachToTarget creates an effect that attaches the source (aura or equipment) to the target permanent.
func AttachToTarget() Effect {
	return &attachToTargetEffect{}
}

func (e *attachToTargetEffect) Text() string                 { return "attach to target" }
func (e *attachToTargetEffect) Properties() EffectProperties { return EffectProperties{} }

// controlChangeTargetEffect gains control of a target permanent.
type controlChangeTargetEffect struct{}

// ControlChangeTarget creates an effect that gives the controller permanent control of a target (e.g. Control Magic).
func ControlChangeTarget() Effect {
	return &controlChangeTargetEffect{}
}

func (e *controlChangeTargetEffect) Text() string { return "gain control of target permanent" }
func (e *controlChangeTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// extraTurnEffect gives the controller an extra turn.
type extraTurnEffect struct{}

// ExtraTurn creates an effect that gives the controller an extra turn (e.g. Time Walk).
func ExtraTurn() Effect {
	return &extraTurnEffect{}
}

func (e *extraTurnEffect) Text() string { return "take an extra turn after this one" }
func (e *extraTurnEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// changeColorEffect changes a target permanent's color.
type changeColorEffect struct {
	color Color
}

// ChangeColorEffect creates an effect that changes a target permanent's color.
func ChangeColorEffect(color Color) Effect {
	return &changeColorEffect{color: color}
}

func (e *changeColorEffect) Text() string {
	return fmt.Sprintf("Target permanent becomes %s", e.color)
}
func (e *changeColorEffect) Properties() EffectProperties { return EffectProperties{} }

// counterUnlessPayEffect counters a target spell unless its controller pays a cost.
type counterUnlessPayEffect struct {
	cost string
}

// CounterUnlessPay creates an effect that counters a target spell unless its
// controller pays the specified mana cost (e.g. Force Spike, Mana Leak).
func CounterUnlessPay(cost string) Effect {
	return &counterUnlessPayEffect{cost: cost}
}

func (e *counterUnlessPayEffect) Text() string {
	return fmt.Sprintf("Counter target spell unless its controller pays %s", e.cost)
}
func (e *counterUnlessPayEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// forcefieldEffect activates a Forcefield shield on the controller for this turn.
type forcefieldEffect struct{}

// ForcefieldEffect creates an effect that reduces all unblocked combat damage to the controller
// to 1 for this turn (Forcefield).
func ForcefieldEffect() Effect {
	return &forcefieldEffect{}
}

func (e *forcefieldEffect) Text() string {
	return "prevent all but 1 combat damage from each unblocked creature this turn"
}
func (e *forcefieldEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// --- exec functions ---

func execCounterSpell(ctx *EffectContext, _ *counterSpellEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target spell to counter")
	}
	ctx.Game.CounterSpellOnStack(ctx.Targets[0])
	return nil
}

func execCounterSpellIfColor(ctx *EffectContext, e *counterSpellIfColorEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	// Find the spell on the stack
	obj := ctx.Game.FindStackObject(ctx.Targets[0])
	if obj == nil || obj.Card == nil {
		return nil
	}
	// Check if the spell has the required color
	for _, c := range obj.Card.ManaCost().Colors() {
		if c == e.color {
			ctx.Game.CounterSpellOnStack(ctx.Targets[0])
			return nil
		}
	}
	// Color doesn't match, spell is NOT countered
	return nil
}

func execCounterSpellIfXMeetsCMC(ctx *EffectContext, _ *counterSpellIfXMeetsOrExceedsCMCEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	obj := ctx.Game.FindStackObject(ctx.Targets[0])
	if obj == nil || obj.Card == nil {
		return nil
	}
	cmc := obj.Card.ManaCost().CMC()
	if ctx.Game.XValue() >= cmc {
		ctx.Game.CounterSpellOnStack(ctx.Targets[0])
	}
	return nil
}

func execPowerSink(ctx *EffectContext, _ *powerSinkEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	obj := ctx.Game.FindStackObject(ctx.Targets[0])
	if obj == nil {
		return nil
	}
	// Check if the spell's controller can pay X mana
	spellController := ctx.Game.GetPlayer(obj.Controller)
	if spellController == nil {
		return nil
	}
	// If they have enough mana in pool, they pay and spell resolves
	totalMana := spellController.ManaPool().TotalMana()
	if totalMana >= ctx.Game.XValue() {
		// Opponent can pay - drain X mana but don't counter
		spellController.ManaPool().DrainGeneric(ctx.Game.XValue())
		return nil
	}
	// Can't pay - counter the spell and drain all mana
	spellController.ManaPool().Clear()
	ctx.Game.CounterSpellOnStack(ctx.Targets[0])
	return nil
}

func execAddMana(ctx *EffectContext, e *addManaEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	p.ManaPool().Add(e.color, e.amount)
	return nil
}

func execAddAnyMana(ctx *EffectContext, e *addAnyManaEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	color := p.ChooseManaColor("add mana")
	p.ManaPool().Add(color, e.amount)
	return nil
}

func execCreateToken(ctx *EffectContext, e *createTokenEffect) error {
	count := e.count
	if count <= 0 {
		count = 1
	}
	for i := 0; i < count; i++ {
		token := NewToken(e.name, e.power, e.toughness, e.types, e.subTypes, e.keywords...)
		token.SetOwner(ctx.Controller)
		if len(e.colors) > 0 {
			token.colorOverride = make([]Color, len(e.colors))
			copy(token.colorOverride, e.colors)
		}
		perm := ctx.Game.PutOnBattlefield(token, ctx.Controller)
		perm.IsToken = true
	}
	return nil
}

func execCloneTarget(ctx *EffectContext, e *cloneTargetEffect) error {
	if len(ctx.Targets) == 0 || ctx.Game.GetResolvingCard() == nil {
		return nil
	}
	target := ctx.Game.FindPermanent(ctx.Targets[0])
	if target == nil {
		return nil
	}
	ctx.Game.GetResolvingCard().CloneFrom(target.Card)
	for _, t := range e.additionalTypes {
		ctx.Game.GetResolvingCard().AddType(t)
	}
	return nil
}

func execCopySpellOnStack(ctx *EffectContext, _ *copySpellOnStackEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	// Find the target spell on the stack
	original := ctx.Game.FindStackObject(ctx.Targets[0])
	if original == nil {
		return nil
	}
	// Create a copy of the stack object
	cp := &StackObject{
		ID:         uuid.New(),
		Card:       original.Card,
		Controller: ctx.Controller,
		SourceID:   original.SourceID,
		Effects:    make([]Effect, len(original.Effects)),
		Targets:    make([]uuid.UUID, len(original.Targets)),
		IsAbility:  original.IsAbility,
		XValue:     original.XValue,
	}
	copy(cp.Effects, original.Effects)
	copy(cp.Targets, original.Targets)
	ctx.Game.PushStack(cp)
	return nil
}

func execAttachToTarget(ctx *EffectContext, _ *attachToTargetEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for attach")
	}
	ctx.Game.Attach(ctx.SourceID, ctx.Targets[0])
	return nil
}

func execControlChangeTarget(ctx *EffectContext, _ *controlChangeTargetEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for control change")
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	perm.Controller = ctx.Controller
	return nil
}

func execExtraTurn(ctx *EffectContext, _ *extraTurnEffect) error {
	ctx.Game.GrantExtraTurn(ctx.Controller)
	return nil
}

func execChangeColor(ctx *EffectContext, e *changeColorEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	// Register as a continuous effect so the color change persists
	ce := ColorOverride(perm.ID(), e.color)
	ce.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(ce)
	return nil
}

func execCounterUnlessPay(ctx *EffectContext, e *counterUnlessPayEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	obj := ctx.Game.FindStackObject(ctx.Targets[0])
	if obj == nil {
		return nil
	}
	spellController := ctx.Game.GetPlayer(obj.Controller)
	if spellController == nil {
		return nil
	}
	if ctx.Game.TryPayCostFromLands(obj.Controller, e.cost) {
		return nil // paid, spell resolves
	}
	ctx.Game.CounterSpellOnStack(ctx.Targets[0])
	return nil
}

func execForcefield(ctx *EffectContext, _ *forcefieldEffect) error {
	ctx.Game.AddForcefieldShield(ctx.Controller)
	return nil
}
