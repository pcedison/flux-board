import { startTransition, type PropsWithChildren, useEffect, useMemo, useState } from "react";

import { PreferencesContext, type PreferencesContextValue } from "./preferences-context";

export type AppLocale = "en" | "zh-TW";
export type AppTheme = "light" | "dark";

const localeStorageKey = "flux-board-locale";
const themeStorageKey = "flux-board-theme";

const messages = {
  en: {
    common: {
      appName: "Flux Board",
      loading: "Loading",
      board: "Board",
      settings: "Settings",
      signIn: "Sign In",
      signOut: "Sign out",
      signingOut: "Signing out...",
      setup: "Setup",
      checkingAccess: "Checking access",
      save: "Save",
      saving: "Saving...",
      cancel: "Cancel",
      close: "Close",
      language: "Language",
      theme: "Theme",
      light: "Light",
      dark: "Dark",
      systemStatus: "System status",
      boardSummary: "Board summary",
      archive: "Archive",
      newTask: "New task",
      search: "Search",
      title: "Title",
      dueDate: "Due date",
      priority: "Priority",
      note: "Note",
      password: "Password",
      currentPassword: "Current password",
      newPassword: "New password",
      confirmPassword: "Confirm password",
      thisBrowser: "This browser",
      anotherBrowser: "Another signed-in browser",
      unknown: "unknown",
      openBoard: "Open board",
      noTasks: "No tasks yet.",
      selectTask: "Select a task",
      version: (value: string) => (value === "dev" ? "Unreleased build" : `Version ${value}`),
      environment: (value: string) =>
        value === "development" ? "local environment" : `${value} environment`,
      archiveRetentionIndefinite: "Archived cards stay until you remove them manually.",
      archiveRetentionDays: (days: number) => `Archived cards auto-delete after ${days} days.`,
      generatedAt: (value: string) => `Updated ${value}.`,
      installedAt: (value: string) => `Installed at ${value}.`,
      sessionExpiresAt: (value: string) => `Signed in on this browser until ${value}.`,
      sessionSignedOut: "This browser is currently signed out.",
    },
    shell: {
      skipLink: "Skip to main content",
      navLabel: "Primary routes",
      themeToggleToDark: "Switch to dark mode",
      themeToggleToLight: "Switch to light mode",
    },
    query: {
      loadingTitle: "Loading",
    },
    overview: {
      errorTitle: "Unable to load board overview",
      loadingMessage: "Loading your board summary and app status.",
      ready: "Ready to use",
      needsAttention: "Needs attention",
      nextSteps: "Next steps",
      setupReady: "Setup is complete and the board is ready to use.",
      setupNeeded: "Finish setup to create the board password.",
      cleanupSummary: (sessions: string, archive: string) =>
        `Expired sessions are cleared every ${sessions}, and archived cards are checked every ${archive}.`,
      rollbackSummary: (path: string) => `Need to roll back? The previous app path is still available at ${path}.`,
      primarySetup: "Open setup",
      primarySignIn: "Open sign-in",
    },
    auth: {
      unableToOpenApp: (appName: string) => `Unable to open ${appName}`,
      decidingRoute: "Checking whether to send you to setup, sign in, or your board.",
      verifySessionTitle: "Unable to verify your session",
      verifySessionMessage: "Checking your sign-in before opening the board.",
      checkingSignInTitle: "Checking sign-in state",
      checkingSignInMessage: "Checking whether this board is ready for sign-in.",
      signInErrorTitle: "Unable to open the sign-in route",
      signInHeading: "Sign in to view the board",
      signInMessage:
        "Use the board password to continue. If you were sent here from another page, you will go right back after sign-in.",
      signInPlaceholder: "Enter the current Flux Board password",
      signInSubmitting: "Signing in...",
      signInFailed: "Sign-in failed.",
      signInRequired: "Enter the current Flux Board password to continue.",
      afterSignIn: "After sign-in",
      afterSignInReturnTo: (path: string) => `Returns you to ${path} as soon as access is confirmed.`,
      afterSignInSettings: "Keeps password changes and session controls in Settings.",
      afterSignInSetup: "Sends you to setup first if the board has not been configured yet.",
      preparingSetupTitle: "Preparing setup",
      preparingSetupMessage: "Checking whether this board already has a password.",
      setupErrorTitle: "Unable to open setup",
      setupHeading: "Set the board password",
      setupMessage:
        "This board is built for one person. Create the password once, then use it for daily sign-in.",
      setupPasswordPlaceholder: "Create a strong board password",
      setupConfirmPlaceholder: "Type the password again",
      setupSubmitting: "Finishing setup...",
      setupSubmit: "Finish setup",
      setupFailed: "Setup failed.",
      setupPasswordsMustMatch: "Passwords must match before setup can continue.",
      whatHappensNext: "What happens next",
      nextPasswordSaved: "Your password is saved securely.",
      nextAutoSignIn: "This browser signs in automatically after setup finishes.",
      nextSettings: "You can later change the password and archive policy in Settings.",
    },
    board: {
      errorTitle: "Unable to load the board",
      loadingMessage: "Loading your board.",
      searchTitle: "Search",
      searchHint: "Press / to search and N to jump back to the new task form.",
      searchLabel: "Search tasks",
      searchPlaceholder: "Filter by title or note",
      dragAction: "Drag",
      createButton: "Create task",
      creatingButton: "Creating...",
      titlePlaceholder: "Follow up with design review",
      notePlaceholder: "Add context, links, or handoff notes",
      selectedTaskTitle: "Task details",
      selectedTaskHint: "Select a card to edit details or archive it.",
      selectedTaskEmpty: "Pick a task card from the board to edit it here.",
      saveChanges: "Save changes",
      saveChangesPending: "Saving...",
      clearSelection: "Clear selection",
      archiveTask: "Archive task",
      archiveTaskPending: "Archiving...",
      searchShortcutsAria: "Search and shortcuts",
      dragLabel: (title: string, laneLabel: string) =>
        `Drag ${title} to reorder or move it from ${laneLabel}`,
      due: (value: string) => `Due ${value}`,
      createTitleError: "Enter a task title before creating a card.",
      createDueError: "Choose a due date before creating a card.",
      updateTitleError: "Enter a task title before saving.",
      updateDueError: "Choose a due date before saving.",
      createdStatus: (title: string, laneLabel: string) => `Created ${title} in ${laneLabel}.`,
      movedWithinStatus: (title: string, laneLabel: string) => `Moved ${title} within ${laneLabel}.`,
      movedToStatus: (title: string, laneLabel: string) => `Moved ${title} to ${laneLabel}.`,
      archivedStatus: (title: string) => `Archived ${title}.`,
      restoredStatus: (title: string, laneLabel: string) => `Restored ${title} to ${laneLabel}.`,
      deletedStatus: (title: string) => `Deleted ${title} permanently.`,
      updatedStatus: (title: string) => `Updated ${title}.`,
      actionFailedTitle: "Board action failed.",
      actionUpdatedTitle: "Board updated.",
      defaultActionError: "The board action failed.",
      laneEmpty: "No tasks in this lane yet.",
      laneHiddenSummary: (count: number) => `${count} tasks hidden`,
      laneExpandHint: "Click to expand tasks",
      laneCollapseHint: "Click to collapse tasks",
      laneExpandAction: "Expand tasks",
      laneCollapseAction: "Collapse tasks",
      laneNavigationHint:
        "Use Tab to reach a task card, Arrow keys to move between cards, and the drag control to reorder tasks.",
      archiveTitle: "Archive",
      archiveHint: "Restore archived tasks or remove them for good.",
      archiveCount: (count: number) => `${count} archived ${count === 1 ? "task" : "tasks"}`,
      archiveEmpty: "Nothing is archived right now.",
      restoreTo: (laneLabel: string) => `Return to ${laneLabel}`,
      restore: "Restore",
      deletePermanently: "Delete permanently",
      restoreAria: (title: string) => `Restore ${title}`,
      deleteAria: (title: string) => `Delete ${title} permanently`,
    },
    settings: {
      errorTitle: "Unable to load settings",
      loadingMessage: "Loading security, retention, and backup controls.",
      appearanceTitle: "Appearance & language",
      appearanceHint: "These display preferences stay on this browser.",
      archiveTitle: "Archive policy",
      archiveHint: "Choose whether archived cards stay forever or expire automatically.",
      archiveAutoDelete: "Auto-delete archived cards",
      archiveRetentionLabel: "Retention (days)",
      archiveSave: "Save archive policy",
      archiveSaving: "Saving...",
      archiveSavedForever: "Archived cards will now stay indefinitely.",
      archiveSavedDays: (days: number) => `Archived cards will auto-delete after ${days} days.`,
      settingsSaveFailed: "Unable to save settings.",
      passwordTitle: "Password",
      passwordHint: "Change the board password without running setup again.",
      passwordConfirmLabel: "Confirm new password",
      passwordMismatch: "New passwords must match.",
      passwordUpdate: "Update password",
      passwordUpdating: "Updating...",
      passwordUpdated: "Password updated. Other sessions were signed out.",
      passwordUpdateFailed: "Unable to update password.",
      sessionsTitle: "Sessions",
      sessionsSummary: (count: number) => `${count} active ${count === 1 ? "session" : "sessions"}`,
      sessionMeta: (lastSeen: string, expiresAt: string) =>
        `Last active ${lastSeen} • expires ${expiresAt}`,
      sessionIP: (ip: string) => `IP address ${ip}`,
      revoke: "Revoke",
      signOutHere: "Sign out here",
      backupTitle: "Backup & restore",
      backupHint: "Download a full backup or restore the board from an earlier export.",
      exportButton: "Download backup",
      exportPreparing: "Preparing backup...",
      importLabel: "Restore from export",
      importDownloaded: "Backup downloaded.",
      importRestored: "Board restored from backup.",
      importDownloadFailed: "Unable to download backup.",
      importRestoreFailed: "Unable to restore backup.",
      overviewStateReady: "Ready",
      overviewStateAttention: "Attention needed",
      overviewChecksTitle: "Checks",
      overviewRuntime: (build: string, environment: string) => `${build} in ${environment}`,
      overviewStatusUpdated: (value: string) => `Updated ${value}`,
      overviewInstalledAt: (value: string) => `Installed at ${value}`,
      overviewRetention: "Archive retention",
      overviewBoardCounts: "Task counts",
      overviewArchivedCountLabel: "Archived",
      overviewDisplayThemeLabel: "Theme mode",
    },
    laneLabels: {
      queued: "Queued",
      active: "Active",
      done: "Done",
      archived: "Archived",
    },
    priorityLabels: {
      medium: "Medium",
      high: "High",
      critical: "Critical",
    },
  },
  "zh-TW": {
    common: {
      appName: "Flux Board",
      loading: "載入中",
      board: "看板",
      settings: "設定",
      signIn: "登入",
      signOut: "登出",
      signingOut: "登出中...",
      setup: "初始化",
      checkingAccess: "檢查存取權限",
      save: "儲存",
      saving: "儲存中...",
      cancel: "取消",
      close: "關閉",
      language: "語系",
      theme: "主題",
      light: "淺色",
      dark: "深色",
      systemStatus: "系統狀態",
      boardSummary: "看板摘要",
      archive: "封存",
      newTask: "新增任務",
      search: "搜尋",
      title: "標題",
      dueDate: "到期日",
      priority: "優先級",
      note: "備註",
      password: "密碼",
      currentPassword: "目前密碼",
      newPassword: "新密碼",
      confirmPassword: "確認密碼",
      thisBrowser: "這個瀏覽器",
      anotherBrowser: "其他已登入的瀏覽器",
      unknown: "未知",
      openBoard: "開啟看板",
      noTasks: "目前沒有任務。",
      selectTask: "選擇任務",
      version: (value: string) => (value === "dev" ? "未發佈版本" : `版本 ${value}`),
      environment: (value: string) => (value === "development" ? "本機環境" : `${value} 環境`),
      archiveRetentionIndefinite: "封存卡片會保留到你手動移除為止。",
      archiveRetentionDays: (days: number) => `封存卡片會在 ${days} 天後自動刪除。`,
      generatedAt: (value: string) => `更新時間：${value}`,
      installedAt: (value: string) => `安裝路徑：${value}`,
      sessionExpiresAt: (value: string) => `此瀏覽器登入有效至 ${value}`,
      sessionSignedOut: "此瀏覽器目前未登入。",
    },
    shell: {
      skipLink: "跳至主要內容",
      navLabel: "主要頁面",
      themeToggleToDark: "切換為深色模式",
      themeToggleToLight: "切換為淺色模式",
    },
    query: {
      loadingTitle: "載入中",
    },
    overview: {
      errorTitle: "無法載入看板總覽",
      loadingMessage: "正在載入看板摘要與系統狀態。",
      ready: "可正常使用",
      needsAttention: "需要留意",
      nextSteps: "接下來",
      setupReady: "初始化已完成，看板可以開始使用。",
      setupNeeded: "請先完成初始化並建立看板密碼。",
      cleanupSummary: (sessions: string, archive: string) =>
        `過期登入會每 ${sessions} 清理一次，封存卡片會每 ${archive} 檢查一次。`,
      rollbackSummary: (path: string) => `如需回復，先前版本仍可從 ${path} 存取。`,
      primarySetup: "前往初始化",
      primarySignIn: "前往登入",
    },
    auth: {
      unableToOpenApp: (appName: string) => `無法開啟 ${appName}`,
      decidingRoute: "正在判斷應帶你前往初始化、登入，還是直接開啟看板。",
      verifySessionTitle: "無法驗證你的登入狀態",
      verifySessionMessage: "正在確認登入狀態後再開啟看板。",
      checkingSignInTitle: "檢查登入狀態",
      checkingSignInMessage: "正在確認這個看板是否可以登入。",
      signInErrorTitle: "無法開啟登入頁",
      signInHeading: "登入以查看看板",
      signInMessage: "輸入看板密碼即可繼續；若你是從其他頁面被導回來，登入後會自動回到原頁面。",
      signInPlaceholder: "輸入目前的 Flux Board 密碼",
      signInSubmitting: "登入中...",
      signInFailed: "登入失敗。",
      signInRequired: "請先輸入目前的 Flux Board 密碼。",
      afterSignIn: "登入後",
      afterSignInReturnTo: (path: string) => `確認存取權限後，會自動帶你回到 ${path}。`,
      afterSignInSettings: "密碼與登入工作階段管理都放在設定頁。",
      afterSignInSetup: "若尚未初始化，系統會先帶你前往初始化頁。",
      preparingSetupTitle: "準備初始化",
      preparingSetupMessage: "正在確認這個看板是否已經設定過密碼。",
      setupErrorTitle: "無法開啟初始化頁",
      setupHeading: "設定看板密碼",
      setupMessage: "這個看板是為單人使用設計。先設定一次密碼，之後每日登入都使用同一組密碼即可。",
      setupPasswordPlaceholder: "建立一組安全的看板密碼",
      setupConfirmPlaceholder: "再次輸入密碼",
      setupSubmitting: "初始化中...",
      setupSubmit: "完成初始化",
      setupFailed: "初始化失敗。",
      setupPasswordsMustMatch: "兩次輸入的密碼必須一致才能繼續。",
      whatHappensNext: "接下來會發生什麼",
      nextPasswordSaved: "你的密碼會被安全地儲存。",
      nextAutoSignIn: "初始化完成後，此瀏覽器會自動登入。",
      nextSettings: "之後可在設定頁調整密碼與封存規則。",
    },
    board: {
      errorTitle: "無法載入看板",
      loadingMessage: "正在載入看板。",
      searchTitle: "搜尋",
      searchHint: "按 / 可快速搜尋，按 N 可回到新增任務表單。",
      searchLabel: "搜尋任務",
      searchPlaceholder: "依標題或備註篩選",
      dragAction: "拖曳",
      createButton: "建立任務",
      creatingButton: "建立中...",
      titlePlaceholder: "追蹤設計審查後續",
      notePlaceholder: "補充說明、連結或交接資訊",
      selectedTaskTitle: "任務詳情",
      selectedTaskHint: "選取卡片後可在這裡編輯內容或封存。",
      selectedTaskEmpty: "先從左側看板選一張任務卡片，再於此處編輯。",
      saveChanges: "儲存變更",
      saveChangesPending: "儲存中...",
      clearSelection: "清除選取",
      archiveTask: "封存任務",
      archiveTaskPending: "封存中...",
      searchShortcutsAria: "搜尋與快捷鍵",
      dragLabel: (title: string, laneLabel: string) => `拖曳 ${title} 以排序或移動到其他狀態（目前：${laneLabel}）`,
      due: (value: string) => `到期 ${value}`,
      createTitleError: "建立卡片前請先輸入任務標題。",
      createDueError: "建立卡片前請先選擇到期日。",
      updateTitleError: "儲存前請先輸入任務標題。",
      updateDueError: "儲存前請先選擇到期日。",
      createdStatus: (title: string, laneLabel: string) => `已將 ${title} 建立到 ${laneLabel}。`,
      movedWithinStatus: (title: string, laneLabel: string) => `已在 ${laneLabel} 內移動 ${title}。`,
      movedToStatus: (title: string, laneLabel: string) => `已將 ${title} 移動到 ${laneLabel}。`,
      archivedStatus: (title: string) => `已封存 ${title}。`,
      restoredStatus: (title: string, laneLabel: string) => `已將 ${title} 還原到 ${laneLabel}。`,
      deletedStatus: (title: string) => `已永久刪除 ${title}。`,
      updatedStatus: (title: string) => `已更新 ${title}。`,
      actionFailedTitle: "看板操作失敗。",
      actionUpdatedTitle: "看板已更新。",
      defaultActionError: "看板操作失敗。",
      laneEmpty: "這個狀態欄目前沒有任務。",
      laneHiddenSummary: (count: number) => `已收合 ${count} 項任務`,
      laneExpandHint: "點一下展開任務清單",
      laneCollapseHint: "點一下收合任務清單",
      laneExpandAction: "展開任務",
      laneCollapseAction: "收合任務",
      laneNavigationHint: "可用 Tab 聚焦卡片、方向鍵切換卡片，並使用拖曳控制來排序任務。",
      archiveTitle: "封存",
      archiveHint: "可將已封存任務還原，或永久移除。",
      archiveCount: (count: number) => `共 ${count} 項已封存任務`,
      archiveEmpty: "目前沒有封存項目。",
      restoreTo: (laneLabel: string) => `還原到 ${laneLabel}`,
      restore: "還原",
      deletePermanently: "永久刪除",
      restoreAria: (title: string) => `還原 ${title}`,
      deleteAria: (title: string) => `永久刪除 ${title}`,
    },
    settings: {
      errorTitle: "無法載入設定",
      loadingMessage: "正在載入安全性、保存規則與備份控制。",
      appearanceTitle: "顯示與語言",
      appearanceHint: "這些顯示偏好只會保留在目前瀏覽器。",
      archiveTitle: "封存規則",
      archiveHint: "設定封存卡片要永久保留，或在指定天數後自動刪除。",
      archiveAutoDelete: "自動刪除封存卡片",
      archiveRetentionLabel: "保留天數",
      archiveSave: "儲存封存規則",
      archiveSaving: "儲存中...",
      archiveSavedForever: "封存卡片現在會永久保留。",
      archiveSavedDays: (days: number) => `封存卡片會在 ${days} 天後自動刪除。`,
      settingsSaveFailed: "無法儲存設定。",
      passwordTitle: "密碼",
      passwordHint: "無需重新初始化，即可更新看板密碼。",
      passwordConfirmLabel: "確認新密碼",
      passwordMismatch: "新密碼必須一致。",
      passwordUpdate: "更新密碼",
      passwordUpdating: "更新中...",
      passwordUpdated: "密碼已更新，其他登入工作階段已被登出。",
      passwordUpdateFailed: "無法更新密碼。",
      sessionsTitle: "登入工作階段",
      sessionsSummary: (count: number) => `目前有 ${count} 個有效工作階段`,
      sessionMeta: (lastSeen: string, expiresAt: string) => `最後活動：${lastSeen} • 到期：${expiresAt}`,
      sessionIP: (ip: string) => `IP 位址 ${ip}`,
      revoke: "撤銷",
      signOutHere: "在此登出",
      backupTitle: "備份與還原",
      backupHint: "下載完整備份，或從先前匯出的檔案還原整個看板。",
      exportButton: "下載備份",
      exportPreparing: "準備備份中...",
      importLabel: "從匯出檔還原",
      importDownloaded: "備份已下載。",
      importRestored: "已從備份還原看板。",
      importDownloadFailed: "無法下載備份。",
      importRestoreFailed: "無法還原備份。",
      overviewStateReady: "正常",
      overviewStateAttention: "需留意",
      overviewChecksTitle: "檢查項目",
      overviewRuntime: (build: string, environment: string) => `${build}｜${environment}`,
      overviewStatusUpdated: (value: string) => `更新時間：${value}`,
      overviewInstalledAt: (value: string) => `安裝位置：${value}`,
      overviewRetention: "封存保留規則",
      overviewBoardCounts: "任務數量",
      overviewArchivedCountLabel: "已封存",
      overviewDisplayThemeLabel: "目前主題",
    },
    laneLabels: {
      queued: "待處理",
      active: "進行中",
      done: "已完成",
      archived: "已封存",
    },
    priorityLabels: {
      medium: "中",
      high: "高",
      critical: "緊急",
    },
  },
} as const;

