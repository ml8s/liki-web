<template>
  <div
    class="app-flow-shell"
    :class="sidecarOpen ? `app-flow-shell--sidecar-${sidecarMode}` : undefined"
  >
    <a
      :href="`#${mainId}`"
      class="flow-skip-link"
    >
      {{ $t('a11y.skipToContent') }}
    </a>

    <header class="flow-status">
      <div class="flow-status__context">
        <span class="flow-status__person">{{ personLabel || $t('flow.personEmpty') }}</span>
        <span
          class="flow-status__separator"
          aria-hidden="true"
        >·</span>
        <span class="flow-status__intent-group">{{ intentGroupLabel }}</span>
      </div>

      <AppLiveRegion
        class="flow-status__announcement"
        screen-only
      >
        {{ contextAnnouncement }}
      </AppLiveRegion>

      <button
        v-if="restoreVisible"
        type="button"
        class="flow-restore"
        @click="emit('restore')"
      >
        {{ $t('flow.restore') }}
      </button>

      <button
        type="button"
        class="flow-overflow-trigger"
        :aria-label="$t('flow.contextMenu')"
        aria-haspopup="dialog"
        :aria-expanded="contextMenuOpen"
        @click="contextMenuOpen = true"
      >
        <span aria-hidden="true">⋯</span>
      </button>
    </header>

    <nav
      class="flow-context-rail"
      :aria-label="$t('flow.contextNav')"
    >
      <div class="flow-context-rail__group">
        <button
          v-for="item in contextItems"
          :key="item.id"
          type="button"
          class="flow-context-rail__item"
          @click="handleContextSelect(item.id)"
        >
          {{ item.label }}
        </button>
      </div>
    </nav>

    <div class="flow-body">
      <main
        :id="mainId"
        ref="mainElement"
        class="flow-main"
        tabindex="-1"
      >
        <slot />
      </main>

      <div class="flow-action-dock">
        <slot name="actions" />
      </div>
    </div>

    <div class="flow-sidecar">
      <slot name="sidecar" />
    </div>

    <FlowContextMenu
      v-model="contextMenuOpen"
      :items="contextItems"
      @select="handleContextSelect"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useId } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLiveRegion from '@/components/app/AppLiveRegion.vue'
import FlowContextMenu from './FlowContextMenu.vue'

const props = defineProps<{
  personLabel?: string
  intentGroupLabel?: string
  partnerVisible?: boolean
  restoreVisible?: boolean
  sidecarOpen?: boolean
  sidecarMode?: 'peek' | 'pinned'
}>()

const emit = defineEmits<{
  restore: []
  'context-select': [id: string]
}>()

const mainId = useId()
const mainElement = ref<HTMLElement | null>(null)
const contextMenuOpen = ref(false)
const { t } = useI18n()

const contextItems = computed(() => {
  const items = [
    { id: 'partner', label: t('router.partnerCenter') },
    { id: 'me', label: t('nav.me') },
    { id: 'history', label: t('router.chat') },
    { id: 'orders', label: t('router.orders') },
    { id: 'account', label: t('router.account') },
    { id: 'classic', label: t('nav.classic') },
  ]
  return props.partnerVisible ? items : items.filter(item => item.id !== 'partner')
})

const contextAnnouncement = computed(() => t('flow.contextAnnouncement', {
  person: props.personLabel || t('flow.personEmpty'),
  group: props.intentGroupLabel,
}))

function getMainElement() {
  return mainElement.value
}

function handleContextSelect(id: string) {
  contextMenuOpen.value = false
  emit('context-select', id)
}

async function scrollToBottom({ focusMain = false }: { focusMain?: boolean } = {}) {
  await nextTick()
  const element = mainElement.value
  if (!element) return
  if (focusMain) element.focus({ preventScroll: true })
  if (typeof element.scrollTo === 'function') {
    const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)')?.matches ?? false
    element.scrollTo({
      top: element.scrollHeight,
      behavior: reducedMotion ? 'auto' : 'smooth',
    })
  } else {
    element.scrollTop = element.scrollHeight
  }
}

defineExpose({
  getMainElement,
  scrollToBottom,
})
</script>

<style scoped>
.app-flow-shell {
  /* The shell—not the document—owns the viewport, keeping the dock visible
     and making the consultation document the only primary scroll region. */
  height: 100vh;
  height: 100dvh;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  background: var(--liki-paper);
}

.flow-skip-link {
  position: absolute;
  inset-inline-start: -9999px;
  z-index: var(--z-skip-link);
  padding: 10px var(--space-4);
  border-radius: 0 0 var(--radius-md) 0;
  background: var(--liki-primary);
  color: var(--liki-on-primary);
  font: inherit;
  font-size: var(--font-size-label);
  text-decoration: none;
}

.flow-skip-link:focus-visible {
  inset-inline-start: 0;
  outline: 3px solid var(--liki-ink);
  outline-offset: -2px;
}

.flow-status {
  min-width: 0;
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  flex-wrap: wrap;
  background: var(--liki-ink);
  color: var(--liki-on-ink);
}

@supports (padding: max(0px)) {
  .flow-status {
    padding-top: max(var(--space-2), env(safe-area-inset-top));
    padding-inline: max(var(--space-3), env(safe-area-inset-left))
      max(var(--space-3), env(safe-area-inset-right));
  }
}

.flow-status__context {
  flex: 1 1 12rem;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--font-size-label);
}

