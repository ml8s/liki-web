# Bring chat controls up to the mobile touch-target contract

Written against: `d02d8fe` plus the current uncommitted working tree

## Evidence chain

- Surface: `/app/chat/:sessionId`, mobile widths through `768px`.
- Problem: The native “jump to latest” and answer-copy controls are respectively `34px` tall and effectively about `24px` tall, below the app's mobile button target contract.
- Design evidence: `FRONTEND.md` requires primary controls to have sufficient touch targets and safe-area-aware mobile layout. `web/app/src/styles/app.css` codifies the product implementation as a `44px` minimum for `.el-button` at `max-width: 768px`; `web/app/src/components/app/RecordActions.vue` applies the same `44px` minimum to native record-action controls at mobile widths.
- Owner: `ChatSessionView.vue` owns the jump control; `ChatMessageItem.vue` owns the answer-copy action.
- Scope and affected surfaces: only the authenticated chat session page and its assistant message actions.
- Uncertainty: none. No documented exception permits smaller chat controls.

## Design decision

Apply the existing `44px` mobile touch-target decision to the two native chat controls. Keep their desktop dimensions and visual identity unchanged. Preserve the jump control's `12px` clearance above the composer by moving it from `top: -46px` to `top: -56px` only at mobile widths, matching the `10px` height increase.

Do not create a new shared primitive: these are two local actions with different compositions, and the repository already expresses the decision through component-local mobile rules and `RecordActions.vue`.

## Reuse

- `--font-size-caption`
- `--radius-full` for `.jump-latest`
- `--radius-small` for `.copy`
- Existing color, border, background, hover, and focus contracts
- Exemplar: `web/app/src/components/app/RecordActions.vue:61-63`

## Changes

1. `web/app/src/views/ChatSessionView.vue`
   - Change: add a `@media (max-width: 768px)` block for `.jump-latest` with:
     - `top: -56px`
     - `min-height: 44px`
     - `padding: 10px 16px`
   - Preserve: `34px` height and `top: -46px` above `768px`; existing tokens, border, background, shadow, and hover state.
   - Verify: after scrolling the thread away from the bottom on a `390px`-wide viewport, the jump control's rendered box is at least `44px` tall and does not overlap the composer.

2. `web/app/src/components/chat/ChatMessageItem.vue`
   - Change: add a `@media (max-width: 768px)` block for `.copy` with:
     - `display: inline-flex`
     - `align-items: center`
     - `min-height: 44px`
     - `padding: 10px 14px`
     - `margin: 8px 0 0 -14px`
   - Preserve: desktop `4px 8px` padding and `-8px` optical offset; existing copied-state label, color, border radius, and hover/focus behavior.
   - Verify: every assistant answer with a visible copy action has a rendered box of at least `44px` in height on a `390px`-wide viewport, while its label remains aligned with answer content.

3. `web/e2e/journeys/chat-session.spec.ts`
   - Change: extend the existing mobile chat-session test to measure the copy control and, after scrolling the message region away from the bottom, the jump control with `getBoundingClientRect()`.
   - Preserve: existing assertions for horizontal overflow, message sending, viewport ownership, and composer visibility.
   - Verify: both controls measure at least `44px` in height; the jump control remains strictly above the composer box.

## Scope

- Inherit: `/app/chat/:sessionId` at widths through `768px`.
- Verify: chat messages with short and long assistant content, the near-bottom and scrolled-away states, and desktop widths above `768px`.
- Exclude: Element Plus buttons, record actions, safe-area insets, the nonstandard login breakpoint, and unrelated chat behavior.

## Validation

- Product: open a chat session on a `390px` viewport, scroll up until “jump to latest” appears, and tap either that control or an answer's copy action without accidentally hitting the composer or adjacent message content.
- Interface: verify assistant messages with short and long Markdown, the jumped/earlier-message states, and the `768px` breakpoint edge; confirm desktop geometry is unchanged.
- System: confirm the implementation reuses the existing touch-target decision and does not introduce a parallel global button style.
- Repository: `cd web && npx playwright test --config e2e/playwright.config.ts e2e/journeys/chat-session.spec.ts` → all chat-session tests pass, including the new geometry assertions.
- Repository: `make lint-js && make typecheck && make check-web-static` → all pass.

## Stop conditions

- Stop if changing the jump control's position reveals overlap with the thread's last message or composer rather than preserving its `12px` clearance.
- Stop if the `44px` copy target causes clipping or horizontal overflow inside long assistant messages; resolve the containing layout before shipping the touch-target change.
- Stop if chat control ownership moves away from these two components or a shared chat action primitive is introduced.

## Design documentation

- After acceptance and validation: add native chat action controls to the mobile touch-target note in `FRONTEND.md`, or record an explicit exception if product review chooses a different target.
