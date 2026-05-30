<!--
  FadeIn — 通用淡入动画组件

  职责：
  - 为包裹的内容提供入场淡入动画
  - SSR 时默认可见，避免空白
  - 首屏内容直接显示；滚动进入视口的内容播放入场动画

  Props:
    direction     'up' | 'down' | 'left' | 'right' | 'none' — 入场方向
    delay         number — 延迟时间（秒）
    duration      number — 动画时长（秒）
    distance      number — 位移距离（px）
    once          boolean — 是否只触发一次
    threshold     number — 进入视口多少比例时触发（0-1）
-->
<template>
  <div
    ref="elRef"
    class="transition-all"
    :style="transitionStyle"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'

const props = withDefaults(defineProps<{
  direction?: 'up' | 'down' | 'left' | 'right' | 'none'
  delay?: number
  duration?: number
  distance?: number
  once?: boolean
  threshold?: number
}>(), {
  direction: 'up',
  delay: 0,
  duration: 0.6,
  distance: 24,
  once: true,
  threshold: 0.1,
})

const elRef = ref<HTMLElement | null>(null)

// SSR 默认可见，避免空白；
// 客户端视口外的元素会在 onMounted 中设为 hidden，由 IntersectionObserver 触发显示。
const isVisible = ref(true)

const initialTransform = computed(() => {
  switch (props.direction) {
    case 'up':    return `translateY(${props.distance}px)`
    case 'down':  return `translateY(-${props.distance}px)`
    case 'left':  return `translateX(${props.distance}px)`
    case 'right': return `translateX(-${props.distance}px)`
    case 'none':  return 'none'
    default:      return 'none'
  }
})

const transitionStyle = computed(() => {
  const opacity = isVisible.value ? 1 : 0
  const transform = isVisible.value ? 'translate(0, 0)' : initialTransform.value

  return {
    opacity,
    transform,
    transition: `opacity ${props.duration}s cubic-bezier(0.25, 0.1, 0.25, 1) ${props.delay}s, transform ${props.duration}s cubic-bezier(0.25, 0.1, 0.25, 1) ${props.delay}s`,
    willChange: isVisible.value ? 'auto' : 'opacity, transform',
  }
})

let observer: IntersectionObserver | null = null

onMounted(() => {
  if (!elRef.value) return

  const rect = elRef.value.getBoundingClientRect()
  const inViewport = rect.top < window.innerHeight && rect.bottom > 0

  if (!inViewport) {
    // 视口外：先隐藏，等 IntersectionObserver 触发时再播放入场动画
    isVisible.value = false
  }
  // 视口内：保持 isVisible = true，直接显示，不做入场动画（避免闪烁）

  observer = new IntersectionObserver(
    ([entry]) => {
      if (entry.isIntersecting) {
        isVisible.value = true
        if (props.once && observer && elRef.value) {
          observer.unobserve(elRef.value)
        }
      }
      else if (!props.once) {
        isVisible.value = false
      }
    },
    { threshold: props.threshold },
  )

  observer.observe(elRef.value)
})
</script>
