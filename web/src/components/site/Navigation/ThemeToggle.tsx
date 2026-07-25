"use client";

import { Moon, Sun } from "lucide-react";

import { useTheme } from "@/lib/theme/ThemeProvider";
import { css } from "@/styled-system/css";

export function ThemeToggle() {
  const { resolvedTheme, toggleTheme } = useTheme();
  const isDark = resolvedTheme === "dark";

  const label = isDark
    ? "Zum hellen Design wechseln"
    : "Zum dunklen Design wechseln";

  return (
    <button
      type="button"
      onClick={toggleTheme}
      aria-label={label}
      title={label}
      className={css({
        position: "relative",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        w: "9",
        h: "9",
        borderRadius: "md",
        border: "none",
        cursor: "pointer",
        color: "fg.muted",
        background: "transparent",
        _hover: {
          background: "bg.subtle",
          color: "fg.default",
        },
        _active: {
          background: "bg.muted",
        },
      })}
      style={{ transition: "background 0.2s, color 0.2s" }}
    >
      {/* Moon icon – visible in dark mode */}
      <span
        aria-hidden="true"
        style={{
          display: "block",
          position: "absolute",
          transition: "opacity 0.25s, transform 0.3s",
          opacity: isDark ? 1 : 0,
          transform: isDark ? "rotate(0deg) scale(1)" : "rotate(90deg) scale(0.7)",
          pointerEvents: "none",
        }}
      >
        <Moon size={18} />
      </span>

      {/* Sun icon – visible in light mode */}
      <span
        aria-hidden="true"
        style={{
          display: "block",
          position: "absolute",
          transition: "opacity 0.25s, transform 0.3s",
          opacity: isDark ? 0 : 1,
          transform: isDark ? "rotate(-90deg) scale(0.7)" : "rotate(0deg) scale(1)",
          pointerEvents: "none",
        }}
      >
        <Sun size={18} />
      </span>
    </button>
  );
}
