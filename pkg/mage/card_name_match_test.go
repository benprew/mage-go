package mage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type cardNameMatchProbe struct {
	matcher CardNameMatcher
	names   []string
	want    bool
	t       *testing.T
}

func (*cardNameMatchProbe) Text() string                 { return "match a chosen name" }
func (*cardNameMatchProbe) Properties() EffectProperties { return EffectProperties{} }
func (e *cardNameMatchProbe) Apply(ctx *EffectContext) error {
	if got := e.matcher.Match(e.names, ctx); got != e.want {
		e.t.Fatalf("match=%v want=%v names=%v", got, e.want, e.names)
	}
	return nil
}
func (e *cardNameMatchProbe) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"matcher", e.matcher}}}
}
func TestCardNameMatchEquivalence(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"", "", false}, {"Island", "Island", true}, {"Island", "island", false},
		{"Cecily, Haunted Mage", "Eleven, the Mage", true},
		{"Eleven, the Mage", "Cecily, Haunted Mage", true},
		{"Cecily, Haunted Mage", "Sophina, Spearsage Deserter", false},
		{"Godzilla, King of the Monsters", "Zilortha, Strength Incarnate", false},
		{"Havengul Laboratory", "Hawkins National Laboratory", true},
		{"Havengul Mystery", "The Upside Down", true},
		{"Havengul Laboratory", "The Upside Down", false},
	} {
		if got := OracleNamesInterchangeable(tc.a, tc.b); got != tc.want {
			t.Errorf("%q / %q = %v", tc.a, tc.b, got)
		}
	}
}
func TestCardNameMatchContract(t *testing.T) {
	base, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &cardNameChoiceTester{Player: base, name: "Eleven, the Mage"}
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	binding := BindCardName("chosen")
	matcher := MatchesChosenCardName(FromCardName(binding))
	if matcher.Match([]string{"Eleven, the Mage"}, ctx) {
		t.Fatal("missing result matched")
	}
	if err := ChooseCardName(You, OracleCardNameCatalog(), binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	for _, names := range [][]string{{"Cecily, Haunted Mage"}, {"Other", "Cecily, Haunted Mage"}, {"Eleven, the Mage"}} {
		if !matcher.Match(names, ctx) {
			t.Fatalf("names did not match: %v", names)
		}
	}
	for _, names := range [][]string{nil, {""}, {"Other"}} {
		if matcher.Match(names, ctx) {
			t.Fatal("absent or different name matched")
		}
	}
	clone := g.Clone()
	a.name = "Ice"
	if err := ChooseCardName(You, OracleCardNameCatalog(), binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if matcher.Match([]string{"Cecily, Haunted Mage"}, ctx) || !matcher.Match([]string{"Cecily, Haunted Mage"}, &EffectContext{Game: clone}) {
		t.Fatal("cloned results were shared")
	}
	clone.resolution.Clear()
	if matcher.Match([]string{"Cecily, Haunted Mage"}, &EffectContext{Game: clone}) {
		t.Fatal("cleared result matched")
	}
	if a.calls != 2 {
		t.Fatal("matching made a choice")
	}
}
func TestCardNameMatchDeclarations(t *testing.T) {
	binding := BindCardName("chosen")
	probe := &cardNameMatchProbe{matcher: MatchesChosenCardName(FromCardName(binding)), names: []string{"Ice"}, want: true, t: t}
	choice := ChooseCardName(You, OracleCardNameCatalog(), binding)
	for _, effect := range []Effect{probe, Sequence(probe, choice), &cardNameMatchProbe{matcher: MatchesChosenCardName(nil), t: t}, &cardNameMatchProbe{matcher: MatchesChosenCardName(FromCardName(nil)), t: t}} {
		card := NewInstant("Invalid", "", Spell(effect))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	card := NewInstant("Valid", "", Spell(Sequence(choice, probe)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestCardNameMatchPendingAndFrames(t *testing.T) {
	for _, state := range []string{"departed", "returned", "phased"} {
		t.Run(state, func(t *testing.T) {
			base, b := NewBasePlayer("A"), NewBasePlayer("B")
			a := &cardNameChoiceTester{Player: base, name: "Eleven, the Mage"}
			g := NewGame(a, b)
			binding := BindCardName("chosen")
			matcher := MatchesChosenCardName(FromCardName(binding))
			body := Sequence(ChooseCardName(You, OracleCardNameCatalog(), binding), &cardNameMatchProbe{matcher: matcher, names: []string{"Cecily, Haunted Mage"}, want: true, t: t})
			p := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(), body))), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if state == "phased" {
				g.PhaseOut(p.ID())
			} else {
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(p.ID())
					g.PutOnBattlefield(p.Card, a.PlayerID())
				}
			}
			clone := g.Clone()
			ca := &cardNameChoiceTester{Player: clone.GetPlayer(a.PlayerID()), name: "Cecily, Haunted Mage"}
			clone.SetPlayerAt(0, ca)
			if a.calls != 0 || matcher.Match([]string{"Cecily, Haunted Mage"}, &EffectContext{Game: g}) {
				t.Fatal("choice or result before resolution")
			}
			g.ResolveStack()
			clone.ResolveStack()
			if a.calls != 1 || ca.calls != 1 || matcher.Match([]string{"Cecily, Haunted Mage"}, &EffectContext{Game: g}) {
				t.Fatal("pending choice or frame cleanup failed")
			}
		})
	}
	base, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &cardNameChoiceTester{Player: base, name: "Eleven, the Mage"}
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	binding := BindCardName("chosen")
	matcher := MatchesChosenCardName(FromCardName(binding))
	choice := ChooseCardName(You, OracleCardNameCatalog(), binding)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	a.name = "Ice"
	body := Sequence(choice, &cardNameMatchProbe{matcher: matcher, names: []string{"Ice"}, want: true, t: t})
	if err := ApplyEffect(g, body, ctx.SourceID, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(NewArtifact("Member", ""), a.PlayerID())
	if err := ForEach(MatchingPermanents(AnyPermanent), BindPermanent("member"), body).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !matcher.Match([]string{"Cecily, Haunted Mage"}, ctx) {
		t.Fatal("nested or loop result escaped its frame")
	}
}

func TestCardNameMatchDiscardBatch(t *testing.T) {
	for _, mode := range []string{"normal", "library", "prevent"} {
		t.Run(mode, func(t *testing.T) {
			base, b := NewBasePlayer("A"), NewBasePlayer("B")
			a := &cardNameChoiceTester{Player: base, name: "Eleven, the Mage"}
			g := NewGame(a, b)
			first, second := NewArtifact("Cecily, Haunted Mage", ""), NewArtifact("Eleven, the Mage", "")
			for _, card := range []Card{first, second} {
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
			}
			if mode != "normal" {
				g.effects.AddReplacement(&handOperationReplacement{card: second.ID(), prevent: mode == "prevent"})
			}
			binding := BindCardName("chosen")
			body := Sequence(ChooseCardName(You, OracleCardNameCatalog(), binding), DiscardHandMatchingName(You, MatchesChosenCardName(FromCardName(binding))))
			card := NewInstant("Batch", "", Spell(body))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			seen := 0
			watcher := NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				seen++
				if mode == "normal" && (len(a.Hand()) != 0 || len(a.Graveyard()) != 2) {
					t.Error("event observed a partial batch")
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher)), a.PlayerID())
			if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if mode == "normal" {
				if len(a.Graveyard()) != 2 || seen != 2 {
					t.Fatal("discard batch missing")
				}
			} else {
				wantEvents := 2
				if mode == "prevent" {
					wantEvents = 1
				}
				if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != first.ID() || seen != wantEvents {
					t.Fatal("replacement missed initial hand")
				}
				if mode == "library" && (len(a.Hand()) != 0 || len(a.Library()) != 1) {
					t.Fatal("discard redirection failed")
				}
				if mode == "prevent" && (len(a.Hand()) != 1 || a.Hand()[0].ID() != second.ID()) {
					t.Fatal("discard prevention failed")
				}
			}
		})
	}
	for _, body := range []Effect{DiscardHandMatchingName(You, nil), DiscardHandMatchingName(nil, MatchesChosenCardName(FromCardName(BindCardName("name"))))} {
		card := NewInstant("Invalid", "", Spell(body))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid discard declaration accepted")
		}
	}
}

