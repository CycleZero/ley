<!--
  标签页 /tags

  展示所有标签，按文章数排序，大字号标签云。
-->
<template>
  <div class="max-w-4xl mx-auto px-6 py-16">
    <!-- 页面标题 -->
    <FadeIn direction="up" :delay="0">
      <div class="text-center mb-12">
        <h1 class="text-3xl font-serif-jp font-semibold text-heading tracking-widest">
          标签
        </h1>
        <p class="mt-3 text-sm text-placeholder tracking-wide">
          共 {{ tagStore.tags.length }} 个标签
        </p>
      </div>
    </FadeIn>

    <!-- 加载中 -->
    <div v-if="tagStore.loading && !tagStore.tags.length" class="flex justify-center py-20">
      <WasLoading />
    </div>

    <!-- 标签云 -->
    <FadeIn v-else-if="tagStore.tags.length" direction="up" :delay="100">
      <div class="flex flex-wrap items-center justify-center gap-3">
        <NuxtLink
          v-for="tag in tagStore.sortedTags"
          :key="tag.id"
          :to="`/articles?tag=${tag.id}`"
          class="px-4 py-2 text-sm border border-default rounded-sm text-muted
                 hover:border-accent hover:text-accent hover:bg-accent-subtle
                 transition-all duration-300"
          :style="{ fontSize: tagSize(tag.articleCount) }"
        >
          {{ tag.name }}
          <span class="text-xs text-placeholder ml-1">{{ tag.articleCount }}</span>
        </NuxtLink>
      </div>
    </FadeIn>

    <!-- 空状态 -->
    <WasEmpty v-else description="暂无标签" />
  </div>
</template>

<script setup lang="ts">
const tagStore = useTagStore()

// 页面标题
useHead({
  title: '标签',
})

// 即时请求：静态部署下 payload 可能是旧数据
onMounted(() => {
  tagStore.fetchTags()
})

// 根据文章数计算标签字号
function tagSize(count: string): string {
  const c = Number(count)
  if (c >= 20) return '1.125rem'  // text-lg
  if (c >= 10) return '1rem'      // text-base
  if (c >= 5) return '0.9375rem'  // text-[15px]
  return '0.875rem'               // text-sm
}
</script>
