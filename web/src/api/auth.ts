// Real auth: реальные эндпоинты ядра (POST /auth/register, /auth/login,
// /auth/verify, GET /auth/me). JWT хранится в IndexedDB и подставляется
// в Authorization-заголовок всех защищённых запросов.
//
// Email-верификация без SMTP: при регистрации сервер возвращает
// confirm_token в body — этот токен надо вручную ввести (или, в
// dev-режиме, его сразу применяет UI). В проде токен будет отправляться
// письмом и в response не попадать.

import { get, set, del } from "idb-keyval";

const KEY_TOKEN = "auth_token";
const KEY_USER = "auth_user";

export interface AuthUser {
  email: string;
  userId: string;
  confirmed?: boolean;
}

export interface RegisterResult {
  user_id: string;
  email: string;
  confirm_token: string;
  confirm_required: boolean;
}

export interface LoginResult {
  user_id: string;
  email: string;
  token: string;
  email_confirmed: boolean;
}

let inMemoryToken: string | null = null;

export async function getToken(): Promise<string | null> {
  if (inMemoryToken) return inMemoryToken;
  const t = (await get<string>(KEY_TOKEN)) ?? null;
  inMemoryToken = t;
  return t;
}

async function setToken(token: string): Promise<void> {
  inMemoryToken = token;
  await set(KEY_TOKEN, token);
}

async function clearToken(): Promise<void> {
  inMemoryToken = null;
  await del(KEY_TOKEN);
}

export async function loadAuth(): Promise<AuthUser | null> {
  const token = await getToken();
  if (!token) return null;
  const u = await get<AuthUser>(KEY_USER);
  return u ?? null;
}

async function saveAuth(u: AuthUser): Promise<void> {
  await set(KEY_USER, u);
}

export async function clearAuth(): Promise<void> {
  await clearToken();
  await del(KEY_USER);
}

export async function authHeader(): Promise<Record<string, string>> {
  const t = await getToken();
  return t ? { Authorization: `Bearer ${t}` } : {};
}

export async function postRegister(
  email: string,
  password: string,
): Promise<RegisterResult> {
  const r = await fetch("/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const body = await r.json().catch(() => null);
  if (!r.ok) {
    throw new Error(body?.error ?? `register ${r.status}`);
  }
  return body as RegisterResult;
}

export async function postLogin(
  email: string,
  password: string,
): Promise<AuthUser> {
  const r = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const body = await r.json().catch(() => null);
  if (!r.ok) {
    throw new Error(body?.error ?? `login ${r.status}`);
  }
  const u: AuthUser = {
    email: (body as LoginResult).email,
    userId: (body as LoginResult).user_id,
    confirmed: (body as LoginResult).email_confirmed,
  };
  await setToken((body as LoginResult).token);
  await saveAuth(u);
  return u;
}

export async function postVerify(token: string): Promise<void> {
  const r = await fetch("/api/auth/verify", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  });
  const body = await r.json().catch(() => null);
  if (!r.ok) {
    throw new Error(body?.error ?? `verify ${r.status}`);
  }
}

// Валидация на стороне фронта (та же политика что в core/internal/auth).
export function validateEmail(email: string): string | null {
  const e = email.trim();
  if (!e) return "Введите email.";
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(e)) return "Неверный формат email.";
  return null;
}

export function validatePassword(password: string): string | null {
  if (password.length < 8) return "Пароль должен быть не менее 8 символов.";
  if (/\s/.test(password)) return "Пароль не должен содержать пробелов.";
  if (/[А-Яа-яЁё]/.test(password)) return "Пароль не должен содержать кириллицу.";
  if (!/[a-z]/.test(password)) return "Нужна хотя бы одна строчная латинская буква.";
  if (!/[A-Z]/.test(password)) return "Нужна хотя бы одна прописная латинская буква.";
  if (!/\d/.test(password)) return "Нужна хотя бы одна цифра.";
  return null;
}
