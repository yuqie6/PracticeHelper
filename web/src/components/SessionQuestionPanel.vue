<template>
  <section class="neo-panel session-question-panel">
    <div class="session-section-head">
      <div class="space-y-2">
        <p class="neo-kicker bg-[var(--neo-red)]">
          {{ t('session.currentQuestion') }}
        </p>
        <h2 class="session-section-title">
          {{ turnLabel }}
        </h2>
      </div>
      <span class="neo-badge bg-white">{{ currentStatusLabel }}</span>
    </div>

    <div v-if="followupIntent" class="session-intent-box">
      <p class="neo-subheading">{{ t('session.followupIntentTitle') }}</p>
      <p class="text-sm font-semibold leading-6">{{ followupIntent }}</p>
    </div>

    <h3
      class="session-question-title"
      :class="{ 'session-question-title-dense': questionDense }"
    >
      {{ question }}
    </h3>

    <form
      v-if="canAnswerCurrentSession"
      class="session-answer-form"
      @submit.prevent="emit('submit')"
    >
      <Transition name="neo-turn" mode="out-in">
        <div v-if="showSubmittedAnswer" key="submitted" class="session-submitted-box">
          <p class="neo-subheading">
            {{ t('session.submittedAnswerTitle') }}
          </p>
          <p class="whitespace-pre-wrap text-sm font-semibold leading-6">
            {{ submittedAnswer }}
          </p>
          <p class="neo-note">
            {{ t('session.submittedAnswerDescription') }}
          </p>
        </div>
        <div v-else key="draft" class="space-y-4">
          <textarea
            :model-value="draftAnswer"
            class="neo-textarea session-answer-input"
            :placeholder="placeholderText"
            :aria-label="placeholderText"
            :disabled="isSubmitting || isBackgroundProcessing"
            @update:model-value="noop"
            @input="emitDraft($event)"
            @keydown="emit('answer-keydown', $event)"
          />
          <div class="session-answer-footer">
            <p class="neo-note">
              {{ t('session.submitShortcutHint') }}
            </p>
            <button
              type="submit"
              class="neo-button-dark w-full sm:w-auto"
              :class="{
                'animate-[neo-working_600ms_ease_infinite]': isSubmitting,
              }"
              :aria-busy="isSubmitting"
              :disabled="isSubmitting || isBackgroundProcessing"
            >
              {{ isSubmitting ? t('common.submitting') : t('common.submit') }}
            </button>
          </div>
        </div>
      </Transition>
    </form>
    <p v-else class="neo-note">
      {{ t('session.answerLockedWhileProcessing') }}
    </p>
  </section>
</template>

<script setup lang="ts">
import { Transition } from 'vue';
import { useI18n } from 'vue-i18n';

defineProps<{
  turnLabel: string;
  currentStatusLabel: string;
  followupIntent: string;
  question: string;
  questionDense: boolean;
  canAnswerCurrentSession: boolean;
  showSubmittedAnswer: boolean;
  submittedAnswer: string;
  draftAnswer: string;
  placeholderText: string;
  isSubmitting: boolean;
  isBackgroundProcessing: boolean;
}>();

const emit = defineEmits<{
  (event: 'submit'): void;
  (event: 'update:draftAnswer', value: string): void;
  (event: 'answer-keydown', value: KeyboardEvent): void;
}>();

const { t } = useI18n();

function emitDraft(event: Event) {
  emit('update:draftAnswer', (event.target as HTMLTextAreaElement).value);
}

function noop() {}
</script>

<style scoped>
.session-question-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.session-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.session-section-title {
  font-size: 1.25rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.session-intent-box,
.session-submitted-box {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
}

.session-question-title {
  font-size: clamp(1.6rem, 4vw, 2.8rem);
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1.15;
  margin: 0;
  text-wrap: balance;
}

.session-question-title-dense {
  font-size: clamp(1.2rem, 2.4vw, 1.8rem);
  line-height: 1.38;
  max-width: 34ch;
  text-wrap: pretty;
}

.session-answer-form {
  display: grid;
  gap: 1rem;
}

.session-answer-input {
  min-height: 18rem;
}

.session-answer-footer {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  justify-content: space-between;
}

@media (min-width: 768px) {
  .session-answer-footer {
    align-items: center;
    flex-direction: row;
  }
}
</style>
