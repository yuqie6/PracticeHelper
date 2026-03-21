<template>
  <div
    class="neo-panel space-y-2"
    :class="tone === 'error' ? 'bg-[var(--neo-red)] text-black' : 'bg-[var(--neo-yellow)] text-black'"
    role="alert"
  >
    <div class="flex items-start justify-between gap-3">
      <p class="neo-kicker bg-white">{{ title }}</p>
      <button
        v-if="dismissible"
        type="button"
        class="text-lg font-black leading-none"
        @click="$emit('dismiss')"
      >
        &times;
      </button>
    </div>
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
  dismissible?: boolean;
}>();

defineEmits<{
  dismiss: [];
}>();
</script>
