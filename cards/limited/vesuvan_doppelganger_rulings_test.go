package limited

import (
	"slices"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type doppelgangerRulingsPlayer struct {
	*mage.BasePlayer
	targetChoices    []uuid.UUID
	targetCandidates [][]uuid.UUID
	targetBounds     [][2]int
	permanentChoice  uuid.UUID
	mayCopy          bool
	mayCalls         int
}

func (p *doppelgangerRulingsPlayer) ChooseTargets(possible []uuid.UUID, minimum, maximum int, _ *mage.Game) []uuid.UUID {
	p.targetCandidates = append(p.targetCandidates, slices.Clone(possible))
	p.targetBounds = append(p.targetBounds, [2]int{minimum, maximum})
	if len(p.targetChoices) == 0 {
		return nil
	}
	selected := p.targetChoices[0]
	p.targetChoices = p.targetChoices[1:]
	if !slices.Contains(possible, selected) {
		return nil
	}
	return []uuid.UUID{selected}
}

func (p *doppelgangerRulingsPlayer) ChoosePermanent(candidates []*mage.Permanent, _ string, _ mage.GameReader) *mage.Permanent {
	for _, candidate := range candidates {
		if candidate.ID() == p.permanentChoice {
			return candidate
		}
	}
	return nil
}

func (p *doppelgangerRulingsPlayer) ChooseMayAbility(_ string) bool {
	p.mayCalls++
	return p.mayCopy
}

func newDoppelgangerRulingsGame() (*mage.Game, *doppelgangerRulingsPlayer) {
	player := &doppelgangerRulingsPlayer{BasePlayer: mage.NewBasePlayer("A"), mayCopy: true}
	return mage.NewGame(player, mage.NewBasePlayer("B")), player
}

func doppelgangerRulingsCard(t *testing.T, owner uuid.UUID) *mage.Card {
	t.Helper()
	card, err := mage.CreateCard("Vesuvan Doppelganger")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(owner)
	return card
}

func doppelgangerRulingsCreature(g *mage.Game, player *doppelgangerRulingsPlayer, card *mage.Card) *mage.Permanent {
	card.SetOwner(player.PlayerID())
	return g.PutOnBattlefield(card, player.PlayerID())
}

// Choose the initial model before tests check changes during upkeep.
func doppelgangerRulingsCopy(t *testing.T, g *mage.Game, player *doppelgangerRulingsPlayer, model *mage.Permanent) *mage.Permanent {
	t.Helper()
	player.permanentChoice = model.ID()
	source := g.PutOnBattlefield(doppelgangerRulingsCard(t, player.PlayerID()), player.PlayerID())
	player.permanentChoice = uuid.Nil()
	expectedName := model.Name()
	if model.FaceDown {
		expectedName = ""
	}
	if source.Name() != expectedName {
		t.Fatalf("entry copy = %q, want %q", source.Name(), expectedName)
	}
	player.mayCalls = 0
	return source
}

func doppelgangerRulingsUpkeep(g *mage.Game, player *doppelgangerRulingsPlayer) {
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: player.PlayerID()})
	g.PutTriggersOnStack()
}

func doppelgangerRulingsUpkeepCount(permanent *mage.Permanent) int {
	count := 0
	for _, ability := range permanent.RuntimeAbilities {
		if trigger, ok := mage.UnwrapAbility(ability).(mage.TriggeredAbility); ok && trigger.CheckEventType(core.EvtUpkeep) {
			count++
		}
	}
	return count
}

func TestVesuvanDoppelgangerRulingsEntryIsUntargeted(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	model := doppelgangerRulingsCreature(g, player, mage.NewCreature("Untargetable model", "{G}", 3, 4, mage.WithKeyword(core.Shroud)))
	player.permanentChoice = model.ID()
	card := doppelgangerRulingsCard(t, player.PlayerID())
	g.ResolveStackObject(&mage.StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: player.PlayerID()})
	copied := g.FindPermanent(card.ID())
	if copied == nil {
		t.Fatal("Doppelganger did not enter as a copy of the chosen creature with shroud")
	}
	if copied.Name() != model.Name() || copied.CurrentPower(g) != 3 {
		t.Errorf("entry copy = %q %d power, want model with 3 power", copied.Name(), copied.CurrentPower(g))
	}
	if len(player.targetCandidates) != 0 {
		t.Error("entry copy requested a target")
	}
}

