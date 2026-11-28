package mtgish

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// MtgishTypeline contains the card types from mtgish JSON.
type MtgishTypeline struct {
	Supertypes []string
	Cardtypes  []string
	Subtypes   []string
}

// MtgishManaSymbol contains one mana symbol. Amount is set only for generic mana.
// Separate symbols remain separate entries, including repeated X symbols.
type MtgishManaSymbol struct {
	Kind   string `json:"_ManaSymbol"`
	Amount *int32 `json:"args,omitempty"`
}

// UnmarshalJSON checks the symbol and its argument before it stores the value.
func (m *MtgishManaSymbol) UnmarshalJSON(data []byte) error {
	if err := validateJSON(data, ""); err != nil {
		return err
	}
	symbol, err := decodeManaSymbol(data, "")
	if err != nil {
		return err
	}
	*m = symbol
	return nil
}

func decodeManaSymbol(data []byte, path string) (MtgishManaSymbol, error) {
	node, err := decodeTaggedNode(data, "_ManaSymbol", path)
	if err != nil {
		return MtgishManaSymbol{}, err
	}
	symbol := MtgishManaSymbol{Kind: node.Tag}
	if node.Tag == "ManaCostGeneric" {
		if err := decodeValue(node.Args, pointerPath(path, "args"), &symbol.Amount); err != nil {
			return MtgishManaSymbol{}, withNode(err, node)
		}
	} else {
		if err := symbol.validate(path); err != nil {
			return MtgishManaSymbol{}, err
		}
		if len(node.Args) != 0 {
			return MtgishManaSymbol{}, diagnostic(pointerPath(path, "args"), InvalidInput, node.Family, node.Tag, "mana symbol does not accept args")
		}
	}
	if err := symbol.validate(path); err != nil {
		return MtgishManaSymbol{}, err
	}
	return symbol, nil
}

func (m MtgishManaSymbol) validate(path string) error {
	if m.Kind == "ManaCostGeneric" {
		if m.Amount == nil || *m.Amount < 0 {
			return diagnostic(pointerPath(path, "args"), InvalidInput, "_ManaSymbol", m.Kind, "generic mana requires a nonnegative 32-bit integer")
		}
		return nil
	}
	switch m.Kind {
	case "ManaCostW", "ManaCostU", "ManaCostB", "ManaCostR", "ManaCostG",
		"ManaCostC", "ManaCostS", "ManaCostX",
		"ManaCostWP", "ManaCostUP", "ManaCostBP", "ManaCostRP", "ManaCostGP",
		"ManaCostRWP", "ManaCostRGP", "ManaCostGWP", "ManaCostGUP",
		"ManaCost2W", "ManaCost2U", "ManaCost2B", "ManaCost2R", "ManaCost2G",
		"ManaCostCW", "ManaCostCU", "ManaCostCB", "ManaCostCR", "ManaCostCG",
		"ManaCostWU", "ManaCostUB", "ManaCostBR", "ManaCostRG", "ManaCostGW",
		"ManaCostWB", "ManaCostUR", "ManaCostBG", "ManaCostRW", "ManaCostGU":
		if m.Amount != nil {
			return diagnostic(pointerPath(path, "args"), InvalidInput, "_ManaSymbol", m.Kind, "mana symbol does not accept args")
		}
		return nil
	case "":
		return diagnostic(pointerPath(path, "_ManaSymbol"), InvalidInput, "_ManaSymbol", "", "expected a nonempty mana symbol tag")
	default:
		return diagnostic(path, UnsupportedConstruct, "_ManaSymbol", m.Kind, "unsupported mana symbol")
	}
}

// MtgishCard contains card data from mtgish JSON.
// Loyalty and Defense are retained so the compiler can report engine gaps.
type MtgishCard struct {
	Kind           string `json:"_OracleCard"`
	Name           string
	Typeline       MtgishTypeline
	ManaCost       []MtgishManaSymbol
	CardPT         *MtgishCardPT
	ColorIndicator []string
	Loyalty        json.RawMessage `json:",omitempty"`
	Defense        json.RawMessage `json:",omitempty"`
	Rules          []json.RawMessage
}

