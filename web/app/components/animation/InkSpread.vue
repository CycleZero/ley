<!--
  InkSpread — 墨迹扩散 Hover 效果

  职责：
  - 鼠标悬停时在元素背景上产生跟随鼠标的径向渐变
  - 模拟水墨在宣纸上晕染的效果
  - 被包裹的组件完全不感知此效果

  Props:
    color         string — 墨迹颜色（默认淡蓝 #5d8c8c）
    opacity       number — 墨迹透明度（0-1），默认 0.04
    spreadSize    number — 扩散半径（px），默认 600

  使用方式：
    <InkSpread>
      <ArticleCard :article="article" />
    </InkSpread>
-->
<template>
  <div
    ref="containerRef"
    class="relative group cursor-pointer"
    @mousemove="handleMouseMove"
  >
    <!-- 墨迹背景层 -->
    <div
      class="pointer-events-none absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 rounded-sm"
      :style="inkStyle"
    />

    <!-- 内容层 -->
    <div class="relative">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

const props = withDefaults(defineProps<{
  color?: string
  opacity?: number
  spreadSize?: number
}>(), {
  color: '#5d8c8c',
  opacity: 0.04,
  spreadSize: 600,
})

const containerRef = ref<HTMLElement | null>(null)
const mousePos = ref({ x: 0, y: 0 })

const inkStyle = computed(() => {
  return {
    background: `radial-gradient(${props.spreadSize}px circle at ${mousePos.value.x}px ${mousePos.value.y}px, ${props.color}${Math.round(props.opacity * 255).toString(16).padStart(2, '0')} 0%, transparent 60%)`,
  }
})

function handleMouseMove(e: MouseEvent) {
  if (!containerRef.value) return
  const rect = containerRef.value.getBoundingClientRect()
  mousePos.value = {
    x: e.clientX - rect.left,
    y: e.clientY - rect.top,
  }
}
</script>
