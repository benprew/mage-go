package mage

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOriginalPrintingNamesRules(t *testing.T) {
	rules, err := os.ReadFile("../../docs/comprehensive-rules.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		set, rule string
		count     int
	}{{"ARN", "206.3a", 77}, {"ATQ", "206.3b", 85}} {
		t.Run(tc.set, func(t *testing.T) {
			catalog, err := OriginalPrintingNameCatalog(tc.set)
			if err != nil {
				t.Fatal(err)
			}
			var want []string
			for line := range strings.SplitSeq(string(rules), "\n") {
				if strings.HasPrefix(line, tc.rule+" ") {
					_, names, ok := strings.Cut(line, "Those names are ")
					if !ok {
						t.Fatal("missing rules list")
					}
					names = strings.TrimSuffix(names, ".")
					names = strings.ReplaceAll(names, ", and ", ", ")
					names = strings.ReplaceAll(names, "’", "'")
					want = strings.Split(names, ", ")
				}
			}
			slices.Sort(want)
			got := catalog.Names()
			if len(got) != tc.count || !slices.Equal(got, want) {
				t.Fatalf("membership differs from CR %s: %v", tc.rule, got)
			}
			for _, name := range want {
				if !catalog.Matches([]string{name}) {
					t.Fatalf("missing %s", name)
				}
			}
			got[0] = "Changed"
			if slices.Equal(got, catalog.Names()) {
				t.Fatal("names share mutable storage")
			}
			for _, names := range [][]string{nil, {""}, {"Mountain"}, {"Island"}, {"Black Lotus"}} {
				if catalog.Matches(names) {
					t.Fatalf("unlisted name matched: %v", names)
				}
			}
		})
	}
	for _, set := range []string{"", "arn", "LEA", "UNKNOWN"} {
		if catalog, err := OriginalPrintingNameCatalog(set); err == nil || catalog != nil {
			t.Fatal("unsupported catalog accepted", set)
		}
	}
	var missing *CardNameCatalog
	if missing.Matches([]string{"Atog"}) || HasNameIn(missing).Match(NewPermanent(NewArtifact("Atog", ""), uuid.Nil), nil) {
		t.Fatal("missing catalog matched")
	}
	aliasCatalog := &CardNameCatalog{names: []string{"Cecily, Haunted Mage"}}
	if !aliasCatalog.Matches([]string{"Other", "Eleven, the Mage"}) || aliasCatalog.Matches([]string{"Godzilla, King of the Monsters"}) {
		t.Fatal("English-name equivalence")
	}
}

func TestOriginalPrintingNamesCurrentCharacteristics(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	catalog, err := OriginalPrintingNameCatalog("ATQ")
	if err != nil {
		t.Fatal(err)
	}
	filter := HasNameIn(catalog)
	p := g.PutOnBattlefield(NewArtifact("Atog", ""), a.PlayerID())
	token := g.PutOnBattlefield(NewToken("Atog", 1, 1, []core.CardType{core.TypeCreature}, nil), b.PlayerID())
	if !filter.Match(p, g) || !filter.Match(token, g) || filter.Match(nil, g) {
		t.Fatal("name filter used type, controller, or token status")
	}
	clone := g.Clone()
	g.MutablePermanent(p.ID()).FaceDown = true
	if filter.Match(g.FindPermanent(p.ID()), g) || !filter.Match(clone.FindPermanent(p.ID()), clone) {
		t.Fatal("face-down name or clone isolation")
	}
	g.MutablePermanent(p.ID()).FaceDown = false
	other := NewArtifact("Black Lotus", "")
	other.SetID(p.ID())
	other.SetOwner(p.Card.Owner())
	g.MutablePermanent(p.ID()).Card = other
	if filter.Match(g.FindPermanent(p.ID()), g) {
		t.Fatal("filter retained printed name")
	}
	changed := NewArtifact("Ornithopter", "")
	changed.SetID(p.ID())
	changed.SetOwner(p.Card.Owner())
	g.MutablePermanent(p.ID()).Card = changed
	if !filter.Match(g.FindPermanent(p.ID()), g) {
		t.Fatal("filter did not read changed name")
	}
	if !catalog.Matches([]string{"Other", "Atog"}) {
		t.Fatal("additional name failed")
	}
}

func TestOriginalPrintingNamesEntryCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	catalog, err := OriginalPrintingNameCatalog("ATQ")
	if err != nil {
		t.Fatal(err)
	}
	filter := HasNameIn(catalog)
	plan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Atog", "", 1, 1) })
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	values, err := g.CaptureCopiableValues(source)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := BuildEntryCopy(entryCopyPhysical(t), values)
	if err != nil {
		t.Fatal(err)
	}
	subject := g.PutOnBattlefield(copied, b.PlayerID())
	if !filter.Match(subject, g) {
		t.Fatal("copy used physical printed name")
	}
	if g.LKI(subject.ID()) != nil {
		t.Fatal("unexpected departure information")
	}
	g.BouncePermanentToHand(subject)
	lki := g.LKI(subject.ID())
	if lki == nil || !filter.Match(lki.Snapshot, g) {
		t.Fatal("departure lost copied name")
	}
	restored := b.Hand()[0]
	b.RemoveFromHand(restored.ID())
	returned := g.PutOnBattlefield(restored, b.PlayerID())
	if filter.Match(returned, g) {
		t.Fatal("returned physical card kept copied name")
	}
}

func TestOriginalPrintingNamesFaceDownLayerRecheck(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	catalog, err := OriginalPrintingNameCatalog("ATQ")
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(NewCreature("Atog", "", 2, 2), b.PlayerID())
	g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Static(While(Controls(AnyPermanent), ModifyEachPT(MatchingPermanents(HasNameIn(catalog)), Fixed(1), Fixed(1)))))), a.PlayerID())
	if g.FindPermanent(p.ID()).CurrentPower(g) != 3 {
		t.Fatal("missing named layer")
	}
	clone := g.Clone()
	g.MutablePermanent(p.ID()).FaceDown = true
	g.MutablePermanent(p.ID()).BasePTOverride = &[2]int{2, 2}
	for range 3 {
		g.ApplyContinuousEffects()
		if g.FindPermanent(p.ID()).CurrentPower(g) != 2 || g.FindPermanent(p.ID()).BasePTOverride == nil {
			t.Fatal("face-down permanent retained a former name bonus")
		}
	}
	if clone.FindPermanent(p.ID()).CurrentPower(clone) != 3 {
		t.Fatal("layer reset changed clone")
	}
}