func TestVesuvanDoppelgangerRulingsEntryMayBeDeclined(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	doppelgangerRulingsCreature(g, player, mage.NewCreature("Model", "{G}", 3, 4))
	player.mayCopy = false
	source := g.PutOnBattlefield(doppelgangerRulingsCard(t, player.PlayerID()), player.PlayerID())
	if source.Name() != "Vesuvan Doppelganger" || source.CurrentPower(g) != 0 {
		t.Error("declined entry copy changed characteristics")
	}
	if count := doppelgangerRulingsUpkeepCount(source); count != 0 {
		t.Errorf("declined entry granted %d upkeep abilities, want 0", count)
	}
}

func TestVesuvanDoppelgangerRulingsCharacteristics(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	model := doppelgangerRulingsCreature(g, player, mage.NewCreature("Elf model", "{2}{G}", 3, 4, mage.WithSubTypes("Elf")))
	copied := doppelgangerRulingsCopy(t, g, player, model)
	if copied.Name() != model.Name() || copied.CurrentPower(g) != 3 || copied.CurrentToughness(g) != 4 {
		t.Error("copy did not acquire the model's name and power/toughness")
	}
	if copied.HasSubType("Shapeshifter") || !copied.HasSubType("Elf") {
		t.Error("copy did not replace Shapeshifter with Elf")
	}
	if !slices.Equal(copied.Colors(), []core.Color{core.Blue}) {
		t.Errorf("copy colors = %v, want blue", copied.Colors())
	}
	if copied.Card.ManaCost().String() != model.Card.ManaCost().String() {
		t.Errorf("copy mana cost = %v, want %v", copied.Card.ManaCost(), model.Card.ManaCost())
	}
}

func TestVesuvanDoppelgangerRulingsChangingFormDoesNotLeave(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	first := doppelgangerRulingsCreature(g, player, mage.NewCreature("First", "{G}", 3, 4))
	second := doppelgangerRulingsCreature(g, player, mage.NewCreature("Second", "{R}", 4, 5))
	copied := doppelgangerRulingsCopy(t, g, player, first)
	id := copied.ID()
	watcher := mage.NewCreature("Leave watcher", "{0}", 1, 1, mage.WithAbility(mage.NewTriggered(core.EvtZoneChange, false, mage.GainLife(1)).SetCondition(func(event *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
		return event.SourceID == id && event.FromZone == core.ZoneBattlefield
	})))
	doppelgangerRulingsCreature(g, player, watcher)
	life := player.Life()
	copied.Damage = 1
	copied.Tapped = true
	g.UpdateCopyEffect(id, second)
	g.ApplyContinuousEffects()
	g.ResolveStack()
	if copied.Damage != 1 {
		t.Error("form change removed damage")
	}
	current := g.FindPermanent(id)
	if current == nil {
		t.Fatal("form change removed the permanent")
	}
	if !current.Tapped {
		t.Error("form change untapped the permanent")
	}
	if player.Life() != life {
		t.Error("form change triggered a leave ability")
	}
	g.BouncePermanentToHand(copied)
	g.ResolveStack()
	if player.Life() != life+1 {
		t.Errorf("actual departure gained %d life, want 1", player.Life()-life)
	}
}

func TestVesuvanDoppelgangerRulingsTextChangeAppliesToNewForm(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	first := doppelgangerRulingsCreature(g, player, mage.NewCreature("First walker", "{B}", 3, 4, mage.WithKeyword(core.Swampwalk)))
	second := doppelgangerRulingsCreature(g, player, mage.NewCreature("Second walker", "{B}", 4, 5, mage.WithKeyword(core.Swampwalk)))
	copied := doppelgangerRulingsCopy(t, g, player, first)
	g.AddContinuousEffect(mage.KeywordReplacement(copied.ID(), core.Swampwalk, core.Forestwalk))
	g.UpdateCopyEffect(copied.ID(), second)
	g.ApplyContinuousEffects()
	if copied.HasKeyword(core.Swampwalk) || !copied.HasKeyword(core.Forestwalk) {
		t.Error("existing word replacement did not apply to the new form")
	}
}

