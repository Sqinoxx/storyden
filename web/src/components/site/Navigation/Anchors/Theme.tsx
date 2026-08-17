"use client";

import { Moon, Sun } from "lucide-react";

import { Item } from "@/components/ui/menu";
import { useTheme } from "@/lib/theme/ThemeProvider";

export const ThemeID = "theme";

type Props = {
  closeOnSelect?: boolean;
};

export function ThemeMenuItem({ closeOnSelect = false }: Props) {
  const { resolvedTheme, toggleTheme } = useTheme();
  const isDark = resolvedTheme === "dark";

  const label = isDark
    ? "Zum hellen Design wechseln"
    : "Zum dunklen Design wechseln";

  return (
    <Item value={ThemeID} asChild closeOnSelect={closeOnSelect}>
      <button
        type="button"
        onClick={toggleTheme}
        title={label}
        style={{ cursor: "pointer", width: "100%" }}
      >
        {isDark ? (
          <Moon size={16} style={{ display: "inline" }} />
        ) : (
          <Sun size={16} style={{ display: "inline" }} />
        )}
        &nbsp;<span>{label}</span>
      </button>
    </Item>
  );
}
