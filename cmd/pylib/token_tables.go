package main

/*
#include <stdlib.h>
#include "abi.h"
*/
import "C" //nolint:gocritic // C and unsafe are distinct despite cgo's synthetic package metadata.

import (
	"fmt"
	"sync"
	"unsafe" //nolint:gocritic // Required for C buffer views; this is not a duplicate C import.
)

// tokenTables is the Go-side mirror of the Python TokenTables wire format.
// Pointers are borrowed from Python; the underlying tensors must outlive any
// code that reads them. Phase 4 of the assembler-port replaces the
// render-plan opcode walker with a dispatch loop that indexes into these
// tables and memcpys int32 token spans into the output buffer.
//
// All fields are kept in their wire shape (flat int32/int64 slices over the
// raw C buffers via unsafe.Slice). Lookups go through accessor helpers that
// resolve (kind, value) -> []int32 token span without copying.
type tokenTables struct {
	// Static structural fragments (Frag enum values).
	structuralTokens  []int32
	structuralOffsets []int32

	turnMin        int32
	turnMax        int32
	stepCount      int32
	turnStepTokens []int32
	turnStepOff    []int32

	lifeMin         int32
	lifeMax         int32
	ownerCount      int32
	lifeOwnerTokens []int32
	lifeOwnerOff    []int32

	abilityMin    int32
	abilityMax    int32
	abilityTokens []int32
	abilityOff    []int32

	countMin    int32
	countMax    int32
	countTokens []int32
	countOff    []int32

	zoneCount      int32
	zoneOpenTokens []int32
	zoneOpenOff    []int32
	zoneCloseTok   []int32
	zoneCloseOff   []int32

	actionVerbCount  int32
	actionVerbTokens []int32
	actionVerbOff    []int32

	manaColorCount int32
	manaTokens     []int32
	manaOff        []int32

	cardRefCount int32
	cardRefIDs   []int32

	padID         int32
	optionID      int32
	targetOpenID  int32
	targetCloseID int32
	tappedID      int32
	untappedID    int32

	cardCloser     []int32
	statusTapped   []int32
	statusUntapped []int32

	cardRowCount int32
	cardBodyToks []int32
	cardBodyOff  []int64
	cardNameToks []int32
	cardNameOff  []int64

	// v2 dedup. dictEntryIDs is one int32 per card row (the
	// ``<dict-entry:R>`` token id) — used by the legacy row-keyed render-plan
	// path. dictSlotIDs is one int32 per per-snapshot slot (the
	// ``<dict-slot:S>`` token id) — used by the direct emitter so the model
	// sees a slot index that's reassigned per snapshot in insertion order
	// rather than a stable per-card identity. NULL/empty when v2 is disabled.
	dictOpenID   int32
	dictCloseID  int32
	cardOpenID   int32
	dictEntryIDs []int32
	dictSlotIDs  []int32

	// Singletons used by the Go emitter for byte-equal parity with the
	// Python emit_render_plan path. ``selfID`` / ``oppID`` are written
	// inside ``<target>...</target>`` blocks for player-target options;
	// ``stack_*`` / ``command_*`` open and close the shared (non-per-
	// player) stack / command zones once per snapshot.
	selfID         int32
	oppID          int32
	stackOpenID    int32
	stackCloseID   int32
	commandOpenID  int32
	commandCloseID int32
}

var (
	tokenTablesMu      sync.RWMutex
	currentTokenTables *tokenTables
)

// sliceI32 wraps a borrowed C int32 buffer as a Go slice. Returns an empty
// slice when ptr is nil or length is non-positive.
func sliceI32(ptr *C.int32_t, length int) []int32 {
	if ptr == nil || length <= 0 {
		return nil
	}
	return unsafe.Slice((*int32)(unsafe.Pointer(ptr)), length)
}

// sliceI64 wraps a borrowed C int64 buffer as a Go slice.
func sliceI64(ptr *C.int64_t, length int) []int64 {
	if ptr == nil || length <= 0 {
		return nil
	}
	return unsafe.Slice((*int64)(unsafe.Pointer(ptr)), length)
}

