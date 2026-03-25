<template>
  <aside class="profile-side neo-stagger-list">
    <section
      v-if="savedSummary"
      class="neo-panel profile-side-panel bg-[var(--neo-green)]"
    >
      <p class="neo-kicker bg-white">{{ t('profile.saveSuccess') }}</p>
      <h2 class="profile-section-title">{{ t('profile.saveSuccess') }}</h2>
      <p class="neo-note">{{ savedSummary }}</p>
      <div class="profile-side-actions">
        <RouterLink to="/train" class="neo-button-dark w-full">
          {{ t('common.start') }}
        </RouterLink>
        <RouterLink to="/" class="neo-button w-full bg-white">
          {{ t('app.nav.home') }}
        </RouterLink>
      </div>
    </section>

    <section class="neo-panel profile-side-panel">
      <p class="neo-kicker bg-[var(--neo-yellow)]">
        {{ t('profile.fields.systemWeaknesses') }}
      </p>
      <h2 class="profile-section-title">
        {{ t('profile.fields.linkedProjects') }}
      </h2>

      <div v-if="systemWeaknesses.length" class="profile-badge-cloud">
        <span v-for="w in systemWeaknesses" :key="w.id" class="profile-badge">
          {{ w.label }}
          <span class="text-xs font-bold opacity-60">{{ w.severity.toFixed(1) }}</span>
        </span>
      </div>
      <p v-else class="neo-note">{{ t('profile.noSystemWeaknesses') }}</p>

      <div v-if="projects.length" class="profile-badge-cloud">
        <RouterLink
          v-for="p in projects"
          :key="p.id"
          to="/projects"
          class="profile-link-chip"
        >
          {{ p.name }}
        </RouterLink>
      </div>
      <p v-else class="neo-note">
        {{ t('profile.noProjects') }}
        <RouterLink to="/projects" class="font-black underline">
          {{ t('profile.goImportProject') }}
        </RouterLink>
      </p>
    </section>

    <section class="neo-panel profile-side-panel">
      <p class="neo-kicker bg-[var(--neo-blue)]">
        {{ t('profile.hero.kicker') }}
      </p>
      <h2 class="profile-section-title">{{ targetRole || '—' }}</h2>
      <p class="neo-note">
        {{
          t('profile.hero.returningDescription', {
            role: targetRole || '—',
            company: companyType || '—',
            stage: currentStage || '—',
          })
        }}
      </p>
      <div class="profile-side-actions">
        <button
          type="submit"
          class="neo-button-dark w-full"
          :disabled="isSaving"
        >
          {{
            isSaving
              ? t('common.saving')
              : isReturningUser
                ? t('profile.saveAction')
                : t('profile.saveAndTrain')
          }}
        </button>
      </div>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';

import type { ProjectProfile, WeaknessTag } from '../api/client';

defineProps<{
  savedSummary: string;
  systemWeaknesses: WeaknessTag[];
  projects: ProjectProfile[];
  targetRole: string;
  companyType: string;
  currentStage: string;
  isSaving: boolean;
  isReturningUser: boolean;
}>();

const { t } = useI18n();
</script>

<style scoped>
.profile-side {
  display: grid;
  gap: 1rem;
}

.profile-side-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.profile-section-title {
  font-size: 1.35rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.profile-badge-cloud,
.profile-side-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.profile-badge,
.profile-link-chip {
  align-items: center;
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: inline-flex;
  gap: 0.5rem;
  min-height: 2.75rem;
  padding: 0.65rem 0.9rem;
}

.profile-link-chip {
  transition:
    transform var(--motion-duration-base) var(--motion-ease-standard),
    box-shadow var(--motion-duration-base) var(--motion-ease-standard),
    background-color var(--motion-duration-fast) var(--motion-ease-soft);
}

.profile-link-chip:hover {
  box-shadow: 5px 5px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(var(--motion-lift-sm), var(--motion-lift-sm));
}

@media (min-width: 1280px) {
  .profile-side {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
