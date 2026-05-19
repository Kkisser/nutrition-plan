// Типы по docs/КОНТРАКТ_API.md §2.

export type Sex = "male" | "female";
export type Kfa = "I" | "II" | "III" | "IV";
export type Goal = "deficit" | "maintain" | "surplus";
export type Diet =
  | "classic"
  | "keto"
  | "vegetarian"
  | "vegan"
  | "paleo"
  | "fasting";
export type Meal = "breakfast" | "lunch" | "dinner" | "snack";
export type Unit = "g" | "ml" | "pcs";
export type Allergen =
  | "milk"
  | "eggs"
  | "fish"
  | "gluten"
  | "peanut"
  | "sesame"
  | "shellfish"
  | "soy"
  | "nuts";

export interface Profile {
  sex: Sex;
  age: number;
  height_cm: number;
  weight_kg: number;
  kfa_group: Kfa;
  goal: Goal;
  diet_type: Diet;
  allergens: Allergen[];
  excluded_products: string[];
  meals: Meal[];
  persons: number;
}

export interface PinnedDish {
  day: number;     // 1..7
  meal: Meal;
  dish_id: string;
}

export interface ManualTargetsOverride {
  enabled: boolean;
  kcal?: number;
  protein_g?: number;
  fat_g?: number;
  carb_g?: number;
}

export interface PlanRequest {
  user_id: string;
  profile: Profile;
  pinned_dishes?: PinnedDish[];
  excluded_dishes?: string[];
  manual_targets_override?: ManualTargetsOverride;
}

export interface MealSlot {
  meal: Meal;
  dish_id: string;
  dish_title: string;
  portions: number;
  pinned: boolean;
  kcal: number;
  protein_g: number;
  fat_g: number;
  carb_g: number;
}

export interface DayTotals {
  kcal: number;
  protein_g: number;
  fat_g: number;
  carb_g: number;
}

export interface Day {
  day: number;
  meals: MealSlot[];
  day_totals: DayTotals;
}

export interface ShoppingItem {
  ingredient_name: string;
  category?: string;
  amount: number;
  unit: Unit;
}

export interface Violation {
  day: number;
  metric: string;
  value: number;
  target: number;
  comment?: string;
}

export interface Compliance {
  in_corridor: boolean;
  violations: Violation[];
  message?: string;
}

export interface MicroDeficit {
  nutrient_id: string;
  deficit_per_day: number;
}

export interface PlanResponse {
  user_id: string;
  week_ref: string;
  plan: Day[];
  shopping_list: ShoppingItem[];
  compliance: Compliance;
  micronutrient_carryover_next: { week_ref: string; deficits: MicroDeficit[] };
}
