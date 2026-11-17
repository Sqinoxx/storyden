"use client";

import { useSyncExternalStore } from "react";

export type ExternalDragKind = "files" | "link";

const LINK_TYPES = ["downloadurl", "text/uri-list", "text/x-moz-url"];

// Browsers fire dragover continuously while something hovers the page, but
// give no reliable signal when a drag leaves the window or is cancelled, so
// the state lapses once the stream of dragover events stops.
const IDLE_MS = 350;

let kind: ExternalDragKind | null = null;
let internal = false;
let idleTimer: ReturnType<typeof setTimeout> | undefined;
let installed = false;
const listeners = new Set<() => void>();

function set(next: ExternalDragKind | null) {
  if (next === kind) return;
  kind = next;
  listeners.forEach((l) => l());
}

export function classifyDrag(
  dt: DataTransfer | null,
): ExternalDragKind | null {
  if (!dt || internal) return null;

  const types = Array.from(dt.types ?? []).map((t) => t.toLowerCase());
  if (types.includes("files")) return "files";
  if (types.some((t) => LINK_TYPES.includes(t))) return "link";

  return null;
}

export function isInternalDrag() {
  return internal;
}

function reset() {
  clearTimeout(idleTimer);
  set(null);
}

function install() {
  if (installed || typeof window === "undefined") return;
  installed = true;

  window.addEventListener(
    "dragstart",
    () => {
      internal = true;
    },
    true,
  );

  const end = () => {
    internal = false;
    reset();
  };
  window.addEventListener("dragend", end, true);
  window.addEventListener("drop", end, true);

  const track = (e: DragEvent) => {
    const next = classifyDrag(e.dataTransfer);
    if (!next) return;

    set(next);
    clearTimeout(idleTimer);
    idleTimer = setTimeout(reset, IDLE_MS);
  };
  window.addEventListener("dragenter", track, true);
  window.addEventListener("dragover", track, true);
}

function subscribe(listener: () => void) {
  install();
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function useExternalDrag(): ExternalDragKind | null {
  return useSyncExternalStore(
    subscribe,
    () => kind,
    () => null,
  );
}
