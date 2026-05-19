import {
  createContext,
  useCallback,
  useContext,
  useRef,
  useState,
} from "react";

export type ToastKind = "success" | "error" | "info";
interface Toast {
  id: number;
  kind: ToastKind;
  text: string;
}

interface Ctx {
  push: (kind: ToastKind, text: string) => void;
  success: (text: string) => void;
  error: (text: string) => void;
  info: (text: string) => void;
}

const ToastCtx = createContext<Ctx | null>(null);

export function useToast(): Ctx {
  const ctx = useContext(ToastCtx);
  if (!ctx) throw new Error("useToast must be used within ToastProvider");
  return ctx;
}

const COLORS: Record<ToastKind, string> = {
  success: "#2e7d32",
  error: "#a05a00",
  info: "#1565c0",
};

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<Toast[]>([]);
  const nextId = useRef(1);

  const push = useCallback((kind: ToastKind, text: string) => {
    const id = nextId.current++;
    setItems((xs) => [...xs, { id, kind, text }]);
    setTimeout(
      () => setItems((xs) => xs.filter((t) => t.id !== id)),
      kind === "error" ? 5000 : 3000,
    );
  }, []);

  const api: Ctx = {
    push,
    success: (t) => push("success", t),
    error: (t) => push("error", t),
    info: (t) => push("info", t),
  };

  return (
    <ToastCtx.Provider value={api}>
      {children}
      <div
        aria-live="polite"
        style={{
          position: "fixed",
          right: 16,
          bottom: 16,
          display: "flex",
          flexDirection: "column",
          gap: 8,
          zIndex: 9999,
          maxWidth: "min(360px, calc(100vw - 32px))",
        }}
      >
        {items.map((t) => (
          <div
            key={t.id}
            role="alert"
            style={{
              background: "white",
              borderLeft: `4px solid ${COLORS[t.kind]}`,
              color: "#222",
              padding: "10px 14px",
              borderRadius: 4,
              boxShadow: "0 4px 12px rgba(0,0,0,0.15)",
              fontSize: 14,
            }}
          >
            {t.text}
          </div>
        ))}
      </div>
    </ToastCtx.Provider>
  );
}

export function Spinner({ size = 16 }: { size?: number }) {
  return (
    <span
      aria-label="загрузка"
      style={{
        display: "inline-block",
        width: size,
        height: size,
        border: "2px solid currentColor",
        borderRightColor: "transparent",
        borderRadius: "50%",
        animation: "toast-spin 0.7s linear infinite",
        verticalAlign: "middle",
      }}
    />
  );
}

// Inject keyframes once. Vite позволяет такой подход без CSS-файла.
const SPIN_STYLE_ID = "toast-spinner-style";
if (typeof document !== "undefined" && !document.getElementById(SPIN_STYLE_ID)) {
  const s = document.createElement("style");
  s.id = SPIN_STYLE_ID;
  s.textContent = `@keyframes toast-spin { to { transform: rotate(360deg); } }`;
  document.head.appendChild(s);
}

