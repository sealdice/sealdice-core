// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';
import { appPinia } from '@/pinia';
import { ApiError, markApiErrorHandled } from '@/api/errors';
import { useThemeStore } from '@/features/theme/store';
import { createApiFeedback } from './apiFeedback';
import { notifyError } from './notifyError';

let feedback: ReturnType<typeof createApiFeedback> | undefined;
afterEach(() => {
  feedback?.dispose();
  feedback = undefined;
});

it('已经全局反馈的错误不再触发页面 toast，业务错误仍然可见', () => {
  const messages = { error: vi.fn() };
  const error = new Error('服务器异常');
  markApiErrorHandled(error);
  notifyError(error, messages, '保存失败');
  expect(messages.error).not.toHaveBeenCalled();
  notifyError(new Error('字段不能为空'), messages, '保存失败');
  expect(messages.error).toHaveBeenCalledOnce();
  expect(messages.error).toHaveBeenCalledWith('字段不能为空');
});

it('相同全局故障只展示一个弹窗，已经打开的弹窗跟随主题变化', async () => {
  const theme = useThemeStore(appPinia);
  theme.setThemeMode('light');
  feedback = createApiFeedback();
  const error = new ApiError({ status: 500, statusText: '', data: { detail: '存储失败' } });
  feedback.report(error);
  feedback.report(error);
  await nextTick();
  expect(document.querySelectorAll('.n-dialog')).toHaveLength(1);
  const dialog = document.querySelector('.n-dialog')!;
  expect(dialog.textContent).toContain('存储失败');
  const lightStyle = dialog.getAttribute('style');
  theme.setThemeMode('dark');
  await nextTick();
  expect(dialog.getAttribute('style')).not.toBe(lightStyle);
});
