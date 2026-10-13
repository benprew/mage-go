package ron

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type optionFactory func() mage.CardOption
type effectFactory func(instanceBindings) mage.Effect
type targetFactory func(instanceBindings) mage.Target
type costFactory func() mage.Cost

type scope struct {
	drawReplacementBody  bool
	drawReplacement      *drawReplacementPlan
	playerTap            *playerTapPlan
	playerDiscard        *playerDiscardPlan
	sacrificeSubtype     *sacrificeSubtypePlan
	colorWords           *textWordChoicePlan
	landWords            *textWordChoicePlan
	sourceCardPayment    *sourceCardPaymentPlan
	groupedExile         *groupedExilePlan
	exileLink            *mage.ExileLink
	drawnCard            *handRevealPlan
	handReveal           *handRevealPlan
	libraryReveal        *libraryRevealPlan
	chosenName           *cardNameChoicePlan
	anteSourcePayment    bool
	chosenPlayer         *playerChoicePlan
	chosenPermanent      *permanentChoicePlan
	permanentChoices     *permanentChoiceHistoryPlan
	createdValue         *createdValuePlan
	chosenDamageSource   *chosenDamageSourcePlan
	spellTargets         bool
	destructionCount     *destructionCountPlan
	chosenAbility        *temporaryAbilityPlan
	sourceTapped         bool
	sourceLeaves         bool
	counterStateSource   bool
	opponentEntryChoice  *opponentEntryChoicePlan
	basicLandEntryChoice *basicLandEntryChoicePlan

	activationHistory *activationHistoryPlan
	enteringSource    bool
	actionPlayer      *playerActionsPlan
	damageRecipient   *playerReferencePlan
	activation        bool
	targetSurcharge   int
	castingCondition  mage.StateCondition
	castingCost       *Node
	damagePlayer      *damagePlayerPlan
	damageAmount      *damageAmountPlan
	receivedSource    bool
	delayedStep       bool
	sourceAttack      bool
	sourceBlock       bool
	beginCombat       bool
	defendingPlayer   *sourceCombatPlayerPlan
	combatOpponent    *permanentEventPlan
	targetNamespace   string
	exilePayment      *exilePaymentPlan
	counterPayment    *counterPaymentPlan
	sacrificeValue    *sacrificeValuePlan
	target            string
	x                 bool
	permanent         bool
	aura              bool
	player            playerReferencePlan
	playerTarget      *playerTargetPlan
	permanentTarget   *permanentTargetPlan
	permanentGroups   map[string]*permanentTargetPlan
	permanentTargets  *permanentTargetsPlan
	spellTarget       *spellTargetPlan
	abilityTarget     *abilityTargetPlan
	cardTarget        *cardTargetPlan
	deathCard         *deathCardPlan
	hostDeath         *hostDeathPlan
	delayedDeath      *hostDeathPlan
	damageTarget      *damageTargetPlan
	damageGroups      map[string]*damageTargetPlan
	damageTargets     *damageTargetsPlan
	castEvent         *castEventPlan
	drawEvent         *drawEventPlan
	stepEvent         *stepEventPlan
	permanentEvent    *permanentEventPlan
	unblockedCreature *permanentEventPlan
	landManaEvent     *landManaEventPlan
	graveyardEvent    *graveyardEventPlan
	landEntryEvent    *landEntryEventPlan
}

type actionPlan struct {
	distribution     *distributedDamagePlan
	effects          []effectFactory
	targets          []targetFactory
	playerTarget     *playerTargetPlan
	permanentTarget  *permanentTargetPlan
	permanentGroups  []*permanentTargetPlan
	permanentTargets *permanentTargetsPlan
	spellTarget      *spellTargetPlan
	abilityTarget    *abilityTargetPlan
	cardTarget       *cardTargetPlan
	damageTarget     *damageTargetPlan
	damageGroups     []*damageTargetPlan
	damageTargets    *damageTargetsPlan
}

func (p actionPlan) build() Actions {
	return p.buildWithEvents(nil)
}

func (p actionPlan) buildWithEvents(events map[*castEventPlan]castEventBindings) Actions {
	return p.buildWithPayments(events, nil)
}

func (p actionPlan) buildWithPayments(events map[*castEventPlan]castEventBindings, sacrifices map[*sacrificeValuePlan]sacrificeValueBindings) Actions {
	return p.buildWithBindings(instanceBindings{sacrifices: sacrifices, events: events})
}

func (p actionPlan) buildWithBindings(bindings instanceBindings) Actions {
	if p.distribution != nil {
		return p.distribution.build()
	}
	targets := make([]mage.Target, len(p.targets))
	bindings = initializeActionBindings(bindings)
	var declared []mage.TargetBinding
	for i, makeTarget := range p.targets {
		targets[i] = makeTarget(bindings)
		if target, ok := targets[i].(*mage.PermanentTargetBinding); ok {
			if len(p.permanentGroups) != 0 {
				bindings.permanents[p.permanentGroups[i]] = target
			} else {
				bindings.permanents[p.permanentTarget] = target
			}
		}
		if target, ok := targets[i].(*mage.PermanentTargetsBinding); ok {
			bindings.permanentSets[p.permanentTargets] = target
		}
		if target, ok := targets[i].(*mage.PlayerTargetBinding); ok {
			bindings.players[p.playerTarget] = target
		}
		if target, ok := targets[i].(*mage.CardTargetBinding); ok {
			bindings.cards[p.cardTarget] = target
		}
		if target, ok := targets[i].(*mage.AbilityTargetBinding); ok {
			bindings.abilities[p.abilityTarget] = target
		}
		if target, ok := targets[i].(*mage.SpellTargetBinding); ok {
			bindings.spells[p.spellTarget] = target
		}
		if target, ok := targets[i].(*mage.DamageTargetsBinding); ok {
			bindings.recipientSets[p.damageTargets] = target
		}
		if target, ok := targets[i].(*mage.DamageTargetBinding); ok {
			if len(p.damageGroups) != 0 {
				bindings.recipients[p.damageGroups[i]] = target
			} else {
				bindings.recipients[p.damageTarget] = target
			}
		}
		if target, ok := targets[i].(mage.TargetBinding); ok {
			declared = append(declared, target)
		}
	}
	effects := make([]mage.Effect, len(p.effects))
	for i, makeEffect := range p.effects {
		effects[i] = makeEffect(bindings)
	}
	if len(declared) != 0 {
		targets = mage.Spell(mage.Sequence(effects...), mage.WithTargets(declared...)).Targets()
	}
	return Targeted(targets, ActionList(effects...))
}

func initializeActionBindings(bindings instanceBindings) instanceBindings {
	if bindings.sacrificeSubtypes == nil {
		bindings.sacrificeSubtypes = make(map[*sacrificeSubtypePlan]*mage.NumberBinding)
	}
	bindings.textWords = make(map[*textWordChoicePlan]textWordBindings)
	bindings.chosenNames = make(map[*cardNameChoicePlan]*mage.CardNameBinding)
	bindings.libraryReveals = make(map[*libraryRevealPlan]*mage.CardBinding)
	bindings.players = make(map[*playerTargetPlan]*mage.PlayerTargetBinding)
	bindings.permanents = make(map[*permanentTargetPlan]*mage.PermanentTargetBinding)
	bindings.permanentSets = make(map[*permanentTargetsPlan]*mage.PermanentTargetsBinding)
	bindings.spells = make(map[*spellTargetPlan]*mage.SpellTargetBinding)
	bindings.abilities = make(map[*abilityTargetPlan]*mage.AbilityTargetBinding)
	bindings.cards = make(map[*cardTargetPlan]*mage.CardTargetBinding)
	bindings.recipients = make(map[*damageTargetPlan]*mage.DamageTargetBinding)
	bindings.recipientSets = make(map[*damageTargetsPlan]*mage.DamageTargetsBinding)
	return bindings
}

