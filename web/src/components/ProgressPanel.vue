<template>
  <div class="neo-panel progress-panel bg-white">
    <div class="progress-head">
      <p class="neo-kicker bg-[var(--neo-blue)]">{{ kicker }}</p>
      <h3 class="progress-title">{{ title }}</h3>
      <p class="progress-note">{{ description }}</p>
    </div>

    <ol class="progress-list">
      <li
        v-for="(step, index) in steps"
        :key="step"
        class="progress-row"
        :class="index <= activeIndex ? 'progress-row-active' : ''"
      >
        <span class="progress-index">
          {{ index + 1 }}
        </span>
        <span class="progress-label">{{ step }}</span>
      </li>
    </ol>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  kicker: string;
  title: string;
  description: string;
  steps: string[];
  activeIndex: number;
}>();
</script>

<style scoped>
.progress-panel {
  display: grid;
  gap: 1rem;
}

.progress-head {
  display: grid;
  gap: 0.6rem;
}

.progress-title {
  font-size: 1.1rem;
  font-weight: 900;
  letter-spacing: -0.03em;
  line-height: 1.15;
  margin: 0;
  text-transform: uppercase;
}

.progress-note {
  font-size: 0.95rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
}

.progress-list {
  display: grid;
  gap: 0.75rem;
  margin: 0;
  padding: 0;
}

.progress-row {
  align-items: center;
  background: color-mix(in srgb, var(--neo-paper) 86%, transparent);
  border: 2px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: grid;
  gap: 0.85rem;
  grid-template-columns: auto minmax(0, 1fr);
  list-style: none;
  padding: 0.9rem 1rem;
}

.progress-row-active {
  background: color-mix(in srgb, var(--neo-yellow) 42%, var(--neo-surface));
  border-color: var(--neo-border);
}

.progress-index {
  align-items: center;
  background: var(--neo-surface);
  border: 2px solid var(--neo-border);
  display: inline-flex;
  font-size: 0.75rem;
  font-weight: 900;
  height: 2rem;
  justify-content: center;
  width: 2rem;
}

.progress-row-active .progress-index {
  background: var(--neo-yellow);
}

.progress-label {
  font-size: 0.92rem;
  font-weight: 700;
  line-height: 1.5;
}
</style>
