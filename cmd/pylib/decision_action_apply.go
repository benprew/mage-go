package main

import (
	"fmt"
)

// actionFromDecoderOutput translates a single decoder-shaped action into the
// engine actionRequest. Mirrors magic_ai/text_encoder/actor_critic.py::
// decode_decoder_action: walks pointer steps in decoder order and resolves
// each pointer step's handle via its subject_index (anchorHandles[subject]).
//
// Inputs are sliced per env:
//
//   - tokens           — output_token_ids[i, :output_lens[i]]
//   - pointerSubjects  — output_pointer_subjects[i, :output_lens[i]]
//   - isPointer        — output_is_pointer[i, :output_lens[i]] (0/1)
//   - anchorHandles    — pointer_anchor_handles[i, :pointer_anchor_count[i]]
//
// pending must reflect the engine's current pending request. It's used to
// look up engine ids (uuid strings) from option-index handles.
func actionFromDecoderOutput(
	dt decisionType,
	tokens []int32,
	pointerSubjects []int32,
	isPointer []uint8,
	anchorHandles []int32,
	pending *apiPending,
) (actionRequest, error) {
	if dt == decTypeNone {
		return actionRequest{}, nil
	}
	if pending == nil {
		return actionRequest{}, fmt.Errorf("decoder action: no pending request")
	}

	// Resolve anchorHandles[subject] for the k-th pointer step.
	resolveByOrdinal := func(ordinal int) (int32, bool) {
		// Walk pointer steps to find the ordinal-th one.
		seen := 0
		for i := range tokens {
			if isPointer[i] == 0 {
				continue
			}
			if seen == ordinal {
				subj := pointerSubjects[i]
				if subj < 0 || int(subj) >= len(anchorHandles) {
					return 0, false
				}
				return anchorHandles[subj], true
			}
			seen++
		}
		return 0, false
	}

	switch dt {
	case decTypePriority:
		// Single pointer step → engine option index.
		optIdx, ok := resolveByOrdinal(0)
		if !ok {
			return actionRequest{}, fmt.Errorf("priority decoder action: no pointer step")
		}
		return priorityActionFromOptionIndex(pending, int(optIdx))

	case decTypeDeclareAttackers:
		// Pairs of (LEGAL_ATTACKER ptr, DEFENDER ptr). Build the binary
		// attacker selection by attacker option index. Defender pointers are
		// captured but the engine API only consumes the attacker list today.
		options := pending.Options
		n := len(options)
		selected := make([]bool, n)
		ord := 0
		for i := range tokens {
			if isPointer[i] == 0 {
				continue
			}
			subj := pointerSubjects[i]
			if subj < 0 || int(subj) >= len(anchorHandles) {
				ord++
				continue
			}
			handle := anchorHandles[subj]
			// Even ordinals = attackers, odd = defenders. Skip defenders.
			if ord%2 == 0 {
				if handle >= 0 && int(handle) < n {
					selected[handle] = true
				}
			}
			ord++
		}
		attackers := make([]string, 0, n)
		for i, sel := range selected {
			if !sel {
				continue
			}
			if id := options[i].PermanentID; id != "" {
				attackers = append(attackers, id)
			}
		}
		return actionRequest{Attackers: attackers}, nil

	case decTypeDeclareBlockers:
		// Pairs of (LEGAL_BLOCKER ptr, ATTACKER ptr). The attacker pointer's
		// handle is the attacker-order index (from decision_spec_emitter's
		// attackerOrder) — which equals the index into the blocker's
		// ValidTargets list when the blocker can block that attacker.
		options := pending.Options
		assigns := make([]blockerAssign, 0)
		// Collect (subject_index, ordinal) pairs in pointer-step order.
		var ptrHandles []int32
		for i := range tokens {
			if isPointer[i] == 0 {
				continue
			}
			subj := pointerSubjects[i]
			if subj < 0 || int(subj) >= len(anchorHandles) {
				ptrHandles = append(ptrHandles, -1)
				continue
			}
			ptrHandles = append(ptrHandles, anchorHandles[subj])
		}
		for i := 0; i+1 < len(ptrHandles); i += 2 {
			blkH := ptrHandles[i]
			atkH := ptrHandles[i+1]
			if blkH < 0 || int(blkH) >= len(options) {
				continue
			}
			opt := options[blkH]
			blockerID := opt.PermanentID
			if blockerID == "" {
				continue
			}
			// atkH is an attacker-order index; the blocker's ValidTargets
			// slice is the attacker list this blocker can block. The renderer
			// orders attackers by first-seen across all blockers, so atkH
			// may not directly index into this blocker's ValidTargets. Match
			// by attacker uuid: walk this blocker's ValidTargets and pick
			// the one whose attacker-order index equals atkH.
			//
			// Approximation: for tests with a single blocker, attacker-order
			// IS the ValidTargets order. For the general case we look up the
			// attacker uuid via the global attackerOrder reconstruction.
			attackerID := lookupBlockerAttacker(options, int(blkH), int(atkH))
			if attackerID == "" {
				continue
			}
			assigns = append(assigns, blockerAssign{Blocker: blockerID, Attacker: attackerID})
		}
		return actionRequest{Blockers: assigns}, nil

	case decTypeChooseTargets:
		// One pointer step → option index in pending.Options.
		optIdx, ok := resolveByOrdinal(0)
		if !ok {
			return actionRequest{SelectedIDs: nil}, nil
		}
		options := pending.Options
		if optIdx < 0 || int(optIdx) >= len(options) {
			return actionRequest{}, fmt.Errorf("choose_targets decoder action: option index %d out of range (%d options)", optIdx, len(options))
		}
		selectedID := options[optIdx].ID
		if selectedID == "" {
			return actionRequest{SelectedIDs: nil}, nil
		}
		return actionRequest{SelectedIDs: []string{selectedID}}, nil

	case decTypeMay:
		// Look for YES / NO grammar tokens.
		for i, t := range tokens {
			if isPointer[i] != 0 {
				continue
			}
			if t == gYes {
				return actionRequest{Accepted: true}, nil
			}
			if t == gNo {
				return actionRequest{Accepted: false}, nil
			}
		}
		return actionRequest{Accepted: false}, nil

	case decTypeChooseMode, decTypeChooseX:
		// Walk digit tokens (gDigit0..gDigit9) and parse to int.
		val := 0
		for i, t := range tokens {
			if isPointer[i] != 0 {
				continue
			}
			if t >= gDigit0 && t <= gDigit9 {
				val = val*10 + int(t-gDigit0)
			}
		}
		if dt == decTypeChooseX {
			return actionRequest{X: val, SelectedIndex: val}, nil
		}
		return actionRequest{SelectedIndex: val}, nil
	}

	return actionRequest{}, fmt.Errorf("applyDecoderAction: unknown decision_type %d", int32(dt))
}

