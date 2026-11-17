// 关闭/刷新标签页/退出时的拦截判定。
// 注意：全局上传队列让上传在后台继续，切换路由不会丢上传，所以这里只在
// beforeunload（关标签/刷新/退出）生效，不参与路由守卫。
export type UnloadGuardInput = {
  controlledReloadArmed: boolean;
  hasUnsavedChanges: boolean;
  hasActiveUploads: boolean;
};

export function shouldBlockUnload(input: UnloadGuardInput): boolean {
  if (input.controlledReloadArmed) return false;
  return input.hasUnsavedChanges || input.hasActiveUploads;
}