// validateTablePack ensures an offsets table is consistent with its token
// buffer: monotonically non-decreasing, terminal value matches token length.
func validateTablePack(name string, tokens, offsets []int32, expectedEntries int) error {
	if len(offsets) != expectedEntries+1 {
		return fmt.Errorf("%s offsets length=%d, want %d", name, len(offsets), expectedEntries+1)
	}
	if expectedEntries == 0 {
		return nil
	}
	for i := 1; i < len(offsets); i++ {
		if offsets[i] < offsets[i-1] {
			return fmt.Errorf("%s offsets[%d]=%d < offsets[%d]=%d", name, i, offsets[i], i-1, offsets[i-1])
		}
	}
	if int(offsets[len(offsets)-1]) != len(tokens) {
		return fmt.Errorf("%s offsets[-1]=%d != len(tokens)=%d", name, offsets[len(offsets)-1], len(tokens))
	}
	return nil
}

func validateTablePack64(name string, tokens []int32, offsets []int64, expectedEntries int) error {
	if len(offsets) != expectedEntries+1 {
		return fmt.Errorf("%s offsets length=%d, want %d", name, len(offsets), expectedEntries+1)
	}
	if expectedEntries == 0 {
		return nil
	}
	for i := 1; i < len(offsets); i++ {
		if offsets[i] < offsets[i-1] {
			return fmt.Errorf("%s offsets[%d]=%d < offsets[%d]=%d", name, i, offsets[i], i-1, offsets[i-1])
		}
	}
	if offsets[len(offsets)-1] != int64(len(tokens)) {
		return fmt.Errorf("%s offsets[-1]=%d != len(tokens)=%d", name, offsets[len(offsets)-1], len(tokens))
	}
	return nil
}

// fragmentSpan returns the token-id span for a structural fragment id.
func (t *tokenTables) fragmentSpan(fragID int32) []int32 {
	if t == nil || fragID < 0 || fragID >= int32(len(t.structuralOffsets)-1) {
		return nil
	}
	start := t.structuralOffsets[fragID]
	end := t.structuralOffsets[fragID+1]
	return t.structuralTokens[start:end]
}

// turnStepSpan returns tokens for `f" turn={turn} step={step} "`.
func (t *tokenTables) turnStepSpan(turn, stepID int32) []int32 {
	if t == nil || turn < t.turnMin || turn > t.turnMax || stepID < 0 || stepID >= t.stepCount {
		return nil
	}
	idx := (turn-t.turnMin)*t.stepCount + stepID
	start := t.turnStepOff[idx]
	end := t.turnStepOff[idx+1]
	return t.turnStepTokens[start:end]
}

// lifeOwnerSpan returns tokens for `f"<self/opp> life={life} mana="`.
func (t *tokenTables) lifeOwnerSpan(life, owner int32) []int32 {
	if t == nil || life < t.lifeMin || life > t.lifeMax || owner < 0 || owner >= t.ownerCount {
		return nil
	}
	idx := (life-t.lifeMin)*t.ownerCount + owner
	start := t.lifeOwnerOff[idx]
	end := t.lifeOwnerOff[idx+1]
	return t.lifeOwnerTokens[start:end]
}

func (t *tokenTables) abilitySpan(n int32) []int32 {
	if t == nil || n < t.abilityMin || n > t.abilityMax {
		return nil
	}
	idx := n - t.abilityMin
	return t.abilityTokens[t.abilityOff[idx]:t.abilityOff[idx+1]]
}

func (t *tokenTables) countSpan(n int32) []int32 {
	if t == nil || n < t.countMin || n > t.countMax {
		return nil
	}
	idx := n - t.countMin
	return t.countTokens[t.countOff[idx]:t.countOff[idx+1]]
}

func (t *tokenTables) zoneOpenSpan(zone, owner int32) []int32 {
	if t == nil || zone < 0 || zone >= t.zoneCount || owner < 0 || owner >= t.ownerCount {
		return nil
	}
	idx := zone*t.ownerCount + owner
	return t.zoneOpenTokens[t.zoneOpenOff[idx]:t.zoneOpenOff[idx+1]]
}

func (t *tokenTables) zoneCloseSpan(zone, owner int32) []int32 {
	if t == nil || zone < 0 || zone >= t.zoneCount || owner < 0 || owner >= t.ownerCount {
		return nil
	}
	idx := zone*t.ownerCount + owner
	return t.zoneCloseTok[t.zoneCloseOff[idx]:t.zoneCloseOff[idx+1]]
}

func (t *tokenTables) actionVerbSpan(kindID int32) []int32 {
	if t == nil || kindID < 0 || kindID >= t.actionVerbCount {
		return nil
	}
	return t.actionVerbTokens[t.actionVerbOff[kindID]:t.actionVerbOff[kindID+1]]
}

