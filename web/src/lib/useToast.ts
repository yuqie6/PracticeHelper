import { ref } from 'vue';

export interface Toast {
  id: number;
  message: string;
  tone: 'success' | 'error' | 'info';
}

const toasts = ref<Toast[]>([]);
let nextId = 0;

export function useToast() {
  function show(message: string, tone: Toast['tone'] = 'info') {
    const id = nextId++;
    toasts.value = [...toasts.value, { id, message, tone }];
    window.setTimeout(() => dismiss(id), 3000);
  }

  function dismiss(id: number) {
    toasts.value = toasts.value.filter((t) => t.id !== id);
  }

  return { toasts, show, dismiss };
}
