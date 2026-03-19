<template>
  <div class="neo-panel bg-white space-y-4">
    <div>
      <p class="neo-kicker bg-[var(--neo-blue)]">{{ kicker }}</p>
      <h3 class="text-xl font-black uppercase tracking-[0.06em]">{{ title }}</h3>
      <p class="mt-2 text-base font-semibold">{{ description }}</p>
    </div>

    <article
      v-for="(section, index) in sections"
      :key="section.id"
      class="space-y-4 border-2 border-black bg-[var(--neo-paper)] px-3 py-3 md:border-4"
    >
      <div class="space-y-2">
        <p class="text-xs font-black uppercase tracking-[0.08em] text-black/65">
          {{ t('session.streamSectionCounter', { index: index + 1 }) }}
        </p>
        <h4 class="text-lg font-black">
          {{ resolveSectionTitle(section) }}
        </h4>
        <p
          v-if="section.contexts.length"
          class="text-sm font-semibold leading-6 text-black/80"
        >
          {{ describeContexts(section.contexts) }}
        </p>
      </div>

      <div v-if="section.reasoning.length" class="space-y-2">
        <p class="neo-subheading">{{ reasoningTitle }}</p>
        <ul class="space-y-2">
          <li
            v-for="(item, reasoningIndex) in section.reasoning"
            :key="`${section.id}-reasoning-${reasoningIndex}`"
            class="border-2 border-black bg-white px-3 py-3 text-sm font-semibold md:border-4"
          >
            {{ item }}
          </li>
        </ul>
      </div>

      <div class="space-y-3">
        <p class="neo-subheading">{{ contentTitle }}</p>

        <template v-if="getQuestionPayload(section)">
          <div class="space-y-3">
            <p class="text-base font-semibold leading-7">
              {{ getQuestionPayload(section)?.question }}
            </p>
            <ul
              v-if="getQuestionPayload(section)?.expectedPoints.length"
              class="space-y-2"
            >
              <li
                v-for="item in getQuestionPayload(section)?.expectedPoints"
                :key="`${section.id}-${item}`"
                class="border-2 border-black bg-white px-3 py-2 text-sm font-semibold md:border-4"
              >
                {{ item }}
              </li>
            </ul>
          </div>
        </template>

        <template v-else-if="getEvaluationPayload(section)">
          <div class="space-y-3">
            <p class="text-lg font-black">
              {{ t('session.streamFields.score', { score: getEvaluationPayload(section)?.score }) }}
            </p>
            <div
              v-if="Object.keys(getEvaluationPayload(section)?.scoreBreakdown ?? {}).length"
              class="space-y-2"
            >
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('session.streamFields.scoreBreakdown') }}
              </p>
              <ul class="space-y-2">
                <li
                  v-for="(score, label) in getEvaluationPayload(section)?.scoreBreakdown"
                  :key="`${section.id}-score-${label}`"
                  class="flex items-center justify-between border-2 border-black bg-white px-3 py-2 text-sm font-semibold md:border-4"
                >
                  <span>{{ label }}</span>
                  <span>{{ score }}</span>
                </li>
              </ul>
            </div>
            <div v-if="getEvaluationPayload(section)?.strengths.length" class="space-y-2">
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('session.strengths') }}
              </p>
              <ul class="space-y-2">
                <li
                  v-for="item in getEvaluationPayload(section)?.strengths"
                  :key="`${section.id}-strength-${item}`"
                  class="border-2 border-black bg-white px-3 py-2 text-sm font-semibold md:border-4"
                >
                  {{ item }}
                </li>
              </ul>
            </div>
            <div v-if="getEvaluationPayload(section)?.gaps.length" class="space-y-2">
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('session.gaps') }}
              </p>
              <ul class="space-y-2">
                <li
                  v-for="item in getEvaluationPayload(section)?.gaps"
                  :key="`${section.id}-gap-${item}`"
                  class="border-2 border-black bg-white px-3 py-2 text-sm font-semibold md:border-4"
                >
                  {{ item }}
                </li>
              </ul>
            </div>
            <div
              v-if="getEvaluationPayload(section)?.followupQuestion"
              class="space-y-2"
            >
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('session.streamFields.followupQuestion') }}
              </p>
              <p class="border-2 border-black bg-white px-3 py-3 text-sm font-semibold md:border-4">
                {{ getEvaluationPayload(section)?.followupQuestion }}
              </p>
              <ul
                v-if="getEvaluationPayload(section)?.followupExpectedPoints.length"
                class="space-y-2"
              >
                <li
                  v-for="item in getEvaluationPayload(section)?.followupExpectedPoints"
                  :key="`${section.id}-followup-${item}`"
                  class="border-2 border-black bg-white px-3 py-2 text-sm font-semibold md:border-4"
                >
                  {{ item }}
                </li>
              </ul>
            </div>
          </div>
        </template>

        <template v-else-if="getReviewPayload(section)">
          <div class="space-y-3">
            <p class="text-base font-semibold leading-7">
              {{ getReviewPayload(section)?.overall }}
            </p>
            <div
              v-if="Object.keys(getReviewPayload(section)?.scoreBreakdown ?? {}).length"
              class="space-y-2"
            >
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('session.streamFields.scoreBreakdown') }}
              </p>
              <ul class="space-y-2">
                <li
                  v-for="(score, label) in getReviewPayload(section)?.scoreBreakdown"
                  :key="`${section.id}-review-score-${label}`"
                  class="flex items-center justify-between border-2 border-black bg-white px-3 py-2 text-sm font-semibold md:border-4"
                >
                  <span>{{ label }}</span>
                  <span>{{ score }}</span>
                </li>
              </ul>
            </div>
            <div v-if="getReviewPayload(section)?.highlights.length" class="space-y-2">
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('review.highlights') }}
              </p>
              <ul class="space-y-2">
                <li
                  v-for="item in getReviewPayload(section)?.highlights"
                  :key="`${section.id}-review-highlight-${item}`"
                  class="border-2 border-black bg-white px-3 py-2 text-sm font-semibold md:border-4"
                >
                  {{ item }}
                </li>
              </ul>
            </div>
            <div v-if="getReviewPayload(section)?.gaps.length" class="space-y-2">
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('review.gaps') }}
              </p>
              <ul class="space-y-2">
                <li
                  v-for="item in getReviewPayload(section)?.gaps"
                  :key="`${section.id}-review-gap-${item}`"
                  class="border-2 border-black bg-white px-3 py-2 text-sm font-semibold md:border-4"
                >
                  {{ item }}
                </li>
              </ul>
            </div>
            <div
              v-if="getReviewPayload(section)?.nextTrainingFocus.length"
              class="space-y-2"
            >
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('review.nextFocus') }}
              </p>
              <ul class="space-y-2">
                <li
                  v-for="item in getReviewPayload(section)?.nextTrainingFocus"
                  :key="`${section.id}-review-focus-${item}`"
                  class="border-2 border-black bg-white px-3 py-2 text-sm font-semibold md:border-4"
                >
                  {{ item }}
                </li>
              </ul>
            </div>
          </div>
        </template>

        <p v-else class="text-sm font-semibold text-black/70">
          {{ t('session.streamPending') }}
        </p>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  parseStreamPayload,
  type ParsedStreamPayload,
  type StreamSection,
} from '../lib/streaming';