func TestVesuvanDoppelgangerRulingsTargetChosenBeforeResolution(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	first := doppelgangerRulingsCreature(g, player, mage.NewCreature("First", "{G}", 3, 4))
	second := doppelgangerRulingsCreature(g, player, mage.NewCreature("Second", "{R}", 4, 5))
	copied := doppelgangerRulingsCopy(t, g, player, first)
	player.targetChoices = []uuid.UUID{second.ID()}
	doppelgangerRulingsUpkeep(g, player)
	obj := g.Stack().Peek()
	if obj == nil {
		t.Fatal("upkeep did not put an ability on the stack")
	}
	if !slices.Equal(obj.Targets, []uuid.UUID{second.ID()}) {
		t.Errorf("stack targets = %v, want chosen creature %s", obj.Targets, second.ID())
	}
	if len(player.targetBounds) != 1 || player.targetBounds[0] != [2]int{1, 1} {
		t.Error("upkeep did not require exactly one target when placed on the stack")
	}
	if player.mayCalls != 0 {
		t.Error("copy decision was requested before resolution")
	}
	if copied.Name() != first.Name() {
		t.Error("copy changed before resolution")
	}
}

func TestVesuvanDoppelgangerRulingsCopyMayBeDeclinedAtResolution(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	first := doppelgangerRulingsCreature(g, player, mage.NewCreature("First", "{G}", 3, 4))
	second := doppelgangerRulingsCreature(g, player, mage.NewCreature("Second", "{R}", 4, 5))
	copied := doppelgangerRulingsCopy(t, g, player, first)
	player.targetChoices = []uuid.UUID{second.ID()}
	doppelgangerRulingsUpkeep(g, player)
	player.mayCopy = false
	g.ResolveTopOfStack()
	g.ApplyContinuousEffects()
	if player.mayCalls != 1 {
		t.Errorf("resolution requested %d copy decisions, want 1", player.mayCalls)
	}
	if copied.Name() != first.Name() {
		t.Errorf("declined copy became %q, want %q", copied.Name(), first.Name())
	}
}

func TestVesuvanDoppelgangerRulingsCanChooseCurrentModel(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	model := doppelgangerRulingsCreature(g, player, mage.NewCreature("Model", "{G}", 3, 4))
	copied := doppelgangerRulingsCopy(t, g, player, model)
	model.Card.SetBasePT(5, 6)
	player.targetChoices = []uuid.UUID{model.ID()}
	doppelgangerRulingsUpkeep(g, player)
	obj := g.Stack().Peek()
	if obj == nil || !slices.Equal(obj.Targets, []uuid.UUID{model.ID()}) {
		t.Fatal("upkeep did not allow the current model as its target")
	}
	if len(player.targetCandidates) != 1 {
		t.Fatal("upkeep did not request a target")
	}
	if !slices.Contains(player.targetCandidates[0], model.ID()) {
		t.Error("current model was excluded from legal targets")
	}
	g.ResolveTopOfStack()
	g.ApplyContinuousEffects()
	if copied.Name() != model.Name() || copied.CurrentPower(g) != 5 || copied.CurrentToughness(g) != 6 {
		t.Error("copying the current model did not refresh its copiable characteristics")
	}
}

func TestVesuvanDoppelgangerRulingsCanTargetSelf(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	model := doppelgangerRulingsCreature(g, player, mage.NewCreature("Model", "{G}", 3, 4))
	copied := doppelgangerRulingsCopy(t, g, player, model)
	player.targetChoices = []uuid.UUID{copied.ID()}
	doppelgangerRulingsUpkeep(g, player)
	obj := g.Stack().Peek()
	if obj == nil || !slices.Equal(obj.Targets, []uuid.UUID{copied.ID()}) {
		t.Fatal("upkeep did not allow Doppelganger to target itself")
	}
	g.ResolveTopOfStack()
	g.ApplyContinuousEffects()
	if count := doppelgangerRulingsUpkeepCount(copied); count != 2 {
		t.Errorf("self copy has %d upkeep abilities, want 2", count)
	}
}

