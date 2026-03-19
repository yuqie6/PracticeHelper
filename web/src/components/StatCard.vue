<template>
  <article class="neo-panel">
    <p class="neo-kicker" :class="kickerClass">{{ kicker }}</p>
    <h3 class="text-xl font-black uppercase tracking-[0.05em]">{{ title }}</h3>
    <p class="mt-3 text-sm font-semibold leading-6 text-black/80">
      {{ description }}
    </p>
  </article>
</template>

<script setup lang="ts">
defineProps<{
  kicker: string;
  title: string;
  description: string;
  kickerClass?: string;
}>();
</script>