const props = defineProps<{
  kicker: string;
  title: string;
  description: string;
  reasoningTitle: string;
  contentTitle: string;
  sections: StreamSection[];
}>();

const { t } = useI18n();

const parsedPayloadMap = computed(() =>
  Object.fromEntries(
    props.sections.map((section) => [section.id, parseStreamPayload(section.rawContent)]),
  ) as Record<string, ParsedStreamPayload | null>,
);

function getParsedPayload(section: StreamSection): ParsedStreamPayload | null {
  return parsedPayloadMap.value[section.id] ?? null;
}

function getQuestionPayload(section: StreamSection): Extract<ParsedStreamPayload, { kind: 'question' }> | null {
  const payload = getParsedPayload(section);
  return payload?.kind === 'question' ? payload : null;
}

function getEvaluationPayload(
  section: StreamSection,
): Extract<ParsedStreamPayload, { kind: 'evaluation' }> | null {
  const payload = getParsedPayload(section);
  return payload?.kind === 'evaluation' ? payload : null;
}

function getReviewPayload(section: StreamSection): Extract<ParsedStreamPayload, { kind: 'review' }> | null {
  const payload = getParsedPayload(section);
  return payload?.kind === 'review' ? payload : null;
}

function resolveSectionTitle(section: StreamSection): string {
  const payload = getParsedPayload(section);
  if (payload?.kind === 'question') {
    return t('session.streamKinds.question');
  }
  if (payload?.kind === 'evaluation') {
    return t('session.streamKinds.evaluation');
  }
  if (payload?.kind === 'review') {
    return t('session.streamKinds.review');
  }

  switch (section.phase) {
    case 'prepare_context':
      return t('session.streamKinds.prepare');
    case 'call_model':
      return t('session.streamKinds.drafting');
    case 'parse_result':
      return t('session.streamKinds.finalizing');
    default:
      return t('session.streamKinds.processing');
  }
}

function describeContexts(contexts: string[]): string {
  return contexts
    .map((item) => t(`session.streamContexts.${item}`))
    .map((item) => item.replace(/^session\.streamContexts\./, ''))
    .join(' / ');
}
</script>
