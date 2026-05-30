<!--
  PageTransition — 页面切换动画（淡墨效果）

  职责：
  - 在路由切换时提供如水墨在宣纸上晕开的过渡效果
  - 淡入 + 轻微模糊 + 垂直位移

  使用方式：
    在 layouts/default.vue 或 layouts/clean.vue 中：
    <PageTransition>
      <NuxtPage />
    </PageTransition>

  注意：
    - 该组件使用 CSS transition 而非 GSAP，避免路由切换时 GSAP 上下文清理问题
    - 使用 Vue 的 <Transition> 组件配合 NuxtPage 的插槽
-->
<template>
  <Transition
    mode="out-in"
    @before-enter="beforeEnter"
    @enter="enter"
    @leave="leave"
  >
    <slot />
  </Transition>
</template>

<script setup lang="ts">
/**
 * 页面进入前：设置初始状态
 */
function beforeEnter(el: Element) {
  const htmlEl = el as HTMLElement
  htmlEl.style.opacity = '0'
  htmlEl.style.transform = 'translateY(24px)'
  htmlEl.style.filter = 'blur(6px)'
}

/**
 * 页面进入：淡墨晕开
 */
function enter(el: Element, done: () => void) {
  const htmlEl = el as HTMLElement

  // 强制回流以确保 transition 生效
  htmlEl.offsetHeight

  htmlEl.style.transition = 'opacity 0.6s cubic-bezier(0.25, 0.1, 0.25, 1), transform 0.6s cubic-bezier(0.25, 0.1, 0.25, 1), filter 0.6s cubic-bezier(0.25, 0.1, 0.25, 1)'
  htmlEl.style.opacity = '1'
  htmlEl.style.transform = 'translateY(0)'
  htmlEl.style.filter = 'blur(0px)'

  setTimeout(() => {
    // 清理内联样式
    htmlEl.style.transition = ''
    htmlEl.style.opacity = ''
    htmlEl.style.transform = ''
    htmlEl.style.filter = ''
    done()
  }, 600)
}

/**
 * 页面离开：淡墨消散
 */
function leave(el: Element, done: () => void) {
  const htmlEl = el as HTMLElement

  htmlEl.style.transition = 'opacity 0.35s cubic-bezier(0.25, 0.1, 0.25, 1), transform 0.35s cubic-bezier(0.25, 0.1, 0.25, 1), filter 0.35s cubic-bezier(0.25, 0.1, 0.25, 1)'
  htmlEl.style.opacity = '0'
  htmlEl.style.transform = 'translateY(-16px)'
  htmlEl.style.filter = 'blur(4px)'

  setTimeout(() => {
    htmlEl.style.transition = ''
    htmlEl.style.opacity = ''
    htmlEl.style.transform = ''
    htmlEl.style.filter = ''
    done()
  }, 350)
}
</script>

<style scoped>
/* 确保 transition 期间内容不被裁剪 */
:deep(.page-enter-active),
:deep(.page-leave-active) {
  will-change: opacity, transform, filter;
}
</style>