func (t *tokenTables) manaGlyphSpan(colorID int32) []int32 {
	if t == nil || colorID < 0 || colorID >= t.manaColorCount {
		return nil
	}
	return t.manaTokens[t.manaOff[colorID]:t.manaOff[colorID+1]]
}

func (t *tokenTables) cardBodySpan(row int32) []int32 {
	if t == nil || row < 0 || row >= t.cardRowCount {
		return nil
	}
	return t.cardBodyToks[t.cardBodyOff[row]:t.cardBodyOff[row+1]]
}

func (t *tokenTables) cardNameSpan(row int32) []int32 {
	if t == nil || row < 0 || row >= t.cardRowCount {
		return nil
	}
	return t.cardNameToks[t.cardNameOff[row]:t.cardNameOff[row+1]]
}

// getTokenTables returns the currently-registered tables, or nil if Python
// has not yet called MageRegisterTokenTables.
func getTokenTables() *tokenTables {
	tokenTablesMu.RLock()
	defer tokenTablesMu.RUnlock()
	return currentTokenTables
}

// registerTokenTables parses the C view and stores it. Returns an error if
// any offset table is internally inconsistent (a defensive check — Python
// builds these from the same source-of-truth as the Go-side spans, so a
// mismatch is a wire-format bug).
func registerTokenTables(c *C.MageTokenTables) error {
	if c == nil {
		tokenTablesMu.Lock()
		currentTokenTables = nil
		tokenTablesMu.Unlock()
		return nil
	}

	t := &tokenTables{
		turnMin:         int32(c.turn_min),
		turnMax:         int32(c.turn_max),
		stepCount:       int32(c.step_count),
		lifeMin:         int32(c.life_min),
		lifeMax:         int32(c.life_max),
		ownerCount:      int32(c.owner_count),
		abilityMin:      int32(c.ability_min),
		abilityMax:      int32(c.ability_max),
		countMin:        int32(c.count_min),
		countMax:        int32(c.count_max),
		zoneCount:       int32(c.zone_count),
		actionVerbCount: int32(c.action_verb_count),
		manaColorCount:  int32(c.mana_color_count),
		cardRefCount:    int32(c.card_ref_count),
		padID:           int32(c.pad_id),
		optionID:        int32(c.option_id),
		targetOpenID:    int32(c.target_open_id),
		targetCloseID:   int32(c.target_close_id),
		tappedID:        int32(c.tapped_id),
		untappedID:      int32(c.untapped_id),
		cardRowCount:    int32(c.card_row_count),
		dictOpenID:      int32(c.dict_open_id),
		dictCloseID:     int32(c.dict_close_id),
		cardOpenID:      int32(c.card_open_id),
		selfID:          int32(c.self_id),
		oppID:           int32(c.opp_id),
		stackOpenID:     int32(c.stack_open_id),
		stackCloseID:    int32(c.stack_close_id),
		commandOpenID:   int32(c.command_open_id),
		commandCloseID:  int32(c.command_close_id),
	}

	fragmentCount := int(c.fragment_count)
	t.structuralOffsets = sliceI32(c.structural_offsets, fragmentCount+1)
	if len(t.structuralOffsets) > 0 {
		t.structuralTokens = sliceI32(c.structural_tokens, int(t.structuralOffsets[fragmentCount]))
	}
	validationErr := validateTablePack("structural", t.structuralTokens, t.structuralOffsets, fragmentCount)
	if validationErr != nil {
		return validationErr
	}

	turnEntries := int((t.turnMax - t.turnMin + 1) * t.stepCount)
	t.turnStepOff = sliceI32(c.turn_step_offsets, turnEntries+1)
	if len(t.turnStepOff) > 0 {
		t.turnStepTokens = sliceI32(c.turn_step_tokens, int(t.turnStepOff[turnEntries]))
	}
	validationErr = validateTablePack("turn_step", t.turnStepTokens, t.turnStepOff, turnEntries)
	if validationErr != nil {
		return validationErr
	}

	lifeEntries := int((t.lifeMax - t.lifeMin + 1) * t.ownerCount)
	t.lifeOwnerOff = sliceI32(c.life_owner_offsets, lifeEntries+1)
	if len(t.lifeOwnerOff) > 0 {
		t.lifeOwnerTokens = sliceI32(c.life_owner_tokens, int(t.lifeOwnerOff[lifeEntries]))
	}
	validationErr = validateTablePack("life_owner", t.lifeOwnerTokens, t.lifeOwnerOff, lifeEntries)
	if validationErr != nil {
		return validationErr
	}

	abEntries := int(t.abilityMax - t.abilityMin + 1)
	t.abilityOff = sliceI32(c.ability_offsets, abEntries+1)
	if len(t.abilityOff) > 0 {
		t.abilityTokens = sliceI32(c.ability_tokens, int(t.abilityOff[abEntries]))
	}
	validationErr = validateTablePack("ability", t.abilityTokens, t.abilityOff, abEntries)
	if validationErr != nil {
		return validationErr
	}

	cnEntries := int(t.countMax - t.countMin + 1)
	t.countOff = sliceI32(c.count_offsets, cnEntries+1)
	if len(t.countOff) > 0 {
		t.countTokens = sliceI32(c.count_tokens, int(t.countOff[cnEntries]))
	}
	validationErr = validateTablePack("count", t.countTokens, t.countOff, cnEntries)
	if validationErr != nil {
		return validationErr
	}

	zoneEntries := int(t.zoneCount * t.ownerCount)
	t.zoneOpenOff = sliceI32(c.zone_open_offsets, zoneEntries+1)
	if len(t.zoneOpenOff) > 0 {
		t.zoneOpenTokens = sliceI32(c.zone_open_tokens, int(t.zoneOpenOff[zoneEntries]))
	}
	validationErr = validateTablePack("zone_open", t.zoneOpenTokens, t.zoneOpenOff, zoneEntries)
	if validationErr != nil {
		return validationErr
	}
	t.zoneCloseOff = sliceI32(c.zone_close_offsets, zoneEntries+1)
	if len(t.zoneCloseOff) > 0 {
		t.zoneCloseTok = sliceI32(c.zone_close_tokens, int(t.zoneCloseOff[zoneEntries]))
	}
	validationErr = validateTablePack("zone_close", t.zoneCloseTok, t.zoneCloseOff, zoneEntries)
	if validationErr != nil {
		return validationErr
	}

	avEntries := int(t.actionVerbCount)
	t.actionVerbOff = sliceI32(c.action_verb_offsets, avEntries+1)
	if len(t.actionVerbOff) > 0 {
		t.actionVerbTokens = sliceI32(c.action_verb_tokens, int(t.actionVerbOff[avEntries]))
	}
	validationErr = validateTablePack("action_verb", t.actionVerbTokens, t.actionVerbOff, avEntries)
	if validationErr != nil {
		return validationErr
	}

	mcEntries := int(t.manaColorCount)
	t.manaOff = sliceI32(c.mana_glyph_offsets, mcEntries+1)
	if len(t.manaOff) > 0 {
		t.manaTokens = sliceI32(c.mana_glyph_tokens, int(t.manaOff[mcEntries]))
	}
	validationErr = validateTablePack("mana_glyph", t.manaTokens, t.manaOff, mcEntries)
	if validationErr != nil {
		return validationErr
	}

	t.cardRefIDs = sliceI32(c.card_ref_ids, int(t.cardRefCount))

	t.cardCloser = sliceI32(c.card_closer, int(c.card_closer_len))
	t.statusTapped = sliceI32(c.status_tapped, int(c.status_tapped_len))
	t.statusUntapped = sliceI32(c.status_untapped, int(c.status_untapped_len))

	rowCount := int(t.cardRowCount)
	t.cardBodyOff = sliceI64(c.card_body_offsets, rowCount+1)
	if len(t.cardBodyOff) > 0 {
		t.cardBodyToks = sliceI32(c.card_body_tokens, int(t.cardBodyOff[rowCount]))
	}
	validationErr = validateTablePack64("card_body", t.cardBodyToks, t.cardBodyOff, rowCount)
	if validationErr != nil {
		return validationErr
	}
	t.cardNameOff = sliceI64(c.card_name_offsets, rowCount+1)
	if len(t.cardNameOff) > 0 {
		t.cardNameToks = sliceI32(c.card_name_tokens, int(t.cardNameOff[rowCount]))
	}
	validationErr = validateTablePack64("card_name", t.cardNameToks, t.cardNameOff, rowCount)
	if validationErr != nil {
		return validationErr
	}

	// dict_entry_ids is sized rowCount (one entry per card row), or NULL
	// when v2 dedup is disabled. The native side never indexes past
	// rowCount, so the same length bound is safe.
	t.dictEntryIDs = sliceI32(c.dict_entry_ids, rowCount)

	tokenTablesMu.Lock()
	currentTokenTables = t
	tokenTablesMu.Unlock()
	return nil
}
