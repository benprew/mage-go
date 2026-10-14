package mage

import (
	"fmt"
	"slices"
	"strings"
)

// PermanentTargetDomain stores immutable subtype predicates. Other restrictions
// remain unknown. A zero value is an invalid declaration.
type PermanentTargetDomain struct {
	kind     targetDomainKind
	subtype  string
	children []PermanentTargetDomain
}

type targetDomainKind uint8

const (
	_ targetDomainKind = iota
	unknownTargetDomain
	requiredSubtypeDomain
	excludedSubtypeDomain
	conjunctionTargetDomain
	disjunctionTargetDomain
)

// UnknownPermanentDomain supplies no subtype proof.
func UnknownPermanentDomain() PermanentTargetDomain {
	return PermanentTargetDomain{kind: unknownTargetDomain}
}

// SubtypeDomain requires a subtype.
func SubtypeDomain(subtype string) PermanentTargetDomain {
	return PermanentTargetDomain{kind: requiredSubtypeDomain, subtype: subtype}
}

// NonSubtypeDomain excludes a subtype.
func NonSubtypeDomain(subtype string) PermanentTargetDomain {
	return PermanentTargetDomain{kind: excludedSubtypeDomain, subtype: subtype}
}

// AllPermanentDomains requires every operand.
func AllPermanentDomains(domains ...PermanentTargetDomain) PermanentTargetDomain {
	return PermanentTargetDomain{kind: conjunctionTargetDomain, children: slices.Clone(domains)}
}

// AnyPermanentDomain requires at least one operand.
func AnyPermanentDomain(domains ...PermanentTargetDomain) PermanentTargetDomain {
	return PermanentTargetDomain{kind: disjunctionTargetDomain, children: slices.Clone(domains)}
}

// TargetWithDomain enforces a subtype domain in a permanent target declaration.
func TargetWithDomain(domain PermanentTargetDomain) PermanentTargetOption { return domain }
func (PermanentTargetDomain) permanentTargetOption()                      {}
func (d PermanentTargetDomain) applyPermanentTarget(t *PermanentTarget) {
	t.subtypeDomains = append(slices.Clone(t.subtypeDomains), d)
}
func (d PermanentTargetDomain) inspectDefinition() definitionNode {
	n := definitionNode{Role: "target-domain", Text: d.subtype}
	switch d.kind {
	case unknownTargetDomain:
	case requiredSubtypeDomain, excludedSubtypeDomain:
		if d.subtype == "" || strings.TrimSpace(d.subtype) != d.subtype {
			n.problem = "target subtype requires a nonempty name without outer spaces"
		}
	case conjunctionTargetDomain, disjunctionTargetDomain:
		if len(d.children) == 0 {
			n.problem = "target domain requires operands"
		}
		for i, child := range d.children {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("domains[%d]", i), child})
		}
	default:
		n.problem = "invalid target domain"
	}
	return n
}
func (d PermanentTargetDomain) matches(p *Permanent) bool {
	switch d.kind {
	case unknownTargetDomain:
		return true
	case requiredSubtypeDomain:
		return p.HasSubType(d.subtype)
	case excludedSubtypeDomain:
		return !p.HasSubType(d.subtype)
	case conjunctionTargetDomain:
		return !slices.ContainsFunc(d.children, func(c PermanentTargetDomain) bool { return !c.matches(p) })
	case disjunctionTargetDomain:
		return slices.ContainsFunc(d.children, func(c PermanentTargetDomain) bool { return c.matches(p) })
	}
	return false
}
func (d PermanentTargetDomain) same(other PermanentTargetDomain) bool {
	if d.kind != other.kind || d.subtype != other.subtype || len(d.children) != len(other.children) {
		return false
	}
	for i, child := range d.children {
		if !child.same(other.children[i]) {
			return false
		}
	}
	return true
}
func sameTargetDomains(a, b []PermanentTargetDomain) bool {
	if len(a) != len(b) {
		return false
	}
	for i, d := range a {
		if !d.same(b[i]) {
			return false
		}
	}
	return true
}

// TargetRequiresSubtype proves that a declaration can select only that subtype.
// Unknown predicates cannot supply proof. Subtype contradictions supply no
// proof. Counts, choices and current game state do not affect this query.
func TargetRequiresSubtype(target Target, subtype string) bool {
	if subtype == "" {
		return false
	}
	spec := permanentDomainTarget(target)
	if spec == nil {
		return false
	}
	domains := slices.Clone(spec.subtypeDomains)
	for _, required := range spec.requiredSubtypes {
		domains = append(domains, SubtypeDomain(required))
	}
	if len(domains) == 0 {
		return false
	}
	domain := AllPermanentDomains(domains...)
	return domain.satisfiable(map[string]bool{subtype: true}) && !domain.satisfiable(map[string]bool{subtype: false})
}
func permanentDomainTarget(target Target) *PermanentTarget {
	switch t := target.(type) {
	case *PermanentTarget:
		return t
	case *targetDeclaration:
		if t != nil {
			return permanentDomainTarget(t.spec)
		}
	case *countBoundTarget:
		if t != nil {
			return permanentDomainTarget(t.inner)
		}
	case *randomTarget:
		if t != nil {
			return permanentDomainTarget(t.inner)
		}
	case *opponentChosenTarget:
		if t != nil {
			return permanentDomainTarget(t.inner)
		}
	case TargetBinding:
		if !nilDefinition(t) {
			return permanentDomainTarget(t.targetSpec())
		}
	}
	return nil
}

// Opaque predicates supply no subtype constraints. This prevents them from
// supplying proof, while subtype contradictions still remove impossible paths.
func (d PermanentTargetDomain) satisfiable(values map[string]bool) bool {
	result, unknown := d.evaluate(values)
	if unknown == "" {
		return result
	}
	values[unknown] = true
	if d.satisfiable(values) {
		delete(values, unknown)
		return true
	}
	values[unknown] = false
	result = d.satisfiable(values)
	delete(values, unknown)
	return result
}
func (d PermanentTargetDomain) evaluate(values map[string]bool) (matched bool, unassignedSubtype string) {
	switch d.kind {
	case unknownTargetDomain:
		return true, ""
	case requiredSubtypeDomain, excludedSubtypeDomain:
		value, known := values[d.subtype]
		if !known {
			return false, d.subtype
		}
		return value == (d.kind == requiredSubtypeDomain), ""
	case conjunctionTargetDomain, disjunctionTargetDomain:
		var unknown string
		for _, child := range d.children {
			value, name := child.evaluate(values)
			if name != "" {
				unknown = name
				continue
			}
			if d.kind == conjunctionTargetDomain && !value {
				return false, ""
			}
			if d.kind == disjunctionTargetDomain && value {
				return true, ""
			}
		}
		return d.kind == conjunctionTargetDomain, unknown
	}
	return false, ""
}
