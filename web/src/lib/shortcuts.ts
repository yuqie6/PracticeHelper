export interface SubmitShortcutEventLike {
  key: string;
  ctrlKey: boolean;
  altKey?: boolean;
  shiftKey?: boolean;
  metaKey?: boolean;
  isComposing?: boolean;
}

export function isSubmitShortcut(event: SubmitShortcutEventLike): boolean {
  return (
    event.key === 'Enter' &&
    event.ctrlKey &&
    !event.altKey &&
    !event.shiftKey &&
    !event.metaKey &&
    !event.isComposing
  );
}
