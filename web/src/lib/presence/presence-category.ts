import { useEffect, useSyncExternalStore } from "react";

export type PresenceCategory = {
  id: string;
  name: string;
};

let current: PresenceCategory | undefined;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((l) => l());
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getSnapshot() {
  return current;
}

export function useCurrentPresenceCategory() {
  return useSyncExternalStore(subscribe, getSnapshot, () => undefined);
}

export function usePresenceCategory(category?: PresenceCategory) {
  const id = category?.id;
  const name = category?.name;

  useEffect(() => {
    if (!id || !name) return;

    const next = { id, name };
    current = next;
    emit();

    return () => {
      if (current === next) {
        current = undefined;
        emit();
      }
    };
  }, [id, name]);
}
