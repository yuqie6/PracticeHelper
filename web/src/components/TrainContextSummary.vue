<template>
  <aside class="train-side neo-stagger-list">
    <section class="neo-panel train-side-panel">
      <p class="neo-kicker bg-[var(--neo-blue)]">
        {{ focusKicker }}
      </p>
      <h2 class="train-section-title">{{ focusTitle }}</h2>
      <p class="neo-note">{{ focusHint }}</p>
    </section>

    <section class="neo-panel train-side-panel">
      <p class="neo-kicker bg-[var(--neo-green)]">
        {{ t('train.fields.jobTarget') }}
      </p>
      <h2 class="train-section-title">
        {{ jobTargetTitle }}
      </h2>
      <p class="neo-note">
        {{ jobTargetDescription }}
      </p>
    </section>

    <section class="neo-panel train-side-panel">
      <p class="neo-kicker bg-[var(--neo-yellow)]">
        {{ t('train.fields.promptSet') }}
      </p>
      <h2 class="train-section-title">
        {{ promptSetTitle }}
      </h2>
      <p class="neo-note">
        {{ promptSetDescription }}
      </p>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

defineProps<{
  focusKicker: string;
  focusTitle: string;
  focusHint: string;
  jobTargetTitle: string;
  jobTargetDescription: string;
  promptSetTitle: string;
  promptSetDescription: string;
}>();

const { t } = useI18n();
</script>

<style scoped>
.train-side {
  display: grid;
  gap: 1rem;
}

.train-side-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.train-section-title {
  font-size: 1.35rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

@media (min-width: 1280px) {
  .train-side {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
