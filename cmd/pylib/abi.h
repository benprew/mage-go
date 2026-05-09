#ifndef MAGE_PYLIB_ABI_H
#define MAGE_PYLIB_ABI_H

#include <stdint.h>

typedef struct {
    int64_t n;
    const int64_t* handles;
    const int64_t* perspective_player_idx;
} MageBatchRequest;

typedef struct {
    int64_t* ready;
    int64_t* game_over;
    int64_t* pending_player_idx;
    int64_t* winner_player_idx;
} MageBatchPollOutputs;

typedef struct {
    int64_t n;
    int64_t max_options;
    int64_t max_targets_per_option;
    const int64_t* handles;
    const int64_t* decision_start;
    const int64_t* decision_count;
    const int64_t* selected_choice_cols;
    const int64_t* may_selected;
} MageStepChoiceRequest;

typedef struct {
    int64_t max_options;
    int64_t max_targets_per_option;
    int64_t max_cached_choices;
    int64_t zone_slot_count;
    int64_t game_info_dim;
    int64_t option_scalar_dim;
    int64_t target_scalar_dim;
    int64_t decision_capacity;
    int64_t emit_render_plan;
    int64_t render_plan_capacity;
    /* When set, the render-plan emitter switches to v2 (``<dict>`` opcodes
       21-24): each unique card body is spliced once at the top, and per-zone
       occurrences become short ``<card-ref>``-anchored references. The
       native token assembler must understand these opcodes for the encoded
       tokens to round-trip correctly. */
    int64_t dedup_card_bodies;
} MageEncodeConfig;

typedef struct {
    int64_t* trace_kind_id;
    int64_t* slot_card_rows;
    float* slot_occupied;
    float* slot_tapped;
    float* game_info;
    int64_t* pending_kind_id;
    int64_t* num_present_options;
    int64_t* option_kind_ids;
    float* option_scalars;
    float* option_mask;
    int64_t* option_ref_slot_idx;
    int64_t* option_ref_card_row;
    float* target_mask;
    int64_t* target_type_ids;
    float* target_scalars;
    float* target_overflow;
    int64_t* target_ref_slot_idx;
    uint8_t* target_ref_is_player;
    uint8_t* target_ref_is_self;
    uint8_t* may_mask;
    int64_t* decision_start;
    int64_t* decision_count;
    int64_t* decision_option_idx;
    int64_t* decision_target_idx;
    uint8_t* decision_mask;
    uint8_t* uses_none_head;
    int32_t* render_plan;
    int64_t* render_plan_lengths;
    int64_t* render_plan_overflow;
} MageEncodeOutputs;

typedef struct {
    int64_t decision_rows_written;
    int64_t error_code;
    char* error_message;
} MageEncodeResult;

typedef struct {
    int64_t n;
    const int64_t* handles;
    const int64_t* slot_ids;
    const int64_t* episode_ids;
    int64_t max_steps_per_game;
    int64_t max_options;
    int64_t max_targets_per_option;
    int64_t max_cached_choices;
    int64_t zone_slot_count;
    int64_t game_info_dim;
    int64_t option_scalar_dim;
    int64_t target_scalar_dim;
    int64_t render_plan_capacity;
    int64_t dedup_card_bodies;
    int64_t max_tokens;
    int64_t max_card_refs;
    int64_t ready_queue_capacity;
    int64_t terminal_queue_capacity;
} MageTextRolloutStartRequest;

typedef struct {
    int64_t rows_written;
    int64_t terminal_events_written;
    int64_t decision_rows_written;
    int64_t error_code;
    char* error_message;
} MageTextReadyBatchResult;

typedef struct {
    int64_t n;
    const int64_t* request_ids;
    const int64_t* decision_count;
    const int64_t* selected_choice_cols;
    const int64_t* may_selected;
} MageTextChoiceSubmitRequest;

/*
 * Token-table registration: Python ships the closed-vocabulary token tables
 * needed by the future native text-encoder assembler. The wire format is a
 * collection of int32 buffers + int32/int64 offset tables. All pointers are
 * borrowed: Python owns the underlying tensors and must keep them alive for
 * the lifetime of the registration. Calling MageRegisterTokenTables again
 * replaces the prior registration.
 *
 * Most fields are length-prefixed via an offsets table: tokens for entry K
 * live at ``tokens[offsets[K]:offsets[K+1]]``. ``offsets`` always has length
 * ``count + 1`` so ``offsets[count]`` equals the total token buffer length.
 */
