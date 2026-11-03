package mtgish

import (
	"bytes"
	"encoding/json"
	"fmt"

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
	var raw struct {
		Kind string          `json:"_ManaSymbol"`
		Args json.RawMessage `json:"args"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	symbol := MtgishManaSymbol{Kind: raw.Kind}
	switch raw.Kind {
	case "ManaCostGeneric":
		if len(raw.Args) == 0 {
			return fmt.Errorf("mtgish generic mana requires args")
		}
		if err := json.Unmarshal(raw.Args, &symbol.Amount); err != nil {
			return fmt.Errorf("decode mtgish generic mana: %w", err)
		}
		if symbol.Amount == nil || *symbol.Amount < 0 {
			return fmt.Errorf("mtgish generic mana requires a nonnegative integer")
		}
	case "ManaCostW", "ManaCostU", "ManaCostB", "ManaCostR", "ManaCostG",
		"ManaCostC", "ManaCostS", "ManaCostX",
		"ManaCostWP", "ManaCostUP", "ManaCostBP", "ManaCostRP", "ManaCostGP",
		"ManaCostRWP", "ManaCostRGP", "ManaCostGWP", "ManaCostGUP",
		"ManaCost2W", "ManaCost2U", "ManaCost2B", "ManaCost2R", "ManaCost2G",
		"ManaCostCW", "ManaCostCU", "ManaCostCB", "ManaCostCR", "ManaCostCG",
		"ManaCostWU", "ManaCostUB", "ManaCostBR", "ManaCostRG", "ManaCostGW",
		"ManaCostWB", "ManaCostUR", "ManaCostBG", "ManaCostRW", "ManaCostGU":
		if len(raw.Args) != 0 {
			return fmt.Errorf("mtgish mana symbol %s does not accept args", raw.Kind)
		}
	default:
		return fmt.Errorf("unknown mtgish mana symbol %q", raw.Kind)
	}
	*m = symbol
	return nil
}

// MtgishCard contains card data from mtgish JSON.
type MtgishCard struct {
	Kind           string `json:"_OracleCard"`
	Name           string
	Typeline       MtgishTypeline
	ManaCost       []MtgishManaSymbol
	CardPT         *MtgishCardPT
	ColorIndicator []string
	Rules          []json.RawMessage
	Raw            json.RawMessage `json:"-"`
}

// MtgishCardPT contains printed power and toughness.
type MtgishCardPT struct {
	Power     int32
	Toughness int32
}

// ToCardDefinitionSpec copies supported printed characteristics into a spec.
// Rules and starting counters require a rules compiler and are not copied.
// The compiler must complete and validate the spec before publication.
func (c MtgishCard) ToCardDefinitionSpec() (mage.CardDefinitionSpec, error) {
	if c.Kind != "Card" {
		return mage.CardDefinitionSpec{}, fmt.Errorf("mtgish card kind %q cannot be converted to mage.CardDefinitionSpec", c.Kind)
	}
	if c.Name == "" || len(c.Typeline.Cardtypes) == 0 {
		return mage.CardDefinitionSpec{}, fmt.Errorf("mtgish Card requires a name and card types")
	}
	cost, err := mtgishBaseManaCost(c.ManaCost)
	if err != nil {
		return mage.CardDefinitionSpec{}, fmt.Errorf("mtgish card %q: %w", c.Name, err)
	}
	spec := mage.CardDefinitionSpec{Name: c.Name, ManaCost: cost}
	for _, name := range c.Typeline.Cardtypes {
		kind, err := core.CardTypeString(name)
		if err != nil {
			return mage.CardDefinitionSpec{}, fmt.Errorf("mtgish card type %q: %w", name, err)
		}
		spec.TypeLine.CardTypes = append(spec.TypeLine.CardTypes, kind)
	}
	for _, name := range c.Typeline.Supertypes {
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
			return mage.CardDefinitionSpec{}, fmt.Errorf("unsupported mtgish supertype %q", name)
		}
		spec.TypeLine.SuperTypes = append(spec.TypeLine.SuperTypes, kind)
	}
	spec.TypeLine.SubTypes = append([]string(nil), c.Typeline.Subtypes...)
	if c.CardPT != nil {
		spec.Power, spec.Toughness = int(c.CardPT.Power), int(c.CardPT.Toughness)
	}
	for _, name := range c.ColorIndicator {
		color, err := core.ColorString(name)
		if err != nil || color == core.Colorless || color == core.AnyColor {
			return mage.CardDefinitionSpec{}, fmt.Errorf("unsupported mtgish color indicator %q", name)
		}
		spec.Colors = append(spec.Colors, color)
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
	for _, symbol := range symbols {
		if symbol.Kind != "ManaCostGeneric" && symbol.Amount != nil {
			return core.ManaCost{}, fmt.Errorf("mtgish mana symbol %s does not accept args", symbol.Kind)
		}
		switch symbol.Kind {
		case "ManaCostGeneric":
			if symbol.Amount == nil || *symbol.Amount < 0 {
				return core.ManaCost{}, fmt.Errorf("mtgish generic mana requires a nonnegative integer")
			}
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
				return core.ManaCost{}, fmt.Errorf("mana symbol %q is not supported by ManaCost", symbol.Kind)
			}
		}
	}
	return cost, nil
}

// ParseMtgishJSON reads one card from mtgish JSON output.
// Raw keeps all fields for later use by the rules compiler.
func ParseMtgishJSON(data []byte) (MtgishCard, error) {
	var card MtgishCard
	if err := json.Unmarshal(data, &card); err != nil {
		return MtgishCard{}, fmt.Errorf("decode mtgish JSON: %w", err)
	}
	if card.Kind == "" {
		return MtgishCard{}, fmt.Errorf("mtgish JSON requires _OracleCard")
	}
	if card.Kind == "Card" && card.Name == "" {
		return MtgishCard{}, fmt.Errorf("mtgish Card requires Name")
	}
	card.Raw = append(json.RawMessage(nil), data...)
	return card, nil
}
