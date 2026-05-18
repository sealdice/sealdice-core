<template>
  <main class="plain-layout">
    <slot />
  </main>
</template>

<style scoped>
.plain-layout {
  min-height: 100vh;
  background:
    radial-gradient(circle at 50% 0%, var(--sd-bg-selected), transparent 28rem),
    var(--sd-bg-app);
}
</style>