func problem(n Node, message string) error {
	return fmt.Errorf("%s: %s (%s)", n.Position, message, n.Name)
}
func unsupported(n Node, role string) error { return problem(n, "unsupported "+role) }
func arity(n Node, count int) error {
	if n.Kind != Tuple || len(n.Args) != count {
		return problem(n, fmt.Sprintf("expected %d positional arguments", count))
	}
	return nil
}
func list(n Node) ([]Node, error) {
	if n.Kind != List {
		return nil, problem(n, "expected list")
	}
	return n.Args, nil
}
func symbol(n Node) (string, error) {
	if n.Kind != Symbol {
		return "", problem(n, "expected symbol")
	}
	return n.Name, nil
}
func fields(n Node, required, optional []string) error {
	if n.Kind != Struct {
		return problem(n, "expected named fields")
	}
	for _, key := range required {
		if _, ok := n.Fields[key]; !ok {
			return problem(n, "missing field "+key)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(n.Fields)) {
		if !slices.Contains(required, key) && !slices.Contains(optional, key) {
			return problem(n, "unknown field "+key)
		}
	}
	return nil
}

// Compile validates a complete card and returns a factory with private instance state.
// No declaration can use a registered Go card as a substitute for a RON rule.
func Compile(n Node) (mage.CardFactory, error) {
	n = cloneTextNode(n)
	if n.Name != "Card" {
		return nil, problem(n, "expected Card")
	}
	if err := fields(n, []string{"Name", "Typeline"}, []string{"ManaCost", "Rules", "CardPT", "ColorIndicator"}); err != nil {
		return nil, err
	}
	nameNode := n.Fields["Name"]
	if nameNode.Kind != String || strings.TrimSpace(nameNode.String) == "" {
		return nil, problem(nameNode, "expected nonempty card name")
	}
	name := nameNode.String
	typeline := n.Fields["Typeline"]
	if err := fields(typeline, []string{"Supertypes", "Cardtypes", "Subtypes"}, nil); err != nil {
		return nil, err
	}
	typeNodes, err := list(typeline.Fields["Cardtypes"])
	if err != nil {
		return nil, err
	}
	if len(typeNodes) == 0 {
		return nil, problem(typeline, "card must have a type")
	}
	types := make([]core.CardType, len(typeNodes))
	for i, typeNode := range typeNodes {
		types[i], err = cardType(typeNode)
		if err != nil {
			return nil, err
		}
		if slices.Contains(types[:i], types[i]) {
			return nil, problem(typeNode, "duplicate card type")
		}
	}
	var supers []core.SuperType
	superNodes, err := list(typeline.Fields["Supertypes"])
	if err != nil {
		return nil, err
	}
	for _, superNode := range superNodes {
		value, err := symbol(superNode)
		if err != nil {
			return nil, err
		}
		super, ok := map[string]core.SuperType{"Basic": core.SuperBasic, "Legendary": core.SuperLegendary, "Snow": core.SuperSnow, "World": core.SuperWorld}[value]
		if !ok {
			return nil, unsupported(superNode, "supertype")
		}
		supers = append(supers, super)
	}
	var subtypes []string
	subNodes, err := list(typeline.Fields["Subtypes"])
	if err != nil {
		return nil, err
	}
	for _, subNode := range subNodes {
		subtype, err := symbol(subNode)
		if err != nil {
			return nil, err
		}
		subtypes = append(subtypes, subtypeName(subtype))
	}
	var cost string
	var hasX bool
	if mana, ok := n.Fields["ManaCost"]; ok {
		cost, err = manaCost(mana)
		if err != nil {
			return nil, err
		}
		hasX = core.ParseManaCost(cost).HasX
	}
	if slices.Contains(types, core.TypeLand) && cost != "" {
		return nil, unsupported(n.Fields["ManaCost"], "mana cost on a land")
	}
	var indicator []core.Color
	if colorNode, ok := n.Fields["ColorIndicator"]; ok {
		colors, err := list(colorNode)
		if err != nil {
			return nil, err
		}
		for _, entry := range colors {
			value, err := color(entry)
			if err != nil {
				return nil, err
			}
			indicator = append(indicator, value)
		}
	}
	power, toughness := 0, 0
	if pt, ok := n.Fields["CardPT"]; ok {
		if err := fields(pt, []string{"Power", "Toughness"}, nil); err != nil {
			return nil, err
		}
		if pt.Fields["Power"].Kind != IntegerNode || pt.Fields["Toughness"].Kind != IntegerNode {
			return nil, problem(pt, "base power and toughness must be integers")
		}
		power, toughness = pt.Fields["Power"].Integer, pt.Fields["Toughness"].Integer
	} else if slices.Contains(types, core.TypeCreature) {
		return nil, problem(n, "creature requires CardPT")
	}
	permanent := !slices.Contains(types, core.TypeInstant) && !slices.Contains(types, core.TypeSorcery)
	ctx := scope{x: hasX, permanent: permanent, aura: slices.Contains(subtypes, "Aura"), player: playerReferencePlan{}}
	var options []optionFactory
	attachedObjectText := false
	landAuraSpellText := false
	staticActivationSpellText := false
	protectionPTSpellText := false
	intrinsicCastSpellText := false
	intrinsicCreatureSpellText := false
	copyEntrySpellText := false
	mixedStaticEventSpellText := false
	globalStaticSpellText := false
	manaTapSpellText := false
	upkeepObjectText := false
	upkeepSpellText := false
	spellCastSpellText := false
	spellCount, enchantCount, deckCount := 0, 0, 0
	if rulesNode, ok := n.Fields["Rules"]; ok {
		rules, err := list(rulesNode)
		if err != nil {
			return nil, err
		}
		mixedStaticEventSpellText = permanent && !slices.Contains(types, core.TypeLand) && !ctx.aura && mixedStaticEventSpellTextRules(rules)
		manaTapSpellText = permanent && !slices.Contains(types, core.TypeLand) && !ctx.aura && manaTapTextRules(rules)
		upkeepObjectText = permanent && !slices.Contains(types, core.TypeLand) && !ctx.aura && upkeepObjectTextRules(rules)
		upkeepSpellText = upkeepObjectText && upkeepSpellTextRules(rules)
		spellCastSpellText = permanent && !slices.Contains(types, core.TypeLand) && !ctx.aura && spellCastTextRules(rules)
		globalStaticSpellText = permanent && !slices.Contains(types, core.TypeLand) && !ctx.aura && (globalStaticSpellTextRules(rules) || globalGrantSpellTextRules(rules))
		copyEntrySpellText = permanent && copyEntrySpellTextRules(rules)
		attachedObjectText = slices.Equal(types, []core.CardType{core.TypeEnchantment}) && slices.Equal(subtypes, []string{"Aura"}) && attachedObjectTextRules(rules)
		landAuraSpellText = slices.Equal(types, []core.CardType{core.TypeEnchantment}) && slices.Equal(subtypes, []string{"Aura"}) && landAuraSpellTextRules(rules)
		staticActivationSpellText = slices.Equal(types, []core.CardType{core.TypeCreature}) && !ctx.aura && staticActivationTextRules(rules)
		protectionPTSpellText = slices.Equal(types, []core.CardType{core.TypeCreature}) && !ctx.aura && protectionPTTextRules(rules)
		intrinsicCastSpellText = slices.Equal(types, []core.CardType{core.TypeCreature}) && !ctx.aura && intrinsicCastSpellTextRules(rules)
		intrinsicCreatureSpellText = slices.Equal(types, []core.CardType{core.TypeCreature}) && intrinsicCreatureSpellTextRules(rules)
		ctx.exileLink, err = compileExileAssociation(rules, permanent)
		if err != nil {
			return nil, err
		}
		characteristic, err := compileCharacteristicCounts(rules, permanent && slices.Contains(types, core.TypeCreature))
		if err != nil {
			return nil, err
		}
		if characteristic != nil {
			options = append(options, definitionAbilityGroupOption(0, characteristic))
		}
		ctx.opponentEntryChoice, err = compileOpponentEntryDeclaration(rules, permanent)
		if err != nil {
			return nil, err
		}
		ctx.basicLandEntryChoice, err = compileBasicLandEntryDeclaration(rules, permanent)
		if err != nil {
			return nil, err
		}
		castingRule := false
		for _, rule := range rules {
			if rule.Name != "CastEffect" {
				continue
			}
			if permanent {
				return nil, unsupported(rule, "casting window on a permanent")
			}
			if castingRule {
				return nil, problem(rule, "duplicate CastEffect rule")
			}
			castingRule = true
			if rule.Kind == Tuple && len(rule.Args) == 1 && rule.Args[0].Name == "AdditionalCastingCost" {
				ctx.castingCost, ctx.sacrificeValue, err = compileAdditionalSacrifice(rule.Args[0], rules)
				if err != nil {
					return nil, err
				}
				continue
			}
			if rule.Kind == Tuple && len(rule.Args) == 1 && rule.Args[0].Name == "AdditionalCastingCostForEachTargetBeyondTheFirst" {
				ctx.targetSurcharge, err = compileTargetSurcharge(rule.Args[0])
				if err != nil {
					return nil, err
				}
				continue
			}
			ctx.castingCondition, err = compileCastingWindow(rule)
			if err != nil {
				return nil, err
			}
		}
		for ruleIndex, rule := range rules {
			if rule.Name == "CastEffect" || rule.Name == "CDA_Power" || rule.Name == "CDA_Toughness" {
				continue
			}
			if rule.Name == "DeckConstruction" {
				deckCount++
				if deckCount > 1 {
					return nil, problem(rule, "duplicate DeckConstruction rule")
				}
				option, err := compileDeckConstruction(rule)
				if err != nil {
					return nil, err
				}
				options = append(options, definitionAbilityGroupOption(ruleIndex+1, option))
				continue
			}
			if rule.Name == "SpellActions" {
				spellCount++
			}
			if rule.Name == "EnchantPermanent" {
				enchantCount++
			}
			option, err := compileRule(rule, ctx)
			if err != nil {
				return nil, err
			}
			options = append(options, definitionAbilityGroupOption(ruleIndex+1, option))
		}
	}
	if !permanent && spellCount != 1 {
		return nil, problem(n, "instant or sorcery requires one SpellActions rule")
	}
	if permanent && spellCount != 0 {
		return nil, problem(n, "permanent cannot declare SpellActions")
	}
	if slices.Contains(subtypes, "Aura") && enchantCount != 1 {
		return nil, problem(n, "Aura requires one enchant rule")
	}
	if enchantCount > 0 && (!slices.Contains(types, core.TypeEnchantment) || !slices.Contains(subtypes, "Aura")) {
		return nil, problem(n, "enchant rule requires an Aura enchantment")
	}
	if types[0] == core.TypePlaneswalker {
		return nil, unsupported(n, "planeswalker metadata")
	}
	_, costPresent := n.Fields["ManaCost"]
	var definition *mage.CardDefinitionPlan
	factory := func() mage.Card {
		opts := make([]mage.CardOption, 0, 1+len(types)+len(options))
		opts = append(opts, mage.WithSubTypes(subtypes...), mage.WithSuperTypes(supers...))
		for _, kind := range types[1:] {
			opts = append(opts, mage.WithCardType(kind))
		}
		for _, buildOption := range options {
			opts = append(opts, buildOption())
		}
		if mixedStaticEventSpellText {
			opts = append(opts, mage.WithMixedStaticEventSpellTextContinuation())
		}
		if manaTapSpellText {
			opts = append(opts, mage.WithManaTapSpellTextContinuation())
		}
		if upkeepObjectText {
			opts = append(opts, mage.WithUpkeepObjectTextRevision())
			if upkeepSpellText {
				opts = append(opts, mage.WithUpkeepSpellTextContinuation())
			}
		}
		if spellCastSpellText {
			opts = append(opts, mage.WithSpellCastSpellTextContinuation())
		}
		if globalStaticSpellText {
			opts = append(opts, mage.WithGlobalStaticSpellTextContinuation())
		}
		if attachedObjectText {
			opts = append(opts, mage.WithAttachedObjectText())
		}
		if landAuraSpellText {
			opts = append(opts, mage.WithLandTypeAuraSpellTextContinuation())
		}
		if copyEntrySpellText {
			opts = append(opts, mage.WithCopyEntrySpellTextContinuation())
		}
		if staticActivationSpellText {
			opts = append(opts, mage.WithStaticActivationSpellTextContinuation())
		}
		if protectionPTSpellText {
			opts = append(opts, mage.WithProtectionPTSpellTextContinuation())
		}
		if intrinsicCastSpellText {
			opts = append(opts, mage.WithIntrinsicCastEventSpellTextContinuation())
		}
		if intrinsicCreatureSpellText {
			opts = append(opts, mage.WithIntrinsicCreatureSpellTextContinuation())
		}
		var card *mage.BaseCard
		switch types[0] {
		case core.TypeCreature:
			card = mage.NewCreature(name, cost, power, toughness, opts...)
		case core.TypeArtifact:
			card = mage.NewArtifact(name, cost, opts...)
		case core.TypeEnchantment:
			card = mage.NewEnchantment(name, cost, opts...)
		case core.TypeLand:
			card = mage.NewLand(name, opts...)
		case core.TypeInstant:
			card = mage.NewInstant(name, cost)
			for _, opt := range opts {
				opt(card)
			}
		case core.TypeSorcery:
			card = mage.NewSorcery(name, cost)
			for _, opt := range opts {
				opt(card)
			}
		}
		if types[0] != core.TypeCreature {
			card.SetBasePT(power, toughness)
		}
		if len(indicator) > 0 {
			card.SetColorOverride(indicator)
		}
		mage.WithTextDefinitionBuilder(func(changes []mage.TextChange) (mage.CardFactory, error) {
			changed, err := transformTextDefinition(n, changes)
			if err != nil {
				return nil, err
			}
			return Compile(changed)
		})(card)
		mage.WithManaCostPresence(costPresent)(card)
		mage.WithCardDefinitionPlan(definition)(card)
		mage.WithPrintedCardDefinitionPlan(definition)(card)
		mage.WithCopiableValues(definition.CopiableValues())(card)
		return card
	}
	definition, err = mage.NewCardDefinitionPlan(factory)
	if err != nil {
		return nil, problem(n, "invalid card definition plan: "+err.Error())
	}
	return factory, nil
}

func cardType(n Node) (core.CardType, error) {
	name, err := symbol(n)
	if err != nil {
		return 0, err
	}
	kind, ok := map[string]core.CardType{"Creature": core.TypeCreature, "Artifact": core.TypeArtifact, "Land": core.TypeLand, "Enchantment": core.TypeEnchantment, "Instant": core.TypeInstant, "Sorcery": core.TypeSorcery}[name]
	if !ok {
		return 0, unsupported(n, "card type")
	}
	return kind, nil
}

func manaCost(n Node) (string, error) {
	nodes, err := list(n)
	if err != nil {
		return "", err
	}
	var cost strings.Builder
	for _, part := range nodes {
		if part.Name == "ManaCostGeneric" {
			if err := arity(part, 1); err != nil {
				return "", err
			}
			if part.Args[0].Kind != IntegerNode || part.Args[0].Integer < 0 {
				return "", problem(part, "generic mana must be a nonnegative integer")
			}
			cost.WriteString("{" + strconv.Itoa(part.Args[0].Integer) + "}")
			continue
		}
		name, err := symbol(part)
		if err != nil {
			return "", err
		}
		value, ok := map[string]string{"ManaCostW": "W", "ManaCostU": "U", "ManaCostB": "B", "ManaCostR": "R", "ManaCostG": "G", "ManaCostX": "X"}[name]
		if !ok {
			return "", unsupported(part, "mana symbol")
		}
		cost.WriteString("{" + value + "}")
	}
	return cost.String(), nil
}

func compileRule(n Node, ctx scope) (optionFactory, error) {
	if !ctx.permanent && n.Name != "SpellActions" {
		return nil, unsupported(n, "non-spell rule on an instant or sorcery")
	}
	if n.Kind == Symbol {
		kw, ok := keywords[n.Name]
		if !ok {
			return nil, unsupported(n, "rule")
		}
		return func() mage.CardOption { return mage.WithKeyword(kw) }, nil
	}
	switch n.Name {
	case "FromStack":
		return compileStackSelfCast(n, ctx)
	case "ReplaceWouldDraw":
		return compileDrawReplacement(n, ctx)
	case "ReplaceWouldBeginDrawStep":
		return compileDrawStepLifeReplacement(n)
	case "ActivatedAbilityEffect":
		return compileActivationCostModifier(n, ctx)
	case "EachPlayerEffect":
		return compileAllPlayerRule(n)
	case "PlayerEffect":
		if isHandSizeRule(n) {
			return compileHandSizeRule(n, ctx)
		}
		return compilePlayerCostReduction(n)
	case "PreventDamage":
		return compileStaticPrevention(n, ctx)
	case "If", "Unless", "IfElse":
		return compileConditionalStatic(n, ctx)
	case "Rampage":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		amount := n.Args[0]
		if amount.Kind != IntegerNode || amount.Integer < 0 {
			return nil, problem(amount, "rampage requires a nonnegative integer literal")
		}
		return func() mage.CardOption { return mage.WithAbility(mage.RampageTrigger(amount.Integer)) }, nil
	case "AsPermanentEnters":
		return compilePermanentEntry(n)
	case "TriggerA":
		return compileTriggerRule(n, ctx)
	case "TriggerI":
		return compileTappedUpkeep(n, ctx)
	case "EachPermanentRuleEffect":
		return compileCollectionPermanentRule(n)
	case "PermanentRuleEffect":
		return compilePermanentBlockingRule(n, ctx)
	case "PermanentLayerEffect", "EachPermanentLayerEffect":
		return compileStaticLayer(n, ctx)
	case "SpellActions":
		ctx.spellTargets = true
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if ctx.targetSurcharge > 0 && (n.Args[0].Name == "Targeted_Modal" || n.Args[0].Name == "Modal_ChooseOne") {
			return nil, unsupported(n.Args[0], "target surcharge on a modal spell")
		}
		if n.Args[0].Name == "Targeted_Modal" {
			return compileObjectTargetBody(n.Args[0], ctx)
		}
		if n.Args[0].Name == "Modal_ChooseOne" {
			if ctx.castingCost != nil {
				return nil, unsupported(n.Args[0], "additional sacrifice cost on a modal spell")
			}
			return compileModalSpell(n.Args[0], ctx)
		}
		var plan actionPlan
		var err error
		if n.Args[0].Name == "TargetedDistributed" {
			plan, err = compileDistributedDamage(n.Args[0], ctx)
		} else {
			plan, err = compileActions(n.Args[0], ctx)
		}
		if err != nil {
			return nil, err
		}
		var costs []boundCostFactory
		if ctx.castingCost != nil {
			costs, _, err = compileBoundCosts(*ctx.castingCost, ctx.sacrificeValue)
			if err != nil {
				return nil, err
			}
		}
		return func() mage.CardOption {
			bindings := instanceBindings{sacrifices: make(map[*sacrificeValuePlan]sacrificeValueBindings)}
			if ctx.sacrificeValue != nil {
				bindings.sacrifices[ctx.sacrificeValue] = sacrificeValueBindings{power: mage.BindNumber("paid sacrifice power"), value: mage.BindNumber("paid sacrifice value")}
			}
			options := actionOptions(plan.buildWithBindings(bindings))
			if ctx.castingCost == nil && ctx.targetSurcharge == 0 {
				options = append(options, mage.WithSpellTextContinuation())
			} else if ctx.sacrificeValue != nil && ctx.sacrificeValue.name == "ManaValueOfTheSacrificedPermanent" {
				options = append(options, mage.WithSpellSacrificeManaValueContinuation(bindings.sacrifices[ctx.sacrificeValue].value))
			}
			if ctx.targetSurcharge > 0 {
				options = append(options, mage.WithAdditionalManaPerTargetBeyondFirst(ctx.targetSurcharge))
			}
			if ctx.castingCondition != nil {
				options = append(options, mage.WithCastingCondition(ctx.castingCondition))
			}
			action := mage.NewAction(mage.ActionSpell, options...)
			return func(card *mage.BaseCard) {
				mage.WithAbility(action)(card)
				for _, cost := range costs {
					mage.WithAdditionalCost(cost(bindings))(card)
				}
			}
		}, nil
	case "Activated", "ActivatedWithModifiers":
		return compileActivatedRule(n, ctx)
	case "EnchantPermanent":
		return compileEnchantPermanent(n, ctx)
	case "Protection":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		colors, err := protectionColors(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(colors) == 0 {
			return nil, problem(n, "printed protection requires at least one color")
		}
		return func() mage.CardOption {
			return mage.WithAbility(mage.ProtectionFromColors(append([]core.Color(nil), colors...)...).WithLayeredColorTextRevision())
		}, nil
	case "Landwalk":
		kw, err := compileLandwalkKeyword(n)
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption {
			if kw == core.Desertwalk || kw == core.LegendaryLandwalk {
				return mage.WithKeyword(kw)
			}
			return mage.WithLayeredPrintedBasicLandwalk(kw)
		}, nil
	default:
		return nil, unsupported(n, "rule")
	}
}

func compileLandwalkKeyword(n Node) (core.Attr, error) {
	if err := arity(n, 1); err != nil {
		return 0, err
	}
	filter := n.Args[0]
	if filter.Name == "IsSupertype" {
		if err := arity(filter, 1); err != nil {
			return 0, err
		}
		if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Legendary" {
			return 0, unsupported(filter.Args[0], "landwalk supertype")
		}
		return core.LegendaryLandwalk, nil
	}
	if filter.Name != "IsLandType" {
		return 0, unsupported(filter, "landwalk filter")
	}
	if err := arity(filter, 1); err != nil {
		return 0, err
	}
	name, err := symbol(filter.Args[0])
	if err != nil {
		return 0, err
	}
	kw, ok := map[string]core.Attr{"Plains": core.Plainswalk, "Island": core.Islandwalk, "Swamp": core.Swampwalk, "Mountain": core.Mountainwalk, "Forest": core.Forestwalk, "Desert": core.Desertwalk}[name]
	if !ok {
		return 0, unsupported(filter, "landwalk type")
	}
	return kw, nil
}

var keywords = map[string]core.Attr{
	"Flying": core.Flying, "FirstStrike": core.FirstStrike, "Trample": core.Trample,
	"Vigilance": core.Vigilance, "Haste": core.Haste, "Reach": core.Reach,
	"Banding": core.Banding, "Defender": core.Defender, "Fear": core.Fear,
	"Deathtouch": core.Deathtouch, "Lifelink": core.Lifelink, "Shroud": core.Shroud,
	"Hexproof": core.Hexproof, "Menace": core.Menace, "Indestructible": core.Indestructible,
}

func compileCosts(n Node) ([]costFactory, bool, error) {
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return nil, false, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, false, err
		}
		var costs []costFactory
		x := false
		for _, part := range parts {
			compiled, hasX, err := compileCosts(part)
			if err != nil {
				return nil, false, err
			}
			costs = append(costs, compiled...)
			x = x || hasX
		}
		return costs, x, nil
	}
	switch n.Name {
	case "DiscardACard":
		if _, err := symbol(n); err != nil {
			return nil, false, err
		}
		return []costFactory{func() mage.Cost { return mage.DiscardCost(1) }}, false, nil
	case "DiscardACardAtRandom":
		if _, err := symbol(n); err != nil {
			return nil, false, err
		}
		return []costFactory{func() mage.Cost { return mage.DiscardRandomCost(1) }}, false, nil
	case "RemoveCounters":
		cost, err := compileCounterRemovalCost(n)
		if err != nil {
			return nil, false, err
		}
		return []costFactory{cost}, false, nil
	case "TapAPermanent", "TapNumberPermanents":
		cost, err := compileMatchingTapCost(n)
		if err != nil {
			return nil, false, err
		}
		return []costFactory{cost}, false, nil
	case "SacrificeNumberPermanents":
		cost, err := compileSacrificeNumberCost(n)
		if err != nil {
			return nil, false, err
		}
		return []costFactory{cost}, false, nil
	case "SacrificeAPermanent":
		if err := arity(n, 1); err != nil {
			return nil, false, err
		}
		filter, err := compileSacrificeCostFilter(n.Args[0])
		if err != nil {
			return nil, false, err
		}
		return []costFactory{func() mage.Cost {
			return mage.SacrificeMatchingIncludingSourceCost(filter, "Sacrifice a matching permanent")
		}}, false, nil
	case "PayLife":
		if err := arity(n, 1); err != nil {
			return nil, false, err
		}
		amount := n.Args[0]
		if amount.Name != "Integer" {
			return nil, false, unsupported(amount, "fixed life cost amount")
		}
		if err := arity(amount, 1); err != nil {
			return nil, false, err
		}
		if amount.Args[0].Kind != IntegerNode || amount.Args[0].Integer < 0 {
			return nil, false, problem(amount, "life cost requires a nonnegative integer")
		}
		value := amount.Args[0].Integer
		return []costFactory{func() mage.Cost { return mage.LifePayCost(value) }}, false, nil
	case "PayMana", "PayManaX":
		count := 1
		if n.Name == "PayManaX" {
			count = 2
		}
		if err := arity(n, count); err != nil {
			return nil, false, err
		}
		cost, err := manaCost(n.Args[0])
		if err != nil {
			return nil, false, err
		}
		parsed := core.ParseManaCost(cost)
		if n.Name == "PayManaX" {
			if n.Args[1].Kind != Symbol || n.Args[1].Name != "ValueX" {
				return nil, false, unsupported(n.Args[1], "announced X payment binding")
			}
			if !parsed.HasX {
				return nil, false, problem(n, "PayManaX requires an X mana symbol")
			}
		} else if parsed.HasX {
			return nil, false, problem(n, "X payment requires PayManaX binding")
		}
		return []costFactory{func() mage.Cost { return PayMana(parsed) }}, parsed.HasX, nil
	case "TapPermanent", "SacrificePermanent":
		if err := arity(n, 1); err != nil {
			return nil, false, err
		}
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" {
			return nil, false, unsupported(n.Args[0], "cost reference")
		}
		if n.Name == "TapPermanent" {
			return []costFactory{func() mage.Cost { return mage.Tap() }}, false, nil
		}
		return []costFactory{mage.SacrificeSourceCost}, false, nil
	default:
		return nil, false, unsupported(n, "cost")
	}
}

