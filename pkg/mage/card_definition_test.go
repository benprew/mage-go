package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func boltDefinition(t *testing.T) *CardDefinition {
	t.Helper()
	def, err := NewCardDefinition(CardDefinitionSpec{
		Name: "Definition Bolt", TypeLine: TypeLine{CardTypes: []CardType{TypeInstant}}, ManaCost: ParseManaCost("{R}"),
		Abilities: []Ability{NewSpell(DealDamage(Fixed(3)), WithTarget(TargetDamageAnyTarget()))},
	})
	if err != nil {
		t.Fatal(err)
	}
	return def
}

func TestCardDefinitionPublication(t *testing.T) {
	types := []CardType{TypeCreature}
	subtypes := []string{"Bear"}
	cost := ParseManaCost("{W/U}")
	seeds := []Ability{StaticKeyword(Flying)}
	spec := CardDefinitionSpec{Name: "Printed Bear", TypeLine: TypeLine{CardTypes: types, SubTypes: subtypes}, ManaCost: cost, Power: 2, Toughness: 2, Abilities: seeds}
	def, err := NewCardDefinition(spec)
	if err != nil {
		t.Fatal(err)
	}
	types[0] = TypeLand
	subtypes[0] = "Goblin"
	cost.Hybrid[0].A = Red
	seeds[0] = StaticKeyword(Haste)
	def.Types()[0] = TypeLand
	def.SubTypes()[0] = "Goblin"
	def.ManaCost().Hybrid[0].A = Green

	if def.Types()[0] != TypeCreature || def.SubTypes()[0] != "Bear" || def.ManaCost().Hybrid[0].A != White || countCardAbility(def.NewInstance(uuid.Nil()), Flying) != 1 {
		t.Fatal("definition has writable aliases")
	}
}

func TestCardDefinitionCompiledKeywords(t *testing.T) {
	abilities := []Ability{keywordRule(Flying), keywordRule(FirstStrike), keywordRule(Flying)}
	def, err := NewCardDefinition(CardDefinitionSpec{
		Name: "Compiled Knight", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}},
		Abilities: abilities,
	})
	if err != nil {
		t.Fatal(err)
	}
	abilities[0] = keywordRule(Haste)

	a, b := def.NewInstance(uuid.New()), def.NewInstance(uuid.New())
	WithKeyword(Haste)(a)
	for _, kw := range []Attr{Flying, FirstStrike} {
		want := 1
		if kw == Flying {
			want = 2
		}
		if countCardAbility(def.NewInstance(uuid.Nil()), kw) != want || countCardAbility(b, kw) != want || !NewPermanent(b, b.Owner()).HasKeyword(kw) {
			t.Fatalf("compiled keyword %s did not reach the instance", kw)
		}
	}
	if countCardAbility(def.NewInstance(uuid.Nil()), Flying) != 2 || countCardAbility(def.NewInstance(uuid.Nil()), FirstStrike) != 1 || countCardAbility(b, Haste) != 0 {
		t.Fatal("compiled abilities share writable state")
	}
}

func TestCardDefinitionInstances(t *testing.T) {
	def := boltDefinition(t)
	a, b := def.NewInstance(uuid.New()), def.NewInstance(uuid.New())
	if a.Definition() != def || b.Definition() != def || a.ID() == b.ID() || a.Owner() == b.Owner() {
		t.Fatal("instances must share only their definition")
	}
	aa := a.Abilities()[0].(*ActionDefinition)
	bb := b.Abilities()[0].(*ActionDefinition)

	if aa != bb {
		t.Fatal("ability context is shared")
	}
	cloned := a.Copy()
	if cloned.ID() == a.ID() || cloned.Definition() != def || cloned.Abilities()[0] != aa {
		t.Fatal("fresh cloned has shared runtime state")
	}
	a.SetBasePT(8, 9)
	if a.Power() != 8 || def.Power() != 0 || b.Power() != 0 {
		t.Fatal("instance change altered printed characteristics")
	}
}

