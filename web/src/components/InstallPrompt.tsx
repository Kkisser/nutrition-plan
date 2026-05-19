import { useEffect, useState } from "react";

// Кастомный onboarding-баннер «добавить на главный экран».
//
// Поведение по платформам:
// - Chrome / Edge / Samsung: ловим событие beforeinstallprompt и предлагаем
//   вызвать его при клике на «Установить». Браузер сам покажет system-диалог.
// - iOS Safari: события beforeinstallprompt нет (Apple их не реализовал),
//   поэтому показываем текстовую инструкцию «Поделиться → На экран Домой».
// - PWA уже установлена (display-mode: standalone) — баннер скрыт.
// - Пользователь закрыл — запоминаем в localStorage, баннер не возвращается
//   в этой сессии и в течение 30 дней.

const DISMISS_KEY = "pwa_install_dismissed_at";
const DISMISS_TTL_MS = 30 * 24 * 60 * 60 * 1000;

interface BeforeInstallPromptEvent extends Event {
  readonly platforms: string[];
  readonly userChoice: Promise<{ outcome: "accepted" | "dismissed"; platform: string }>;
  prompt(): Promise<void>;
}

function isStandalone(): boolean {
  if (typeof window === "undefined") return false;
  // iOS legacy + современный API.
  if ((navigator as Navigator & { standalone?: boolean }).standalone) return true;
  return window.matchMedia?.("(display-mode: standalone)").matches ?? false;
}

function isIOS(): boolean {
  if (typeof navigator === "undefined") return false;
  return /iphone|ipad|ipod/i.test(navigator.userAgent);
}

function dismissedRecently(): boolean {
  try {
    const at = Number(localStorage.getItem(DISMISS_KEY));
    if (!at) return false;
    return Date.now() - at < DISMISS_TTL_MS;
  } catch {
    return false;
  }
}

export default function InstallPrompt() {
  const [deferred, setDeferred] = useState<BeforeInstallPromptEvent | null>(null);
  const [showIOSHint, setShowIOSHint] = useState(false);
  const [hidden, setHidden] = useState(true);

  useEffect(() => {
    if (isStandalone()) return;
    if (dismissedRecently()) return;

    if (isIOS()) {
      setShowIOSHint(true);
      setHidden(false);
      return;
    }

    const handler = (e: Event) => {
      // По спеке prevent сохраняет событие для пользовательского вызова позже.
      e.preventDefault();
      setDeferred(e as BeforeInstallPromptEvent);
      setHidden(false);
    };
    window.addEventListener("beforeinstallprompt", handler);
    return () => window.removeEventListener("beforeinstallprompt", handler);
  }, []);

  const dismiss = () => {
    try {
      localStorage.setItem(DISMISS_KEY, String(Date.now()));
    } catch {
      // приватный режим без storage — просто скрываем для текущей сессии.
    }
    setHidden(true);
  };

  const install = async () => {
    if (!deferred) return;
    await deferred.prompt();
    const { outcome } = await deferred.userChoice;
    if (outcome === "accepted") {
      setHidden(true);
    } else {
      dismiss();
    }
  };

  if (hidden) return null;

  return (
    <div
      role="dialog"
      aria-label="Установить приложение"
      style={{
        position: "fixed",
        left: 12,
        right: 12,
        bottom: 12,
        background: "#2e7d32",
        color: "white",
        padding: "12px 16px",
        borderRadius: 8,
        boxShadow: "0 4px 16px rgba(0,0,0,0.2)",
        zIndex: 9998,
        display: "flex",
        alignItems: "center",
        gap: 12,
        flexWrap: "wrap",
      }}
    >
      <div style={{ flex: 1, minWidth: 180, fontSize: 14, lineHeight: 1.35 }}>
        {showIOSHint ? (
          <>
            <b>Добавьте на главный экран</b> для офлайн-доступа: нажмите{" "}
            <span style={{ whiteSpace: "nowrap" }}>«Поделиться» ⤴</span> →{" "}
            <span style={{ whiteSpace: "nowrap" }}>«На экран Домой».</span>
          </>
        ) : (
          <>
            <b>Установить как приложение?</b> Работает офлайн, иконка на
            домашнем экране, открывается без браузерной строки.
          </>
        )}
      </div>
      {deferred && !showIOSHint && (
        <button
          type="button"
          onClick={install}
          style={{
            background: "white",
            color: "#2e7d32",
            border: "none",
            padding: "6px 12px",
            borderRadius: 4,
            fontWeight: 600,
            cursor: "pointer",
          }}
        >
          Установить
        </button>
      )}
      <button
        type="button"
        onClick={dismiss}
        aria-label="Закрыть"
        style={{
          background: "transparent",
          color: "white",
          border: "1px solid rgba(255,255,255,0.5)",
          padding: "6px 12px",
          borderRadius: 4,
          cursor: "pointer",
        }}
      >
        Позже
      </button>
    </div>
  );
}
