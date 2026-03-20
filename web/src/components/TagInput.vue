<template>
  <div>
    <div
      class="neo-input flex flex-wrap items-center gap-2 !py-2"
      @click="focusInput"
    >
      <span
        v-for="(tag, index) in modelValue"
        :key="tag"
        class="inline-flex items-center gap-1 border-2 border-black bg-[var(--neo-yellow)] px-2 py-0.5 text-sm font-bold"
      >
        {{ tag }}
        <button
          type="button"
          class="ml-0.5 text-xs font-black leading-none hover:text-[var(--neo-red)]"
          @click.stop="remove(index)"
        >
          &times;
        </button>
      </span>

      <input
        ref="inputRef"
        v-model="inputValue"
        class="min-w-[120px] flex-1 border-none bg-transparent text-sm outline-none"
        :placeholder="modelValue.length ? '' : placeholder"
        @keydown="onKeydown"
        @blur="commitInput"
      />
    </div>

    <div
      v-if="filteredSuggestions.length && inputValue.trim()"
      class="mt-1 border-2 border-black bg-white shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] md:border-4"
    >
      <button
        v-for="item in filteredSuggestions"
        :key="item"
        type="button"
        class="block w-full px-3 py-2 text-left text-sm font-semibold hover:bg-[var(--neo-yellow)]"
        @mousedown.prevent="addTag(item)"
      >
        {{ item }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';

const props = withDefaults(
  defineProps<{
    modelValue: string[];
    placeholder?: string;
    suggestions?: string[];
  }>(),
  {
    placeholder: '',
    suggestions: () => [],
  },
);

const emit = defineEmits<{
  'update:modelValue': [value: string[]];
}>();

const inputRef = ref<HTMLInputElement | null>(null);
const inputValue = ref('');

const filteredSuggestions = computed(() => {
  const query = inputValue.value.trim().toLowerCase();
  if (!query) return [];
  const existing = new Set(props.modelValue.map((t) => t.toLowerCase()));
  return props.suggestions.filter(
    (s) => s.toLowerCase().includes(query) && !existing.has(s.toLowerCase()),
  );
});

function focusInput() {
  inputRef.value?.focus();
}

function addTag(tag: string) {
  const trimmed = tag.trim();
  if (!trimmed) return;
  if (props.modelValue.some((t) => t.toLowerCase() === trimmed.toLowerCase()))
    return;
  emit('update:modelValue', [...props.modelValue, trimmed]);
  inputValue.value = '';
}

function remove(index: number) {
  const next = [...props.modelValue];
  next.splice(index, 1);
  emit('update:modelValue', next);
}

function commitInput() {
  if (inputValue.value.trim()) {
    addTag(inputValue.value);
  }
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' || event.key === 'Tab' || event.key === ',') {
    if (inputValue.value.trim()) {
      event.preventDefault();
      addTag(inputValue.value);
    }
  } else if (
    event.key === 'Backspace' &&
    !inputValue.value &&
    props.modelValue.length
  ) {
    remove(props.modelValue.length - 1);
  }
}
</script>
