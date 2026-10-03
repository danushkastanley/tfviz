import { useEffect, useState } from "react";

export type ThemePreference = "system" | "light" | "dark";

// Preferences are per-viewer conveniences only: no report data is stored.
const THEME_KEY = "tfviz.theme";
const MOTION_KEY = "tfviz.reduceMotion";

function read(key: string): string | null {
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key: string, value: string): void {
  try {
    window.localStorage.setItem(key, value);
  } catch {
    // Storage can be unavailable (private mode, file:// policies); preferences then last for the session.
  }
}

export function useThemePreference(initial: ThemePreference): [ThemePreference, (t: ThemePreference) => void] {
  const [theme, setTheme] = useState<ThemePreference>(() => {
    const stored = read(THEME_KEY);
    return stored === "light" || stored === "dark" || stored === "system" ? stored : initial;
  });
  useEffect(() => {
    const root = document.documentElement;
    if (theme === "system") root.removeAttribute("data-theme");
    else root.setAttribute("data-theme", theme);
    write(THEME_KEY, theme);
  }, [theme]);
  return [theme, setTheme];
}

export function useReducedMotion(): [boolean, (value: boolean) => void] {
  const query = typeof window.matchMedia === "function" ? window.matchMedia("(prefers-reduced-motion: reduce)") : undefined;
  const [systemReduced, setSystemReduced] = useState(query?.matches ?? false);
  const [userReduced, setUserReduced] = useState(() => read(MOTION_KEY) === "true");
  useEffect(() => {
    if (!query) return;
    const listener = (event: MediaQueryListEvent) => setSystemReduced(event.matches);
    query.addEventListener("change", listener);
    return () => query.removeEventListener("change", listener);
  }, [query]);
  const update = (value: boolean) => {
    setUserReduced(value);
    write(MOTION_KEY, String(value));
  };
  return [systemReduced || userReduced, update];
}