func TestCardNameMatchSnapshot(t *testing.T) {
	var source struct {
		AssetSHA256 string `json:"asset_sha256"`
		SourceCards int    `json:"source_cards"`
		FaceGroups  int    `json:"face_groups"`
	}
	data, err := os.ReadFile("oracle_name_equivalences_source.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(oracleNameEquivalencesText))
	if hex.EncodeToString(digest[:]) != source.AssetSHA256 {
		t.Fatal("snapshot hash does not match source evidence")
	}
	lines := strings.Split(strings.TrimSuffix(oracleNameEquivalencesText, "\n"), "\n")
	if source.SourceCards != 30 || source.FaceGroups != 31 || len(lines) != source.FaceGroups || !slices.IsSorted(lines) {
		t.Fatal("incomplete or unsorted snapshot")
	}
	seen := map[string]bool{}
	for _, line := range lines {
		pair := strings.Split(line, "\t")
		if len(pair) != 2 || !slices.IsSorted(pair) {
			t.Fatal("invalid equivalence group")
		}
		for _, name := range pair {
			if name == "" || seen[name] || !OracleCardNameCatalog().Contains(name) || !OracleNamesInterchangeable(pair[0], name) || !OracleNamesInterchangeable(name, pair[0]) {
				t.Fatalf("invalid or missing name %q", name)
			}
			seen[name] = true
		}
	}
	if len(OracleCardNameCatalog().Names()) != 35950 {
		t.Fatal("supplemented catalog has incorrect size")
	}
}
