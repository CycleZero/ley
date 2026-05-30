<!--
  StaggerList — 列表 Stagger 入场动画组件

  职责：
  - 为列表子项提供级联入场动画（如落樱般依次浮现）
  - 通过 GSAP 实现精确的时间线控制
  - 被包裹的列表项需设置 data-index 属性用于计算延迟

  Props:
    staggerDelay    number — 每项之间的延迟（秒），默认 0.08
    duration        number — 单个项的动画时长（秒），默认 0.7
    direction       'up' | 'down' — 入场方向

  使用方式：
    <StaggerList>
      <ArticleCard
        v-for="(article, index) in articles"
        :key="article.id"
        :article="article"
        :data-index="index"
      />
    </StaggerList>
-->
<template>
  <div ref="containerRef" class="stagger-list-wrapper">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import gsap from 'gsap'

const props = withDefaults(defineProps<{
  staggerDelay?: number
  duration?: number
  direction?: 'up' | 'down'
}>(), {
  staggerDelay: 0.08,
  duration: 0.7,
  direction: 'up',
})

const containerRef = ref<HTMLElement | null>(null)
let ctx: gsap.Context | null = null

onMounted(async () => {
  await nextTick()

  if (!containerRef.value) return

  // 获取所有带有 data-index 属性的子项
  const items = containerRef.value.querySelectorAll('[data-index]')
  if (!items.length) return

  // 初始化状态：隐藏 + 偏移
  const yOffset = props.direction === 'up' ? 32 : -32

  gsap.set(items, {
    opacity: 0,
    y: yOffset,
    rotateX: -6,
    transformOrigin: 'center top',
  })

  // 使用 ScrollTrigger 在滚动到容器时触发
  // 为避免引入 ScrollTrigger 插件增加复杂度，这里用 IntersectionObserver
  const observer = new IntersectionObserver(
    ([entry]) => {
      if (entry.isIntersecting) {
        // Stagger 动画
        gsap.to(items, {
          opacity: 1,
          y: 0,
          rotateX: 0,
          duration: props.duration,
          stagger: props.staggerDelay,
          ease: 'power3.out',
        })
        observer.disconnect()
      }
    },
    { threshold: 0.05 },
  )

  observer.observe(containerRef.value)

  // 保存引用用于清理
  ctx = gsap.context(() => {}, containerRef.value)
})

onUnmounted(() => {
  if (ctx) {
    ctx.revert()
  }
})
</script>
