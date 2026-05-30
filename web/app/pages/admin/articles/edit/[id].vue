<!--
  文章编辑页 /admin/articles/edit/[id]

  支持新建（id=new）和编辑。
-->
<template>
  <div>
    <!-- 页面标题栏 -->
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center gap-3">
        <button
          class="text-placeholder hover:text-body transition-colors"
          @click="navigateTo('/admin/articles')"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
        </button>
        <h1 class="text-2xl font-serif-jp font-semibold text-heading tracking-wider">
          {{ isNew ? '写文章' : '编辑文章' }}
        </h1>
      </div>
      <div class="flex items-center gap-2">
        <WasButton
          variant="secondary"
          :loading="saving"
          @click="saveAsDraft"
        >
          保存草稿
        </WasButton>
        <WasButton
          variant="primary"
          :loading="publishing"
          @click="saveAndPublish"
        >
          {{ isNew ? '发布' : '更新' }}
        </WasButton>
      </div>
    </div>

    <!-- 编辑表单 -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- 主编辑区 -->
      <div class="lg:col-span-2 space-y-6">
        <!-- 标题 -->
        <div class="surface-card p-5">
          <WasInput
            v-model="form.title"
            label="标题"
            placeholder="请输入文章标题"
            :error="errors.title"
          />
        </div>

        <!-- 内容 -->
        <div class="surface-card p-5">
          <label class="block text-sm text-muted mb-2">内容（Markdown）</label>
          <textarea
            v-model="form.content"
            rows="20"
            class="w-full bg-base text-body border border-default rounded-sm p-3 text-sm font-mono leading-washi focus:outline-none focus:border-accent transition-colors resize-y"
            placeholder="开始写作..."
          />
          <p v-if="errors.content" class="mt-1.5 text-xs text-error">{{ errors.content }}</p>
        </div>

        <!-- 摘要 -->
        <div class="surface-card p-5">
          <WasTextarea
            v-model="form.excerpt"
            label="摘要"
            placeholder="文章摘要，为空则自动生成"
            rows="3"
          />
        </div>
      </div>

      <!-- 侧边设置区 -->
      <div class="space-y-6">
        <!-- 分类 -->
        <div class="surface-card p-5">
          <label class="block text-sm text-muted mb-2">分类</label>
          <select
            v-model="form.categoryId"
            class="w-full bg-base border border-default text-body text-sm rounded-sm px-3 py-2.5 focus:outline-none focus:border-accent"
          >
            <option value="">未分类</option>
            <option
              v-for="cat in categoryStore.flatCategories"
              :key="cat.category.id"
              :value="cat.category.id"
            >
              {{ '　'.repeat(cat.depth) }}{{ cat.category.name }}
            </option>
          </select>
        </div>

        <!-- 标签 -->
        <div class="surface-card p-5">
          <label class="block text-sm text-muted mb-2">标签</label>
          <div class="flex flex-wrap gap-2 mb-3">
            <WasTag
              v-for="tagName in form.tagNames"
              :key="tagName"
              :label="tagName"
              variant="ai"
              removable
              @remove="removeTag(tagName)"
            />
          </div>
          <div class="flex gap-2">
            <WasInput
              v-model="newTagInput"
              placeholder="添加标签"
              size="sm"
              class="flex-1"
              @keyup.enter="addTag"
            />
            <WasButton variant="secondary" size="sm" @click="addTag">
              添加
            </WasButton>
          </div>
          <div class="mt-3 flex flex-wrap gap-1.5">
            <button
              v-for="tag in availableTags"
              :key="tag.id"
              class="text-xs px-2 py-1 border border-default rounded-sm text-muted hover:border-accent hover:text-accent transition-colors"
              @click="addExistingTag(tag.name)"
            >
              {{ tag.name }}
            </button>
          </div>
        </div>

        <!-- 封面图 -->
        <div class="surface-card p-5">
          <label class="block text-sm text-muted mb-2">封面图 URL</label>
          <WasInput
            v-model="form.coverImage"
            placeholder="https://..."
          />
          <div v-if="form.coverImage" class="mt-3 aspect-video bg-surface rounded-sm overflow-hidden">
            <img :src="form.coverImage" class="w-full h-full object-cover" alt="封面预览">
          </div>
        </div>

        <!-- 选项 -->
        <div class="surface-card p-5">
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              v-model="form.isTop"
              type="checkbox"
              class="w-4 h-4 accent-accent rounded border-default"
            >
            <span class="text-sm text-body">置顶文章</span>
          </label>
        </div>

        <!-- 文章信息（编辑模式） -->
        <div v-if="!isNew && currentArticle" class="surface-card p-5 text-xs text-placeholder space-y-2">
          <div class="flex justify-between">
            <span>状态</span>
            <span :class="statusClass(currentArticle.status)">{{ statusText(currentArticle.status) }}</span>
          </div>
          <div class="flex justify-between">
            <span>浏览</span>
            <span>{{ currentArticle.viewCount }}</span>
          </div>
          <div class="flex justify-between">
            <span>点赞</span>
            <span>{{ currentArticle.likeCount }}</span>
          </div>
          <div class="flex justify-between">
            <span>创建</span>
            <span>{{ formatDateTime(currentArticle.createdAt) }}</span>
          </div>
          <div class="flex justify-between">
            <span>更新</span>
            <span>{{ formatDateTime(currentArticle.updatedAt) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'admin',
  middleware: 'admin',
  pageTransition: false,
})

