package mage

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed original_names_arn.txt
var originalArabianNamesText string

//go:embed original_names_atq.txt
var originalAntiquitiesNamesText string

var originalArabianNames = &CardNameCatalog{names: strings.Split(strings.TrimSuffix(originalArabianNamesText, "\n"), "\n")}
var originalAntiquitiesNames = &CardNameCatalog{names: strings.Split(strings.TrimSuffix(originalAntiquitiesNamesText, "\n"), "\n")}

// OriginalPrintingNameCatalog supplies the fixed names from CR 206.3a and 206.3b.
// The lists do not depend on physical printings or the active card registry.
func OriginalPrintingNameCatalog(setCode string) (*CardNameCatalog, error) {
	switch setCode {
	case "ARN":
		return originalArabianNames, nil
	case "ATQ":
		return originalAntiquitiesNames, nil
	default:
		return nil, fmt.Errorf("no original-printing name list for %q", setCode)
	}
}

// Matches checks supplied English names, including interchangeable names.
// A missing catalog or a nameless object cannot match.
func (c *CardNameCatalog) Matches(names []string) bool {
	if c == nil {
		return false
	}
	for _, name := range names {
		if name == "" {
			continue
		}
		if c.Contains(name) {
			return true
		}
		group, known := oracleNameEquivalences[name]
		if known {
			for alias := range strings.SplitSeq(group, "\t") {
				if c.Contains(alias) {
					return true
				}
			}
		}
	}
	return false
}

// HasNameIn checks a permanent's current name against a retained name catalog.
// Face-down permanents have no name. Tokens retain their names.
func HasNameIn(catalog *CardNameCatalog) PermanentFilter {
	return NewPermanentFilter("with a name in the supplied catalog", func(p *Permanent, _ *Game) bool {
		return p != nil && !p.FaceDown && catalog.Matches([]string{p.Name()})
	})
}
