package dice

// registerAdapterEventCompat 将旧 IMSession.OnXxx 通知逻辑注册为总线兼容订阅器。
// Task 6 会替换为真实实现；此占位保证装配顺序先于 JS 加载。
func (d *Dice) registerAdapterEventCompat() {}
