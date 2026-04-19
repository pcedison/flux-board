import { createContext } from "react";

import type { TaskPriority, TaskStatus } from "./api";
import type { AppLocale, AppTheme, Messages } from "./preferences";

export type PreferencesContextValue = {
  copy: Messages;
  formatDate: (value: string) => string;
  formatDateTime: (value: number) => string;
  isDarkTheme: boolean;
  locale: AppLocale;
  priorityLabel: (value: TaskPriority) => string;
  setLocale: (locale: AppLocale) => void;
  setTheme: (theme: AppTheme) => void;
  statusLabel: (value: TaskStatus) => string;
  theme: AppTheme;
  toggleTheme: () => void;
};

export const PreferencesContext = createContext<PreferencesContextValue | null>(null);
