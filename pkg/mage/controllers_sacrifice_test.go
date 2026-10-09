package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type sacrificeBatchOwner struct {
	*BasePlayer
	check func()
	calls int
}

func (p *sacrificeBatchOwner) ChooseCardFromLibrary(cards []Card, _ string, _ GameReader) Card {
	p.calls++
	if p.check != nil {
		p.check()
	}
	return cards[len(cards)-1]
}

func TestControllersSacrificeBatch(t *testing.T) {
	a := &sacrificeBatchOwner{BasePlayer: NewBasePlayer("A")}
	b := &sacrificeBatchOwner{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	var subjects []*Permanent
	for i := range 4 {
		owner, controller := Player(a), Player(b)
		if i >= 2 {
			owner, controller = b, a
		}
		card := NewCreature("Subject", "", 1, 1, WithKeyword(core.Indestructible))
		card.SetOwner(owner.PlayerID())
		p := g.PutOnBattlefield(card, controller.PlayerID())
		g.AddRegenerationShield(p.ID())
		subjects = append(subjects, p)
	}
	var order []uuid.UUID
	for _, owner := range []*sacrificeBatchOwner{a, b} {
		owner.check = func() {
			order = append(order, owner.PlayerID())
			for _, p := range subjects {
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("order choice saw partial movement")
				}
			}
		}
	}
	observed := 0
	watcher := NewLand("Watcher", WithAbility(NewTriggered(core.EvtSacrifice, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		observed++
		for _, p := range subjects {
			if g.FindPermanent(p.ID()) != nil || reader.FindPermanent(p.ID()) == nil {
				t.Fatal("batch event lost live or former battlefield")
			}
		}
		for _, p := range subjects {
			if evt.SourceID == p.ID() && evt.PlayerID != p.ControllerID() {
				t.Fatal("sacrificing player was the owner or effect controller")
			}
		}
		return true
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	clone := g.Clone()
	if err := ControllersSacrificeEach(fixedTapSet{append(subjects, subjects[0])}).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if observed != 4 || a.calls != 1 || b.calls != 1 || len(order) != 2 || order[0] != g.ActivePlayerObj().PlayerID() || order[1] == order[0] {
		t.Fatal("batch choices or events", observed, order)
	}
	if len(a.Graveyard()) != 2 || len(b.Graveyard()) != 2 || a.Graveyard()[0].ID() != subjects[1].ID() || b.Graveyard()[0].ID() != subjects[3].ID() {
		t.Fatal("owner graveyard ordering")
	}
	for _, p := range subjects {
		if clone.FindPermanent(p.ID()) == nil {
			t.Fatal("movement changed clone")
		}
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("sacrifice events did not resolve", a.Life())
	}
}

func TestControllersSacrificeIdentityAndDefinition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Returned", "", 1, 1)
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	phased := g.PutOnBattlefield(NewArtifact("Phased", ""), a.PlayerID())
	g.PhaseOut(phased.ID())
	if err := ControllersSacrificeEach(fixedTapSet{[]*Permanent{old, phased, nil}}).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(current.ID()) == nil || g.FindPermanentIncludingPhased(phased.ID()) == nil {
		t.Fatal("stale or phased subject moved")
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewSorcery("Good", "", Spell(ControllersSacrificeEach(MatchingPermanents(IsArtifact))))},
		{Name: "nil", Card: NewSorcery("Bad", "", Spell(ControllersSacrificeEach(nil))), WantErr: true},
	})
}

func TestControllersSacrificeReplacement(t *testing.T) {
	for _, mode := range []int{0, 1} {
		a := &choiceRecordingPlayer{BasePlayer: NewBasePlayer("A"), chosenMode: mode}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Replaced", "", 1, 1), a.PlayerID())
		other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), b.PlayerID())
		first, second := uuid.New(), uuid.New()
		g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), first)
		g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), second)
		if err := ControllersSacrificeEach(MatchingPermanents(IsCreature)).Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		want := first
		if mode == 1 {
			want = second
		}
		if len(g.GetExile()) != 1 || g.GetExile()[0].ExiledBy != want || g.FindPermanent(other.ID()) != nil || len(b.Graveyard()) != 1 {
			t.Fatal("replacement or independent sacrifice failed")
		}
	}
}

func TestControllersSacrificeDepartingReplacementSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Replacement source", "", 1, 1), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), b.PlayerID())
	g.effects.AddReplacement(&requiredSacrificeSourceReplacement{replacementBase: replacementBase{sourceID: source.ID()}})
	if err := ApplyEffect(g, ControllersSacrificeEach(MatchingPermanents(IsCreature)), source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.GetExile()) != 2 || g.FindPermanent(other.ID()) != nil || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
		t.Fatal("departing replacement source stopped before all replacements")
	}
}

func TestControllersSacrificeWaitingPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	reads := 0
	filter := NewPermanentFilter("query must not select", func(*Permanent, *Game) bool { reads++; return false })
	body := ControllersSacrificeEach(FilterPermanentSet(SingletonPermanent(This), filter))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(source)
	clone := g.Clone()
	for range 3 {
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || !got[0].SameObject(saved) || reads != 0 || len(a.Graveyard()) != 0 {
			t.Fatal("query executed action or lost source")
		}
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("waiting clones shared lifetimes")
	}
}

func TestControllersSacrificeTokenAndSourceTriggers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(LeavesBattlefield(This), GainLifeFor(You, Fixed(3)))))
	g.PutOnBattlefield(card, b.PlayerID())
	token := g.PutOnBattlefield(NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil), a.PlayerID())
	observer := NewLand("State observer", WithAbility(NewStateTriggered(false, GainLifeFor(You, Fixed(10))).SetCondition(func(_ *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		return reader.FindPermanent(token.ID()) == nil && reader.FindPermanent(card.ID()) != nil
	})))
	g.PutOnBattlefield(observer, a.PlayerID())
	if err := ApplyEffect(g, ControllersSacrificeEach(MatchingPermanents(IsCreature)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 23 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 1 || g.FindPermanent(token.ID()) != nil {
		t.Fatal("token, source LKI or intermediate state trigger")
	}
}

type collectionSacrificeReplacement struct {
	replacementBase
	blocked uuid.UUID
	inspect func(Action)
}

func (r *collectionSacrificeReplacement) Matches(action Action, _ GameReader) bool {
	_, ok := action.(*SacrificePermanentAction)
	return ok
}
func (r *collectionSacrificeReplacement) Replace(action Action, _ *Game) Action {
	if r.inspect != nil {
		r.inspect(action)
	}
	if action.(*SacrificePermanentAction).PermanentID() == r.blocked {
		return nil
	}
	return action
}
func (*collectionSacrificeReplacement) IsActive(GameReader) bool   { return true }
func (r *collectionSacrificeReplacement) Clone() ReplacementEffect { cloned := *r; return &cloned }

func TestControllersSacrificePreparationOrderAndCancellation(t *testing.T) {
	for _, active := range []int{0, 1} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.SetActivePlayerIndex(active)
		first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
		second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), b.PlayerID())
		var order []uuid.UUID
		replacement := &collectionSacrificeReplacement{blocked: first.ID(), inspect: func(action Action) {
			if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil {
				t.Fatal("replacement preparation saw partial moves")
			}
			order = append(order, g.FindPermanent(action.(*SacrificePermanentAction).PermanentID()).ControllerID())
		}}
		g.effects.AddReplacement(replacement)
		if err := ApplyEffect(g, ControllersSacrificeEach(fixedTapSet{[]*Permanent{second, first}}), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(order) != 2 || order[0] != g.ActivePlayerObj().PlayerID() || order[1] != g.NonActivePlayerObj().PlayerID() {
			t.Fatal("replacement preparation did not use APNAP", order)
		}
		if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) != nil || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 1 {
			t.Fatal("canceled sacrifice changed another subject")
		}
	}
}