typedef struct {
    /* Static structural fragments (Frag enum values 0..fragment_count-1). */
    int32_t fragment_count;
    const int32_t* structural_tokens;
    const int32_t* structural_offsets; /* length fragment_count + 1 */

    /* turn × step. ``turn_step_offsets`` has length
       (turn_max - turn_min + 1) * step_count + 1. */
    int32_t turn_min;
    int32_t turn_max;
    int32_t step_count;
    const int32_t* turn_step_tokens;
    const int32_t* turn_step_offsets;

    /* life × owner. ``life_owner_offsets`` has length
       (life_max - life_min + 1) * owner_count + 1. */
    int32_t life_min;
    int32_t life_max;
    int32_t owner_count;
    const int32_t* life_owner_tokens;
    const int32_t* life_owner_offsets;

    /* ability index. */
    int32_t ability_min;
    int32_t ability_max;
    const int32_t* ability_tokens;
    const int32_t* ability_offsets; /* length ability_max - ability_min + 2 */

    /* counter count. */
    int32_t count_min;
    int32_t count_max;
    const int32_t* count_tokens;
    const int32_t* count_offsets;

    /* zone × owner open/close pairs. ``*_offsets`` length zone_count*owner_count + 1. */
    int32_t zone_count;
    const int32_t* zone_open_tokens;
    const int32_t* zone_open_offsets;
    const int32_t* zone_close_tokens;
    const int32_t* zone_close_offsets;

    /* action verb prefix (with leading space). */
    int32_t action_verb_count;
    const int32_t* action_verb_tokens;
    const int32_t* action_verb_offsets;

    /* mana glyph per color id. */
    int32_t mana_color_count;
    const int32_t* mana_glyph_tokens;
    const int32_t* mana_glyph_offsets;

    /* card-ref single ids. */
    int32_t card_ref_count;
    const int32_t* card_ref_ids;

    /* Singletons. */
    int32_t pad_id;
    int32_t option_id;
    int32_t target_open_id;
    int32_t target_close_id;
    int32_t tapped_id;
    int32_t untapped_id;

    /* Small fixed-length lists. */
    int32_t card_closer_len;
    const int32_t* card_closer;
    int32_t status_tapped_len;
    const int32_t* status_tapped;
    int32_t status_untapped_len;
    const int32_t* status_untapped;

    /* Per-card body / display-name tables.
       ``card_body_offsets`` and ``card_name_offsets`` have length
       ``card_row_count + 1``. */
    int32_t card_row_count;
    const int32_t* card_body_tokens;
    const int64_t* card_body_offsets;
    const int32_t* card_name_tokens;
    const int64_t* card_name_offsets;

    /* v2 card-body deduplication. The dict-entry table is one int32 per
       cache row (the ``<dict-entry:R>`` token id for row R), aligned to
       ``card_row_count``. ``dict_open_id`` / ``dict_close_id`` /
       ``card_open_id`` are the singleton ids for ``<dict>`` / ``</dict>`` /
       ``<card>``. All zero (and dict_entry_ids = NULL) is acceptable when
       ``cfg.dedup_card_bodies`` is never set. */
    int32_t dict_open_id;
    int32_t dict_close_id;
    int32_t card_open_id;
    const int32_t* dict_entry_ids;

    /* Singletons used by the structured Go emitter to reach byte-for-byte
       parity with the Python emit_render_plan path:
         - ``self_id`` / ``opp_id`` are emitted inside ``<target>...</target>``
           blocks when an option targets a player.
         - ``stack_*`` / ``command_*`` open/close the shared (non-per-player)
           stack and command zones, emitted once per snapshot. */
    int32_t self_id;
    int32_t opp_id;
    int32_t stack_open_id;
    int32_t stack_close_id;
    int32_t command_open_id;
    int32_t command_close_id;
} MageTokenTables;

/* Token-assembler dimensions shared by packed token outputs. */
typedef struct {
    int32_t max_tokens;
    int32_t max_options;
    int32_t max_targets;
    int32_t max_card_refs;
} MageTokenAssemblerConfig;

/*
 * Packed (varlen) token-assembler outputs. Caller allocates the token-
 * shaped arrays at capacity ``B * max_tokens`` (the worst case where
 * every row fills its budget). Anchor arrays carry absolute offsets
 * into ``token_ids`` (i.e. they are already shifted by cu_seqlens[b]).
 *
 * After a successful call, ``cu_seqlens[B]`` is the total live token
 * count; ``token_ids[0 : cu_seqlens[B]]`` is the live region. The trailing
 * portion of the buffer is unspecified. ``seq_id`` and ``pos_in_seq`` are
 * derivable from ``cu_seqlens`` and are intentionally not written by Go.
 */
