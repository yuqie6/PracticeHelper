<template>
  <header
    class="neo-panel-hero profile-stage"
    :class="isReturningUser ? 'bg-[var(--neo-blue)]' : 'bg-[var(--neo-yellow)]'"
  >
    <div class="profile-stage-copy">
      <p class="neo-kicker bg-white">{{ t('profile.hero.kicker') }}</p>
      <h1 class="profile-stage-title">
        {{ isReturningUser ? t('profile.hero.returningTitle') : t('profile.hero.newUserTitle') }}
      </h1>
      <p class="profile-stage-note">
        {{
          isReturningUser
            ? t('profile.hero.returningDescription', {
                role: targetRole || '—',
                company: companyType || '—',
                stage: currentStage || '—',
              })
            : t('profile.hero.newUserDescription')
        }}
      </p>
    </div>

    <div class="profile-stage-stats">
      <article v-for="item in stats" :key="item.label" class="profile-stage-stat">
        <span>{{ item.value }}</span>
        <small>{{ item.label }}</small>
      </article>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

defineProps<{
  isReturningUser: boolean;
  targetRole: string;
  companyType: string;
  currentStage: string;
  stats: Array<{
    label: string;
    value: string | number;
  }>;
}>();

const { t } = useI18n();
</script>

<style scoped>
.profile-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
}

.profile-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.profile-stage-copy,
.profile-stage-stats {
  position: relative;
  z-index: 1;
}

.profile-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.profile-stage-title {
  font-size: clamp(2.1rem, 6vw, 4.6rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.95;
  margin: 0;
  max-width: 12ch;
  text-transform: uppercase;
}

.profile-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 38rem;
}

.profile-stage-stats {
  display: grid;
  gap: 0.75rem;
}

.profile-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.profile-stage-stat span {
  font-size: clamp(2.4rem, 8vw, 4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.profile-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

@media (min-width: 768px) {
  .profile-stage-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .profile-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.1fr) minmax(18rem, 0.9fr);
  }
}
</style>