.flow-status__person,
.flow-status__intent-group {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.flow-status__person { font-weight: 600; }
.flow-status__intent-group { color: var(--liki-on-ink-subtle); }
.flow-status__separator { flex: 0 0 auto; color: var(--liki-on-ink-muted); }

.flow-restore {
  flex: 0 1 auto;
  min-width: 0;
  max-width: 100%;
  min-height: var(--control-height-touch);
  padding-inline: var(--space-3);
  border: 1px solid var(--liki-on-ink-line);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--liki-on-ink);
  font: inherit;
  font-size: var(--font-size-label);
  overflow-wrap: anywhere;
  white-space: normal;
  text-wrap: balance;
  cursor: pointer;
}

.flow-restore:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}

.flow-overflow-trigger {
  flex: 0 0 auto;
  width: var(--control-height-touch);
  height: var(--control-height-touch);
  display: inline-grid;
  place-items: center;
  border: 1px solid var(--liki-on-ink-line);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--liki-on-ink);
  font: inherit;
  font-size: var(--font-size-title-md);
  line-height: 1;
  cursor: pointer;
}
.flow-overflow-trigger:hover {
  border-color: var(--liki-on-ink-hover);
  background: var(--liki-on-ink-hover);
}
.flow-overflow-trigger:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
.flow-context-rail {
  display: none;
}

.flow-body {
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
}

.flow-main {
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
  padding: var(--space-4);
}

.flow-main:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: -2px;
}

.flow-action-dock {
  min-width: 0;
  display: grid;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3) calc(var(--space-3) + env(safe-area-inset-bottom));
  border-top: 1px solid var(--liki-line);
  background: var(--liki-surface);
}

/* Expand the dock's interactive safety area over device curves without
   pushing visible content off screen. */
@supports (padding: max(0px)) {
  .flow-action-dock {
    padding-inline: max(var(--space-3), env(safe-area-inset-left))
      max(var(--space-3), env(safe-area-inset-right));
  }
}

@media (max-width: 479.98px) {
  .flow-status {
    flex-wrap: nowrap;
    height: 44px;
    min-height: 44px;
    padding-block: 0;
  }

  .flow-action-dock {
    gap: 0;
    padding: 0 var(--space-3) 3px;
  }
}

@media (min-width: 1024px) {
  .app-flow-shell {
    grid-template-columns: 184px minmax(400px, 1fr);
    grid-template-rows: minmax(0, 1fr);
  }

  .app-flow-shell--sidecar-peek {
    grid-template-columns: 184px minmax(400px, 1fr) minmax(320px, 360px);
  }

  .app-flow-shell--sidecar-pinned {
    grid-template-columns: 184px minmax(400px, 1fr) minmax(384px, 440px);
  }

  .flow-status {
    grid-column: 1;
    grid-row: 1;
    min-height: 0;
    align-items: stretch;
    justify-content: flex-start;
    flex-direction: column;
    gap: var(--space-4);
    padding: var(--space-5) var(--space-4);
    border-right: 1px solid var(--liki-line);
    background: var(--liki-surface);
    color: var(--liki-ink);
  }

  .flow-status {
    height: auto;
    min-height: 0;
    flex-wrap: wrap;
  }

  .flow-action-dock {
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3) calc(var(--space-3) + env(safe-area-inset-bottom));
  }

  .flow-status__context {
    position: relative;
    align-items: flex-start;
    flex-direction: column;
    gap: var(--space-1);
    min-height: 0;
  }

  /* A rail reads as two context rows; the compact-bar separator is redundant. */
  .flow-status__separator { display: none; }

  .flow-status__person,
  .flow-status__intent-group {
    overflow-wrap: anywhere;
    white-space: normal;
  }

  .flow-status__intent-group { color: var(--liki-text-2); }

  .flow-restore {
    align-self: flex-start;
    border-color: var(--liki-line);
    color: var(--liki-ink);
  }

  .flow-restore:hover { border-color: var(--liki-primary-border); }

  .flow-overflow-trigger { display: none; }

  .flow-context-rail {
    grid-column: 1;
    grid-row: 2;
    min-width: 0;
    display: grid;
    gap: var(--space-5);
    align-content: start;
    overflow-y: auto;
    padding: var(--space-4);
    border-right: 1px solid var(--liki-line);
    background: var(--liki-surface);
  }
  .flow-context-rail__group {
    min-width: 0;
    display: grid;
    gap: var(--space-1);
  }
  .flow-context-rail__item {
    min-width: 0;
    min-height: var(--control-height-touch);
    display: flex;
    align-items: center;
    padding: 8px var(--space-2);
    border: 0;
    border-radius: var(--radius-md);
    background: transparent;
    color: var(--liki-text-2);
    font: inherit;
    font-size: var(--font-size-body);
    text-align: start;
    cursor: pointer;
  }
  .flow-context-rail__item:hover {
    background: var(--liki-primary-tint);
    color: var(--liki-primary);
  }
  .flow-context-rail__item:focus-visible {
    outline: 3px solid var(--liki-primary);
    outline-offset: 2px;
  }
  .flow-context-rail__item.is-selected {
    background: var(--liki-primary-tint);
    color: var(--liki-primary);
    font-weight: 600;
  }

  .flow-body {
    grid-column: 2;
    grid-row: 1 / span 2;
    width: min(100%, 760px);
    margin-inline: auto;
  }

  .flow-sidecar { display: none; }

  .app-flow-shell--sidecar-peek .flow-sidecar,
  .app-flow-shell--sidecar-pinned .flow-sidecar {
    grid-column: 3;
    grid-row: 1 / span 2;
    min-width: 0;
    min-height: 0;
    display: grid;
    align-items: start;
    padding-inline-start: var(--space-3);
  }

  .flow-main,
  .flow-action-dock {
    padding-inline: var(--space-3);
  }
}
</style>