func TestVesuvanDoppelgangerRulingsIllegalTargetDoesNotSelectAnother(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	first := doppelgangerRulingsCreature(g, player, mage.NewCreature("First", "{G}", 3, 4))
	second := doppelgangerRulingsCreature(g, player, mage.NewCreature("Second", "{R}", 4, 5))
	doppelgangerRulingsCreature(g, player, mage.NewCreature("Alternative", "{B}", 5, 6))
	copied := doppelgangerRulingsCopy(t, g, player, first)
	player.targetChoices = []uuid.UUID{second.ID()}
	doppelgangerRulingsUpkeep(g, player)
	g.BouncePermanentToHand(second)
	g.ResolveTopOfStack()
	g.ApplyContinuousEffects()
	if copied.Name() != first.Name() {
		t.Errorf("ability with an illegal target selected %q instead", copied.Name())
	}
}

func TestVesuvanDoppelgangerRulingsFaceDownModel(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	model := doppelgangerRulingsCreature(g, player, mage.NewCreature("Hidden", "{G}", 7, 8, mage.WithSubTypes("Elf"), mage.WithKeyword(core.Flying)))
	model.FaceDown = true
	copied := doppelgangerRulingsCopy(t, g, player, model)
	if copied.FaceDown {
		t.Error("copy is face down")
	}
	if copied.Name() != "" {
		t.Errorf("face-down model copied hidden name %q", copied.Name())
	}
	if copied.CurrentPower(g) != 2 || copied.CurrentToughness(g) != 2 {
		t.Errorf("face-down copy is %d/%d, want 2/2", copied.CurrentPower(g), copied.CurrentToughness(g))
	}
	if !slices.Equal(copied.Colors(), []core.Color{core.Blue}) {
		t.Error("face-down copy is not blue")
	}
	if !copied.Card.ManaCost().IsZero() {
		t.Error("face-down copy retained the hidden mana cost")
	}
	if len(copied.Card.SubTypes()) != 0 || copied.HasKeyword(core.Flying) || len(copied.RuntimeAbilities) != 1 || doppelgangerRulingsUpkeepCount(copied) != 1 {
		t.Error("face-down copy retained hidden types or abilities")
	}
}

func TestVesuvanDoppelgangerRulingsCopiedDoppelganger(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	model := doppelgangerRulingsCreature(g, player, mage.NewCreature("Model", "{G}", 3, 4, mage.WithSubTypes("Elf")))
	source := doppelgangerRulingsCopy(t, g, player, model)
	card := mage.NewCreature("Clone", "{3}{U}", 0, 0)
	card.SetOwner(player.PlayerID())
	g.Resolution().SetResolvingCard(card)
	if err := mage.CloneTarget().Apply(&mage.EffectContext{Game: g, Controller: player.PlayerID(), Targets: []uuid.UUID{source.ID()}}); err != nil {
		t.Fatal(err)
	}
	g.Resolution().SetResolvingCard(nil)
	copied := g.PutOnBattlefield(card, player.PlayerID())
	if copied.Name() != model.Name() || copied.CurrentPower(g) != 3 || !copied.HasSubType("Elf") {
		t.Error("another copy did not acquire Doppelganger's copied characteristics")
	}
	if !slices.Equal(copied.Colors(), []core.Color{core.Blue}) {
		t.Error("another copy did not acquire Doppelganger's blue color")
	}
	if doppelgangerRulingsUpkeepCount(copied) != 1 {
		t.Error("another copy did not acquire the upkeep ability")
	}
	t.Run("inherited_ability_preserves_color_and_itself", func(t *testing.T) {
		next := doppelgangerRulingsCreature(g, player, mage.NewCreature("Next", "{R}", 4, 5))
		g.BouncePermanentToHand(source)
		player.targetChoices = []uuid.UUID{next.ID()}
		doppelgangerRulingsUpkeep(g, player)
		g.ResolveTopOfStack()
		g.ApplyContinuousEffects()
		if copied.Name() != next.Name() {
			t.Errorf("inherited upkeep copied %q, want selected %q", copied.Name(), next.Name())
		}
		if !slices.Equal(copied.Colors(), []core.Color{core.Blue}) {
			t.Errorf("inherited copy changed color to %v", copied.Colors())
		}
		if doppelgangerRulingsUpkeepCount(copied) != 1 {
			t.Error("inherited copy lost its upkeep ability")
		}
	})
}

