<template>
  <section class="neo-panel jobs-editor-panel">
    <div class="jobs-section-head">
      <div class="space-y-2">
        <p class="neo-kicker bg-[var(--neo-red)]">
          {{ selectedJobTarget ? t('jobs.editorTitle') : t('jobs.createTitle') }}
        </p>
        <h2 class="jobs-section-title">
          {{ selectedJobTarget?.title || t('jobs.createTitle') }}
        </h2>
      </div>
      <div v-if="selectedJobTarget" class="jobs-editor-actions">
        <button
          type="button"
          class="neo-button w-full bg-white sm:w-auto"
          :disabled="isActivating"
          @click="emit('toggle-active')"
        >
          {{ isActiveSelection ? t('jobs.clearActiveAction') : t('jobs.activateAction') }}
        </button>
        <button
          type="button"
          class="neo-button-dark w-full sm:w-auto"
          :disabled="isAnalyzing"
          @click="emit('run-analysis')"
        >
          {{
            isAnalyzing
              ? t('jobs.analyzing')
              : selectedLatestAnalysis
                ? t('jobs.reanalyzeAction')
                : t('jobs.analyzeAction')
          }}
        </button>
      </div>
    </div>

    <p v-if="selectedJobTarget && isActiveSelection" class="neo-note">
      {{ activeSelectionDescription }}
    </p>

    <form class="jobs-editor-form" @submit.prevent="emit('submit')">
      <div class="jobs-editor-grid">
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('jobs.fields.title') }}</span>
          <input
            :value="editor.title"
            class="neo-input"
            :placeholder="t('jobs.placeholders.title')"
            @input="emitField('title', $event)"
          />
        </label>

        <label class="space-y-2">
          <span class="neo-subheading">{{ t('jobs.fields.companyName') }}</span>
          <input
            :value="editor.company_name"
            class="neo-input"
            :placeholder="t('jobs.placeholders.companyName')"
            @input="emitField('company_name', $event)"
          />
        </label>
      </div>

      <label class="space-y-2">
        <span class="neo-subheading">{{ t('jobs.fields.sourceText') }}</span>
        <textarea
          :value="editor.source_text"
          class="neo-textarea min-h-[240px]"
          :placeholder="t('jobs.placeholders.sourceText')"
          @input="emitField('source_text', $event)"
        />
      </label>

      <button
        type="submit"
        class="neo-button-dark w-full sm:w-auto"
        :disabled="isSaving || isCreating"
      >
        {{ isSaving || isCreating ? t('common.saving') : t('jobs.saveAction') }}
      </button>
    </form>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { JobTarget } from '../api/client';

defineProps<{
  selectedJobTarget: JobTarget | null;
  selectedLatestAnalysis: unknown;
  editor: {
    title: string;
    company_name: string;
    source_text: string;
  };
  isActiveSelection: boolean;
  isCreating: boolean;
  isSaving: boolean;
  isAnalyzing: boolean;
  isActivating: boolean;
  activeSelectionDescription: string;
}>();

const emit = defineEmits<{
  (event: 'submit'): void;
  (event: 'run-analysis'): void;
  (event: 'toggle-active'): void;
  (event: 'update:field', payload: { field: string; value: string }): void;
}>();

const { t } = useI18n();

function emitField(field: string, event: Event) {
  emit('update:field', {
    field,
    value: (event.target as HTMLInputElement | HTMLTextAreaElement).value,
  });
}
</script>

<style scoped>
.jobs-editor-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.jobs-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.jobs-section-title {
  font-size: 1.35rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.jobs-editor-actions,
.jobs-editor-form,
.jobs-editor-grid {
  display: grid;
  gap: 1rem;
}

@media (min-width: 768px) {
  .jobs-editor-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
