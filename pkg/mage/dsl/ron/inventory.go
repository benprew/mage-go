package ron

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
	"strings"
)

// Inventory keeps syntax evidence separate from rule execution evidence.
type Inventory struct {
	SchemaVersion    int                  `json:"schema_version"`
	Records          int                  `json:"records"`
	UniqueCards      int                  `json:"unique_cards"`
	Compiled         int                  `json:"compiled"`
	EngineValidated  *int                 `json:"engine_validated"`
	BehaviorVerified *int                 `json:"behavior_verified"`
	Files            []InventoryFile      `json:"files"`
	Issues           []Issue              `json:"issues"`
	Signatures       []InventorySignature `json:"signatures"`
}

type InventoryFile struct {
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Records int    `json:"records"`
}

// InventorySignature groups expressions by their declaration context.
// Support stays unverified because compilation cannot prove execution.
type InventorySignature struct {
	Role                string             `json:"role"`
	Name                string             `json:"name"`
	Kind                string             `json:"kind"`
	Arity               int                `json:"arity"`
	Shape               string             `json:"shape"`
	Context             string             `json:"context"`
	Support             string             `json:"support"`
	Occurrences         int                `json:"occurrences"`
	CompiledOccurrences int                `json:"compiled_record_occurrences"`
	Cards               []string           `json:"cards"`
	Examples            []InventoryExample `json:"examples"`
}

type InventoryExample struct {
	Card   string   `json:"card"`
	Path   string   `json:"path"`
	Source Position `json:"source"`
}

// InspectFS reads all expressions, including children of rejected declarations.
// It does not build a registry or change the compiler's binding environment.
func InspectFS(files fs.FS, pattern string) (*Inventory, error) {
	paths, err := fs.Glob(files, pattern)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no RON files match %q", pattern)
	}
	report := &Inventory{SchemaVersion: 1, Files: []InventoryFile{}, Issues: []Issue{}, Signatures: []InventorySignature{}}
	entries := map[string]*InventorySignature{}
	names := map[string]bool{}
	definitions := map[string]string{}
	positions := map[string]Position{}
	for _, path := range paths {
		data, err := fs.ReadFile(files, path)
		if err != nil {
			report.Issues = append(report.Issues, Issue{Source: Position{File: path}, Message: err.Error()})
			continue
		}
		file := InventoryFile{Path: path, SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
		nodes, err := Parse(strings.NewReader(string(data)), path)
		if err != nil {
			report.Files = append(report.Files, file)
			report.Issues = append(report.Issues, Issue{Source: Position{File: path}, Message: err.Error()})
			continue
		}
		file.Records = len(nodes)
		report.Files = append(report.Files, file)
		if len(nodes) == 0 {
			report.Issues = append(report.Issues, Issue{Source: Position{File: path}, Message: "empty card file"})
		}
		for _, node := range nodes {
			report.Records++
			name := node.Fields["Name"].String
			if name != "" {
				names[name] = true
			}
			_, compileErr := Compile(node)
			if compileErr == nil {
				key := definitionKey(node)
				if previous, ok := definitions[name]; ok && previous != key {
					compileErr = fmt.Errorf("conflicting definition; first definition at %s", positions[name])
				} else {
					if _, exists := definitions[name]; !exists {
						definitions[name], positions[name] = key, node.Position
					}
				}
			}
			if compileErr != nil {
				report.Issues = append(report.Issues, Issue{Card: name, Source: node.Position, Message: compileErr.Error()})
			} else {
				report.Compiled++
			}
			walkInventory(node, name, "Card", "card", "collection", compileErr == nil, entries)
		}
	}
	report.UniqueCards = len(names)
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		entry := entries[key]
		slices.Sort(entry.Cards)
		report.Signatures = append(report.Signatures, *entry)
	}
	return report, nil
}

func inventoryKind(n Node) string {
	switch n.Kind {
	case Symbol:
		return "symbol"
	case String:
		return "string"
	case IntegerNode:
		return "integer"
	case List:
		return "list"
	case Tuple:
		return "tuple"
	case Struct:
		return "struct"
	default:
		return "unknown"
	}
}

