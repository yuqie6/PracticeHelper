<template>
  <section class="neo-panel projects-editor-panel">
    <div class="projects-section-head">
      <div class="space-y-2">
        <p class="neo-kicker bg-[var(--neo-red)]">
          {{ t('projects.editorTitle') }}
        </p>
        <h2 class="projects-section-title">
          {{ project.name }}
        </h2>
      </div>
      <span class="neo-badge bg-white">
        {{ formatImportStatusLabel(t, project.import_status) }}
      </span>
    </div>

    <p class="neo-note projects-editor-summary">
      {{ project.summary || project.repo_url || t('projects.hero.description') }}
    </p>

    <div class="projects-metadata-grid">
      <article class="projects-metadata-card">
        <span class="projects-metadata-label">{{ t('projects.meta.repo') }}</span>
        <p class="projects-metadata-value break-all">{{ project.repo_url }}</p>
      </article>
      <article class="projects-metadata-card">
        <span class="projects-metadata-label">{{ t('projects.meta.branch') }}</span>
        <p class="projects-metadata-value">
          {{ project.default_branch || t('common.notProvided') }}
        </p>
      </article>
      <article class="projects-metadata-card">
        <span class="projects-metadata-label">{{ t('projects.meta.commit') }}</span>
        <p class="projects-metadata-value">
          {{ shortCommit }}
        </p>
      </article>
    </div>

    <form class="projects-editor-form" @submit.prevent="emit('submit')">
      <div class="projects-editor-grid">
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('projects.fields.name') }}</span>
          <input v-model="editor.name" class="neo-input" />
        </label>

        <label class="space-y-2">
          <span class="neo-subheading">
            {{ t('projects.fields.techStack') }}
          </span>
          <input v-model="editor.tech_stack" class="neo-input" />
        </label>
      </div>

      <label class="space-y-2">
        <span class="neo-subheading">{{ t('projects.fields.summary') }}</span>
        <textarea v-model="editor.summary" class="neo-textarea" />
      </label>

      <div class="projects-detail-grid">
        <label class="space-y-2">
          <span class="neo-subheading">
            {{ t('projects.fields.highlights') }}
          </span>
          <textarea v-model="editor.highlights" class="neo-textarea" />
        </label>

        <label class="space-y-2">
          <span class="neo-subheading">
            {{ t('projects.fields.challenges') }}
          </span>
          <textarea v-model="editor.challenges" class="neo-textarea" />
        </label>

        <label class="space-y-2">
          <span class="neo-subheading">
            {{ t('projects.fields.tradeoffs') }}
          </span>
          <textarea v-model="editor.tradeoffs" class="neo-textarea" />
        </label>

        <label class="space-y-2">
          <span class="neo-subheading">
            {{ t('projects.fields.ownership') }}
          </span>
          <textarea v-model="editor.ownership_points" class="neo-textarea" />
        </label>
      </div>

      <label class="space-y-2">
        <span class="neo-subheading">
          {{ t('projects.fields.followups') }}
        </span>
        <textarea v-model="editor.followup_points" class="neo-textarea" />
      </label>

      <button
        class="neo-button-dark w-full sm:w-auto"
        type="submit"
        :disabled="isUpdating"
      >
        {{ isUpdating ? t('common.saving') : t('projects.saveAction') }}
      </button>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { ProjectProfile } from '../api/client';
import { formatImportStatusLabel } from '../lib/labels';

type ProjectEditorState = {
  name: string;
  summary: string;
  tech_stack: string;
  highlights: string;
  challenges: string;
  tradeoffs: string;
  ownership_points: string;
  followup_points: string;
};

const props = defineProps<{
  project: ProjectProfile;
  editor: ProjectEditorState;
  isUpdating: boolean;
}>();

const emit = defineEmits<{
  (event: 'submit'): void;
}>();

const { t } = useI18n();

const shortCommit = computed(() => {
  if (!props.project.import_commit) {
    return t('common.notProvided');
  }

  return props.project.import_commit.slice(0, 8);
});
</script>

<style scoped>
.projects-editor-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.projects-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.projects-section-title {
  font-size: 1.4rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.projects-editor-summary {
  line-height: 1.7;
  margin: 0;
  max-width: 54rem;
}

.projects-metadata-grid,
.projects-editor-grid,
.projects-detail-grid,
.projects-editor-form {
  display: grid;
  gap: 1rem;
}

.projects-metadata-grid {
  grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
}

.projects-metadata-card {
  background: color-mix(in srgb, var(--neo-surface) 86%, transparent);
  border: 2px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: grid;
  gap: 0.35rem;
  padding: 0.9rem 1rem;
}

.projects-metadata-label {
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  opacity: 0.65;
  text-transform: uppercase;
}

.projects-metadata-value {
  font-size: 0.95rem;
  font-weight: 700;
  line-height: 1.5;
  margin: 0;
}

@media (min-width: 768px) {
  .projects-editor-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .projects-detail-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