// MtgishCardPT contains printed power and toughness.
// Pointers keep a printed zero distinct from a missing value.
type MtgishCardPT struct {
	Power     *int32
	Toughness *int32
}

func (c MtgishCard) validateMetadata() error {
	if c.Kind == "" {
		return diagnostic("/_OracleCard", InvalidInput, "_OracleCard", "", "expected a nonempty card form")
	}
	if c.Kind != "Card" {
		return diagnostic("/_OracleCard", UnsupportedConstruct, "_OracleCard", c.Kind, "unsupported card form")
	}
	if c.Name == "" {
		return diagnostic("/Name", InvalidInput, "", "", "expected a nonempty card name")
	}
	if len(c.Typeline.Cardtypes) == 0 {
		return diagnostic("/Typeline/Cardtypes", InvalidInput, "", "", "expected at least one card type")
	}
	for _, field := range []struct {
		path   string
		values []string
	}{
		{"/Typeline/Supertypes", c.Typeline.Supertypes},
		{"/Typeline/Cardtypes", c.Typeline.Cardtypes},
		{"/Typeline/Subtypes", c.Typeline.Subtypes},
		{"/ColorIndicator", c.ColorIndicator},
	} {
		for i, value := range field.values {
			if value == "" {
				return diagnostic(indexPath(field.path, i), InvalidInput, "", "", "expected a nonempty string")
			}
		}
	}
	if c.ManaCost != nil && len(c.ManaCost) == 0 {
		return diagnostic("/ManaCost", InvalidInput, "", "", "an explicit mana cost requires at least one symbol")
	}
	for i, symbol := range c.ManaCost {
		if err := symbol.validate(indexPath("/ManaCost", i)); err != nil {
			return err
		}
	}
	if c.CardPT == nil {
		if slices.Contains(c.Typeline.Cardtypes, "Creature") {
			return diagnostic("/CardPT", InvalidInput, "", "", "a creature requires printed power and toughness")
		}
	} else {
		if c.CardPT.Power == nil {
			return diagnostic("/CardPT/Power", InvalidInput, "", "", "printed power is required")
		}
		if c.CardPT.Toughness == nil {
			return diagnostic("/CardPT/Toughness", InvalidInput, "", "", "printed toughness is required")
		}
	}
	for _, field := range []struct {
		path string
		data json.RawMessage
	}{
		{"/Loyalty", c.Loyalty},
		{"/Defense", c.Defense},
	} {
		if field.data != nil {
			if err := validateJSONDepth(field.data, field.path, 1); err != nil {
				return err
			}
			if isNull(field.data) {
				return diagnostic(field.path, InvalidInput, "", "", "metadata must not be null")
			}
		}
	}
	return nil
}

