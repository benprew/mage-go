package mage

import (
	_ "embed"
	"slices"
	"strings"
)

//go:embed oracle_name_equivalences.txt
var oracleNameEquivalencesText string

var oracleNameEquivalences = loadOracleNameEquivalences()

func loadOracleNameEquivalences() map[string]string {
	groups := make(map[string]string)
	for line := range strings.SplitSeq(strings.TrimSuffix(oracleNameEquivalencesText, "\n"), "\n") {
		for name := range strings.SplitSeq(line, "\t") {
			groups[name] = line
		}
	}
	return groups
}

func loadOracleCardNameCatalog() *CardNameCatalog {
	names := strings.Split(strings.TrimSuffix(oracleCardNamesText, "\n"), "\n")
	for name := range oracleNameEquivalences {
		names = append(names, name)
	}
	slices.Sort(names)
	return &CardNameCatalog{names: slices.Compact(names)}
}

// OracleNamesInterchangeable checks English names with the retained Oracle aliases.
// Empty names cannot match, because a nameless object has no name in common.
func OracleNamesInterchangeable(first, second string) bool {
	if first == "" || second == "" {
		return false
	}
	if first == second {
		return true
	}
	group, known := oracleNameEquivalences[first]
	return known && group == oracleNameEquivalences[second]
}

// CardNameMatcher checks supplied characteristic names in a resolution frame.
// The caller supplies current or retained names and keeps object identity.
type CardNameMatcher interface {
	Match([]string, *EffectContext) bool
	inspectDefinition() definitionNode
}

type chosenCardNameMatcher struct{ name CardNameRef }

// MatchesChosenCardName declares a pure test of a saved name against object names.
// A missing choice cannot match. The test does not select or move an object.
func MatchesChosenCardName(name CardNameRef) CardNameMatcher {
	return chosenCardNameMatcher{name: name}
}

func (m chosenCardNameMatcher) Match(names []string, ctx *EffectContext) bool {
	chosen, ok := m.name.Resolve(ctx)
	if !ok {
		return false
	}
	for _, name := range names {
		if OracleNamesInterchangeable(name, chosen) {
			return true
		}
	}
	return false
}

func (m chosenCardNameMatcher) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-name-matcher", children: []definitionEdge{{"name", m.name}}}
}
