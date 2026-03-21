<template>
  <TransitionGroup
    name="neo-toast"
    tag="div"
    class="fixed bottom-6 right-6 z-50 flex flex-col gap-3"
  >
    <div
      v-for="toast in toasts"
      :key="toast.id"
      class="neo-panel flex items-start gap-3 border-l-8 px-4 py-3"
      :style="{ borderLeftColor: `var(${toneColor(toast.tone)})` }"
    >
      <p class="text-sm font-bold">{{ toast.message }}</p>
      <button
        type="button"
        class="ml-auto shrink-0 text-sm font-black"
        @click="dismiss(toast.id)"
      >
        &times;
      </button>
    </div>
  </TransitionGroup>
</template>

<script setup lang="ts">
import { useToast } from '../lib/useToast';

const { toasts, dismiss } = useToast();

function toneColor(tone: 'success' | 'error' | 'info'): string {
  switch (tone) {
    case 'success':
      return '--neo-green';
    case 'error':
      return '--neo-red';
    default:
      return '--neo-blue';
  }
}
</script>
