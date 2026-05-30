<!--
  标签管理页 /admin/tags
-->
<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-serif-jp font-semibold text-heading tracking-wider">
        标签管理
      </h1>
      <div class="flex items-center gap-2">
        <WasInput
          v-model="newTagName"
          placeholder="新标签名称"
          class="w-40"
          @keyup.enter="createTag"
        />
        <WasButton variant="primary" @click="createTag">
          <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v16m8-8H4" />
          </svg>
          添加
        </WasButton>
      </div>
    </div>

    <!-- 标签列表 -->
    <div class="surface-card overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-subtle text-left text-placeholder">
            <th class="px-4 py-3 font-normal">名称</th>
            <th class="px-4 py-3 font-normal w-32 hidden sm:table-cell">Slug</th>
            <th class="px-4 py-3 font-normal w-24">文章数</th>
            <th class="px-4 py-3 font-normal w-20 text-right">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-subtle">
          <tr
            v-for="tag in tagStore.tags"
            :key="tag.id"
            class="hover:bg-surface-hover transition-colors"
          >
            <td class="px-4 py-3">
              <span class="text-body">{{ tag.name }}</span>
            </td>
            <td class="px-4 py-3 text-placeholder hidden sm:table-cell">{{ tag.slug }}</td>
            <td class="px-4 py-3 text-placeholder">{{ tag.articleCount }}</td>
            <td class="px-4 py-3 text-right">
              <button
                class="text-xs text-placeholder hover:text-error transition-colors"
                @click="confirmDelete(tag)"
              >
                删除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!tagStore.tags.length" class="py-16 text-center text-placeholder text-sm">
        暂无标签
      </div>
    </div>

    <!-- 删除确认 -->
    <WasModal
      v-model="deleteModalOpen"
      title="确认删除"
      confirm-text="删除"
      confirm-variant="danger"
      @confirm="doDelete"
    >
      <p class="text-sm text-muted">
        确定要删除标签「{{ tagToDelete?.name }}」吗？此操作不可撤销。
      </p>
    </WasModal>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'admin',
  middleware: 'admin',
  pageTransition: false,
})

const tagStore = useTagStore()
const ui = useUiStore()

// 加载标签
await tagStore.fetchTags()

// ---------- 新建 ----------

const newTagName = ref('')

async function createTag() {
  const name = newTagName.value.trim()
  if (!name) {
    ui.toast('请输入标签名称', 'error')
    return
  }

  // 检查是否已存在
  if (tagStore.tags.some(t => t.name === name)) {
    ui.toast('标签已存在', 'error')
    return
  }

  try {
    await tagStore.createTag({ name })
    ui.toast('标签已创建', 'success')
    newTagName.value = ''
  }
  catch (e: any) {
    ui.toast(e?.message || '创建失败', 'error')
  }
}

// ---------- 删除 ----------

const deleteModalOpen = ref(false)
const tagToDelete = ref<TagInfo | null>(null)

function confirmDelete(tag: TagInfo) {
  tagToDelete.value = tag
  deleteModalOpen.value = true
}

async function doDelete() {
  if (!tagToDelete.value) return
  try {
    await tagStore.deleteTag(tagToDelete.value.id)
    ui.toast('标签已删除', 'success')
  }
  catch (e: any) {
    ui.toast(e?.message || '删除失败', 'error')
  }
  finally {
    deleteModalOpen.value = false
    tagToDelete.value = null
  }
}

useHead({
  title: '标签管理 - 管理后台',
})
</script>
