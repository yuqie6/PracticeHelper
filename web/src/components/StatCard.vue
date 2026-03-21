<template>
  <article class="neo-panel">
    <p class="neo-kicker" :class="kickerClass">{{ kicker }}</p>
    <h3 class="text-xl font-black uppercase tracking-[0.05em]">{{ title }}</h3>
    <p class="neo-note mt-3 leading-6">
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