// ToCardDefinitionSpec copies supported printed characteristics into a spec.
// Rules require a rules compiler and are not copied.
// The compiler must complete the spec before publication.
func (c MtgishCard) ToCardDefinitionSpec() (spec mage.CardDefinitionSpec, err error) {
	defer func() {
		if err != nil {
			spec = mage.CardDefinitionSpec{}
			err = withCard(err, c.Name)
		}
	}()
	if err := c.validateMetadata(); err != nil {
		return spec, err
	}
	spec.Name = c.Name
	for i, name := range c.Typeline.Cardtypes {
		path := indexPath("/Typeline/Cardtypes", i)
		if name == "Battle" {
			return spec, diagnostic(path, EngineGap, "", "", "the engine does not support Battles")
		}
		kind, err := core.CardTypeString(name)
		if err != nil || kind.String() != name {
			return spec, diagnostic(path, UnsupportedConstruct, "", "", fmt.Sprintf("unsupported card type %q", name))
		}
		spec.TypeLine.CardTypes = append(spec.TypeLine.CardTypes, kind)
	}
	for i, name := range c.Typeline.Supertypes {
		var kind core.SuperType
		switch name {
		case "Legendary":
			kind = core.SuperLegendary
		case "Basic":
			kind = core.SuperBasic
		case "Snow":
			kind = core.SuperSnow
		case "World":
			kind = core.SuperWorld
		default:
			return spec, diagnostic(indexPath("/Typeline/Supertypes", i), UnsupportedConstruct, "", "", fmt.Sprintf("unsupported supertype %q", name))
		}
		spec.TypeLine.SuperTypes = append(spec.TypeLine.SuperTypes, kind)
	}
	spec.TypeLine.SubTypes = slices.Clone(c.Typeline.Subtypes)
	if c.ManaCost == nil && !slices.Contains(spec.TypeLine.CardTypes, core.TypeLand) {
		return spec, diagnostic("/ManaCost", EngineGap, "", "", "the engine cannot preserve an absent nonland mana cost")
	}
	spec.ManaCost, err = mtgishBaseManaCost(c.ManaCost)
	if err != nil {
		return spec, err
	}
	if c.CardPT != nil {
		spec.Power, spec.Toughness = int(*c.CardPT.Power), int(*c.CardPT.Toughness)
	}
	if c.ColorIndicator != nil {
		spec.Colors = make([]core.Color, 0, len(c.ColorIndicator))
	}
	for i, name := range c.ColorIndicator {
		color, err := core.ColorString(name)
		if err != nil || color.String() != name || color == core.Colorless || color == core.AnyColor {
			return spec, diagnostic(indexPath("/ColorIndicator", i), UnsupportedConstruct, "", "", fmt.Sprintf("unsupported color indicator %q", name))
		}
		spec.Colors = append(spec.Colors, color)
	}
	if c.Loyalty != nil {
		return spec, diagnostic("/Loyalty", EngineGap, "", "", "loyalty metadata has no supported lowering")
	}
	if c.Defense != nil {
		return spec, diagnostic("/Defense", EngineGap, "", "", "defense metadata has no supported lowering")
	}
	return spec, nil
}

func mtgishBaseManaCost(symbols []MtgishManaSymbol) (core.ManaCost, error) {
	var cost core.ManaCost
	hybrids := map[string]core.HybridSymbol{
		"ManaCostWU": {A: core.White, B: core.Blue},
		"ManaCostUB": {A: core.Blue, B: core.Black},
		"ManaCostBR": {A: core.Black, B: core.Red},
		"ManaCostRG": {A: core.Red, B: core.Green},
		"ManaCostGW": {A: core.Green, B: core.White},
		"ManaCostWB": {A: core.White, B: core.Black},
		"ManaCostUR": {A: core.Blue, B: core.Red},
		"ManaCostBG": {A: core.Black, B: core.Green},
		"ManaCostRW": {A: core.Red, B: core.White},
		"ManaCostGU": {A: core.Green, B: core.Blue},
	}
	var total int64
	for i, symbol := range symbols {
		path := indexPath("/ManaCost", i)
		if err := symbol.validate(path); err != nil {
			return core.ManaCost{}, err
		}
		amount := int64(1)
		if symbol.Kind == "ManaCostGeneric" {
			amount = int64(*symbol.Amount)
		}
		total += amount
		if total > math.MaxInt32 {
			return core.ManaCost{}, diagnostic(path, InvalidInput, "_ManaSymbol", symbol.Kind, "mana cost total exceeds a 32-bit integer")
		}
		switch symbol.Kind {
		case "ManaCostGeneric":
			cost.Generic += int(*symbol.Amount)
		case "ManaCostW":
			cost.White++
		case "ManaCostU":
			cost.Blue++
		case "ManaCostB":
			cost.Black++
		case "ManaCostR":
			cost.Red++
		case "ManaCostG":
			cost.Green++
		case "ManaCostC":
			cost.Colorless++
		case "ManaCostX":
			cost.HasX = true
			cost.XCount++
		default:
			if hybrid, ok := hybrids[symbol.Kind]; ok {
				cost.Hybrid = append(cost.Hybrid, hybrid)
			} else {
				return core.ManaCost{}, diagnostic(path, EngineGap, "_ManaSymbol", symbol.Kind, "the engine cannot represent this mana symbol")
			}
		}
	}
	return cost, nil
}