func compileActions(n Node, ctx scope) (actionPlan, error) {
	var plan actionPlan
	if n.Name == "Targeted" {
		if err := arity(n, 2); err != nil {
			return plan, err
		}
		targets, err := list(n.Args[0])
		if err != nil {
			return plan, err
		}
		if len(targets) == 1 && targets[0].Name == "AnyNumberOfAnyTargets" {
			return compilePluralUnionDamage(n, targets[0], ctx)
		}
		if len(targets) > 1 {
			if isUnionDamageTarget(targets[0]) {
				return compileUnionDamageGroups(n, targets, ctx)
			}
			return compileSingularPermanentGroups(n, targets, ctx)
		}
		if len(targets) == 0 {
			return plan, problem(n, "target declaration list must not be empty")
		}
		buildTarget, binding, err := compileTarget(targets[0], ctx)
		if err != nil {
			return plan, err
		}
		ctx.target = binding
		if binding == "Ref_TargetAbility" {
			ctx.abilityTarget = &abilityTargetPlan{name: binding}
		}
		if binding == "Ref_TargetPermanent" {
			ctx.permanentTarget = &permanentTargetPlan{name: binding}
		}
		if binding == "Ref_TargetPermanents" {
			filter := targets[0].Args[len(targets[0].Args)-1]
			ctx.permanentTargets = &permanentTargetsPlan{name: binding, filter: filter}
		}
		if binding == "Ref_TargetPlayer" {
			ctx.playerTarget = &playerTargetPlan{name: binding}
		}
		if binding == "Ref_TargetAnteCard" {
			ctx.cardTarget = &cardTargetPlan{name: binding, ante: true}
		}
		if binding == "Ref_TargetGraveyardCard" {
			ctx.cardTarget = &cardTargetPlan{name: binding, creatureOnly: graveyardFilterRequiresCreature(targets[0].Args[0])}
		}
		if binding == "Ref_TargetSpell" {
			ctx.spellTarget = &spellTargetPlan{name: binding}
		}
		if binding == "Ref_AnyTarget" || binding == "Ref_TargetPlayerOrPermanent" {
			ctx.damageTarget = &damageTargetPlan{name: binding}
		}
		plan, err = compileActions(n.Args[1], ctx)
		if err != nil {
			return plan, err
		}
		if len(plan.targets) != 0 {
			return plan, problem(n, "nested target declarations are not supported")
		}
		plan.targets = []targetFactory{buildTarget}
		plan.playerTarget = ctx.playerTarget
		plan.permanentTarget = ctx.permanentTarget
		plan.permanentTargets = ctx.permanentTargets
		plan.spellTarget = ctx.spellTarget
		plan.abilityTarget = ctx.abilityTarget
		plan.cardTarget = ctx.cardTarget
		plan.damageTarget = ctx.damageTarget
		return plan, nil
	}
	if n.Name != "ActionList" {
		return plan, unsupported(n, "action list")
	}
	if err := arity(n, 1); err != nil {
		return plan, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return plan, err
	}
	plan.effects, err = compileActionSequence(parts, ctx)
	return plan, err
}

