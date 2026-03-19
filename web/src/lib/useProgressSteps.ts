import { computed, onBeforeUnmount, ref, watch, type Ref } from 'vue';

export interface ProgressStep {
  afterMs: number;
  label: string;
}

export function useProgressSteps(active: Ref<boolean>, steps: Ref<ProgressStep[]>) {
  const startedAt = ref<number | null>(null);
  const now = ref(Date.now());
  let timer: number | null = null;

  const stop = () => {
    if (timer != null) {
      window.clearInterval(timer);
      timer = null;
    }
  };

  watch(
    active,
    (value) => {
      if (!value) {
        startedAt.value = null;
        stop();
        return;
      }

      startedAt.value = Date.now();
      now.value = startedAt.value;
      stop();
      timer = window.setInterval(() => {
        now.value = Date.now();
      }, 800);
    },
    { immediate: true },
  );

  onBeforeUnmount(stop);

  const activeIndex = computed(() => {
    if (!active.value || startedAt.value == null) {
      return 0;
    }

    const elapsed = now.value - startedAt.value;
    let index = 0;
    for (const [stepIndex, step] of steps.value.entries()) {
      if (elapsed >= step.afterMs) {
        index = stepIndex;
      }
    }

    return index;
  });

  return { activeIndex };
}
