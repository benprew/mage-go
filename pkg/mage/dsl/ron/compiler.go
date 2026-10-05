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
type targetFactory func() mage.Target
type costFactory func() mage.Cost

type scope struct {
	actionPlayer     *playerActionsPlan
	damageRecipient  *playerReferencePlan
	activation       bool
	castingCondition mage.StateCondition
	damagePlayer     *damagePlayerPlan
	damageAmount     *damageAmountPlan
	delayedStep      bool
	sourceAttack     bool
	sourceBlock      bool
	combatOpponent   *permanentEventPlan
	targetNamespace  string
	counterPayment   *counterPaymentPlan
	sacrificeValue   *sacrificeValuePlan
	target           string
	x                bool
	permanent        bool
	aura             bool
	player           playerReferencePlan
	playerTarget     *playerTargetPlan
	permanentTarget  *permanentTargetPlan
	permanentTargets *permanentTargetsPlan
	spellTarget      *spellTargetPlan
	cardTarget       *cardTargetPlan
	damageTarget     *damageTargetPlan
	castEvent        *castEventPlan
	stepEvent        *stepEventPlan
	permanentEvent   *permanentEventPlan
	landManaEvent    *landManaEventPlan
	graveyardEvent   *graveyardEventPlan
}

type actionPlan struct {
	effects          []effectFactory
	targets          []targetFactory
	playerTarget     *playerTargetPlan
	permanentTarget  *permanentTargetPlan
	permanentTargets *permanentTargetsPlan
	spellTarget      *spellTargetPlan
	cardTarget       *cardTargetPlan
	damageTarget     *damageTargetPlan
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
	targets := make([]mage.Target, len(p.targets))
	bindings.players = make(map[*playerTargetPlan]*mage.PlayerTargetBinding)
	bindings.permanents = make(map[*permanentTargetPlan]*mage.PermanentTargetBinding)
	bindings.permanentSets = make(map[*permanentTargetsPlan]*mage.PermanentTargetsBinding)
	bindings.spells = make(map[*spellTargetPlan]*mage.SpellTargetBinding)
	bindings.cards = make(map[*cardTargetPlan]*mage.CardTargetBinding)
	bindings.recipients = make(map[*damageTargetPlan]*mage.DamageTargetBinding)
	var declared []mage.TargetBinding
	for i, makeTarget := range p.targets {
		targets[i] = makeTarget()
		if target, ok := targets[i].(*mage.PermanentTargetBinding); ok {
			bindings.permanents[p.permanentTarget] = target
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
		if target, ok := targets[i].(*mage.SpellTargetBinding); ok {
			bindings.spells[p.spellTarget] = target
		}
		if target, ok := targets[i].(*mage.DamageTargetBinding); ok {
			bindings.recipients[p.damageTarget] = target
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
	spellCount, enchantCount := 0, 0
	if rulesNode, ok := n.Fields["Rules"]; ok {
		rules, err := list(rulesNode)
		if err != nil {
			return nil, err
		}
		for _, rule := range rules {
			if rule.Name != "CastEffect" {
				continue
			}
			if permanent {
				return nil, unsupported(rule, "casting window on a permanent")
			}
			if ctx.castingCondition != nil {
				return nil, problem(rule, "duplicate CastEffect rule")
			}
			ctx.castingCondition, err = compileCastingWindow(rule)
			if err != nil {
				return nil, err
			}
		}
		for _, rule := range rules {
			if rule.Name == "CastEffect" {
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
			options = append(options, option)
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
	return func() mage.Card {
		opts := make([]mage.CardOption, 0, 1+len(types)+len(options))
		opts = append(opts, mage.WithSubTypes(subtypes...), mage.WithSuperTypes(supers...))
		for _, kind := range types[1:] {
			opts = append(opts, mage.WithCardType(kind))
		}
		for _, buildOption := range options {
			opts = append(opts, buildOption())
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
		return card
	}, nil
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
	case "PreventDamage":
		return compileStaticPrevention(n, ctx)
	case "If":
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
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Name == "Modal_ChooseOne" {
			return compileModalSpell(n.Args[0], ctx)
		}
		plan, err := compileActions(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption {
			options := actionOptions(plan.build())
			if ctx.castingCondition != nil {
				options = append(options, mage.WithCastingCondition(ctx.castingCondition))
			}
			return mage.WithAbility(mage.NewAction(mage.ActionSpell, options...))
		}, nil
	case "Activated", "ActivatedWithModifiers":
		return compileActivatedRule(n, ctx)
	case "EnchantPermanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		filter, err := compileFilter(n.Args[0])
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption { return mage.WithCastTarget(mage.TargetPermanent(filter)) }, nil
	case "Protection":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		colors, err := protectionColors(n.Args[0])
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption {
			return mage.WithAbility(mage.ProtectionFromColors(append([]core.Color(nil), colors...)...))
		}, nil
	case "Landwalk":
		kw, err := compileLandwalkKeyword(n)
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption { return mage.WithKeyword(kw) }, nil
	default:
		return nil, unsupported(n, "rule")
	}
}

func compileLandwalkKeyword(n Node) (core.Attr, error) {
	if err := arity(n, 1); err != nil {
		return 0, err
	}
	filter := n.Args[0]
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
		if len(targets) != 1 {
			return plan, problem(n, "multiple target bindings are not supported")
		}
		buildTarget, binding, err := compileTarget(targets[0], ctx)
		if err != nil {
			return plan, err
		}
		ctx.target = binding
		if binding == "Ref_TargetPermanent" {
			ctx.permanentTarget = &permanentTargetPlan{name: binding}
		}
		if binding == "Ref_TargetPermanents" {
			ctx.permanentTargets = &permanentTargetsPlan{name: binding}
		}
		if binding == "Ref_TargetPlayer" {
			ctx.playerTarget = &playerTargetPlan{name: binding}
		}
		if binding == "Ref_TargetGraveyardCard" {
			ctx.cardTarget = &cardTargetPlan{name: binding, creatureOnly: graveyardFilterRequiresCreature(targets[0].Args[0])}
		}
		if binding == "Ref_TargetSpell" {
			ctx.spellTarget = &spellTargetPlan{name: binding}
		}
		if binding == "Ref_AnyTarget" {
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
	if n.Name == "NumberTargetPermanents" {
		return compileNumberedPermanentTarget(n, ctx)
	}
	if n.Kind == Symbol && n.Name == "AnyTarget" {
		return func() mage.Target { return mage.DeclareDamageTarget(ctx.targetNamespace + "Ref_AnyTarget") }, "Ref_AnyTarget", nil
	}
	if err := arity(n, 1); err != nil {
		return nil, "", err
	}
	switch n.Name {
	case "OneOrMoreTargetPermanents":
		return compilePluralPermanentTarget(n, ctx)
	case "TargetPermanent":
		filter, options, err := compilePermanentTargetFilter(n.Args[0], ctx)
		if err != nil {
			return nil, "", err
		}
		return func() mage.Target {
			return mage.DeclarePermanentTarget(ctx.targetNamespace+"Ref_TargetPermanent", filter, options...)
		}, "Ref_TargetPermanent", nil
	case "TargetPlayer":
		players := n.Args[0]
		if players.Kind != Symbol || (players.Name != "AnyPlayer" && players.Name != "Opponent") {
			return nil, "", unsupported(players, "player target filter")
		}
		return func() mage.Target {
			var options []mage.PlayerTargetOption
			if players.Name == "Opponent" {
				options = append(options, mage.TargetOpponentOnly())
			}
			return mage.DeclarePlayerTarget(ctx.targetNamespace+"Ref_TargetPlayer", options...)
		}, "Ref_TargetPlayer", nil
	case "TargetGraveyardCard":
		return compileGraveyardTarget(n, ctx)
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
		case "IsBlocking":
			return mage.IsBlocking, nil
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
		if n.Args[0].Kind == Symbol && n.Args[0].Name == "ManaAbility" {
			filter := mage.HasManaAbility
			if n.Name == "DoesntHaveAbility" {
				filter = mage.Not(filter)
			}
			return filter, nil
		}
		keyword, ok := keywords[n.Args[0].Name]
		if n.Args[0].Kind != Symbol || !ok {
			return mage.PermanentFilter{}, unsupported(n.Args[0], "keyword filter")
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
	one := func(build effectFactory) ([]effectFactory, error) { return []effectFactory{build}, nil }
	switch n.Name {
	case "SearchLibrary":
		build, err := compileLibrarySearch(n)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "RemoveCreatureFromCombat", "RemoveCreatureFromCombatAndUnblockBlockers":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		subject, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return one(func(bindings instanceBindings) mage.Effect {
			if n.Name == "RemoveCreatureFromCombatAndUnblockBlockers" {
				return mage.RemovePermanentFromCombatAndUnblock(subject.build(bindings))
			}
			return mage.RemovePermanentFromCombat(subject.build(bindings))
		})
	case "FlipACoin_OnLose", "FlipACoin_OnWinAndLose":
		build, err := compileCoinBranch(n, ctx)
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
	case "PutGraveyardCardIntoHand", "PutGraveyardCardOntoBattlefield":
		build, err := compileGraveyardReturn(n, ctx)
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
	case "CreatePreventDamageUntil":
		build, err := compileCombatPrevention(n, ctx)
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
	case "Exile":
		build, err := compilePermanentExile(n, ctx)
		if err != nil {
			return nil, err
		}
		return one(build)
	case "PutCounters":
		build, err := compileCounterPlacement(n, ctx)
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
	case "SpellDealsDamage", "PermanentDealsDamage":
		if err := arity(n, 3); err != nil {
			return nil, err
		}
		source := "ThisSpell"
		if n.Name == "PermanentDealsDamage" {
			source = "ThisPermanent"
		}
		if source == "ThisPermanent" {
			ref, err := compilePermanentRef(n.Args[0], ctx)
			if err != nil || ref.kind != permanentSource {
				return nil, unsupported(n.Args[0], "damage source")
			}
		} else if n.Args[0].Kind != Symbol || n.Args[0].Name != source || ctx.permanent {
			return nil, unsupported(n.Args[0], "damage source")
		}
		if n.Args[2].Name == "Player" || n.Args[2].Name == "EachPlayer" {
			build, err := compilePlayerRecipientDamage(n, ctx)
			if err != nil {
				return nil, err
			}
			return one(build)
		}
		amount, err := compileInstanceValue(n.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		if n.Args[2].Name == "MultipleRecipients" {
			recipients, err := compileMixedDamageRecipients(n.Args[2])
			if err != nil {
				return nil, err
			}
			return one(func(bindings instanceBindings) mage.Effect {
				return mage.DealDamageEach(mage.This, recipients(), amount(bindings))
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
		if err := requireRef(n.Args[2], ctx.target, ctx); err != nil {
			return nil, err
		}
		if ctx.target != "Ref_AnyTarget" && ctx.target != "Ref_TargetPlayer" && ctx.target != "Ref_TargetPermanent" {
			return nil, unsupported(n.Args[2], "damage recipient")
		}
		if ctx.target == "Ref_AnyTarget" {
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
	case "PutPermanentIntoItsOwnersHand", "DestroyPermanent", "DestroyPermanentNoRegen", "CounterSpell", "TapPermanent", "UntapPermanent", "RegeneratePermanent", "SacrificePermanent", "PermanentDoesntUntapDuringControllersNextUntap":
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
			case "TapPermanent":
				return mage.TapSubject(ref)
			case "UntapPermanent":
				return mage.UntapPermanent(ref)
			case "RegeneratePermanent":
				return mage.Regenerate(ref)
			case "PermanentDoesntUntapDuringControllersNextUntap":
				return mage.SkipNextUntap(ref)
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
	case "UntapEachPermanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Name == "Ref_TargetPermanents" {
			collection, err := compilePermanentTargetsRef(n.Args[0], ctx)
			if err != nil {
				return nil, err
			}
			return one(func(bindings instanceBindings) mage.Effect {
				return mage.UntapEach(bindings.permanentSets[collection])
			})
		}
		var controller *playerReferencePlan
		filter, err := compileCountFilter(n.Args[0], ctx, &controller)
		if err != nil {
			return nil, err
		}
		return one(func(bindings instanceBindings) mage.Effect {
			if controller != nil {
				return mage.UntapEach(mage.PermanentsControlledBy(controller.build(bindings), filter))
			}
			return mage.UntapEach(mage.MatchingPermanents(filter))
		})
	case "TapEachPermanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Name == "Ref_TargetPermanents" {
			collection, err := compilePermanentTargetsRef(n.Args[0], ctx)
			if err != nil {
				return nil, err
			}
			return one(func(bindings instanceBindings) mage.Effect { return mage.TapEach(bindings.permanentSets[collection]) })
		}
		var controller *playerReferencePlan
		filter, err := compileCountFilter(n.Args[0], ctx, &controller)
		if err != nil {
			return nil, err
		}
		return one(func(bindings instanceBindings) mage.Effect {
			if controller != nil {
				return mage.TapEach(mage.PermanentsControlledBy(controller.build(bindings), filter))
			}
			return mage.TapEach(mage.MatchingPermanents(filter))
		})
	case "DestroyEachPermanent", "DestroyEachPermanentNoRegen":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		filter, err := compileFilter(n.Args[0])
		if err != nil {
			return nil, err
		}
		if n.Name == "DestroyEachPermanent" {
			return one(func(bindings instanceBindings) mage.Effect { return DestroyEachPermanent(filter) })
		}
		return one(func(bindings instanceBindings) mage.Effect { return DestroyEachPermanentNoRegen(filter) })
	case "EachPlayerAction":
		return compileEachPlayerAction(n)
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
	case "LoseUnspentMana", "DrawACard", "DrawNumberCards", "DiscardACard", "DiscardNumberCards", "DiscardACardAtRandom", "DiscardNumberCardsAtRandom", "GainLife", "LoseLife":
		return compilePlayerEffect(n, ctx)
	case "AddMana", "AddManaRepeated":
		if n.Name == "AddMana" && len(n.Args) == 1 && n.Args[0].Name == "Trigger_AnyManaTypeProduced" {
			return compilePlayerMana(n, ctx)
		}
		count := 1
		args := 1
		if n.Name == "AddManaRepeated" {
			args = 2
		}
		if err := arity(n, args); err != nil {
			return nil, err
		}
		if args == 2 {
			value := n.Args[1]
			if value.Name == "TheNumberOfCountersRemovedThisWay" {
				return compilePaidCounterMana(n, ctx)
			}
			if value.Name != "Integer" {
				return nil, unsupported(value, "mana amount")
			}
			if err := arity(value, 1); err != nil {
				return nil, err
			}
			if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
				return nil, problem(value, "mana amount must be nonnegative")
			}
			count = value.Args[0].Integer
		}
		productions, err := compileMana(n.Args[0])
		if err != nil {
			return nil, err
		}
		var effects []effectFactory
		for _, production := range productions {
			amount, color := production.Amount*count, production.Color
			effects = append(effects, func(bindings instanceBindings) mage.Effect {
				if color == core.AnyColor {
					return mage.AddManaOfAnyOneColor(mage.Fixed(amount))
				}
				return mage.AddMana(color, amount)
			})
		}
		return effects, nil
	case "CreateEachPermanentLayerEffectUntil":
		if n.Kind == Tuple && len(n.Args) == 3 && n.Args[0].Name == "Ref_TargetPermanents" {
			build, err := compilePluralColor(n, ctx)
			if err != nil {
				return nil, err
			}
			return one(build)
		}
		build, err := compileTemporaryCollection(n)
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
		if err := arity(n, 3); err != nil {
			return nil, err
		}
		duration, err := compileTemporaryDuration(n.Args[2], ctx)
		if err != nil {
			return nil, err
		}
		subject, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		parts, err := list(n.Args[1])
		if err != nil {
			return nil, err
		}
		var modifications []func(instanceBindings) mage.ContinuousModification
		for _, part := range parts {
			if part.Name == "SetPower" {
				build, err := compileTemporaryPower(part, subject)
				if err != nil {
					return nil, err
				}
				modifications = append(modifications, build)
				continue
			}
			if part.Name == "LosesAbility" {
				build, err := compileTemporaryKeywordRemoval(part, subject)
				if err != nil {
					return nil, err
				}
				modifications = append(modifications, build)
				continue
			}
			if part.Name == "SetPT" || part.Name == "AddCardtype" || part.Name == "AddCreatureType" || part.Name == "SetCreatureType" {
				build, err := compileTemporaryType(part, subject)
				if err != nil {
					return nil, err
				}
				modifications = append(modifications, build)
				continue
			}
			if part.Name == "AddAbility" {
				if err := arity(part, 1); err != nil {
					return nil, err
				}
				abilities, err := list(part.Args[0])
				if err != nil {
					return nil, err
				}
				for _, ability := range abilities {
					keyword, err := compileTemporaryKeyword(ability)
					if err != nil {
						return nil, err
					}
					modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
						return mage.GrantKeywordModification(subject.build(bindings), keyword)
					})
				}
				continue
			}
			if part.Name == "AdjustPTX" {
				build, err := compileTemporaryPTX(part, ctx, subject)
				if err != nil {
					return nil, err
				}
				modifications = append(modifications, build)
				continue
			}
			if part.Name != "AdjustPT" {
				return nil, unsupported(part, "temporary layer action")
			}
			power, toughness, err := compilePT(part)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
				return mage.ModifyPT(subject.build(bindings), mage.Fixed(power), mage.Fixed(toughness))
			})
		}
		if len(modifications) == 0 {
			return nil, problem(n.Args[1], "supply at least one temporary modification")
		}
		return one(func(bindings instanceBindings) mage.Effect {
			values := make([]mage.ContinuousModification, len(modifications))
			for i, build := range modifications {
				values[i] = build(bindings)
			}
			return duration(values)
		})
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
	if n.Name == "LoseUnspentMana" {
		if n.Kind != Symbol {
			return nil, problem(n, "LoseUnspentMana requires a symbol")
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect {
			return mage.LoseUnspentManaFor(ctx.player.build(bindings))
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
		if ctx.player.event != nil || ctx.player.spell != nil && ctx.player.spell.event != nil {
			return nil, unsupported(n, "life action in an event player scope")
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect {
			player := ctx.player.build(bindings)
			if n.Name == "GainLife" {
				return mage.GainLifeFor(player, amount(bindings))
			}
			return mage.LoseLifeFor(player, amount(bindings))
		}}, nil
	default:
		return nil, unsupported(n, "player action")
	}
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