func compileTarget(n Node, ctx scope) (targetFactory, string, error) {
	if n.Name == "TargetPlayerOrPermanent" {
		return compilePlayerPlaneswalkerTarget(n, ctx)
	}
	if n.Name == "NumberTargetPermanents" {
		return compileNumberedPermanentTarget(n, ctx)
	}
	if isUnionDamageTarget(n) {
		build, err := compileUnionDamageTarget(n, ctx, "Ref_AnyTarget")
		return build, "Ref_AnyTarget", err
	}
	if isSingularPermanentTarget(n) {
		build, err := compileSingularPermanentTarget(n, ctx, "Ref_TargetPermanent")
		return build, "Ref_TargetPermanent", err
	}
	if err := arity(n, 1); err != nil {
		return nil, "", err
	}
	switch n.Name {
	case "OneOrMoreTargetPermanents":
		return compilePluralPermanentTarget(n, ctx)
	case "TargetPlayer":
		players := n.Args[0]
		if players.Kind != Symbol || (players.Name != "AnyPlayer" && players.Name != "Opponent" && players.Name != "AttackedThisTurn") {
			return nil, "", unsupported(players, "player target filter")
		}
		return func(instanceBindings) mage.Target {
			var options []mage.PlayerTargetOption
			if players.Name == "Opponent" {
				options = append(options, mage.TargetOpponentOnly())
			}
			if players.Name == "AttackedThisTurn" {
				options = append(options, mage.TargetPlayerAttackedThisTurn())
			}
			return mage.DeclarePlayerTarget(ctx.targetNamespace+"Ref_TargetPlayer", options...)
		}, "Ref_TargetPlayer", nil
	case "TargetAnteCard":
		return compileAnteTarget(n, ctx)
	case "TargetGraveyardCard":
		return compileGraveyardTarget(n, ctx)
	case "TargetAbility":
		return compileAbilityTarget(n, ctx)
	case "TargetSpell":
		return compileSpellTarget(n.Args[0], ctx)
	default:
		return nil, "", unsupported(n, "target")
	}
}