func TestVesuvanDoppelgangerRulingsMultipleUpkeepAbilities(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	first := doppelgangerRulingsCreature(g, player, mage.NewCreature("First", "{G}", 3, 4))
	second := doppelgangerRulingsCreature(g, player, mage.NewCreature("Second", "{R}", 4, 5))
	copied := doppelgangerRulingsCopy(t, g, player, first)
	g.UpdateCopyEffect(copied.ID(), copied)
	g.ApplyContinuousEffects()
	if count := doppelgangerRulingsUpkeepCount(copied); count != 2 {
		t.Fatalf("self copy has %d upkeep abilities, want 2", count)
	}
	player.targetChoices = []uuid.UUID{first.ID(), second.ID()}
	doppelgangerRulingsUpkeep(g, player)
	if size := g.Stack().Size(); size != 2 {
		t.Fatalf("stack has %d upkeep abilities, want 2", size)
	}
	objects := g.Stack().Objects()
	if len(objects[0].Targets) != 1 || len(objects[1].Targets) != 1 {
		t.Fatal("each upkeep ability must have its own target")
	}
	topTarget := g.FindPermanent(g.Stack().Peek().Targets[0])
	lastTarget := g.FindPermanent(objects[0].Targets[0])
	g.ResolveTopOfStack()
	g.ApplyContinuousEffects()
	if copied.Name() != topTarget.Name() || g.Stack().Size() != 1 {
		t.Error("first ability did not resolve separately")
	}
	g.ResolveTopOfStack()
	g.ApplyContinuousEffects()
	if copied.Name() != lastTarget.Name() || g.Stack().Size() != 0 {
		t.Error("last ability did not determine the final form")
	}
}

func TestVesuvanDoppelgangerRulingsFlipCards(t *testing.T) {
	for _, modelFlipped := range []bool{false, true} {
		for _, sourceFlipped := range []bool{false, true} {
			name := "unflipped_model"
			if modelFlipped {
				name = "flipped_model"
			}
			if sourceFlipped {
				name += "/flipped_copy"
			} else {
				name += "/unflipped_copy"
			}
			t.Run(name, func(t *testing.T) {
				g, player := newDoppelgangerRulingsGame()
				condition := func(_ *core.GameEvent, g mage.GameReader, source, _ uuid.UUID) bool {
					permanent := g.FindPermanent(source)
					return permanent != nil && permanent.StoredValue == 1
				}
				front := mage.NewCreature("Flip front", "{2}{B}", 2, 4,
					mage.WithAbility(mage.NewStateTriggered(false, mage.FlipSource()).SetCondition(condition)),
					mage.WithFlipFace(mage.NewCreature("Flip back", "{R}", 5, 6)))
				model := doppelgangerRulingsCreature(g, player, front)
				if modelFlipped {
					g.FlipPermanent(model.ID())
				}
				plain := doppelgangerRulingsCreature(g, player, mage.NewCreature("Initial model", "{G}", 3, 4))
				copied := doppelgangerRulingsCopy(t, g, player, plain)
				if sourceFlipped {
					g.FlipPermanent(copied.ID())
				}
				g.UpdateCopyEffect(copied.ID(), model)
				if copied.Flipped != sourceFlipped {
					t.Error("copy changed flipped status")
				}
				if !slices.Equal(copied.Colors(), []core.Color{core.Blue}) {
					t.Error("flip copy did not retain blue")
				}
				if copied.Card.ManaCost().String() != "{2}{B}" {
					t.Errorf("flip copy mana cost = %s, want {2}{B}", copied.Card.ManaCost().String())
				}
				expectedName, expectedPower := "Flip front", 2
				if sourceFlipped {
					expectedName, expectedPower = "Flip back", 5
				}
				if copied.Name() != expectedName || copied.CurrentPower(g) != expectedPower {
					t.Fatalf("copy is %q with %d power, want %q with %d power", copied.Name(), copied.CurrentPower(g), expectedName, expectedPower)
				}
				if !sourceFlipped {
					copied.StoredValue = 1
					g.CheckStateTriggers()
					g.ResolveStack()
					if !copied.Flipped || copied.Name() != "Flip back" || copied.CurrentPower(g) != 5 {
						t.Error("copied flip condition did not select the copied back face")
					}
				}
				if doppelgangerRulingsUpkeepCount(copied) != 1 {
					t.Error("flipping removed the added upkeep ability")
				}
			})
		}
	}
}

