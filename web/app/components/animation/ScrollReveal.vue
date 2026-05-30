<!--
  ScrollReveal — 滚动触发显示组件

  职责：
  - 当元素滚动进入视口时触发动画
  - 支持从指定方向滑入 + 淡入
  - 可配置触发阈值和是否只触发一次

  Props:
    direction     'up' | 'down' | 'left' | 'right' — 滑入方向
    distance      number — 滑入距离（px），默认 30
    duration      number — 动画时长（秒），默认 0.6
    delay         number — 延迟（秒），默认 0
    once          boolean — 是否只触发一次，默认 true
    threshold     number — 触发阈值（0-1），默认 0.1
    easing        string — CSS easing 函数

  使用方式：
    <ScrollReveal direction="up" distance="40" duration="0.8">
      <h2>区块标题</h2>
      <p>内容...</p>
    </ScrollReveal>
-->
<template>
  <div
    ref="elRef"
    :class="[
      'transition-all',
      isRevealed ? 'opacity-100 translate-x-0 translate-y-0' : 'opacity-0',
    ]"
    :style="transformStyle"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

const props = withDefaults(defineProps<{
  direction?: 'up' | 'down' | 'left' | 'right'
  distance?: number
  duration?: number
  delay?: number
  once?: boolean
  threshold?: number
  easing?: string
}>(), {
  direction: 'up',
  distance: 30,
  duration: 0.6,
  delay: 0,
  once: true,
  threshold: 0.1,
  easing: 'cubic-bezier(0.25, 0.1, 0.25, 1)',
})

const elRef = ref<HTMLElement | null>(null)
const isRevealed = ref(false)

// 初始偏移样式
const initialOffset = computed(() => {
  switch (props.direction) {
    case 'up':    return { y: props.distance }
    case 'down':  return { y: -props.distance }
    case 'left':  return { x: props.distance }
    case 'right': return { x: -props.distance }
  }
})

const transformStyle = computed(() => {
  const x = isRevealed.value ? 0 : (initialOffset.value.x || 0)
  const y = isRevealed.value ? 0 : (initialOffset.value.y || 0)

  return {
    transform: `translate(${x}px, ${y}px)`,
    transition: `opacity ${props.duration}s ${props.easing} ${props.delay}s, transform ${props.duration}s ${props.easing} ${props.delay}s`,
    willChange: isRevealed.value ? 'auto' : 'opacity, transform',
  }
})

let observer: IntersectionObserver | null = null

onMounted(() => {
  if (!elRef.value) return

  observer = new IntersectionObserver(
    ([entry]) => {
      if (entry.isIntersecting) {
        isRevealed.value = true
        if (props.once && observer && elRef.value) {
          observer.unobserve(elRef.value)
        }
      }
      else if (!props.once) {
        isRevealed.value = false
      }
    },
    { threshold: props.threshold },
  )

  observer.observe(elRef.value)
})

onUnmounted(() => {
  if (observer && elRef.value) {
    observer.unobserve(elRef.value)
    observer.disconnect()
  }
})
</script>
