"use client";

import { AnimatePresence, motion } from "framer-motion";
import { usePathname, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, useState } from "react";

import { styled } from "@/styled-system/jsx";

const TRICKLE_INTERVAL_MS = 150;
const FINISH_FADE_MS = 200;
const SAFETY_TIMEOUT_MS = 8000;
// Client-side navigations to prefetched routes can resolve in a single tick,
// so without a floor the bar would start and finish before ever painting.
const MIN_VISIBLE_MS = 350;

function RouteProgressBar() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const routeKey = `${pathname}?${searchParams.toString()}`;

  const [progress, setProgress] = useState(0);
  const [visible, setVisible] = useState(false);

  const trickleRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const safetyRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const minVisibleRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const startedAtRef = useRef(0);
  const previousKeyRef = useRef(routeKey);

  useEffect(() => {
    return () => {
      if (trickleRef.current) clearInterval(trickleRef.current);
      if (safetyRef.current) clearTimeout(safetyRef.current);
      if (minVisibleRef.current) clearTimeout(minVisibleRef.current);
    };
  }, []);

  function start() {
    if (trickleRef.current) return;

    if (minVisibleRef.current) {
      clearTimeout(minVisibleRef.current);
      minVisibleRef.current = null;
    }

    startedAtRef.current = Date.now();
    setVisible(true);
    setProgress(15);

    trickleRef.current = setInterval(() => {
      setProgress((p) => (p >= 90 ? p : p + (90 - p) * 0.15));
    }, TRICKLE_INTERVAL_MS);

    safetyRef.current = setTimeout(complete, SAFETY_TIMEOUT_MS);
  }

  function finish() {
    const elapsed = Date.now() - startedAtRef.current;
    const remaining = MIN_VISIBLE_MS - elapsed;

    if (remaining > 0) {
      minVisibleRef.current = setTimeout(complete, remaining);
    } else {
      complete();
    }
  }

  function complete() {
    if (trickleRef.current) {
      clearInterval(trickleRef.current);
      trickleRef.current = null;
    }
    if (safetyRef.current) {
      clearTimeout(safetyRef.current);
      safetyRef.current = null;
    }
    if (minVisibleRef.current) {
      clearTimeout(minVisibleRef.current);
      minVisibleRef.current = null;
    }

    setProgress(100);
    setTimeout(() => setVisible(false), FINISH_FADE_MS);
  }

  useEffect(() => {
    if (previousKeyRef.current !== routeKey) {
      previousKeyRef.current = routeKey;
      finish();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [routeKey]);

  useEffect(() => {
    function onClick(event: MouseEvent) {
      if (event.button !== 0) return;
      if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;

      const anchor = (event.target as HTMLElement)?.closest("a");
      if (!anchor) return;
      if (anchor.target && anchor.target !== "_self") return;
      if (anchor.hasAttribute("download")) return;

      const href = anchor.getAttribute("href");
      if (!href || href.startsWith("#")) return;

      let url: URL;
      try {
        url = new URL(href, window.location.href);
      } catch {
        return;
      }

      if (url.origin !== window.location.origin) return;
      if (url.pathname + url.search === window.location.pathname + window.location.search) return;

      start();
    }

    function onPopState() {
      start();
    }

    document.addEventListener("click", onClick);
    window.addEventListener("popstate", onPopState);
    return () => {
      document.removeEventListener("click", onClick);
      window.removeEventListener("popstate", onPopState);
    };
  }, []);

  return (
    <AnimatePresence>
      {visible && (
        <styled.div
          position="fixed"
          top="0"
          left="0"
          right="0"
          height="0.5"
          zIndex="banner"
          pointerEvents="none"
          overflow="hidden"
        >
          <motion.div
            style={{
              height: "100%",
              width: "100%",
              transformOrigin: "left",
              // Flat ramp (not the muted dark-mode ramp) so it stays visible in both themes.
              background: "var(--accent-colour-flat-fill-600)",
              boxShadow: "0 0 var(--spacing-2) var(--accent-colour-flat-fill-600)",
            }}
            initial={{ scaleX: 0 }}
            animate={{ scaleX: progress / 100 }}
            exit={{ opacity: 0, transition: { duration: 0.15, ease: "easeIn" } }}
            transition={{ duration: 0.2, ease: "easeOut" }}
          />
        </styled.div>
      )}
    </AnimatePresence>
  );
}

export function RouteProgress() {
  return (
    <Suspense fallback={null}>
      <RouteProgressBar />
    </Suspense>
  );
}
