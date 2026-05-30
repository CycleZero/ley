<!--
  SmoothScroll — 平滑滚动锚点组件

  职责：
  - 点击锚点时平滑滚动到目标元素
  - 支持偏移量（用于固定导航栏高度补偿）

  Props:
    target        string — 目标元素 ID（如 '#section-1'）
    offset        number — 滚动偏移量（px），默认 80（导航栏高度）
    behavior      'smooth' | 'auto' — 滚动行为，默认 smooth

  Events:
    click         点击事件（可用于外部触发滚动）

  使用方式：
    <!-- 作为链接使用 -->
    <SmoothScroll target="#introduction" :offset="80">
      <a href="#introduction">跳到简介</a>
    </SmoothScroll>

    <!-- 直接触发 -->
    <WasButton @click="smoothScrollTo('#comments', 80)">查看评论</WasButton>
-->
<template>
  <span @click="scroll">
    <slot />
  </span>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  target: string
  offset?: number
  behavior?: 'smooth' | 'auto'
}>(), {
  offset: 80,
  behavior: 'smooth',
})

const emit = defineEmits<{
  click: [event: MouseEvent]
}>()

function scroll(event: MouseEvent) {
  emit('click', event)

  const targetId = props.target.replace(/^#/, '')
  const targetEl = document.getElementById(targetId)

  if (targetEl) {
    const rect = targetEl.getBoundingClientRect()
    const scrollTop = window.scrollY + rect.top - props.offset

    window.scrollTo({
      top: scrollTop,
      behavior: props.behavior,
    })
  }
}
</script>