// ParseMtgishJSON reads one complete card record and checks its structure.
// Rules remain source data until Compile validates and lowers them.
func ParseMtgishJSON(data []byte) (card MtgishCard, err error) {
	if err := validateJSON(data, ""); err != nil {
		return MtgishCard{}, err
	}
	fields, err := objectFields(data, "")
	if err != nil {
		return MtgishCard{}, err
	}
	var name string
	_ = json.Unmarshal(fields["Name"], &name)
	defer func() {
		if err != nil {
			card = MtgishCard{}
			err = withCard(err, name)
		}
	}()
	if err := decodeValue(fields["_OracleCard"], "/_OracleCard", &card.Kind); err != nil {
		return card, err
	}
	if card.Kind != "Card" {
		return card, card.validateMetadata()
	}
	if err := checkFields(fields, "", "_OracleCard", "Name", "Typeline", "ManaCost", "CardPT", "ColorIndicator", "Loyalty", "Defense", "Rules"); err != nil {
		return card, err
	}
	if err := decodeValue(fields["Name"], "/Name", &card.Name); err != nil {
		return card, err
	}
	card.Typeline, err = decodeTypeline(fields["Typeline"])
	if err != nil {
		return card, err
	}
	if raw, present := fields["ManaCost"]; present {
		var symbols []json.RawMessage
		if err := decodeValue(raw, "/ManaCost", &symbols); err != nil {
			return card, err
		}
		card.ManaCost = make([]MtgishManaSymbol, len(symbols))
		for i, raw := range symbols {
			card.ManaCost[i], err = decodeManaSymbol(raw, indexPath("/ManaCost", i))
			if err != nil {
				return card, err
			}
		}
	}
	if raw, present := fields["CardPT"]; present {
		pt, err := objectFields(raw, "/CardPT")
		if err != nil {
			return card, err
		}
		if err := checkFields(pt, "/CardPT", "Power", "Toughness"); err != nil {
			return card, err
		}
		card.CardPT = &MtgishCardPT{}
		if err := decodeValue(pt["Power"], "/CardPT/Power", &card.CardPT.Power); err != nil {
			return card, err
		}
		if err := decodeValue(pt["Toughness"], "/CardPT/Toughness", &card.CardPT.Toughness); err != nil {
			return card, err
		}
	}
	if raw, present := fields["ColorIndicator"]; present {
		card.ColorIndicator, err = decodeStringList(raw, "/ColorIndicator")
		if err != nil {
			return card, err
		}
	}
	card.Loyalty, card.Defense = fields["Loyalty"], fields["Defense"]
	if raw, present := fields["Rules"]; present {
		if err := decodeValue(raw, "/Rules", &card.Rules); err != nil {
			return card, err
		}
		for i, rule := range card.Rules {
			if isNull(rule) {
				return card, diagnostic(indexPath("/Rules", i), InvalidInput, "", "", "a rule must not be null")
			}
		}
	}
	return card, card.validateMetadata()
}

func decodeTypeline(data []byte) (MtgishTypeline, error) {
	fields, err := objectFields(data, "/Typeline")
	if err != nil {
		return MtgishTypeline{}, err
	}
	if err := checkFields(fields, "/Typeline", "Supertypes", "Cardtypes", "Subtypes"); err != nil {
		return MtgishTypeline{}, err
	}
	var typeline MtgishTypeline
	for _, field := range []struct {
		name  string
		value *[]string
	}{
		{"Supertypes", &typeline.Supertypes},
		{"Cardtypes", &typeline.Cardtypes},
		{"Subtypes", &typeline.Subtypes},
	} {
		raw, present := fields[field.name]
		if !present && field.name != "Cardtypes" {
			continue
		}
		*field.value, err = decodeStringList(raw, pointerPath("/Typeline", field.name))
		if err != nil {
			return MtgishTypeline{}, err
		}
	}
	return typeline, nil
}