func TestCardDefinitionRegistryAndExecution(t *testing.T) {
	def := boltDefinition(t)
	registry := NewRegistry()
	if err := registry.RegisterDefinition(def); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterDefinition(def); err == nil {
		t.Fatal("duplicate registration accepted")
	}
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	card, err := registry.CreateInstance(def.Name(), a.PlayerID())
	if err != nil {
		t.Fatal(err)
	}
	a.AddToHand(card)
	a.ManaPool().Add(Red, 1)
	before := b.Life()
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	branch := g.Clone()
	branchCard := branch.stack.Peek().Card
	if branchCard == card || branchCard.Definition() != def || branchCard.ID() != card.ID() {
		t.Fatal("search clone must isolate instances and preserve identity")
	}
	branchCard.SetOwner(b.PlayerID())
	if card.Owner() != a.PlayerID() {
		t.Fatal("ownership leaked from search branch")
	}
	g.ResolveStack()
	if b.Life() != before-3 || len(a.Graveyard()) != 1 {
		t.Fatal("definition-backed spell did not resolve normally")
	}
}

func TestCardDefinitionValidation(t *testing.T) {
	for _, spec := range []CardDefinitionSpec{
		{}, {Name: "No Types"}, {Name: "Bad Rule", TypeLine: TypeLine{CardTypes: []CardType{TypeInstant}}, Abilities: []Ability{nil}},
	} {
		if _, err := NewCardDefinition(spec); err == nil {
			t.Fatal("invalid definition accepted")
		}
	}
}

func TestCardDefinitionCopyCharacteristics(t *testing.T) {
	source := boltDefinition(t).NewInstance(uuid.New())
	base, err := NewCardDefinition(CardDefinitionSpec{Name: "Copy", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}}, Power: 2, Toughness: 2})
	if err != nil {
		t.Fatal(err)
	}
	target := base.NewInstance(source.Owner())
	target.CloneFrom(source)

	if target.Abilities()[0] != source.Abilities()[0] {
		t.Fatal("cloned shares ability context with its source")
	}
	if target.Definition() != base || base.Name() != "Copy" || target.Name() != source.Name() {
		t.Fatal("cloned replaced the printed definition")
	}
	branch := target.CloneInstance()
	if len(branch.Abilities()) != 1 || branch.Abilities()[0] != target.Abilities()[0] {
		t.Fatal("cloned cloned lost its rule state")
	}
}

func TestCardDefinitionCloneZoneAliases(t *testing.T) {
	for _, zone := range []Zone{ZoneHand, ZoneLibrary, ZoneGraveyard, ZoneExile, ZoneAnte, ZoneBattlefield} {
		t.Run(zone.String(), func(t *testing.T) {
			g := newPriorityTestGame()
			owner := g.players[0].PlayerID()
			card := boltDefinition(t).NewInstance(owner)
			switch zone {
			case ZoneHand:
				g.players[0].AddToHand(card)
			case ZoneLibrary:
				g.players[0].AddToLibrary(card)
			case ZoneGraveyard:
				g.players[0].AddToGraveyard(card)
			case ZoneAnte:
				g.players[0].AddToAnte(card)
			case ZoneExile:
				g.zones.exile = append(g.zones.exile, ExiledCard{Card: card})
			case ZoneBattlefield:
				g.zones.battlefield = append(g.zones.battlefield, NewPermanent(card, owner))
			}
			g.stack.Push(&StackObject{Card: card, TargetSource: card})
			branch := g.Clone()
			cloned := branch.FindCardAnywhere(card.ID())
			if cloned == nil || cloned == card || cloned != branch.stack.Peek().Card || cloned != branch.stack.Peek().TargetSource {
				t.Fatal("clone did not preserve private instance aliases")
			}
			cloned.SetOwner(uuid.New())
			if card.Owner() != owner {
				t.Fatal("owner changed in the original game")
			}
		})
	}
}

func TestCardDefinitionRejectsSharedRuleState(t *testing.T) {
	for _, rule := range []Ability{nil} {
		_, err := NewCardDefinition(CardDefinitionSpec{Name: "Bad Rule State", TypeLine: TypeLine{CardTypes: []CardType{TypeInstant}}, Abilities: []Ability{rule}})
		if err == nil {
			t.Fatal("invalid rule state accepted")
		}
	}
}

