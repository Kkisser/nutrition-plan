// Локальное хранилище через IndexedDB (idb-keyval).
// Используется для офлайн-просмотра последнего плана (ФРОНТЕНД.md §4).

import { get, set } from "idb-keyval";
import type { ManualTargetsOverride, PinnedDish, PlanResponse, Profile } from "./types";
import type { Survey } from "../lib/kfa";

const KEY_PROFILE = "profile";
const KEY_KFA_SURVEY = "kfa_survey";
const KEY_LAST_PLAN = "last_plan";
const KEY_PINNED = "pinned_dishes";
const KEY_EXCLUDED = "excluded_dishes";
const KEY_MANUAL_TARGETS = "manual_targets";

export const saveProfile = (p: Profile) => set(KEY_PROFILE, p);
export const loadProfile = () => get<Profile>(KEY_PROFILE);

export const saveKfaSurvey = (s: Survey) => set(KEY_KFA_SURVEY, s);
export const loadKfaSurvey = () => get<Survey>(KEY_KFA_SURVEY);

const KEY_PLAN_HISTORY = "plan_history";

export interface PlanHistoryEntry {
  plan: PlanResponse;
  savedAt: string;
}

export const saveLastPlan = async (p: PlanResponse) => {
  await set(KEY_LAST_PLAN, p);
  const hist = (await get<Record<string, PlanHistoryEntry>>(KEY_PLAN_HISTORY)) ?? {};
  hist[p.week_ref] = { plan: p, savedAt: new Date().toISOString() };
  await set(KEY_PLAN_HISTORY, hist);
};
export const loadLastPlan = () => get<PlanResponse>(KEY_LAST_PLAN);

export const loadPlanHistory = async (): Promise<PlanHistoryEntry[]> => {
  const hist = (await get<Record<string, PlanHistoryEntry>>(KEY_PLAN_HISTORY)) ?? {};
  return Object.values(hist).sort((a, b) =>
    a.savedAt < b.savedAt ? 1 : a.savedAt > b.savedAt ? -1 : 0,
  );
};

export const loadPlanByWeek = async (
  weekRef: string,
): Promise<PlanResponse | undefined> => {
  const hist = (await get<Record<string, PlanHistoryEntry>>(KEY_PLAN_HISTORY)) ?? {};
  return hist[weekRef]?.plan;
};

export const savePinnedDishes = (xs: PinnedDish[]) => set(KEY_PINNED, xs);
export const loadPinnedDishes = async (): Promise<PinnedDish[]> =>
  (await get<PinnedDish[]>(KEY_PINNED)) ?? [];

export const saveExcludedDishes = (xs: string[]) => set(KEY_EXCLUDED, xs);
export const loadExcludedDishes = async (): Promise<string[]> =>
  (await get<string[]>(KEY_EXCLUDED)) ?? [];

export const saveManualTargets = (t: ManualTargetsOverride) =>
  set(KEY_MANUAL_TARGETS, t);
export const loadManualTargets = () =>
  get<ManualTargetsOverride>(KEY_MANUAL_TARGETS);
