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

    /* Inline-blank singletons. Each ``<choose-*>`` token is emitted at the
       cursor position recorded by an EMIT_BLANK opcode; ``chosen_id`` /
       ``yes_id`` / ``no_id`` / ``none_id`` / ``x_end_id`` may appear in legal-
       id lists or as bookkeeping markers in the token stream. ``use_ability``
       is the ability-action variant of ``<choose-play>``. All are independent
       of the cardrow / ability tables — they are pure singletons. */
    int32_t choose_target_id;
    int32_t choose_block_id;
    int32_t choose_damage_order_id;
    int32_t choose_mode_id;
    int32_t choose_may_id;
    int32_t choose_x_digit_id;
    int32_t choose_mana_source_id;
    int32_t choose_play_id;
    int32_t use_ability_id;
    int32_t chosen_id;
    int32_t yes_id;
    int32_t no_id;
    int32_t none_id;
    int32_t x_end_id;
    int32_t mulligan_id;
    int32_t keep_id;

    /* Digit tokens for inline X-cost blanks: ``num_ids[k]`` is the token id
       for digit ``k`` (typically 0..15). Length is given by ``num_count``;
       the count is authoritative and the array may be NULL when count==0. */
    int32_t num_count;
    const int32_t* num_ids;
} MageTokenTables;

/* Token-assembler dimensions shared by packed token outputs. */
typedef struct {
    int32_t max_tokens;
    int32_t max_options;
    int32_t max_targets;
    int32_t max_card_refs;
} MageTokenAssemblerConfig;

typedef struct {
    int32_t max_blanks;
    int32_t max_legal_per_blank;
} MageBlankAssemblerConfig;

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

typedef struct {
    int32_t k_max;
    int32_t v_max;
    int32_t* blank_positions;    /* [B, K] int32, absolute, -1 absent */
    int32_t* blank_kind;         /* [B, K] int32, 0 absent */
    int32_t* blank_group;        /* [B, K] int32, -1 absent */
    int32_t* blank_group_kind;   /* [B, K] int32 */
    int32_t* blank_option_index; /* [B, K] int32, engine option index, -1 absent */
    int32_t* blank_legal_ids;    /* [B, K, V] int32, 0 pad */
    uint8_t* blank_legal_mask;   /* [B, K, V] uint8 */
    int32_t* blank_overflow;     /* [B] int32, count of dropped blanks */
    int32_t* blank_count;        /* [B] int32, live blanks per row */
    int32_t* blank_legal_count;  /* [B, K] int32, live legal ids per blank */
} MagePackedBlankOutputs;

#endif
