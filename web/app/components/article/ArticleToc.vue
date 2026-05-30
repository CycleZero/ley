<!--
  ArticleToc — 文章目录树

  职责：
  - 解析文章正文中的 h2/h3 标题，生成目录树
  - 跟踪当前阅读位置，高亮对应的目录项
  - 点击目录项平滑滚动到对应位置

  Props:
    content     string — 文章内容（用于解析标题）
    offset      number — 滚动偏移量（固定导航栏高度），默认 96

  使用方式：
    <ArticleToc :content="article.content" :offset="96" />
-->
<template>
  <nav v-if="headings.length" class="sticky top-24">
    <h4 class="text-xs font-medium text-placeholder uppercase tracking-wider mb-4">
      目录
    </h4>
    <ul class="space-y-1.5 border-l border-subtle pl-3">
      <li
        v-for="heading in headings"
        :key="heading.id"
        class="transition-colors duration-200"
        :class="[
          heading.level === 2 ? 'text-sm' : 'text-xs pl-3',
          activeId === heading.id
            ? 'text-accent font-medium'
            : 'text-placeholder hover:text-muted',
        ]"
      >
        <a
          :href="`#${heading.id}`"
          class="block py-0.5 transition-colors"
          @click.prevent="scrollToHeading(heading.id)"
        >
          {{ heading.text }}
        </a>
      </li>
    </ul>
  </nav>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

const props = withDefaults(defineProps<{
  content: string
  offset?: number
}>(), {
  offset: 96,
})

const activeId = ref('')

// 从 Markdown 内容中解析标题
const headings = computed(() => {
  const result: Array<{ id: string; text: string; level: number }> = []
  const lines = props.content.split('\n')

  for (const line of lines) {
    const match = line.match(/^(#{2,3})\s+(.+)$/)
    if (match) {
      const level = match[1].length
      const text = match[2].trim()
      // 生成锚点 ID：小写 + 替换空格为连字符 + 去除特殊字符
      const id = text
        .toLowerCase()
        .replace(/\s+/g, '-')
        .replace(/[^\w\u4e00-\u9fa5-]/g, '')
        .replace(/^-+|-+$/g, '')

      if (id && text) {
        result.push({ id, text, level })
      }
    }
  }

  return result
})

// 平滑滚动到标题
function scrollToHeading(id: string) {
  const el = document.getElementById(id)
  if (el) {
    const rect = el.getBoundingClientRect()
    const scrollTop = window.scrollY + rect.top - props.offset
    window.scrollTo({ top: scrollTop, behavior: 'smooth' })
  }
}

// 监听滚动，跟踪当前阅读位置
let ticking = false

function onScroll() {
  if (!ticking) {
    requestAnimationFrame(() => {
      updateActiveHeading()
      ticking = false
    })
    ticking = true
  }
}

function updateActiveHeading() {
  const headingElements = headings.value
    .map(h => document.getElementById(h.id))
    .filter(Boolean) as HTMLElement[]

  if (!headingElements.length) return

  // 找到第一个在视口上方的标题
  let current = ''
  for (const el of headingElements) {
    const rect = el.getBoundingClientRect()
    if (rect.top <= props.offset + 20) {
      current = el.id
    }
    else {
      break
    }
  }

  if (current) {
    activeId.value = current
  }
}

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true })
  // 延迟初始化，等文章内容渲染完成
  setTimeout(updateActiveHeading, 500)
})

onUnmounted(() => {
  window.removeEventListener('scroll', onScroll)
})
</script>
