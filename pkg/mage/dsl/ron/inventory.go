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
		switch n.Name {
		case "SpellActions":
			if len(n.Args) == 1 {
				return "actions"
			}
		case "Activated":
			if len(n.Args) == 2 {
				return []string{"cost", "actions"}[index]
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
	case "actions":
		if n.Name == "Targeted" && len(n.Args) == 2 {
			return []string{"target", "actions"}[index]
		}
		if n.Name == "ActionList" && len(n.Args) == 1 {
			return "action"
		}
	case "target":
		if len(n.Args) == 1 {
			switch n.Name {
			case "TargetPermanent":
				return "permanent_filter"
			case "TargetPlayer":
				return "player_filter"
			case "TargetSpell":
				return "spell_filter"
			}
		}
	case "cost":
		if n.Name == "And" && len(n.Args) == 1 {
			return "cost"
		}
		if (n.Name == "TapPermanent" || n.Name == "SacrificePermanent") && len(n.Args) == 1 {
			return "permanent_reference"
		}
		if n.Name == "PayMana" && len(n.Args) == 1 {
			return "metadata"
		}
	case "action":
		switch n.Name {
		case "TapPermanent", "UntapPermanent", "DestroyPermanent", "DestroyPermanentNoRegen", "RegeneratePermanent", "SacrificePermanent":
			if len(n.Args) == 1 {
				return "permanent_reference"
			}
		case "CounterSpell":
			if len(n.Args) == 1 {
				return "spell_reference"
			}
		case "PlayerAction":
			if len(n.Args) == 2 {
				return []string{"player_reference", "player_action"}[index]
			}
		case "SpellDealsDamage":
			if len(n.Args) == 3 {
				return []string{"spell_reference", "value", "recipient"}[index]
			}
		case "PermanentDealsDamage":
			if len(n.Args) == 3 {
				return []string{"permanent_reference", "value", "recipient"}[index]
			}
		case "DrawNumberCards", "GainLife", "LoseLife":
			if len(n.Args) == 1 {
				return "value"
			}
		}
	case "player_action":
		if (n.Name == "DrawNumberCards" || n.Name == "GainLife" || n.Name == "LoseLife") && len(n.Args) == 1 {
			return "value"
		}
	case "spell_filter":
		if (n.Name == "And" || n.Name == "Or") && len(n.Args) == 1 {
			return role
		}
	case "permanent_filter":
		if (n.Name == "And" || n.Name == "Or") && len(n.Args) == 1 {
			return role
		}
	case "event":
		if len(n.Args) == 1 {
			switch n.Name {
			case "WhenAPermanentEntersTheBattlefield", "WhenAPermanentDies":
				return "permanent_filter"
			case "AtTheBeginningOfAPlayersUpkeep":
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
