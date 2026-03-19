<template>
  <div
    class="neo-panel space-y-2"
    :class="tone === 'error' ? 'bg-[var(--neo-red)] text-black' : 'bg-[var(--neo-yellow)] text-black'"
  >
    <p class="neo-kicker bg-white">{{ title }}</p>
    <p class="text-sm font-semibold leading-6">
      {{ message }}
    </p>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  message: string;
  tone?: 'error' | 'info';
}>();
</script>