export type Messages = (typeof messages)[keyof typeof messages];

export function PreferencesProvider({ children }: PropsWithChildren) {
  const [locale, setLocaleState] = useState<AppLocale>(() => readInitialLocale());
  const [theme, setThemeState] = useState<AppTheme>(() => readInitialTheme());

  useEffect(() => {
    window.localStorage?.setItem?.(localeStorageKey, locale);
    document.documentElement.lang = locale;
  }, [locale]);

  useEffect(() => {
    window.localStorage?.setItem?.(themeStorageKey, theme);
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;
  }, [theme]);

  const value = useMemo<PreferencesContextValue>(() => {
    const copy = messages[locale];
    return {
      copy,
      formatDate: formatDateForLocale(locale),
      formatDateTime: formatDateTimeForLocale(locale),
      isDarkTheme: theme === "dark",
      locale,
      priorityLabel: (priority) => copy.priorityLabels[priority],
      setLocale: (nextLocale) => {
        startTransition(() => setLocaleState(nextLocale));
      },
      setTheme: (nextTheme) => {
        startTransition(() => setThemeState(nextTheme));
      },
      statusLabel: (status) => copy.laneLabels[status],
      theme,
      toggleTheme: () => {
        startTransition(() => setThemeState((current) => (current === "dark" ? "light" : "dark")));
      },
    };
  }, [locale, theme]);

  return <PreferencesContext.Provider value={value}>{children}</PreferencesContext.Provider>;
}

