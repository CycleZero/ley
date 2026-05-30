<!--
  clean.vue — 纯净布局

  使用场景：
  - 文章阅读详情页（沉浸阅读，最大化内容空间）

  特点：
  - 导航栏默认隐藏，滚动时浮现
  - 无页脚（或极简页脚）
  - 主内容区无额外 padding/margin 干扰
-->
<template>
  <div class="min-h-screen bg-base">
    <!-- 极简顶部（滚动浮现） -->
    <header
      class="fixed top-0 left-0 right-0 z-30 transition-all duration-500 ease-ink"
      :class="[
        isVisible ? 'translate-y-0 opacity-100' : '-translate-y-full opacity-0',
        'bg-base/90 backdrop-blur-md border-b border-subtle',
      ]"
    >
      <div class="max-w-4xl mx-auto px-6 h-12 flex items-center justify-between">
        <NuxtLink
          to="/articles"
          class="text-sm text-muted hover:text-heading transition-colors flex items-center gap-1"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 19l-7-7 7-7" />
          </svg>
          返回列表
        </NuxtLink>

        <NuxtLink to="/" class="font-serif-jp text-base font-semibold text-heading tracking-wider">
          Ley
        </NuxtLink>

        <!-- 占位，保持居中对称 -->
        <div class="w-16" />
      </div>
    </header>

    <!-- 主内容区 -->
    <main>
      <slot />
    </main>

    <!-- 回到顶部 -->
    <AppScrollTop />
  </div>
</template>

<script setup lang="ts">
// 复用相同的滚动方向检测逻辑
const { isVisible, scrollY } = useScrollDirection(200)
</script>