func TestVesuvanDoppelgangerRulingsTextChangeOnlyAppliesToMatchingText(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	first := doppelgangerRulingsCreature(g, player, mage.NewCreature("Walker", "{B}", 3, 4, mage.WithKeyword(core.Swampwalk)))
	second := doppelgangerRulingsCreature(g, player, mage.NewCreature("Nonwalker", "{R}", 4, 5))
	copied := doppelgangerRulingsCopy(t, g, player, first)
	g.AddContinuousEffect(mage.KeywordReplacement(copied.ID(), core.Swampwalk, core.Forestwalk))
	g.UpdateCopyEffect(copied.ID(), second)
	g.ApplyContinuousEffects()
	if copied.HasKeyword(core.Forestwalk) || copied.HasKeyword(core.Swampwalk) {
		t.Error("word replacement granted landwalk to a form without matching rules text")
	}
}

func TestVesuvanDoppelgangerRulingsCopyingAnotherDoppelgangerAddsAnAbility(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	model := doppelgangerRulingsCreature(g, player, mage.NewCreature("Model", "{G}", 3, 4))
	first := doppelgangerRulingsCopy(t, g, player, model)
	second := doppelgangerRulingsCopy(t, g, player, model)
	g.UpdateCopyEffect(second.ID(), first)
	g.ApplyContinuousEffects()
	if count := doppelgangerRulingsUpkeepCount(first); count != 1 {
		t.Errorf("model Doppelganger has %d upkeep abilities, want 1", count)
	}
	if count := doppelgangerRulingsUpkeepCount(second); count != 2 {
		t.Errorf("copy of another Doppelganger has %d upkeep abilities, want 2", count)
	}
	g.ApplyContinuousEffects()
	if count := doppelgangerRulingsUpkeepCount(second); count != 2 {
		t.Errorf("reapplication changed the number of upkeep abilities to %d", count)
	}
	doppelgangerRulingsUpkeep(g, player)
	if size := g.Stack().Size(); size != 3 {
		t.Errorf("two Doppelgangers produced %d upkeep triggers, want 3", size)
	}
}

func TestVesuvanDoppelgangerRulingsUpkeepCannotTargetShroud(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	model := doppelgangerRulingsCreature(g, player, mage.NewCreature("Model", "{G}", 3, 4))
	hidden := doppelgangerRulingsCreature(g, player, mage.NewCreature("Untargetable", "{R}", 4, 5, mage.WithKeyword(core.Shroud)))
	copied := doppelgangerRulingsCopy(t, g, player, model)
	player.targetChoices = []uuid.UUID{model.ID()}
	doppelgangerRulingsUpkeep(g, player)
	if len(player.targetCandidates) != 1 {
		t.Fatal("upkeep did not request its required target")
	}
	possible := player.targetCandidates[0]
	if slices.Contains(possible, hidden.ID()) {
		t.Error("target choices include a creature with shroud")
	}
	if !slices.Contains(possible, model.ID()) || !slices.Contains(possible, copied.ID()) {
		t.Error("target choices exclude the current model or Doppelganger itself")
	}
}

func TestVesuvanDoppelgangerRulingsReturnedTargetIsANewObject(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	first := doppelgangerRulingsCreature(g, player, mage.NewCreature("First", "{G}", 3, 4))
	second := doppelgangerRulingsCreature(g, player, mage.NewCreature("Second", "{R}", 4, 5))
	copied := doppelgangerRulingsCopy(t, g, player, first)
	player.targetChoices = []uuid.UUID{second.ID()}
	doppelgangerRulingsUpkeep(g, player)
	g.BouncePermanentToHand(second)
	card, ok := player.RemoveFromHand(second.ID())
	if !ok {
		t.Fatal("bounced target is not in hand")
	}
	g.PutOnBattlefield(card, player.PlayerID())
	g.ResolveTopOfStack()
	g.ApplyContinuousEffects()
	if copied.Name() != first.Name() {
		t.Error("upkeep copied a new object that returned with the target's card ID")
	}
}

