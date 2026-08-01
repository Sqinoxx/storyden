"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

export type Theme = "dark" | "light" | "system";

export type LightPreset =
  | "neutral"
  | "paper"
  | "nordic"
  | "custom";

export type LightBgStyle = "default" | "soft" | "warm";

export const PRESET_CONFIGS: Record<
  LightPreset,
  { warmth: number; bgStyle: LightBgStyle; name: string }
> = {
  neutral: { warmth: 0, bgStyle: "default", name: "Neutral (Notion / GitHub)" },
  paper: { warmth: 25, bgStyle: "warm", name: "Warmes Papier (Substack / Medium)" },
  nordic: { warmth: 0, bgStyle: "soft", name: "Kühles Nordisch (Linear / Arc)" },
  custom: { warmth: 0, bgStyle: "default", name: "Benutzerdefiniert" },
};

type ThemeContextValue = {
  theme: Theme;
  resolvedTheme: "dark" | "light";
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
  warmth: number;
  setWarmth: (value: number) => void;
  lightPreset: LightPreset;
  setLightPreset: (preset: LightPreset) => void;
  lightBgStyle: LightBgStyle;
  setLightBgStyle: (style: LightBgStyle) => void;
};

const STORAGE_KEY = "storyden-theme";
const WARMTH_KEY = "storyden-warmth";
const PRESET_KEY = "storyden-light-preset";
const BG_STYLE_KEY = "storyden-light-bg-style";

const ThemeContext = createContext<ThemeContextValue>({
  theme: "system",
  resolvedTheme: "dark",
  setTheme: () => { },
  toggleTheme: () => { },
  warmth: 0,
  setWarmth: () => { },
  lightPreset: "neutral",
  setLightPreset: () => { },
  lightBgStyle: "default",
  setLightBgStyle: () => { },
});

export function useTheme() {
  return useContext(ThemeContext);
}

function getSystemTheme(): "dark" | "light" {
  if (typeof window === "undefined") return "dark";
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function getStoredTheme(): Theme {
  if (typeof window === "undefined") return "system";
  try {
    const stored = localStorage.getItem(STORAGE_KEY) as Theme | null;
    if (stored === "dark" || stored === "light" || stored === "system") {
      return stored;
    }
  } catch { }
  return "system";
}

function getStoredWarmth(): number {
  if (typeof window === "undefined") return 0;
  try {
    const stored = localStorage.getItem(WARMTH_KEY);
    if (stored !== null) {
      const parsed = parseInt(stored, 10);
      if (!isNaN(parsed) && parsed >= 0 && parsed <= 100) return parsed;
    }
  } catch { }
  return 0;
}

function getStoredPreset(): LightPreset {
  if (typeof window === "undefined") return "neutral";
  try {
    const stored = localStorage.getItem(PRESET_KEY) as LightPreset | null;
    if (stored && stored in PRESET_CONFIGS) return stored;
  } catch { }
  return "neutral";
}

function getStoredBgStyle(): LightBgStyle {
  if (typeof window === "undefined") return "default";
  try {
    const stored = localStorage.getItem(BG_STYLE_KEY) as LightBgStyle | null;
    if (
      stored === "default" ||
      stored === "soft" ||
      stored === "warm"
    ) {
      return stored;
    }
  } catch { }
  return "default";
}

function applyThemeClass(resolved: "dark" | "light") {
  const root = document.documentElement;
  root.classList.remove("dark", "light");
  root.classList.add(resolved);
}

function applyLightStyles(preset: LightPreset, warmth: number, bgStyle: LightBgStyle) {
  const root = document.documentElement;
  root.style.setProperty("--color-warmth", String(warmth));
  root.setAttribute("data-light-preset", preset);
  if (bgStyle === "default") {
    root.removeAttribute("data-light-bg");
  } else {
    root.setAttribute("data-light-bg", bgStyle);
  }
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = useState<Theme>("system");
  const [resolvedTheme, setResolvedTheme] = useState<"dark" | "light">("dark");
  const [warmth, setWarmthState] = useState<number>(0);
  const [lightPreset, setLightPresetState] = useState<LightPreset>("neutral");
  const [lightBgStyle, setLightBgStyleState] = useState<LightBgStyle>("default");

  // On mount, read from localStorage
  useEffect(() => {
    const storedTheme = getStoredTheme();
    const resolved = storedTheme === "system" ? getSystemTheme() : storedTheme;
    setThemeState(storedTheme);
    setResolvedTheme(resolved);
    applyThemeClass(resolved);

    const storedWarmth = getStoredWarmth();
    const storedPreset = getStoredPreset();
    const storedBgStyle = getStoredBgStyle();

    setWarmthState(storedWarmth);
    setLightPresetState(storedPreset);
    setLightBgStyleState(storedBgStyle);

    applyLightStyles(storedPreset, storedWarmth, storedBgStyle);
  }, []);

  // Listen to OS preference changes (only relevant when theme === "system")
  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = (e: MediaQueryListEvent) => {
      if (theme === "system") {
        const resolved = e.matches ? "dark" : "light";
        setResolvedTheme(resolved);
        applyThemeClass(resolved);
      }
    };
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [theme]);

  const setTheme = useCallback((newTheme: Theme) => {
    const resolved = newTheme === "system" ? getSystemTheme() : newTheme;
    try {
      localStorage.setItem(STORAGE_KEY, newTheme);
    } catch { }
    setThemeState(newTheme);
    setResolvedTheme(resolved);
    applyThemeClass(resolved);
  }, []);

  const toggleTheme = useCallback(() => {
    setTheme(resolvedTheme === "dark" ? "light" : "dark");
  }, [resolvedTheme, setTheme]);

  const updateLightState = useCallback(
    (
      newPreset: LightPreset,
      newWarmth: number,
      newBgStyle: LightBgStyle
    ) => {
      try {
        localStorage.setItem(PRESET_KEY, newPreset);
        localStorage.setItem(WARMTH_KEY, String(newWarmth));
        localStorage.setItem(BG_STYLE_KEY, newBgStyle);
      } catch { }
      setLightPresetState(newPreset);
      setWarmthState(newWarmth);
      setLightBgStyleState(newBgStyle);
      applyLightStyles(newPreset, newWarmth, newBgStyle);
    },
    []
  );

  const setLightPreset = useCallback(
    (preset: LightPreset) => {
      if (preset !== "custom" && preset in PRESET_CONFIGS) {
        const cfg = PRESET_CONFIGS[preset];
        updateLightState(preset, cfg.warmth, cfg.bgStyle);
      } else {
        updateLightState("custom", warmth, lightBgStyle);
      }
    },
    [updateLightState, warmth, lightBgStyle]
  );

  const setWarmth = useCallback(
    (value: number) => {
      const clamped = Math.max(0, Math.min(100, Math.round(value)));
      updateLightState("custom", clamped, lightBgStyle);
    },
    [updateLightState, lightBgStyle]
  );

  const setLightBgStyle = useCallback(
    (style: LightBgStyle) => {
      updateLightState("custom", warmth, style);
    },
    [updateLightState, warmth]
  );

  return (
    <ThemeContext.Provider
      value={{
        theme,
        resolvedTheme,
        setTheme,
        toggleTheme,
        warmth,
        setWarmth,
        lightPreset,
        setLightPreset,
        lightBgStyle,
        setLightBgStyle,
      }}
    >
      {children}
    </ThemeContext.Provider>
  );
}


