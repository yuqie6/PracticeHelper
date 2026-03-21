<template>
  <div class="neo-panel stream-trace bg-white">
    <div class="stream-head">
      <p class="neo-kicker bg-[var(--neo-blue)]">{{ kicker }}</p>
      <h3 class="stream-title">
        {{ title }}
      </h3>
      <p class="stream-note">{{ description }}</p>
    </div>

    <article
      v-for="(section, index) in sections"
      :key="section.id"
      class="stream-section"
    >
      <div class="space-y-2">
        <p class="stream-counter">
          {{ t('session.streamSectionCounter', { index: index + 1 }) }}
        </p>
        <h4 class="stream-section-title">
          {{ resolveSectionTitle(section) }}
        </h4>
        <p v-if="section.contexts.length" class="stream-contexts">
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
              {{
                t('session.streamFields.score', {
                  score: getEvaluationPayload(section)?.score,
                })
              }}
            </p>
            <p
              v-if="getEvaluationPayload(section)?.headline"
              class="border-2 border-black bg-white px-3 py-3 text-sm font-semibold md:border-4"
            >
              {{ getEvaluationPayload(section)?.headline }}
            </p>
            <div
              v-if="
                Object.keys(getEvaluationPayload(section)?.scoreBreakdown ?? {})
                  .length
              "
              class="space-y-2"
            >
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('session.streamFields.scoreBreakdown') }}
              </p>
              <ul class="space-y-2">
                <li
                  v-for="(score, label) in getEvaluationPayload(section)
                    ?.scoreBreakdown"
                  :key="`${section.id}-score-${label}`"
                  class="flex items-center justify-between border-2 border-black bg-white px-3 py-2 text-sm font-semibold md:border-4"
                >
                  <span>{{ label }}</span>
                  <span>{{ score }}</span>
                </li>
              </ul>
            </div>
            <div
              v-if="getEvaluationPayload(section)?.strengths.length"
              class="space-y-2"
            >
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
            <div
              v-if="getEvaluationPayload(section)?.gaps.length"
              class="space-y-2"
            >
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
              v-if="getEvaluationPayload(section)?.suggestion"
              class="space-y-2"
            >
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('session.suggestionTitle') }}
              </p>
              <p
                class="border-2 border-black bg-white px-3 py-3 text-sm font-semibold md:border-4"
              >
                {{ getEvaluationPayload(section)?.suggestion }}
              </p>
            </div>
            <div
              v-if="getEvaluationPayload(section)?.followupIntent"
              class="space-y-2"
            >
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('session.followupIntentTitle') }}
              </p>
              <p
                class="border-2 border-black bg-white px-3 py-3 text-sm font-semibold md:border-4"
              >
                {{ getEvaluationPayload(section)?.followupIntent }}
              </p>
            </div>
            <div
              v-if="getEvaluationPayload(section)?.followupQuestion"
              class="space-y-2"
            >
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('session.streamFields.followupQuestion') }}
              </p>
              <p
                class="border-2 border-black bg-white px-3 py-3 text-sm font-semibold md:border-4"
              >
                {{ getEvaluationPayload(section)?.followupQuestion }}
              </p>
              <ul
                v-if="
                  getEvaluationPayload(section)?.followupExpectedPoints.length
                "
                class="space-y-2"
              >
                <li
                  v-for="item in getEvaluationPayload(section)
                    ?.followupExpectedPoints"
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
            <div v-if="getReviewPayload(section)?.topFix" class="space-y-2">
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('review.topFixTitle') }}
              </p>
              <p
                class="border-2 border-black bg-white px-3 py-3 text-sm font-semibold md:border-4"
              >
                {{ getReviewPayload(section)?.topFix }}
              </p>
              <p
                v-if="getReviewPayload(section)?.topFixReason"
                class="border-2 border-black bg-white px-3 py-3 text-sm font-semibold md:border-4"
              >
                {{ getReviewPayload(section)?.topFixReason }}
              </p>
            </div>
            <div
              v-if="
                Object.keys(getReviewPayload(section)?.scoreBreakdown ?? {})
                  .length
              "
              class="space-y-2"
            >
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('session.streamFields.scoreBreakdown') }}
              </p>
              <ul class="space-y-2">
                <li
                  v-for="(score, label) in getReviewPayload(section)
                    ?.scoreBreakdown"
                  :key="`${section.id}-review-score-${label}`"
                  class="flex items-center justify-between border-2 border-black bg-white px-3 py-2 text-sm font-semibold md:border-4"
                >
                  <span>{{ label }}</span>
                  <span>{{ score }}</span>
                </li>
              </ul>
            </div>
            <div
              v-if="getReviewPayload(section)?.highlights.length"
              class="space-y-2"
            >
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
            <div
              v-if="getReviewPayload(section)?.gaps.length"
              class="space-y-2"
            >
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
            <div
              v-if="getReviewPayload(section)?.recommendedNext"
              class="space-y-2"
            >
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('review.recommendedNextTitle') }}
              </p>
              <p
                class="border-2 border-black bg-white px-3 py-3 text-sm font-semibold md:border-4"
              >
                {{ getReviewPayload(section)?.recommendedNext?.reason }}
              </p>
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

const parsedPayloadMap = computed(
  () =>
    Object.fromEntries(
      props.sections.map((section) => [
        section.id,
        parseStreamPayload(section.rawContent),
      ]),
    ) as Record<string, ParsedStreamPayload | null>,
);

function getParsedPayload(section: StreamSection): ParsedStreamPayload | null {
  return parsedPayloadMap.value[section.id] ?? null;
}

function getQuestionPayload(
  section: StreamSection,
): Extract<ParsedStreamPayload, { kind: 'question' }> | null {
  const payload = getParsedPayload(section);
  return payload?.kind === 'question' ? payload : null;
}

function getEvaluationPayload(
  section: StreamSection,
): Extract<ParsedStreamPayload, { kind: 'evaluation' }> | null {
  const payload = getParsedPayload(section);
  return payload?.kind === 'evaluation' ? payload : null;
}

function getReviewPayload(
  section: StreamSection,
): Extract<ParsedStreamPayload, { kind: 'review' }> | null {
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

<style scoped>
.stream-trace {
  display: grid;
  gap: 1rem;
}

.stream-head {
  display: grid;
  gap: 0.6rem;
}

.stream-title {
  font-size: 1.1rem;
  font-weight: 900;
  letter-spacing: -0.03em;
  line-height: 1.15;
  margin: 0;
  text-transform: uppercase;
}

.stream-note {
  font-size: 0.95rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
}

.stream-section {
  background: color-mix(in srgb, var(--neo-paper) 88%, transparent);
  border: 2px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: grid;
  gap: 1rem;
  padding: 1rem;
}

.stream-counter {
  color: color-mix(in srgb, var(--neo-text) 64%, transparent);
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  margin: 0;
  text-transform: uppercase;
}

.stream-section-title {
  font-size: 1rem;
  font-weight: 900;
  letter-spacing: -0.02em;
  line-height: 1.3;
  margin: 0;
}

.stream-contexts {
  font-size: 0.9rem;
  font-weight: 700;
  line-height: 1.6;
  margin: 0;
  opacity: 0.82;
}
</style>
