<template>
  <main class="plain-layout">
    <slot />
  </main>
</template>

<style scoped>
.plain-layout {
  min-height: 100vh;
  background: var(--sd-bg-app);
}
</style>
