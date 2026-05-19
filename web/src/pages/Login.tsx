import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  postLogin,
  postRegister,
  postVerify,
  validateEmail,
  validatePassword,
} from "../api/auth";

type Mode = "login" | "register";

export default function Login() {
  const nav = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [passwordRepeat, setPasswordRepeat] = useState("");
  const [mode, setMode] = useState<Mode>("login");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [confirmToken, setConfirmToken] = useState<string | null>(null);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const emailErr = validateEmail(email);
    if (emailErr) return setError(emailErr);

    const pwdErr = validatePassword(password);
    if (pwdErr) return setError(pwdErr);

    if (mode === "register" && password !== passwordRepeat) {
      return setError("Пароли не совпадают.");
    }

    setBusy(true);
    try {
      if (mode === "register") {
        const r = await postRegister(email, password);
        // В dev: confirm_token приходит в response (нет SMTP).
        // Сразу применим его автоматически и сделаем логин.
        await postVerify(r.confirm_token);
        setConfirmToken(r.confirm_token);
        await postLogin(email, password);
        nav("/survey");
      } else {
        await postLogin(email, password);
        nav("/survey");
      }
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <h2>{mode === "login" ? "Вход" : "Регистрация"}</h2>
      <p style={{ color: "#666", fontSize: 13 }}>
        Поддержка email-верификации работает через одноразовый токен.
        В dev-режиме токен возвращается в ответе на регистрацию (вместо
        отправки письма через SMTP — это раздел развёртывания, не код).
      </p>

      <form onSubmit={onSubmit}>
        <div className="field">
          <label>Email</label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            autoComplete="email"
            disabled={busy}
          />
        </div>

        <div className="field">
          <label>Пароль</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="8+ символов, латиница, цифра"
            autoComplete={mode === "login" ? "current-password" : "new-password"}
            disabled={busy}
          />
        </div>

        {mode === "register" && (
          <div className="field">
            <label>Повторите пароль</label>
            <input
              type="password"
              value={passwordRepeat}
              onChange={(e) => setPasswordRepeat(e.target.value)}
              autoComplete="new-password"
              disabled={busy}
            />
          </div>
        )}

        {error && <div className="warn">{error}</div>}
        {confirmToken && (
          <div className="card" style={{ background: "#eef7ee", borderColor: "#a5d6a7" }}>
            Email подтверждён автоматически (dev-режим). Токен: <code>{confirmToken}</code>
          </div>
        )}

        <button type="submit" disabled={busy}>
          {busy
            ? "…"
            : mode === "login"
              ? "Войти"
              : "Зарегистрироваться"}
        </button>
        {"  "}
        <button
          type="button"
          disabled={busy}
          style={{
            background: "transparent",
            color: "#2e7d32",
            border: "1px solid #2e7d32",
          }}
          onClick={() => {
            setMode(mode === "login" ? "register" : "login");
            setError(null);
            setConfirmToken(null);
          }}
        >
          {mode === "login" ? "Регистрация" : "У меня уже есть аккаунт"}
        </button>
      </form>
    </>
  );
}
