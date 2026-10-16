import { useState } from "react";

export type Theme = "light" | "dark";

const STORAGE_KEY = "koi:theme:v1";

function storedTheme(): Theme {
  try {
    return localStorage.getItem(STORAGE_KEY) === "light" ? "light" : "dark";
  } catch {
    return "dark";
  }
}

export function initializeTheme() {
  const theme = storedTheme();
  document.documentElement.classList.toggle("dark", theme === "dark");
  document.documentElement.style.colorScheme = theme;
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(() =>
    document.documentElement.classList.contains("dark") ? "dark" : "light",
  );

  function changeTheme(next: Theme) {
    document.documentElement.setAttribute("data-theme-switching", "");
    setTheme(next);
    document.documentElement.classList.toggle("dark", next === "dark");
    document.documentElement.style.colorScheme = next;
    requestAnimationFrame(() => document.documentElement.removeAttribute("data-theme-switching"));

    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // The selection still applies to this visit.
    }
  }

  return { theme, changeTheme };
}
