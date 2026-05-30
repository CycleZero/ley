<!--
  AppNav — 纯渲染导航链接组件

  职责：
  - 渲染一组导航链接
  - 根据当前路由自动高亮激活项
  - 支持水平和垂直两种布局方向

  约束：
  - 不维护任何内部状态
  - 不调用任何 Store
  - 只通过 props 接收数据和 emit 发出点击事件
-->
<template>
  <nav :class="navClass">
    <NuxtLink
      v-for="item in items"
      :key="item.to"
      :to="item.to"
      :class="[
        'relative text-sm transition-colors duration-200',
        direction === 'horizontal' ? 'py-1' : 'block py-2',
        // 当前路由高亮：使用 Nuxt 的 router-link-active 类
        'text-placeholder hover:text-body',
        'router-link-active:text-heading router-link-active:font-medium',
      ]"
      active-class="text-heading font-medium"
    >
      <!-- 激活指示器（水平布局） -->
      <span
        v-if="direction === 'horizontal'"
        class="absolute bottom-0 left-0 h-px bg-heading transition-all duration-300 scale-x-0 group-hover:scale-x-100"
        :class="{ 'scale-x-100': isActive(item.to) }"
      />
      {{ item.label }}
    </NuxtLink>
  </nav>
</template>

<script setup lang="ts">
interface NavItem {
  label: string
  to: string
}

withDefaults(defineProps<{
  /** 导航项列表 */
  items: NavItem[]
  /** 布局方向 */
  direction?: 'horizontal' | 'vertical'
}>(), {
  direction: 'horizontal',
})

const route = useRoute()

function isActive(to: string) {
  if (to === '/') {
    return route.path === '/'
  }
  return route.path.startsWith(to)
}

const navClass = computed(() => {
  return 'flex gap-6'
})
</script>