typedef struct {
    int32_t* token_ids;          /* [B*max_tokens] int32, live region */
    int32_t* cu_seqlens;         /* [B+1] int32, exclusive prefix sum */
    int32_t* seq_lengths;        /* [B] int32 */
    int32_t* state_positions;    /* [B] int32, packed-offset of row's first token */
    int32_t* card_ref_positions; /* [B, max_card_refs] int32, absolute, -1 absent */
    int32_t* token_overflow;     /* [B] int32 (1 = row truncated) */
} MagePackedTokenAssemblerOutputs;

/*
 * Per-row decision-spec outputs emitted alongside the packed token stream.
 * The caller pre-allocates buffers at capacity ``B * T_spec_max`` (and
 * similarly for anchors and the legal-edge bitmap). The native side packs
 * each row's emitted spec tokens into the row's slice (0-padded), with
 * pointer_anchor_positions already shifted by the row's emitted state-token
 * length so they reference the combined ``[state_tokens] + [spec_tokens]``
 * stream. ``spec_overflow`` is set to a nonzero value if any per-row cap
 * was exceeded.
 */
typedef struct {
    int32_t* spec_tokens;             /* [B, T_spec_max] int32, 0 = pad */
    int32_t* spec_lens;               /* [B] int32 */
    int32_t* decision_type;           /* [B] int32, -1 = no pending */
    int32_t* pointer_anchor_positions;/* [B, N_anchors_max] int32, -1 = pad */
    int32_t* pointer_anchor_kinds;    /* [B, N_anchors_max] int32, -1 = pad */
    int32_t* pointer_anchor_subjects; /* [B, N_anchors_max] int32 */
    int32_t* pointer_anchor_handles;  /* [B, N_anchors_max] int32 */
    int32_t* pointer_anchor_counts;   /* [B] int32 */
    uint8_t* legal_edge_bitmap;       /* [B, N_blockers_max, N_attackers_max] */
    int32_t* legal_edge_n_blockers;   /* [B] int32 */
    int32_t* legal_edge_n_attackers;  /* [B] int32 */
    int32_t  T_spec_max;
    int32_t  N_anchors_max;
    int32_t  N_blockers_max;
    int32_t  N_attackers_max;
    int32_t  spec_overflow;
} MagePackedSpecOutputs;

/*
 * Decision-spec tag tokens + digit-token lookup table. Caller passes the
 * 12 fixed structural tag ids, the 7 decision-type-name ids, the 16
 * stack-ref ids, and the BPE digit-id table indexed by integer value.
 * Pointers are borrowed; tensors must outlive the registration.
 */
typedef struct {
    int32_t spec_open_id;
    int32_t spec_close_id;
    int32_t decision_type_id;
    int32_t legal_attacker_id;
    int32_t legal_blocker_id;
    int32_t legal_target_id;
    int32_t legal_action_id;
    int32_t for_action_id;
    int32_t max_value_open_id;
    int32_t max_value_close_id;
    int32_t player_ref0_id;
    int32_t player_ref1_id;

    /* dt_name_ids[k] = id for `<dt-{name}>` for decisionType k.
       k in [0..6] (priority, declare_attackers, declare_blockers,
       choose_targets, may, choose_mode, choose_x). */
    const int32_t* dt_name_ids;       /* length 7 */

    /* stack_ref_ids[k] = id for `<stack-ref:k>`. length 16. */
    const int32_t* stack_ref_ids;     /* length 16 */

    /* Digit lookup: value v in [0, max_value_digit_max] maps to the
       int32 sequence at digits[offsets[v]:offsets[v+1]]. */
    int32_t max_value_digit_max;
    const int32_t* max_value_digits;
    const int32_t* max_value_digit_offsets; /* length max_value_digit_max + 2 */
} MageDecisionSpecTokens;

typedef struct {
    int64_t* request_ids;
    int64_t* slot_ids;
    int64_t* episode_ids;
    int64_t* step_indices;
    int64_t* perspective_player_idx;
    int64_t* terminal_slot_ids;
    int64_t* terminal_episode_ids;
    int64_t* terminal_winner_idx;
    int64_t* terminal_is_timeout;
    int64_t* terminal_life_p0;
    int64_t* terminal_life_p1;
    MageEncodeOutputs encode;
    MagePackedTokenAssemblerOutputs packed_tokens;
} MageTextReadyBatchOutputs;

#endif
