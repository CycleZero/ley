<!--
  分类页 /categories

  展示分类树，层级缩进，点击跳转对应分类的文章列表。
-->
<template>
  <div class="max-w-3xl mx-auto px-6 py-16">
    <!-- 页面标题 -->
    <FadeIn direction="up" :delay="0">
      <div class="text-center mb-12">
        <h1 class="text-3xl font-serif-jp font-semibold text-heading tracking-widest">
          分类
        </h1>
        <p class="mt-3 text-sm text-placeholder tracking-wide">
          共 {{ categoryStore.categories.length }} 个分类
        </p>
      </div>
    </FadeIn>

    <!-- 加载中 -->
    <div v-if="categoryStore.loading" class="flex justify-center py-20">
      <WasLoading />
    </div>

    <!-- 分类树 -->
    <FadeIn v-else-if="categoryStore.categories.length" direction="up" :delay="100">
      <div class="space-y-1">
        <NuxtLink
          v-for="item in categoryStore.flatCategories"
          :key="item.category.id"
          :to="`/articles?category=${item.category.id}`"
          class="group flex items-center justify-between py-3 px-4
                 border-b border-subtle last:border-0
                 hover:bg-surface-hover transition-colors duration-200"
        >
          <div class="flex items-center gap-3">
            <!-- 层级缩进 -->
            <span class="inline-block" :style="{ width: `${item.depth * 1.5}rem` }" />
            <!-- 层级指示器 -->
            <span
              v-if="item.depth > 0"
              class="text-placeholder text-xs"
            >
              └
            </span>
            <span class="text-body group-hover:text-accent transition-colors duration-300">
              {{ item.category.name }}
            </span>
            <span
              v-if="item.category.description"
              class="text-xs text-placeholder hidden sm:inline"
            >
              {{ item.category.description }}
            </span>
          </div>
          <span class="text-xs text-placeholder">
            {{ item.category.articleCount }} 篇
          </span>
        </NuxtLink>
      </div>
    </FadeIn>

    <!-- 空状态 -->
    <WasEmpty v-else description="暂无分类" />
  </div>
</template>

<script setup lang="ts">
const categoryStore = useCategoryStore()

// SSR 获取分类
await useAsyncData('categories-page', () => categoryStore.fetchCategories(), {
  server: true,
  lazy: false,
})

// 页面标题
useHead({
  title: '分类',
})

// 客户端兜底：nuxt generate 下 payload 恢复不执行副作用
onMounted(() => {
  if (!categoryStore.categories.length) categoryStore.fetchCategories()
})
</script>
