"use client";

import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import { useTranslation } from "@/lib/i18n";
import { css } from "@/styled-system/css";
import { Box } from "@/styled-system/jsx";

const SEQUENCE = [
  "arrowup",
  "arrowup",
  "arrowdown",
  "arrowdown",
  "arrowleft",
  "arrowright",
  "arrowleft",
  "arrowright",
  "b",
  "a",
];

const EMOJIS = ["🎉", "✨", "🦄", "📜", "🎮", "🌟"];
const PARTICLE_COUNT = 30;
const DURATION_MS = 4000;

const particles = Array.from({ length: PARTICLE_COUNT }, (_, i) => ({
  emoji: EMOJIS[i % EMOJIS.length],
  left: `${(i * 37) % 100}%`,
  delay: `${((i * 53) % 1500) / 1000}s`,
  duration: `${2 + ((i * 29) % 15) / 10}s`,
}));

const particleStyles = css({
  position: "absolute",
  top: "0",
  fontSize: "2xl",
  animationName: "easterEggFall",
  animationTimingFunction: "default",
  animationFillMode: "forwards",
  opacity: "0",
});

function isTypingTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false;
  return (
    target.isContentEditable ||
    target.tagName === "INPUT" ||
    target.tagName === "TEXTAREA" ||
    target.tagName === "SELECT"
  );
}

export function KonamiEasterEgg() {
  const t = useTranslation();
  const progress = useRef(0);
  const [active, setActive] = useState(false);

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (isTypingTarget(e.target)) return;

      const key = e.key.toLowerCase();

      if (key === SEQUENCE[progress.current]) {
        progress.current += 1;
      } else {
        progress.current = key === SEQUENCE[0] ? 1 : 0;
      }

      if (progress.current === SEQUENCE.length) {
        progress.current = 0;
        toast.success(t.easterEgg.unlocked);

        const reduceMotion = window.matchMedia(
          "(prefers-reduced-motion: reduce)",
        ).matches;
        if (!reduceMotion) setActive(true);
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, [t]);

  useEffect(() => {
    if (!active) return;
    const timeout = setTimeout(() => setActive(false), DURATION_MS);
    return () => clearTimeout(timeout);
  }, [active]);

  if (!active) return null;

  return (
    <Box
      data-testid="konami-easter-egg"
      aria-hidden="true"
      position="fixed"
      inset="0"
      overflow="hidden"
      pointerEvents="none"
      zIndex="overlay"
    >
      {particles.map((p, i) => (
        <span
          key={i}
          className={particleStyles}
          style={{
            left: p.left,
            animationDelay: p.delay,
            animationDuration: p.duration,
          }}
        >
          {p.emoji}
        </span>
      ))}
    </Box>
  );
}