func compileFilter(n Node) (mage.PermanentFilter, error) {
	if n.Kind == Symbol {
		switch n.Name {
		case "AnyPermanent":
			return mage.AnyPermanent, nil
		case "IsNonToken":
			return mage.Not(mage.IsToken), nil
		case "SharesANameOriginallyPrintedInAntiquities", "SharesANameOriginallyPrintedInArabianNights":
			setCode := "ATQ"
			if n.Name == "SharesANameOriginallyPrintedInArabianNights" {
				setCode = "ARN"
			}
			catalog, err := mage.OriginalPrintingNameCatalog(setCode)
			if err != nil {
				return mage.PermanentFilter{}, problem(n, err.Error())
			}
			return mage.HasNameIn(catalog), nil
		case "IsEnchanted":
			return mage.IsEnchanted, nil
		case "IsntEnchanted":
			return mage.Not(mage.IsEnchanted), nil
		case "IsTapped":
			return mage.IsTapped, nil
		case "IsUntapped":
			return mage.IsUntapped, nil
		case "IsAttacking":
			return mage.IsAttacking, nil
		case "IsntAttacking":
			return mage.Not(mage.IsAttacking), nil
		case "IsBlocking":
			return mage.IsBlocking, nil
		case "IsUnblocked":
			return mage.DamageSourceIsUnblocked(), nil
		case "AttackedThisTurn":
			return mage.AttackedThisTurn, nil
		case "DidntAttackThisTurn":
			return mage.DidntAttackThisTurn, nil
		default:
			return mage.PermanentFilter{}, unsupported(n, "permanent filter")
		}
	}
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	switch n.Name {
	case "And", "Or":
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.And(filters...), nil
		}
		return mage.Or(filters...), nil
	case "IsSupertype", "IsNonSupertype":
		if n.Args[0].Kind != Symbol {
			return mage.PermanentFilter{}, unsupported(n.Args[0], "permanent supertype")
		}
		var filter mage.PermanentFilter
		switch n.Args[0].Name {
		case "Legendary":
			filter = mage.IsLegendary
		case "Basic":
			filter = mage.NewPermanentFilter("basic", func(p *mage.Permanent, _ *mage.Game) bool {
				return p.Card.HasSuperType(core.SuperBasic)
			})
		default:
			return mage.PermanentFilter{}, unsupported(n.Args[0], "permanent supertype")
		}
		if n.Name == "IsNonSupertype" {
			filter = mage.Not(filter)
		}
		return filter, nil
	case "PowerAndToughnessIsEqualTo":
		return compileFixedPTFilter(n)
	case "ToughnessIs":
		return compileFixedToughnessFilter(n)
	case "PowerIs":
		return compileFixedPowerFilter(n)
	case "IsCardtype", "IsNonCardtype":
		kind, err := cardType(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		filter := IsCardtype(kind)
		if n.Name == "IsNonCardtype" {
			filter = mage.Not(filter)
		}
		return filter, nil
	case "IsCreatureType", "IsLandType", "IsEnchantmentType", "IsNonCreatureType", "IsNonLandType":
		subtype, err := symbol(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		filter := mage.HasSubType(subtypeName(subtype))
		if strings.HasPrefix(n.Name, "IsNon") {
			filter = mage.Not(filter)
		}
		return filter, nil
	case "IsColor", "IsNonColor":
		color, err := color(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		filter := mage.HasColorFilter(color)
		if n.Name == "IsNonColor" {
			filter = mage.Not(filter)
		}
		return filter, nil
	case "HasAbility", "DoesntHaveAbility":
		if n.Args[0].Kind == Symbol && (n.Args[0].Name == "ManaAbility" || n.Args[0].Name == "AnyRampage") {
			filter := mage.HasManaAbility
			if n.Args[0].Name == "AnyRampage" {
				filter = mage.HasRampage
			}
			if n.Name == "DoesntHaveAbility" {
				filter = mage.Not(filter)
			}
			return filter, nil
		}
		keyword, err := compileTemporaryKeyword(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		filter := mage.HasKeywordFilter(keyword)
		if n.Name == "DoesntHaveAbility" {
			filter = mage.Not(filter)
		}
		return filter, nil
	default:
		return mage.PermanentFilter{}, unsupported(n, "permanent filter")
	}
}

func color(n Node) (core.Color, error) {
	name, err := symbol(n)
	if err != nil {
		return 0, err
	}
	value, ok := map[string]core.Color{"White": core.White, "Blue": core.Blue, "Black": core.Black, "Red": core.Red, "Green": core.Green, "Colorless": core.Colorless}[name]
	if !ok {
		return 0, unsupported(n, "color")
	}
	return value, nil
}

func subtypeName(name string) string {
	switch name {
	case "AssemblyWorker":
		return "Assembly-Worker"
	case "Urzas":
		return "Urza's"
	case "PowerPlant":
		return "Power-Plant"
	default:
		return name
	}
}

func protectionColors(n Node) ([]core.Color, error) {
	if n.Name != "FromColor" {
		return nil, unsupported(n, "protection filter")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	n = n.Args[0]
	if n.Name != "Colors" {
		return nil, unsupported(n, "color list")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	var colors []core.Color
	for _, part := range parts {
		value, err := color(part)
		if err != nil {
			return nil, err
		}
		colors = append(colors, value)
	}
	return colors, nil
}

func compileValue(n Node, ctx scope) (mage.ValueSource, error) {
	if n.Kind == Symbol && n.Name == "ValueX" {
		if ctx.createdValue != nil {
			return nil, problem(n, "created ValueX requires a resolution value context")
		}
		if !ctx.x {
			return nil, problem(n, "unbound ValueX")
		}
		return mage.XValue(), nil
	}
	if n.Name != "Integer" {
		return nil, unsupported(n, "value")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != IntegerNode {
		return nil, problem(n, "expected integer literal")
	}
	return Integer(n.Args[0].Integer), nil
}

func requireRef(n Node, name string, ctx scope) error {
	if n.Kind != Symbol || n.Name != name || ctx.target != name {
		return problem(n, "unbound or incompatible target reference")
	}
	return nil
}

func compileEffect(n Node, ctx scope) ([]effectFactory, error) {
	if n.Name == "If" && len(n.Args) > 0 && n.Args[0].Name == "APermanentWasSacrificedThisWay" {
		build, err := compileSacrificeSubtypeBranch(n, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	}
	if (n.Name == "If" || n.Name == "Unless" || n.Name == "IfElse") && len(n.Args) > 0 && n.Args[0].Name == "ACardWasRevealedThisWay" {
		build, err := compileRevealResultBranch(n, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	}
	if n.Name == "CounterAbility" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref, err := compileAbilityRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect { return mage.CounterAbility(bindings.abilities[ref]) }}, nil
	}
	one := func(build effectFactory) ([]effectFactory, error) { return []effectFactory{build}, nil }
	switch n.Name {
	case "RepeatableActions":
		var build effectFactory
		var err error
		if isRepeatedCoinDamage(n) {
			build, err = compileRepeatedCoinDamage(n, ctx)
		} else {
			build, err = compileRepeatedEntry(n, ctx)
		}
		if err != nil {
			return nil, err
		}
		return one(build)
	case "ExchangeControl":
		build, err := compileControlExchange(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "ExchangeOwnershipOfTwoCards":
		build, err := compileCardOwnershipExchange(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "AcquireOwnershipOfCard", "ExchangeAnteCardWithTopCardOfPlayersLibrary":
		build, err := compileAnteExchangeAction(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "PutEachCardFromAnteInGraveyard":
		build, err := compileAnteGraveyard(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "ChooseADamageSource":
		if _, err := compileChosenDamageSourceChoice(n); err != nil {
			return nil, err
		}
		return nil, problem(n, "damage source choice requires an action sequence")
	case "ChooseAnAction":
		build, err := compileActionChoice(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "ShuffleGraveyardIntoLibrary":
		if _, err := symbol(n); err != nil {
			return nil, err
		}
		return one(func(instanceBindings) mage.Effect { return mage.ShuffleGraveyardFor(mage.You) })
	case "IfElse", "Unless":
		if len(n.Args) == 0 || n.Args[0].Name != "PermanentPassesFilter" {
			return nil, unsupported(n, "resolving alternative condition")
		}
		build, err := compileResolvingPermanentBranch(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "If":
		if len(n.Args) > 0 && n.Args[0].Name == "PermanentPassesFilter" {
			build, err := compileResolvingPermanentBranch(n, ctx)
			if err != nil {
				return nil, err
			}
			return one(build)
		}
		build, err := compileActivationHistoryBranch(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)

	case "SearchLibrary":
		build, err := compileLibrarySearch(n)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "RemoveCreatureFromCombat", "RemoveCreatureFromCombatAndUnblockBlockers":
		build, err := compileCombatExit(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "FlipACoin_OnLose", "FlipACoin_OnWinAndLose":
		build, err := compileCoinBranch(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateTriggerUntil":
		build, err := compileDepartureDelay(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateFutureReplaceWouldDraw":
		build, err := compileFutureDrawReplacement(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateFutureTrigger":
		build, err := compileDelayedStep(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "MayAction":
		build, err := compileOptionalTapChoice(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "ReturnACardFromPlayersGraveyardToBattlefield":
		build, err := compileChosenGraveyardReturn(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "PutGraveyardCardIntoHand", "PutGraveyardCardOntoBattlefield":
		build, err := compileGraveyardReturn(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "LookAtPlayersHand":
		build, err := compileHandInspection(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "LookAtTheTopNumberCardsOfPlayersLibrary":
		build, err := compileLibraryInspection(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "MayHavePlayerAction":
		build, err := compileMayShuffleLibrary(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateReplaceWouldProduceManaUntil":
		build, err := compileManaTypeReplacement(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreatePlayerEffectUntil":
		build, err := compilePlayerPayment(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateFuturePreventDamage":
		build, err := compileFinitePrevention(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateFutureReplaceWouldDealDamage":
		build, err := compileSelectedSourceRedirect(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateReplaceWouldDealDamageUntil":
		build, err := compileTemporaryDamageRedirect(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreatePreventDamageUntil":
		build, err := compileCombatPrevention(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateReplaceWouldPutIntoGraveyardUntil":
		build, err := compileTemporaryDeathExile(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateEachPermanentRuleEffectUntil":
		build, err := compileCollectionDamageHistoryRule(n, ctx)
		if len(n.Args) == 3 && n.Args[1].Kind == List && len(n.Args[1].Args) == 1 && (n.Args[1].Args[0].Name == "CantAttack" || n.Args[1].Args[0].Name == "MustAttack") {
			build, err = compileCollectionAttackTurn(n)
		}
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreatePermanentRuleEffectUntil":
		build, err := compileTemporaryCombatRule(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateTokens":
		build, err := compileTokenCreation(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "GainControlOfPermanentUntil":
		build, err := compileSourceControl(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "PutExiledCardIntoOwnersGraveyard":
		effect, err := compileGroupedExileGraveyard(n, ctx)
		return []effectFactory{effect}, err
	case "ReturnEachExiledCardToOwnersHand", "PutExiledCardsIntoOwnersGraveyards", "PutEachExiledCardOntoTheBattlefield":
		effect, err := compileLinkedExileTransport(n, ctx)
		return []effectFactory{effect}, err
	case "Exile":
		build, err := compilePermanentExile(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "PutCountersWithRestriction", "PutCounters", "RemoveCounters":
		build, err := compileCounterAction(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "SpellDealsDamageDividedAmongRecipientsRoundedDown":
		build, err := compileEvenSpellDamage(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "SpellDealsMultipleDamage", "PermanentDealsMultipleDamage":
		build, err := compileDamagePackets(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "DeadGuestPermanentDealsDamage":
		build, err := compileDeadGuestDamage(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "SpellDealsDamage", "PermanentDealsDamage":
		if n.Name == "PermanentDealsDamage" && len(n.Args) == 3 && (n.Args[2].Name == "Permanent" || n.Args[2].Name == "Player" || n.Args[2].Name == "Ref_AnyTarget" || n.Args[2].Name == "Ref_TargetPlayerOrPermanent" || ctx.damageGroups[n.Args[2].Name] != nil) {
			build, err := compileReferencedPermanentDamage(n, ctx)
			if err != nil {
				return nil, err
			}
			return one(build)
		}
		if err := arity(n, 3); err != nil {
			return nil, err
		}
		source, err := compileSelfDamageSource(n.Args[0], ctx, n.Name == "PermanentDealsDamage")
		if err != nil {
			return nil, err
		}
		if n.Args[2].Name == "Player" || n.Args[2].Name == "EachPlayer" {
			build, err := compilePlayerRecipientDamage(n, ctx, source)
			if err != nil {
				return nil, err
			}
			return one(build)
		}
		amount, err := compileInstanceValue(n.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		if n.Args[2].Name == "EachPermanent" {
			subjects, err := compilePermanentDamageCollection(n.Args[2], ctx)
			if err != nil {
				return nil, err
			}
			return one(func(bindings instanceBindings) mage.Effect {
				return mage.DealDamageEach(source(bindings), mage.PermanentDamageRecipients(subjects(bindings)), amount(bindings))
			})
		}
		if n.Args[2].Name == "MultipleRecipients" {
			recipients, err := compileMixedDamageRecipients(n.Args[2])
			if err != nil {
				return nil, err
			}
			return one(func(bindings instanceBindings) mage.Effect {
				return mage.DealDamageEach(source(bindings), recipients(), amount(bindings))
			})
		}
		if n.Args[2].Name == "Permanent" {
			if err := arity(n.Args[2], 1); err != nil {
				return nil, err
			}
			recipient, err := compilePermanentRef(n.Args[2].Args[0], ctx)
			if err != nil {
				return nil, err
			}
			if recipient.kind != permanentTarget {
				return nil, unsupported(n.Args[2].Args[0], "targeted damage recipient")
			}
			return one(func(bindings instanceBindings) mage.Effect {
				return mage.DealDamageFrom(mage.This, bindings.permanents[recipient.target], amount(bindings))
			})
		}
		if ctx.damageGroups[n.Args[2].Name] != nil {
			recipient, err := compileDamageTargetRef(n.Args[2], ctx)
			if err != nil {
				return nil, err
			}
			return one(func(bindings instanceBindings) mage.Effect {
				return mage.DealDamageFrom(source(bindings), bindings.recipients[recipient], amount(bindings))
			})
		}
		if err := requireRef(n.Args[2], ctx.target, ctx); err != nil {
			return nil, err
		}
		if ctx.damageTarget == nil && ctx.target != "Ref_TargetPlayer" && ctx.target != "Ref_TargetPermanent" {
			return nil, unsupported(n.Args[2], "damage recipient")
		}
		if ctx.damageTarget != nil {
			recipient, err := compileDamageTargetRef(n.Args[2], ctx)
			if err != nil {
				return nil, err
			}
			return one(func(bindings instanceBindings) mage.Effect {
				return mage.DealDamageFrom(mage.This, bindings.recipients[recipient], amount(bindings))
			})
		}
		if ctx.target == "Ref_TargetPlayer" {
			return one(func(bindings instanceBindings) mage.Effect {
				return mage.DealDamageFrom(mage.This, bindings.players[ctx.playerTarget], amount(bindings))
			})
		}
		return one(func(bindings instanceBindings) mage.Effect {
			return mage.DealDamageFrom(mage.This, bindings.permanents[ctx.permanentTarget], amount(bindings))
		})
	case "PermanentDoesntUntapDuringControllersNextUntap", "PermanentDoesntUntapDuringControllersNextNumberUntaps", "EachPermanentDoesntUntapDuringControllersNextUntap":
		build, err := compileNextUntaps(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "TapPermanent", "UntapPermanent":
		build, err := compileTapAction(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "Fight":
		build, err := compileFight(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "RegeneratePermanent":
		build, err := compileRegeneration(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "PutPermanentIntoItsOwnersHand", "DestroyPermanent", "DestroyPermanentNoRegen", "CounterSpell", "SacrificePermanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref := n.Args[0]
		if n.Name == "CounterSpell" {
			subject, err := compileSpellRef(ref, ctx)
			if err != nil {
				return nil, err
			}
			return one(func(bindings instanceBindings) mage.Effect { return mage.Counter(subject.build(bindings)) })
		}
		subject, err := compilePermanentRef(ref, ctx)
		if err != nil {
			return nil, err
		}
		return one(func(bindings instanceBindings) mage.Effect {
			ref := subject.build(bindings)
			switch n.Name {
			case "PutPermanentIntoItsOwnersHand":
				return mage.ReturnToHand(ref)
			case "DestroyPermanent":
				return mage.Destroy(ref)
			case "DestroyPermanentNoRegen":
				return mage.DestroyWithoutRegeneration(ref)
			default:
				return mage.Sacrifice(ref)
			}
		})
	case "PutEachPermanentIntoItsOwnersHand":
		build, err := compileReturnCollection(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "SacrificeAPermanent", "SacrificeAPermanentOfAPlayersChoice":
		build, err := compileChosenSacrifice(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "ControllersSacrificeEachPermanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		set, err := compileContextualCollection(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return one(func(bindings instanceBindings) mage.Effect {
			return mage.ControllersSacrificeEach(set(bindings))
		})
	case "TapEachPermanent", "UntapEachPermanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		set, err := compileContextualCollection(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return one(func(bindings instanceBindings) mage.Effect {
			if n.Name == "UntapEachPermanent" {
				return mage.UntapEach(set(bindings))
			}
			return mage.TapEach(set(bindings))
		})
	case "DestroyEachPermanent", "DestroyEachPermanentNoRegen":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		set, err := compileContextualCollection(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if n.Name == "DestroyEachPermanent" {
			return one(func(bindings instanceBindings) mage.Effect { return mage.DestroyEach(set(bindings)) })
		}
		return one(func(bindings instanceBindings) mage.Effect { return mage.DestroyEachWithoutRegeneration(set(bindings)) })
	case "EachPlayerAction":
		return compileEachPlayerAction(n, ctx)
	case "PlayerActions":
		build, err := compilePlayerActions(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "PlayerAction":
		if err := arity(n, 2); err != nil {
			return nil, err
		}
		who, err := compilePlayerRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		ctx.player = who
		return compilePlayerEffect(n.Args[1], ctx)
	case "DiscardTheCardDrawnThisWay":
		return compileDrawnCardDiscard(n, ctx)
	case "DestroyAPermanentNoRegen", "ChooseTwoColorWords", "ChooseTwoBasicLandTypes", "RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType", "RevealHandAndDiscardACardOfTypeAtRandom", "PutTopOfLibraryInHand", "PutTopOfLibraryInGraveyard", "TakeAnExtraTurn", "ChooseACardName", "AnteTopCardOfLibrary", "DiscardHand", "MillACard", "MillNumberCards", "LoseUnspentMana", "DrawACard", "DrawNumberCards", "DiscardACard", "DiscardNumberCards", "DiscardACardAtRandom", "DiscardNumberCardsAtRandom", "GainLife", "GainLifeForEach", "LoseLife":
		return compilePlayerEffect(n, ctx)
	case "AddManaRepeatedWithModifiers":
		return compileRestrictedRepeatedMana(n, ctx)
	case "AddMana", "AddManaRepeated":
		return compileResolvingMana(n, ctx)
	case "CreateEachPermanentLayerEffect":
		build, err := compileCapturedBasePT(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateEachPermanentLayerEffectUntil":
		if n.Kind == Tuple && len(n.Args) == 3 && n.Args[0].Name == "Ref_TargetPermanents" {
			build, err := compilePluralColor(n, ctx)
			if err != nil {
				return nil, err
			}
			return one(build)
		}
		build, err := compileTemporaryCollection(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreateSpellEffect":
		build, err := compileSpellTextAction(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreatePermanentLayerEffect":
		build, err := compileIndefiniteType(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "CreatePermanentLayerEffectUntil":
		build, err := compileTemporaryLayer(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	}
	return nil, unsupported(n, "action")
}

func compilePT(n Node) (power, toughness int, err error) {
	if err := arity(n, 2); err != nil {
		return 0, 0, err
	}
	if n.Args[0].Kind != IntegerNode || n.Args[1].Kind != IntegerNode {
		return 0, 0, problem(n, "AdjustPT requires integer literals")
	}
	return n.Args[0].Integer, n.Args[1].Integer, nil
}

func compilePlayerEffect(n Node, ctx scope) ([]effectFactory, error) {
	if n.Name == "DestroyAPermanentNoRegen" {
		build, err := compileChosenDestruction(n, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	}
	if n.Name == "SacrificeAPermanent" || n.Name == "SacrificeAPermanentOfAPlayersChoice" {
		build, err := compileChosenSacrifice(n, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	}
	if n.Name == "ChooseTwoColorWords" || n.Name == "ChooseTwoBasicLandTypes" {
		build, _, err := compileTextWordChoice(n, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	}
	if n.Name == "RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType" || n.Name == "RevealHandAndDiscardACardOfTypeAtRandom" {
		return compileRandomHandDiscard(n, ctx)
	}
	if n.Name == "PutTopOfLibraryInHand" || n.Name == "PutTopOfLibraryInGraveyard" {
		return compileLibraryTopMove(n, ctx)
	}
	if n.Name == "TakeAnExtraTurn" {
		if n.Kind != Symbol {
			return nil, problem(n, "TakeAnExtraTurn requires a symbol")
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect {
			return mage.ExtraTurnFor(ctx.player.build(bindings))
		}}, nil
	}
	if n.Name == "DiscardTheCardDrawnThisWay" {
		return compileDrawnCardDiscard(n, ctx)
	}
	if n.Name == "DiscardTheCardRevealedThisWay" {
		return compileRevealedHandDiscard(n, ctx)
	}
	if n.Name == "ChooseACardName" {
		build, err := compileCardNameChoice(n, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	}
	if n.Name == "ChooseAPlayer" {
		build, err := compilePlayerChoice(n, nil, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	}
	if n.Name == "ChooseAPermanent" {
		build, err := compilePermanentChoice(n, nil, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	}
	if n.Name == "CreateTokens" {
		build, err := compileTokenCreation(n, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	}
	if n.Name == "ChooseAnAction" {
		build, err := compileActionChoice(n, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	}
	if n.Name == "FlipACoin_OnLose" || n.Name == "FlipACoin_OnWinAndLose" {
		build, err := compileCoinBranch(n, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	}
	if n.Name == "GainLifeForEach" {
		if err := arity(n, 2); err != nil {
			return nil, err
		}
		per, err := compileInstanceValue(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		count, err := compileInstanceValue(n.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		return compileLifeEffect(n, ctx, func(bindings instanceBindings) mage.ResolutionValue {
			return mage.MultiplyValues(per(bindings), count(bindings))
		})
	}
	if n.Name == "MillNumberCards" {
		return compileMillAction(n, ctx)
	}
	if n.Name == "MillACard" {
		if n.Kind != Symbol {
			return nil, problem(n, "MillACard requires a symbol")
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect {
			return mage.Mill(ctx.player.build(bindings), mage.Fixed(1))
		}}, nil
	}
	if n.Name == "LoseUnspentMana" {
		return compileLoseUnspentMana(n, ctx)
	}
	if n.Name == "AnteTopCardOfLibrary" {
		if n.Kind != Symbol {
			return nil, problem(n, "AnteTopCardOfLibrary requires a symbol")
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect {
			return mage.AnteTopFor(ctx.player.build(bindings))
		}}, nil
	}
	if n.Name == "RevealHand" || n.Name == "DiscardCards" || n.Name == "DiscardHand" {
		return compileHandPlayerAction(n, ctx)
	}
	if n.Name == "AddMana" {
		return compilePlayerMana(n, ctx)
	}
	if n.Name == "DiscardACardAtRandom" || n.Name == "DiscardNumberCardsAtRandom" {
		return compileRandomDiscard(n, ctx)
	}
	if n.Name == "DiscardACard" || n.Name == "DiscardNumberCards" {
		return compileFixedDiscard(n, ctx)
	}
	if n.Name == "DrawACard" {
		if n.Kind != Symbol {
			return nil, problem(n, "DrawACard requires a symbol")
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect {
			return mage.DrawFor(ctx.player.build(bindings), mage.Fixed(1))
		}}, nil
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	amount, err := compileInstanceValue(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	switch n.Name {
	case "DrawNumberCards":
		return []effectFactory{func(bindings instanceBindings) mage.Effect {
			return mage.DrawFor(ctx.player.build(bindings), amount(bindings))
		}}, nil
	case "GainLife", "LoseLife":
		return compileLifeEffect(n, ctx, amount)
	default:
		return nil, unsupported(n, "player action")
	}
}

func compileLifeEffect(n Node, ctx scope, amount valueFactory) ([]effectFactory, error) {
	if ctx.player.event != nil || ctx.player.spell != nil && ctx.player.spell.event != nil {
		return nil, unsupported(n, "life action in an event player scope")
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		player := ctx.player.build(bindings)
		if n.Name == "LoseLife" {
			return mage.LoseLifeFor(player, amount(bindings))
		}
		return mage.GainLifeFor(player, amount(bindings))
	}}, nil
}

func compileFixedDiscard(n Node, ctx scope) ([]effectFactory, error) {
	amount := 1
	if n.Name == "DiscardACard" {
		if n.Kind != Symbol {
			return nil, problem(n, "DiscardACard requires a symbol")
		}
	} else {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		value := n.Args[0]
		if value.Name != "Integer" {
			return nil, unsupported(value, "fixed discard amount")
		}
		if err := arity(value, 1); err != nil {
			return nil, err
		}
		if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
			return nil, problem(value, "discard amount must be a nonnegative integer")
		}
		amount = value.Args[0].Integer
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.Discard(ctx.player.build(bindings), mage.Fixed(amount))
	}}, nil
}

func compileMana(n Node) ([]mage.ManaProduction, error) {
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		var values []mage.ManaProduction
		for _, part := range parts {
			value, err := compileMana(part)
			if err != nil {
				return nil, err
			}
			values = append(values, value...)
		}
		return values, nil
	}
	name, err := symbol(n)
	if err != nil {
		return nil, err
	}
	color, ok := map[string]core.Color{"ManaProduceW": core.White, "ManaProduceU": core.Blue, "ManaProduceB": core.Black, "ManaProduceR": core.Red, "ManaProduceG": core.Green, "ManaProduceC": core.Colorless, "AnyManaColor": core.AnyColor}[name]
	if !ok {
		return nil, unsupported(n, "mana production")
	}
	return []mage.ManaProduction{{Color: color, Amount: 1}}, nil
}