func TestVesuvanDoppelgangerRulingsReturnedSourceIsANewObject(t *testing.T) {
	for _, pending := range []bool{false, true} {
		name := "on_stack"
		if pending {
			name = "pending"
		}
		t.Run(name, func(t *testing.T) {
			g, player := newDoppelgangerRulingsGame()
			first := doppelgangerRulingsCreature(g, player, mage.NewCreature("First", "{G}", 3, 4))
			second := doppelgangerRulingsCreature(g, player, mage.NewCreature("Second", "{R}", 4, 5))
			copied := doppelgangerRulingsCopy(t, g, player, first)
			player.targetChoices = []uuid.UUID{second.ID()}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: player.PlayerID()})
			if !pending {
				g.PutTriggersOnStack()
			}
			g.BouncePermanentToHand(copied)
			card, ok := player.RemoveFromHand(copied.ID())
			if !ok {
				t.Fatal("bounced source is not in hand")
			}
			player.permanentChoice = first.ID()
			returned := g.PutOnBattlefield(card, player.PlayerID())
			g.PutTriggersOnStack()
			g.ResolveTopOfStack()
			g.ApplyContinuousEffects()
			if returned.Name() != first.Name() {
				t.Error("old upkeep ability changed the source's new incarnation")
			}
		})
	}
}

func TestVesuvanDoppelgangerRulingsTargetEvents(t *testing.T) {
	t.Run("entry_does_not_target", func(t *testing.T) {
		g, player := newDoppelgangerRulingsGame()
		watched := doppelgangerRulingsCreature(g, player, mage.NewCreature("Watched", "{G}", 3, 4, mage.WithAbility(mage.WheneverBecomesTargetTrigger(mage.GainLife(1), false))))
		life := player.Life()
		doppelgangerRulingsCopy(t, g, player, watched)
		g.ResolveStack()
		if player.Life() != life {
			t.Error("entry copy caused a becomes-target trigger")
		}
	})
	t.Run("upkeep_targets_even_when_copy_is_declined", func(t *testing.T) {
		g, player := newDoppelgangerRulingsGame()
		model := doppelgangerRulingsCreature(g, player, mage.NewCreature("Model", "{G}", 3, 4))
		watched := doppelgangerRulingsCreature(g, player, mage.NewCreature("Watched", "{R}", 4, 5, mage.WithAbility(mage.WheneverBecomesTargetTrigger(mage.GainLife(1), false))))
		copied := doppelgangerRulingsCopy(t, g, player, model)
		life := player.Life()
		player.targetChoices = []uuid.UUID{watched.ID()}
		doppelgangerRulingsUpkeep(g, player)
		if g.Stack().Size() != 2 {
			t.Fatalf("stack contains %d objects, want upkeep and becomes-target triggers", g.Stack().Size())
		}
		g.ResolveTopOfStack()
		if player.Life() != life+1 || copied.Name() != model.Name() {
			t.Error("becomes-target trigger did not resolve before the copy decision")
		}
		player.mayCopy = false
		g.ResolveTopOfStack()
		g.ApplyContinuousEffects()
		if copied.Name() != model.Name() {
			t.Error("declined targeted copy changed form")
		}
	})
}

func TestVesuvanDoppelgangerRulingsNoLegalUpkeepTarget(t *testing.T) {
	g, player := newDoppelgangerRulingsGame()
	model := doppelgangerRulingsCreature(g, player, mage.NewCreature("Model", "{G}", 3, 4))
	copied := doppelgangerRulingsCopy(t, g, player, model)
	g.AddContinuousEffect(mage.TemporaryKeyword(model.ID(), core.Shroud))
	g.AddContinuousEffect(mage.TemporaryKeyword(copied.ID(), core.Shroud))
	doppelgangerRulingsUpkeep(g, player)
	if g.Stack().Size() != 0 {
		t.Error("upkeep ability was put on the stack without a legal required target")
	}
	if player.mayCalls != 0 {
		t.Error("requested a copy decision without a legal target")
	}
}