func inventoryFields(n Node) []string {
	keys := make([]string, 0, len(n.Fields))
	for key := range n.Fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func inventoryShape(n Node) string {
	if n.Kind == Struct {
		return "fields(" + strings.Join(inventoryFields(n), ",") + ")"
	}
	parts := make([]string, len(n.Args))
	for i, arg := range n.Args {
		parts[i] = inventoryKind(arg) + ":" + arg.Name
	}
	return inventoryKind(n) + "(" + strings.Join(parts, ",") + ")"
}

func walkInventory(n Node, card, path, role, context string, compiled bool, entries map[string]*InventorySignature) {
	entry := InventorySignature{Role: role, Name: n.Name, Kind: inventoryKind(n), Arity: len(n.Args), Shape: inventoryShape(n), Context: context, Support: "unverified"}
	keyBytes, _ := json.Marshal(entry)
	key := string(keyBytes)
	current := entries[key]
	if current == nil {
		current = &entry
		entries[key] = current
	}
	current.Occurrences++
	if compiled {
		current.CompiledOccurrences++
	}
	if card != "" && !slices.Contains(current.Cards, card) {
		current.Cards = append(current.Cards, card)
	}
	current.Examples = append(current.Examples, InventoryExample{Card: card, Path: path, Source: n.Position})
	for _, field := range inventoryFields(n) {
		childRole := "unknown"
		if role == "card" && n.Name == "Card" {
			if field == "Rules" {
				childRole = "rule"
			} else {
				childRole = "metadata"
			}
		} else if role == "metadata" {
			childRole = "metadata"
		}
		walkInventory(n.Fields[field], card, path+"."+field, childRole, context+"/"+n.Name+"."+field, compiled, entries)
	}
	for i, arg := range n.Args {
		childPath := fmt.Sprintf("%s.args[%d]", path, i)
		childContext := fmt.Sprintf("%s/%s[%d]", context, n.Name, i)
		childRole := inventoryArgumentRole(n, role, i)
		if n.Kind == List {
			childPath = fmt.Sprintf("%s[%d]", path, i)
			childContext = context + "/[]"
			childRole = role
		}
		walkInventory(arg, card, childPath, childRole, childContext, compiled, entries)
	}
}

// Unknown roles prevent an unsupported parent from inventing a child's contract.
func inventoryArgumentRole(n Node, role string, index int) string {
	if role == "metadata" {
		return role
	}
	if n.Kind != Tuple {
		return "unknown"
	}
	switch role {
	case "rule":
		if n.Name == "PreventDamage" && len(n.Args) == 2 {
			return []string{"prevention_event", "prevention_operation"}[index]
		}
		if n.Name == "Landwalk" && len(n.Args) == 1 {
			return "landwalk_filter"
		}
		switch n.Name {
		case "If":
			if len(n.Args) == 2 {
				return []string{"static_condition", "rule"}[index]
			}
		case "AsPermanentEnters":
			if len(n.Args) == 2 {
				return []string{"permanent_reference", "entry_operation"}[index]
			}
		case "Rampage":
			if len(n.Args) == 1 {
				return "integer_literal"
			}
		case "SpellActions":
			if len(n.Args) == 1 {
				return "actions"
			}
		case "ActivatedWithModifiers":
			if len(n.Args) == 3 {
				return []string{"cost", "actions", "activation_modifier"}[index]
			}
		case "Activated":
			if len(n.Args) == 2 {
				return []string{"cost", "actions"}[index]
			}
		case "EachPermanentRuleEffect":
			if len(n.Args) == 2 {
				return []string{"permanent_filter", "permanent_rule_operation"}[index]
			}
		case "PermanentRuleEffect":
			if len(n.Args) == 2 {
				return []string{"permanent_reference", "permanent_rule_operation"}[index]
			}
		case "TriggerI":
			if len(n.Args) == 3 {
				return []string{"event", "trigger_condition", "actions"}[index]
			}
		case "TriggerA":
			if len(n.Args) == 2 {
				return []string{"event", "actions"}[index]
			}
		case "EnchantPermanent":
			if len(n.Args) == 1 {
				return "permanent_filter"
			}
		case "EachPermanentLayerEffect":
			if len(n.Args) == 2 {
				return []string{"permanent_filter", "static_modification"}[index]
			}
		case "PermanentLayerEffect":
			if len(n.Args) == 2 {
				return []string{"permanent_reference", "static_modification"}[index]
			}
		}
	case "condition":
		if n.Name == "PlayerPassesFilter" && len(n.Args) == 2 {
			return []string{"player_reference", "static_player_filter"}[index]
		}
	case "static_condition":
		if n.Name == "APlayerPassesFilter" && len(n.Args) == 2 {
			return []string{"player_filter", "static_player_filter"}[index]
		}
		if n.Name == "PlayerPassesFilter" && len(n.Args) == 2 {
			return []string{"player_reference", "static_player_filter"}[index]
		}
		if n.Name == "PermanentPassesFilter" && len(n.Args) == 2 {
			return []string{"permanent_reference", "permanent_filter"}[index]
		}
	case "static_player_filter":
		if n.Name == "And" && len(n.Args) == 1 {
			return "static_player_filter"
		}
		if (n.Name == "ControlsA" || n.Name == "ControlsNo") && len(n.Args) == 1 {
			return "permanent_filter"
		}
	case "defending_player_condition":
		if n.Name == "PlayerPassesFilter" && len(n.Args) == 2 {
			return []string{"player_reference", "static_player_filter"}[index]
		}
	case "trigger_condition":
		if n.Name == "NoPermanentsPassFilter" && len(n.Args) == 2 {
			return []string{"permanent_filter", "zone"}[index]
		}
		if n.Name == "PermanentPassesFilter" && len(n.Args) == 2 {
			return []string{"permanent_reference", "permanent_filter"}[index]
		}
	case "activation_modifier":
		if n.Name == "And" && len(n.Args) == 1 {
			return "activation_modifier"
		}
		if n.Name == "ActivateNoMoreThanNumberTimesEachTurn" && len(n.Args) == 1 {
			return "activation_limit_value"
		}
		if n.Name == "ActivateOnlyIf" && len(n.Args) == 1 {
			return "activation_condition"
		}
	case "activation_limit_value":
		if n.Name == "Integer" && len(n.Args) == 1 {
			return "integer_literal"
		}
	case "activation_condition":
		if n.Name == "PlayerPassesFilter" && len(n.Args) == 2 {
			return []string{"player_reference", "activation_player_filter"}[index]
		}
	case "actions":
		if n.Name == "Modal_ChooseOne" && len(n.Args) == 1 {
			return "actions"
		}
		if n.Name == "Targeted" && len(n.Args) == 2 {
			return []string{"target", "actions"}[index]
		}
		if n.Name == "ActionList" && len(n.Args) == 1 {
			return "action"
		}
	case "target":
		if n.Name == "NumberTargetPermanents" && len(n.Args) == 2 {
			return []string{"announcement_value", "permanent_filter"}[index]
		}
		if len(n.Args) == 1 {
			switch n.Name {
			case "TargetPermanent", "OneOrMoreTargetPermanents":
				return "permanent_filter"
			case "TargetPlayer":
				return "player_filter"
			case "TargetGraveyardCard":
				return "card_filter"
			case "TargetSpell":
				return "spell_filter"
			}
		}
	case "announcement_value":
		if n.Name == "Integer" && len(n.Args) == 1 {
			return "integer_literal"
		}
	case "card_filter":
		if n.Name == "IsColor" || n.Name == "IsNonColor" {
			if len(n.Args) == 1 {
				return "color"
			}
		}
		if (n.Name == "And" || n.Name == "Or") && len(n.Args) == 1 {
			return "card_filter"
		}
		if (n.Name == "IsCardtype" || n.Name == "IsNonCardtype") && len(n.Args) == 1 {
			return "card_type"
		}
		if n.Name == "InAPlayersGraveyard" && len(n.Args) == 1 {
			return "player_filter"
		}
	case "cost":
		if n.Name == "RemoveCounters" && len(n.Args) == 1 {
			return "counter_removal_instruction"
		}
		if n.Name == "SacrificeAPermanent" && len(n.Args) == 1 {
			return "permanent_filter"
		}
		if n.Name == "And" && len(n.Args) == 1 {
			return "cost"
		}
		if (n.Name == "TapPermanent" || n.Name == "SacrificePermanent") && len(n.Args) == 1 {
			return "permanent_reference"
		}
		if n.Name == "PayMana" && len(n.Args) == 1 {
			return "metadata"
		}
		if n.Name == "PayLife" && len(n.Args) == 1 {
			return "value"
		}
	case "token_instruction", "token_definition":
		if n.Name == "NumberTokens" && len(n.Args) == 2 {
			return []string{"value", "token_definition"}[index]
		}
		if n.Name == "TokenWithPT" && len(n.Args) == 6 {
			return []string{"token_pt", "token_colors", "token_supertype", "token_card_type", "token_subtypes", "token_keyword"}[index]
		}
		if n.Name == "NamedTokenWithPT" && len(n.Args) == 7 {
			return []string{"token_name", "token_pt", "token_colors", "token_supertype", "token_card_type", "token_subtypes", "token_keyword"}[index]
		}
	case "token_pt":
		if n.Name == "PT" && len(n.Args) == 2 {
			return "integer_literal"
		}
	case "token_colors":
		if n.Name == "Colors" && len(n.Args) == 1 {
			return "token_color"
		}
	case "token_subtypes":
		if n.Name == "SubtypesList" && len(n.Args) == 1 {
			return "token_subtype"
		}
	case "prevention_event":
		if n.Name == "DamageWouldBeDealtByAPermanentToRecipient" && len(n.Args) == 2 {
			return []string{"permanent_filter", "prevention_recipient"}[index]
		}
		if n.Name == "Or" && len(n.Args) == 1 {
			return "prevention_event"
		}
		if (n.Name == "CombatDamageWouldBeDealtToRecipient" || n.Name == "DamageWouldBeDealtToRecipient") && len(n.Args) == 1 {
			return "prevention_recipient"
		}
		if n.Name == "CombatDamageWouldBeDealtByCreature" && len(n.Args) == 1 {
			return "permanent_reference"
		}
	case "prevention_recipient":
		if n.Name == "Player" && len(n.Args) == 1 {
			return "player_reference"
		}
		if n.Name == "Permanent" && len(n.Args) == 1 {
			return "permanent_reference"
		}
	case "exile_subject":
		if n.Name == "EachPermanent" && len(n.Args) == 1 {
			return "permanent_collection_reference"
		}
		if n.Name == "Permanent" && len(n.Args) == 1 {
			return "permanent_reference"
		}
	case "delayed_step_event":
		if n.Name == "AtTheBeginningOfPlayersNextUpkeep" && len(n.Args) == 1 {
			return "player_reference"
		}
	case "action":
		switch n.Name {
		case "CreateFutureTrigger":
			if len(n.Args) == 2 {
				return []string{"delayed_step_event", "actions"}[index]
			}
		case "MayAction":
			if len(n.Args) == 1 {
				return "action"
			}
		case "Exile":
			if len(n.Args) == 2 {
				return []string{"exile_subject", "exile_modifier"}[index]
			}
		case "PutGraveyardCardIntoHand":
			if len(n.Args) == 1 {
				return "card_reference"
			}
		case "PutGraveyardCardOntoBattlefield":
			if len(n.Args) == 2 {
				return []string{"card_reference", "entry_operation"}[index]
			}
		case "CreatePreventDamageUntil":
			if len(n.Args) == 3 {
				return []string{"prevention_event", "prevention_operation", "duration"}[index]
			}
		case "AddMana":
			if len(n.Args) == 1 {
				return "mana_production"
			}
		case "AddManaRepeated":
			if len(n.Args) == 2 {
				return []string{"mana_production", "value"}[index]
			}
		case "CreateTokens":
			if len(n.Args) == 2 {
				return []string{"token_instruction", "token_creation_modifier"}[index]
			}

		case "PutCounters":
			if len(n.Args) == 1 {
				return "counter_instruction"
			}
		case "MayCost":
			if len(n.Args) == 1 {
				return "cost"
			}
		case "PlayerMayCost":
			if len(n.Args) == 2 {
				return []string{"player_reference", "cost"}[index]
			}
		case "Unless", "If":
			if len(n.Args) == 2 {
				return []string{"condition", "action"}[index]
			}
		case "IfElse":
			if len(n.Args) == 3 {
				return []string{"condition", "action", "action"}[index]
			}
		case "CreateEachPermanentLayerEffectUntil":
			if len(n.Args) == 3 {
				if index == 0 && n.Args[0].Name == "Ref_TargetPermanents" {
					return "permanent_collection_reference"
				}
				return []string{"permanent_filter", "temporary_modification", "duration"}[index]
			}
		case "CreatePermanentRuleEffectUntil":
			if len(n.Args) == 3 {
				return []string{"permanent_reference", "permanent_rule_operation", "duration"}[index]
			}
		case "CreatePermanentLayerEffect":
			if len(n.Args) == 2 {
				return []string{"permanent_reference", "resolving_modification"}[index]
			}
		case "CreatePermanentLayerEffectUntil":
			if len(n.Args) == 3 {
				return []string{"permanent_reference", "temporary_modification", "duration"}[index]
			}
		case "TapEachPermanent":
			if len(n.Args) == 1 {
				if n.Args[0].Name == "Ref_TargetPermanents" {
					return "permanent_collection_reference"
				}
				return "permanent_filter"
			}
		case "PutPermanentIntoItsOwnersHand", "TapPermanent", "UntapPermanent", "TapOrUntapPermanent", "DestroyPermanent", "DestroyPermanentNoRegen", "RegeneratePermanent", "SacrificePermanent", "PermanentDoesntUntapDuringControllersNextUntap":
			if len(n.Args) == 1 {
				return "permanent_reference"
			}
		case "CounterSpell":
			if len(n.Args) == 1 {
				return "spell_reference"
			}
		case "EachPlayerAction":
			if len(n.Args) == 2 {
				return []string{"player_filter", "player_action"}[index]
			}
		case "PlayerAction":
			if len(n.Args) == 2 {
				return []string{"player_reference", "player_action"}[index]
			}
		case "SpellDealsMultipleDamage":
			if len(n.Args) == 2 {
				return []string{"spell_reference", "damage_packet"}[index]
			}
		case "PermanentDealsMultipleDamage":
			if len(n.Args) == 2 {
				return []string{"permanent_reference", "damage_packet"}[index]
			}
		case "SpellDealsDamage":
			if len(n.Args) == 3 {
				return []string{"spell_reference", "value", "recipient"}[index]
			}
		case "PermanentDealsDamage":
			if len(n.Args) == 3 {
				return []string{"permanent_reference", "value", "recipient"}[index]
			}
		case "DrawNumberCards", "DiscardNumberCards", "DiscardNumberCardsAtRandom", "GainLife", "LoseLife":
			if len(n.Args) == 1 {
				return "value"
			}
		}
	case "counter_instruction":
		if n.Name == "ACounterOfTypeOnPermanent" && len(n.Args) == 2 {
			return []string{"permanent_counter_type", "permanent_reference"}[index]
		}
		if n.Name == "NumberCountersOfTypeOnPermanent" && len(n.Args) == 3 {
			return []string{"value", "permanent_counter_type", "permanent_reference"}[index]
		}
	case "counter_removal_instruction":
		if (n.Name == "ACounterOfTypeFromPermanent" || n.Name == "AnyNumberOfCountersOfTypeFromPermanent") && len(n.Args) == 2 {
			return []string{"permanent_counter_type", "permanent_reference"}[index]
		}
		if n.Name == "NumberCountersOfTypeFromPermanent" && len(n.Args) == 3 {
			return []string{"value", "permanent_counter_type", "permanent_reference"}[index]
		}
	case "mana_production":
		if n.Name == "And" && len(n.Args) == 1 {
			return "mana_production"
		}
	case "entry_operation":
		if n.Name == "EntersWithNumberCounters" && len(n.Args) == 2 {
			return []string{"entry_value", "permanent_counter_type"}[index]
		}
	case "entry_value":
		if n.Name == "Integer" && len(n.Args) == 1 {
			return "integer_literal"
		}
	case "permanent_counter_type":
		if n.Name == "PTCounter" && len(n.Args) == 2 {
			return "integer_literal"
		}
	case "static_modification", "temporary_modification", "resolving_modification":
		switch n.Name {
		case "SetPower":
			if role == "temporary_modification" && len(n.Args) == 1 {
				return "value"
			}
		case "SetLandType":
			if (role == "static_modification" || role == "resolving_modification") && len(n.Args) == 1 {
				return "land_subtype"
			}
		case "SetController":
			if role == "static_modification" && len(n.Args) == 1 {
				return "player_reference"
			}
		case "SetPT":
			if len(n.Args) == 1 {
				return "fixed_pt"
			}
		case "AddCardtype":
			if len(n.Args) == 1 {
				return "card_type"
			}
		case "AddCreatureType", "SetCreatureType":
			if role == "temporary_modification" && len(n.Args) == 1 {
				return "creature_subtype"
			}
		case "SetColor":
			if (role == "static_modification" || role == "temporary_modification") && len(n.Args) == 1 {
				return "color_list"
			}
		case "AdjustPTX":
			if len(n.Args) == 3 {
				return []string{"pt_coordinate", "pt_coordinate", "value"}[index]
			}
		case "AdjustPT":
			if len(n.Args) == 2 {
				return "value"
			}
		case "AddAbility", "LosesAbility":
			if len(n.Args) == 1 {
				return "keyword"
			}
		}
	case "duration":
		if n.Name == "UntilNextUpkeep" && len(n.Args) == 1 {
			return "player_reference"
		}
		if n.Name == "ForAsLongAsPermanentRemainsTapped" && len(n.Args) == 1 {
			return "permanent_reference"
		}
	case "fixed_pt":
		if n.Name == "PT" && len(n.Args) == 2 {
			return "value"
		}
	case "color_list":
		if n.Name == "SimpleColorList" && len(n.Args) == 1 {
			return "color"
		}
	case "permanent_rule_operation":
		if n.Name == "CantAttackUnlessDefendingPlayer" && len(n.Args) == 1 {
			return "defending_player_condition"
		}
		if (n.Name == "CantBeBlockedExceptByDefenders" || n.Name == "CantBeBlockedByDefenders" || n.Name == "CantBlockAttackers") && len(n.Args) == 1 {
			return "permanent_filter"
		}
		if n.Name == "CanBeBlockedAsThoughItDidntHave" && len(n.Args) == 1 {
			return "keyword"
		}
	case "keyword":
		if n.Name == "Landwalk" && len(n.Args) == 1 {
			return "landwalk_filter"
		}
	case "landwalk_filter":
		if n.Name == "IsLandType" && len(n.Args) == 1 {
			return "land_subtype"
		}
	case "pt_coordinate":
		if n.Name == "Integer" && len(n.Args) == 1 {
			return "value"
		}
	case "value":
		if n.Name == "Twice" && len(n.Args) == 1 {
			return "value"
		}
		if n.Name == "TheNumberOfPermanentsOnTheBattlefield" && len(n.Args) == 1 {
			return "permanent_filter"
		}
		if n.Name == "Minus" && len(n.Args) == 2 {
			return "value"
		}
		if n.Name == "TheNumberOfCardsOfTypeInPlayersHand" && len(n.Args) == 2 {
			return []string{"card_filter", "player_reference"}[index]
		}
		if n.Name == "TheNumberOfCardsInPlayersHand" && len(n.Args) == 1 {
			return "player_reference"
		}
		if n.Name == "NumCreaturesOrPlaneswalkersThatDiedThisTurn" && len(n.Args) == 1 {
			return "permanent_filter"
		}
		if (n.Name == "ManaValueOfPermanent" || n.Name == "PowerOfPermanent") && len(n.Args) == 1 {
			return "permanent_reference"
		}
	case "player_action":
		if n.Name == "DiscardCards" && len(n.Args) == 1 {
			return "card_filter"
		}
		if n.Name == "AddMana" && len(n.Args) == 1 {
			return "mana_production"
		}
		if (n.Name == "DrawNumberCards" || n.Name == "DiscardNumberCards" || n.Name == "DiscardNumberCardsAtRandom" || n.Name == "GainLife" || n.Name == "LoseLife") && len(n.Args) == 1 {
			return "value"
		}
	case "damage_packet":
		if n.Name == "DamageToRecipients" && len(n.Args) == 2 {
			return []string{"value", "recipient"}[index]
		}
	case "recipient":
		if n.Name == "MultipleRecipients" && len(n.Args) == 1 {
			return "recipient_set"
		}
		if n.Name == "Player" && len(n.Args) == 1 {
			return "player_reference"
		}
		if n.Name == "Permanent" && len(n.Args) == 1 {
			return "permanent_reference"
		}
	case "recipient_set":
		if n.Name == "EachPermanent" && len(n.Args) == 1 {
			return "permanent_filter"
		}
		if n.Name == "EachPlayer" && len(n.Args) == 1 {
			return "player_filter"
		}
	case "player_reference":
		if n.Name == "ControllerOfPermanent" && len(n.Args) == 1 {
			return "permanent_reference"
		}
		if n.Name == "ControllerOfSpell" && len(n.Args) == 1 {
			return "spell_reference"
		}
	case "spell_filter":
		if (n.Name == "And" || n.Name == "Or") && len(n.Args) == 1 {
			return role
		}
	case "power_comparison":
		if len(n.Args) == 1 && (n.Name == "LessThanOrEqualTo" || n.Name == "GreaterThanOrEqualTo" || n.Name == "EqualTo") {
			return "value"
		}
	case "card_name":
		if n.Name == "NamedCard" && len(n.Args) == 1 {
			return "metadata"
		}
	case "permanent_filter":
		if n.Name == "AttachedToAPermanent" && len(n.Args) == 1 {
			return role
		}
		if n.Name == "DealtDamageToAnyPlayerThisTurn" && len(n.Args) == 1 {
			return "player_filter"
		}
		if n.Name == "WasDealtDamageByPermanentThisTurn" && len(n.Args) == 1 {
			return "permanent_reference"
		}
		if (n.Name == "IsSupertype" || n.Name == "IsNonSupertype") && len(n.Args) == 1 {
			return "supertype"
		}
		if n.Name == "ExceptFor" && len(n.Args) == 1 {
			return role
		}
		if n.Name == "IsNamed" && len(n.Args) == 1 {
			return "card_name"
		}
		if n.Name == "PowerIs" && len(n.Args) == 1 {
			return "power_comparison"
		}
		if (n.Name == "HasAbility" || n.Name == "DoesntHaveAbility") && len(n.Args) == 1 {
			return "keyword"
		}
		if (n.Name == "IsCreatureType" || n.Name == "IsNonCreatureType") && len(n.Args) == 1 {
			return "creature_subtype"
		}
		if n.Name == "Other" && len(n.Args) == 1 {
			return "permanent_reference"
		}
		if n.Name == "ControlledByAPlayer" && len(n.Args) == 1 {
			return "player_filter"
		}
		if (n.Name == "And" || n.Name == "Or") && len(n.Args) == 1 {
			return role
		}
	case "activation_event_filter":
		if n.Name == "And" && len(n.Args) == 1 {
			return role
		}
		if n.Name == "AbilityOfAPermanent" && len(n.Args) == 1 {
			return "permanent_filter"
		}
		if n.Name == "AbilityOfPermanent" && len(n.Args) == 1 {
			return "permanent_reference"
		}
	case "event":
		if n.Name == "Or" && len(n.Args) == 1 {
			return role
		}
		if (n.Name == "WhenACreatureBlocksACreature" || n.Name == "WhenACreatureBecomesBlockedByACreature") && len(n.Args) == 2 {
			return "permanent_filter"
		}
		if n.Name == "WhenAPlayerControlsNoPermanents" && len(n.Args) == 2 {
			return []string{"player_filter", "permanent_filter"}[index]
		}
		if n.Name == "WhenAPlayerActivatesAnAbility" && len(n.Args) == 2 {
			return []string{"player_filter", "activation_event_filter"}[index]
		}
		if n.Name == "WhenAPermanentIsPutIntoAPlayersGraveyard" && len(n.Args) == 2 {
			return []string{"permanent_filter", "player_filter"}[index]
		}
		if n.Name == "WhenAPlayerTapsAPermanentForMana" && len(n.Args) == 2 {
			return []string{"player_filter", "permanent_filter"}[index]
		}
		if n.Name == "WhenAPlayerCastsASpell" && len(n.Args) == 2 {
			return []string{"player_filter", "spell_filter"}[index]
		}
		if len(n.Args) == 1 {
			switch n.Name {
			case "WhenACreatureAttacks", "WhenACreatureAttacksAndIsntBlocked", "WhenAPermanentDealsDamage", "WhenAPermanentEntersTheBattlefield", "WhenAPermanentDies", "WhenAPermanentBecomesTapped", "WhenAPermanentIsTappedForMana":
				return "permanent_filter"
			case "AtTheBeginningOfAPlayersUpkeep", "AtTheBeginningOfAPlayersEndStep", "AtTheBeginningOfAPlayersDrawStep":
				return "player_filter"
			}
		}
	case "player_filter":
		if n.Name == "SinglePlayer" && len(n.Args) == 1 {
			return "player_reference"
		}
	}
	if role == "permanent_filter" && n.Name == "SinglePermanent" && len(n.Args) == 1 {
		return "permanent_reference"
	}
	return "unknown"
}
