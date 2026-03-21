<template>
  <div class="neo-panel space-y-4 bg-white">
    <div>
      <p class="neo-kicker bg-[var(--neo-blue)]">{{ kicker }}</p>
      <h3 class="text-xl font-black uppercase tracking-[0.06em]">
        {{ title }}
      </h3>
      <p class="mt-2 text-base font-semibold">{{ description }}</p>
    </div>

    <ol class="neo-stagger-list space-y-2">
      <li
        v-for="(step, index) in steps"
        :key="step"
        class="flex items-center gap-3 border-2 border-black bg-[var(--neo-paper)] px-3 py-3 md:border-4"
      >
        <span
          class="inline-flex h-7 w-7 items-center justify-center border-2 border-black text-xs font-black md:border-4"
          :class="index <= activeIndex ? 'bg-[var(--neo-yellow)]' : 'bg-white'"
        >
          {{ index + 1 }}
        </span>
        <span class="text-sm font-semibold">{{ step }}</span>
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
