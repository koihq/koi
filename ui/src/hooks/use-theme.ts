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
    const root = document.documentElement;
    root.setAttribute("data-theme-switching", "");
    setTheme(next);
    root.classList.toggle("dark", next === "dark");
    root.style.colorScheme = next;
    root.getBoundingClientRect();
    requestAnimationFrame(() => root.removeAttribute("data-theme-switching"));

    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // The selection still applies to this visit.
    }
  }

  return { theme, changeTheme };
}