function readInitialLocale(): AppLocale {
  if (typeof window === "undefined") {
    return "en";
  }

  const stored = window.localStorage?.getItem?.(localeStorageKey);
  if (stored === "en" || stored === "zh-TW") {
    return stored;
  }

  const browserLocale = window.navigator.language.toLowerCase();
  if (browserLocale.startsWith("zh")) {
    return "zh-TW";
  }

  return "en";
}

function readInitialTheme(): AppTheme {
  if (typeof window === "undefined") {
    return "light";
  }

  const stored = window.localStorage?.getItem?.(themeStorageKey);
  if (stored === "light" || stored === "dark") {
    return stored;
  }

  if (typeof window.matchMedia === "function" && window.matchMedia("(prefers-color-scheme: dark)").matches) {
    return "dark";
  }

  return "light";
}

function formatDateForLocale(locale: AppLocale) {
  return (value: string) => {
    const parts = value.split("-");
    if (parts.length !== 3) {
      return value;
    }

    if (locale === "zh-TW") {
      return `${parts[0]}/${parts[1]}/${parts[2]}`;
    }

    return `${parts[0]}-${parts[1]}-${parts[2]}`;
  };
}

function formatDateTimeForLocale(locale: AppLocale) {
  const formatter = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  });

  return (value: number) => formatter.format(new Date(value));
}