const route = useRoute()
const articleStore = useArticleStore()
const categoryStore = useCategoryStore()
const tagStore = useTagStore()
const ui = useUiStore()

const articleId = computed(() => route.params.id as string)
const isNew = computed(() => articleId.value === 'new')

// ---------- 加载数据 ----------

// 加载分类和标签（编辑时还需要加载文章）
await Promise.all([
  categoryStore.fetchCategories(),
  tagStore.fetchTags(),
])

// 编辑模式：加载文章详情
const currentArticle = ref<ArticleInfo | null>(null)
if (!isNew.value) {
  try {
    const res = await articleStore.fetchArticle(articleId.value)
    currentArticle.value = res.article
  }
  catch (e) {
    ui.toast('文章加载失败', 'error')
    navigateTo('/admin/articles')
  }
}

// ---------- 表单 ----------

const form = reactive({
  title: currentArticle.value?.title || '',
  content: currentArticle.value?.content || '',
  excerpt: currentArticle.value?.excerpt || '',
  coverImage: currentArticle.value?.coverImage || '',
  categoryId: currentArticle.value?.categoryId || '',
  tagNames: currentArticle.value?.tags.map(t => t.name) || [] as string[],
  isTop: currentArticle.value?.isTop || false,
})

const errors = reactive({
  title: '',
  content: '',
})

const newTagInput = ref('')
const saving = ref(false)
const publishing = ref(false)

// 可用标签（排除已选的）
const availableTags = computed(() => {
  return tagStore.tags.filter(t => !form.tagNames.includes(t.name))
})

// ---------- 标签操作 ----------

function addTag() {
  const name = newTagInput.value.trim()
  if (!name) return
  if (!form.tagNames.includes(name)) {
    form.tagNames.push(name)
  }
  newTagInput.value = ''
}

function addExistingTag(name: string) {
  if (!form.tagNames.includes(name)) {
    form.tagNames.push(name)
  }
}

function removeTag(name: string) {
  form.tagNames = form.tagNames.filter(n => n !== name)
}

// ---------- 验证与保存 ----------

function validate(): boolean {
  errors.title = ''
  errors.content = ''
  let valid = true

  if (!form.title.trim()) {
    errors.title = '请输入标题'
    valid = false
  }
  if (!form.content.trim()) {
    errors.content = '请输入文章内容'
    valid = false
  }

  return valid
}

async function saveAsDraft() {
  if (!validate()) return
  await doSave('draft')
}

async function saveAndPublish() {
  if (!validate()) return
  await doSave('published')
}

async function doSave(targetStatus: string) {
  const isPublish = targetStatus === 'published'
  if (isPublish) {
    publishing.value = true
  }
  else {
    saving.value = true
  }

  try {
    const data: CreateArticleRequest = {
      title: form.title.trim(),
      content: form.content.trim(),
      excerpt: form.excerpt.trim() || undefined,
      coverImage: form.coverImage || undefined,
      categoryId: form.categoryId || undefined,
      tagNames: form.tagNames.length > 0 ? form.tagNames : undefined,
    }

    let articleId: string

    if (isNew.value) {
      const res = await articleStore.createArticle(data)
      articleId = res.article.id
      currentArticle.value = res.article
    }
    else {
      await articleStore.updateArticle(articleId.value, data)
      articleId = articleId.value
      // 重新加载文章
      const res = await articleStore.fetchArticle(articleId)
      currentArticle.value = res.article
    }

    // 发布/归档操作
    if (isPublish && currentArticle.value?.status !== 'published') {
      await articleStore.publishArticle(articleId)
    }

    ui.toast(isNew.value ? '文章创建成功' : '文章更新成功', 'success')

    // 如果是新建，跳转到编辑模式
    if (isNew.value) {
      await navigateTo(`/admin/articles/edit/${articleId}`)
    }
  }
  catch (e: any) {
    ui.toast(e?.message || '保存失败', 'error')
  }
  finally {
    saving.value = false
    publishing.value = false
  }
}

// ---------- 工具函数 ----------

function statusClass(status: string) {
  switch (status) {
    case 'published': return 'text-success'
    case 'draft': return 'text-muted'
    case 'archived': return 'text-placeholder'
    default: return ''
  }
}

function statusText(status: string) {
  switch (status) {
    case 'published': return '已发布'
    case 'draft': return '草稿'
    case 'archived': return '已归档'
    default: return status
  }
}

function formatDateTime(date: string) {
  const d = new Date(date)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

useHead({
  title: `${isNew.value ? '写文章' : '编辑文章'} - 管理后台`,
})
</script>