func TestCardDefinitionRegistrationIsAtomic(t *testing.T) {
	registry := NewRegistry()
	first := boltDefinition(t)
	duplicate := boltDefinition(t)
	if err := registry.RegisterDefinitions(first, duplicate); err == nil {
		t.Fatal("duplicate batch accepted")
	}
	if registry.CardRegistered(first.Name()) {
		t.Fatal("failed batch left a registration")
	}
	if err := registry.RegisterDefinition(first); err != nil {
		t.Fatal(err)
	}
	a, err := registry.CreateCard(first.Name())
	if err != nil {
		t.Fatal(err)
	}
	b, err := registry.CreateCard(first.Name())
	if err != nil {
		t.Fatal(err)
	}
	if a.ID() == b.ID() || a.Definition() != b.Definition() {
		t.Fatal("registry did not share the definition")
	}
}

func TestCardDefinitionActivationCloneState(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	card := NewArtifact("Limited", "{0}", WithAbility(NewActivated(ManaCostOf("{0}"), GainLife(1), WithOncePerTurn())))
	permanent := g.PutOnBattlefield(card, owner)
	if err := g.ActivateAbilityByIndex(owner, permanent.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	branch := g.Clone()
	if err := branch.ActivateAbilityByIndex(owner, permanent.ID(), 0, nil); err == nil {
		t.Fatal("branch lost activation count")
	}
	clear(branch.abilityActivations)
	if err := branch.ActivateAbilityByIndex(owner, permanent.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(owner, permanent.ID(), 0, nil); err == nil {
		t.Fatal("branch reset original count")
	}

}

func TestCardDefinitionSpellCopyChoices(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	def := boltDefinition(t)
	card := def.NewInstance(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.stack.Peek()
	original.CastContext = &CastContext{ColorsSpent: map[Color]int{Red: 1}}
	copied := g.CopySpellOnStack(card.ID(), a.PlayerID(), false)
	if copied == nil || copied.Card.Definition() != def {
		t.Fatal("spell copy lost the shared definition")
	}
	copied.CastContext.ColorsSpent[Red] = 2
	if original.CastContext.ColorsSpent[Red] != 1 {
		t.Fatal("spell copy shares announced payment data")
	}
	if copied.TargetSpecs[0] != original.TargetSpecs[0] {
		t.Fatal("spell copy changed target specification")
	}
	before := b.Life()
	g.ResolveStack()
	if b.Life() != before-6 || len(a.Graveyard()) != 1 {
		t.Fatal("copied spell did not resolve through normal damage and zone rules")
	}
}

func TestCardDefinitionSpellLosesTarget(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	card := boltDefinition(t).NewInstance(a.PlayerID())
	a.AddToHand(card)
	creature := NewCreature("Vanishing Target", "{0}", 4, 4)
	creature.SetOwner(b.PlayerID())
	permanent := g.PutOnBattlefield(creature, b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{permanent.ID()}, 0); err == nil {
		t.Fatal("unpayable spell was accepted")
	}
	if len(a.Hand()) != 1 || !g.stack.IsEmpty() {
		t.Fatal("failed announcement left state behind")
	}
	a.ManaPool().Add(Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{permanent.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanent(permanent)
	g.ResolveStack()
	if len(a.Graveyard()) != 1 || b.Life() != 20 {
		t.Fatal("illegal target did not use normal resolution rules")
	}
}

func TestNewCardUsesSharedDefinition(t *testing.T) {
	def := boltDefinition(t)
	first := NewCard(def)
	second := NewCard(def)
	first.SetOwner(uuid.New())
	if first.Definition() != def || second.Definition() != def {
		t.Fatal("base cards must keep the shared definition")
	}
	if first.ID() == second.ID() || first.Owner() == second.Owner() {
		t.Fatal("base cards share instance identity or ownership")
	}
	if first.Name() != def.Name() || first.ManaCost().Red != 1 || !first.HasType(TypeInstant) {
		t.Fatal("base card did not use the definition")
	}
	if first.Abilities()[0] != second.Abilities()[0] {
		t.Fatal("base cards share ability state")
	}
	first.SetBasePT(4, 5)
	if first.Power() != 4 || second.Power() != def.Power() || def.Power() != 0 {
		t.Fatal("base card changed the shared definition")
	}
}

func TestNewCardSupportsCardOptions(t *testing.T) {
	def, err := NewCardDefinition(CardDefinitionSpec{
		Name: "Definition Creature", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}}, Power: 2, Toughness: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	card := NewCard(def,
		WithSuperTypes(SuperLegendary), WithSubTypes("Spirit"), WithKeyword(Flying), WithWard(ManaCostOf("{1}")),
		WithAdditionalCost(LifePayCost(1)), WithAlternateCost(ZoneHand, ParseManaCost("{G}")),
		WithUncounterable(), WithAuraAIProfile(AuraAIProfile{PowerBoost: 1}), WithCastTarget(TargetCreature()),
	)
	if !card.HasSuperType(SuperLegendary) || !card.HasSubType("Spirit") || countCardAbility(card, Flying) != 1 || countCardAbility(card, Ward) != 1 {
		t.Fatal("card options did not change private characteristics")
	}
	if len(def.SuperTypes()) != 0 || len(def.SubTypes()) != 0 || len(def.NewInstance(uuid.Nil()).Abilities()) != 0 {
		t.Fatal("card options changed the shared definition")
	}
	if len(card.AdditionalCosts()) != 1 || len(card.AlternateCosts()) != 1 || !card.IsUncounterable() || len(card.CastTargets()) != 1 {
		t.Fatal("base card lost casting rules")
	}
	if profile, ok := card.AuraAIProfile(); !ok || profile.PowerBoost != 1 {
		t.Fatal("base card lost its aura profile")
	}
	cloned := card.CloneInstance()
	if cloned.ID() != card.ID() || cloned.Definition() != def {
		t.Fatal("base card clone lost its identity or definition")
	}
	cloned.mutableDefinition().alternateCosts[0].Mana.Green = 2
	cloned.mutableDefinition().castTargets[0] = TargetCreature(IsCreature)
	if card.AlternateCosts()[0].Mana.Green != 1 || len(card.CastTargets()[0].(*CreatureTarget).Filters) != 0 {
		t.Fatal("base card clone shares private casting data")
	}
}

func TestCardCharacteristicSnapshotHasNoCardState(t *testing.T) {
	def := boltDefinition(t)
	card := NewCard(def)
	owner := uuid.New()
	card.SetOwner(owner)
	card.SetBasePT(4, 5)
	if _, isCard := any(&card.cardCharacteristics).(*Card); isCard {
		t.Fatal("characteristic data must not be a card with identity and execution state")
	}
	branch := card.CloneInstance()
	branch.SetBasePT(8, 9)
	branch.AddSubType("Spirit")
	if card.Power() != 4 || card.Toughness() != 5 || card.HasSubType("Spirit") {
		t.Fatal("cloned characteristic data changed the original")
	}
	if branch.ID() != card.ID() || branch.Owner() != owner || branch.Definition() != def {
		t.Fatal("cloning characteristics changed card identity, ownership, or definition")
	}
	if def.Power() != 0 || def.Toughness() != 0 {
		t.Fatal("characteristic changes altered the shared definition")
	}
}

func TestCardDefinitionRejectsNilAbility(t *testing.T) {
	if _, err := NewCardDefinition(CardDefinitionSpec{Name: "Invalid", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}}, Abilities: []Ability{nil}}); err == nil {
		t.Fatal("nil ability accepted")
	}
}

func keywordRule(attr Attr) Ability {
	return NewAttributeAbility(attr)
}

func TestCardDefinitionRejectsTypedNilAbility(t *testing.T) {
	var ability *ActionDefinition
	_, err := NewCardDefinition(CardDefinitionSpec{Name: "Invalid", TypeLine: TypeLine{CardTypes: []CardType{TypeInstant}}, Abilities: []Ability{ability}})
	if err == nil {
		t.Fatal("typed nil ability was published")
	}
}

func TestZoneChangeRestoresThePrintedDefinition(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	card := NewCreature("Changing entry", "{0}", 3, 3, WithAbility(ETBEffect(FuncEffect("choose entry values", EffectProperties{}, func(g *Game, source, _ uuid.UUID, _ []uuid.UUID) error {
		if g.ModeValue() == 1 {
			g.MutablePermanent(source).Card.SetBasePT(2, 2)
		}
		return nil
	}))))
	g.resolution.SetMode(1)
	permanent := g.PutOnBattlefield(card, owner)
	if permanent.CurrentPower(g) != 2 || card.Definition().Power() != 3 {
		t.Fatal("entry did not retain a separate printed face")
	}
	g.RemoveFromBattlefield(permanent)
	g.resolution.SetMode(0)
	returned := g.PutOnBattlefield(card, owner)
	if returned.CurrentPower(g) != 3 {
		t.Fatal("a prior battlefield characteristic survived a zone change")
	}
}
