import { readonly, ref, shallowRef } from 'vue';
import { useIntersectionObserver } from '@vueuse/core';

/**
 * PageHeader 的待处理状态区是否仍在视口内。
 *
 * 全站只有一个锚点：PageHeader 渲染状态区时登记自己，悬浮面板据此决定是否出现。
 * 两个载体互斥 —— 锚点可见时只显示标题旁状态，滚出视口后才显示悬浮面板，
 * 因此永远不会相互遮挡。未登记锚点的页面回落为始终显示悬浮面板。
 */
const anchorEl = shallowRef<HTMLElement | null>(null);
const anchorVisible = ref(false);

// rootMargin 上移 24px：锚点要完全离开视口顶部一段距离才切到悬浮面板，
// 避免滚动到临界点时来回抖动。
useIntersectionObserver(
  anchorEl,
  entries => {
    const entry = entries[0];
    if (!entry) return;
    anchorVisible.value = entry.isIntersecting;
  },
  { rootMargin: '-24px 0px 0px 0px', threshold: 0 }
);

export function setPendingActionAnchor(el: HTMLElement | null): void {
  anchorEl.value = el;
  if (!el) anchorVisible.value = false;
}

export const isPendingActionAnchorVisible = readonly(anchorVisible);
