<template>
  <section class="neo-panel review-audit-panel">
    <div class="review-section-head">
      <div>
        <p class="neo-kicker bg-[var(--neo-yellow)]">
          {{ t('review.auditTitle') }}
        </p>
        <p class="neo-note">
          {{ t('review.auditDescription') }}
        </p>
      </div>
    </div>

    <div v-if="isLoadingEvaluationLogs" class="neo-note">
      {{ t('common.loading') }}
    </div>

    <NoticePanel
      v-else-if="evaluationLogsError"
      tone="error"
      :title="t('review.auditErrorTitle')"
      :message="evaluationLogsError"
    />

    <section
      v-if="retrievalTrace && !isLoadingEvaluationLogs"
      class="review-trace-panel space-y-4"
    >
      <div class="space-y-1">
        <p class="neo-subheading">
          {{ t('review.retrievalTraceTitle') }}
        </p>
        <p class="neo-note">
          {{ t('review.retrievalTraceDescription') }}
        </p>
      </div>

      <div class="review-trace-meta">
        <span class="neo-badge bg-white">
          {{ retrievalTrace.topic || '—' }}
        </span>
        <span class="neo-note">
          {{ new Date(retrievalTrace.generated_at).toLocaleString() }}
        </span>
      </div>

      <div class="review-trace-groups">
        <article
          v-for="group in retrievalTraceGroups"
          :key="group.key"
          class="review-audit-row"
        >
          <div class="space-y-1">
            <p class="text-base font-black">{{ group.title }}</p>
            <p v-if="group.trace" class="neo-note">
              {{ group.trace.strategy }} /
              {{ group.trace.selected_count }} /
              {{ group.trace.candidate_count }}
            </p>
          </div>

          <p v-if="group.trace?.query" class="neo-note">
            {{ group.trace.query }}
          </p>
          <p v-if="group.trace?.fallback_reason" class="neo-note">
            {{ group.trace.fallback_reason }}
          </p>

          <ul
            v-if="group.trace?.hits.length"
            class="review-trace-hit-list neo-stagger-list"
          >
            <li
              v-for="(hit, index) in group.trace?.hits"
              :key="`${group.key}-${hit.ref_id}-${index}`"
              class="review-trace-hit"
            >
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-black">{{ hit.summary || '—' }}</span>
                <code v-if="hit.ref_table" class="review-trace-code">
                  {{ hit.ref_table }}/{{ hit.ref_id || '—' }}
                </code>
              </div>
              <p class="neo-note">
                {{
                  [
                    hit.scope_type || '—',
                    hit.topic || '—',
                    formatTraceScore(hit.final_score),
                  ].join(' / ')
                }}
              </p>
              <p v-if="hit.reason" class="neo-note">{{ hit.reason }}</p>
            </li>
          </ul>

          <p v-else class="neo-note">
            {{ t('review.retrievalTraceEmpty') }}
          </p>
        </article>
      </div>
    </section>

    <div v-if="evaluationLogs.length" class="review-audit-list neo-stagger-list">
      <article
        v-for="item in evaluationLogs"
        :key="item.id"
        class="review-audit-row"
      >
        <div
          class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between"
        >
          <div class="space-y-1">
            <p class="text-base font-black">{{ item.flow_name }}</p>
            <p class="neo-note">
              {{ t('review.auditMeta') }}: {{ item.model_name || '—' }} /
              {{ item.prompt_hash || '—' }} /
              {{ item.latency_ms.toFixed(1) }} ms
            </p>
          </div>
          <p class="neo-note text-xs">
            {{ new Date(item.created_at).toLocaleString() }}
          </p>
        </div>

        <div v-if="item.runtime_trace?.entries.length" class="space-y-2">
          <p class="text-xs font-black uppercase tracking-[0.08em]">
            {{ t('review.runtimeTraceTitle') }}
          </p>
          <RuntimeTraceList :entries="item.runtime_trace.entries" />
        </div>

        <div v-if="hasEvaluationRawOutput(item.raw_output)" class="space-y-2">
          <p class="text-xs font-black uppercase tracking-[0.08em]">
            {{ t('review.auditRawOutput') }}
          </p>
          <pre
            class="max-h-72 overflow-auto border-2 border-black bg-[var(--neo-paper)] px-3 py-3 text-xs leading-6 md:border-4"
          ><code>{{ formatEvaluationRawOutput(item.raw_output) }}</code></pre>
        </div>
      </article>
    </div>

    <p v-else class="neo-note">
      {{ t('review.auditEmpty') }}
    </p>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { EvaluationLogEntry, RetrievalTrace } from '../api/client';
import NoticePanel from './NoticePanel.vue';
import RuntimeTraceList from './RuntimeTraceList.vue';
import {
  formatEvaluationRawOutput,
  hasEvaluationRawOutput,
} from '../lib/evaluationLogs';

defineProps<{
  retrievalTrace: RetrievalTrace | null | undefined;
  retrievalTraceGroups: Array<{
    key: string;
    title: string;
    trace: {
      strategy: string;
      selected_count: number;
      candidate_count: number;
      query?: string;
      fallback_reason?: string;
      hits: Array<{
        ref_id?: string;
        ref_table?: string;
        summary?: string;
        scope_type?: string;
        topic?: string;
        final_score?: number;
        reason?: string;
      }>;
    } | null;
  }>;
  evaluationLogs: EvaluationLogEntry[];
  isLoadingEvaluationLogs: boolean;
  evaluationLogsError: string;
  formatTraceScore: (value?: number) => string;
}>();

const { t } = useI18n();
</script>

<style scoped>
.review-audit-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.review-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.review-audit-row {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
}

.review-trace-panel,
.review-trace-hit-list,
.review-trace-groups,
.review-audit-list {
  display: grid;
  gap: 1rem;
}

.review-trace-meta {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  justify-content: space-between;
}

.review-trace-hit {
  background: color-mix(in srgb, var(--neo-paper) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.5rem;
  padding: 0.9rem 1rem;
}

.review-trace-code {
  background: color-mix(in srgb, var(--neo-surface) 92%, transparent);
  border: 2px solid var(--neo-border);
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 800;
  padding: 0.2rem 0.45rem;
}
</style>