// priorityActionFromOptionIndex maps a 0-based engine option index (as
// produced by the decision_spec emitter for PRIORITY) to the actionRequest
// that routeAction expects. Mirrors priorityActionFromChoiceCol but indexes
// by raw option index without the per-target column expansion.
func priorityActionFromOptionIndex(pending *apiPending, optIdx int) (actionRequest, error) {
	if optIdx < 0 || optIdx >= len(pending.Options) {
		return actionRequest{}, fmt.Errorf("priority decoder action: option index %d out of range (%d options)", optIdx, len(pending.Options))
	}
	option := pending.Options[optIdx]
	switch option.Kind {
	case "pass":
		return actionRequest{Kind: "pass"}, nil
	case "play_land":
		return actionRequest{Kind: "play_land", CardID: option.CardID}, nil
	case "cast_spell":
		req := actionRequest{Kind: "cast_spell", CardID: option.CardID}
		if len(option.ValidTargets) > 0 {
			req.Targets = []string{option.ValidTargets[0].ID}
		}
		return req, nil
	case "activate_ability":
		req := actionRequest{
			Kind:         "activate_ability",
			PermanentID:  option.PermanentID,
			AbilityIndex: option.AbilityIndex,
		}
		if len(option.ValidTargets) > 0 {
			req.Targets = []string{option.ValidTargets[0].ID}
		}
		return req, nil
	}
	return actionRequest{}, fmt.Errorf("priority decoder action: unsupported option kind %q", option.Kind)
}

// lookupBlockerAttacker resolves the (blockerOptIdx, attackerOrderIdx) pair
// from a DECLARE_BLOCKERS decoder action into the engine attacker uuid. The
// renderer (decision_spec_emitter.go::decTypeDeclareBlockers) builds the
// global attackerOrder as the first-seen union of every blocker's
// ValidTargets — we re-derive it here so that attackerOrderIdx matches.
func lookupBlockerAttacker(options []apiOption, blockerOptIdx, attackerOrderIdx int) string {
	// Re-derive attacker order.
	order := make([]string, 0)
	seen := make(map[string]int, 8)
	for _, opt := range options {
		for _, t := range opt.ValidTargets {
			if t.ID == "" {
				continue
			}
			if _, ok := seen[t.ID]; !ok {
				seen[t.ID] = len(order)
				order = append(order, t.ID)
			}
		}
	}
	if attackerOrderIdx < 0 || attackerOrderIdx >= len(order) {
		return ""
	}
	wantedID := order[attackerOrderIdx]
	// Confirm this blocker can actually block that attacker.
	if blockerOptIdx < 0 || blockerOptIdx >= len(options) {
		return ""
	}
	for _, t := range options[blockerOptIdx].ValidTargets {
		if t.ID == wantedID {
			return wantedID
		}
	}
	return ""
}

// applyDecoderAction looks up the handle, builds an actionRequest from the
// decoder output, and routes it through the existing engine-step plumbing.
// Caller is responsible for skipping no-op rows (decisionType == -1) before
// calling.
func applyDecoderAction(
	dt decisionType,
	tokens []int32,
	pointerSubjects []int32,
	isPointer []uint8,
	anchorHandles []int32,
	h *handle,
) error {
	if dt == decTypeNone {
		return nil
	}
	pending := buildPending(h.current)
	action, err := actionFromDecoderOutput(dt, tokens, pointerSubjects, isPointer, anchorHandles, pending)
	if err != nil {
		return err
	}
	if dt == decTypeNone {
		return nil
	}
	return routeAction(h, action)
}
