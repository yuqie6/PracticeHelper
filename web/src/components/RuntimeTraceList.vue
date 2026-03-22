<template>
  <div class="runtime-trace-list">
    <article
      v-for="(entry, index) in entries"
      :key="`${entry.flow}-${entry.phase}-${entry.code}-${index}`"
      class="runtime-trace-row"
    >
      <div class="runtime-trace-head">
        <p class="runtime-trace-phase">
          {{ formatRuntimeTracePhaseLabel(t, entry.phase) }}
        </p>
        <span class="runtime-trace-status">
          {{ formatRuntimeTraceStatusLabel(t, entry.status) }}
        </span>
      </div>

      <div class="runtime-trace-meta">
        <code v-if="entry.code" class="runtime-trace-code">{{
          entry.code
        }}</code>
        <span v-if="entry.tool_name" class="runtime-trace-chip">
          {{ entry.tool_name }}
        </span>
        <span v-if="entry.attempt" class="runtime-trace-chip">
          #{{ entry.attempt }}
        </span>
      </div>

      <p v-if="entry.message" class="runtime-trace-message">
        {{ entry.message }}
      </p>
    </article>

    <p v-if="!entries.length" class="neo-note">
      {{ t('session.traceEmpty') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { RuntimeTraceEntry } from '../api/client';
import {
  formatRuntimeTracePhaseLabel,
  formatRuntimeTraceStatusLabel,
} from '../lib/runtimeTrace';

defineProps<{
  entries: RuntimeTraceEntry[];
}>();

const { t } = useI18n();
</script>

<style scoped>
.runtime-trace-list {
  display: grid;
  gap: 0.75rem;
}

.runtime-trace-row {
  background: color-mix(in srgb, var(--neo-paper) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.6rem;
  padding: 0.9rem 1rem;
}

.runtime-trace-head,
.runtime-trace-meta {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
  justify-content: space-between;
}

.runtime-trace-phase {
  font-size: 0.9rem;
  font-weight: 900;
  letter-spacing: 0.04em;
  margin: 0;
  text-transform: uppercase;
}

.runtime-trace-status,
.runtime-trace-chip,
.runtime-trace-code {
  background: color-mix(in srgb, var(--neo-surface) 92%, transparent);
  border: 2px solid var(--neo-border);
  display: inline-flex;
  font-size: 0.75rem;
  font-weight: 800;
  line-height: 1.2;
  padding: 0.25rem 0.5rem;
}

.runtime-trace-code {
  font-family: var(--font-mono);
}

.runtime-trace-message {
  font-size: 0.9rem;
  font-weight: 700;
  line-height: 1.6;
  margin: 0;
  text-wrap: pretty;
}
</style>
