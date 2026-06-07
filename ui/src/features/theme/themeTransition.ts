import type { InjectionKey } from 'vue';

export type ThemeTransitionSource = DOMRect | MouseEvent;
export type TriggerThemeTransition = (source?: ThemeTransitionSource) => void;

export const triggerThemeTransitionKey: InjectionKey<TriggerThemeTransition> =
  Symbol('triggerThemeTransition');
