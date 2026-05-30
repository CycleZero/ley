<!--
  分类管理页 /admin/categories
-->
<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-serif-jp font-semibold text-heading tracking-wider">
        分类管理
      </h1>
      <WasButton variant="primary" @click="showCreateModal = true">
        <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v16m8-8H4" />
        </svg>
        新建分类
      </WasButton>
    </div>

    <!-- 分类列表 -->
    <div class="surface-card overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-subtle text-left text-placeholder">
            <th class="px-4 py-3 font-normal">名称</th>
            <th class="px-4 py-3 font-normal w-32 hidden sm:table-cell">Slug</th>
            <th class="px-4 py-3 font-normal w-20 hidden md:table-cell">文章数</th>
            <th class="px-4 py-3 font-normal w-20">排序</th>
            <th class="px-4 py-3 font-normal w-24 text-right">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-subtle">
          <tr
            v-for="item in categoryStore.flatCategories"
            :key="item.category.id"
            class="hover:bg-surface-hover transition-colors"
          >
            <td class="px-4 py-3">
              <div class="flex items-center gap-2">
                <span :style="{ width: item.depth * 1.5 + 'rem' }" />
                <span v-if="item.depth > 0" class="text-placeholder text-xs">└</span>
                <span class="text-body">{{ item.category.name }}</span>
              </div>
            </td>
            <td class="px-4 py-3 text-placeholder hidden sm:table-cell">{{ item.category.slug }}</td>
            <td class="px-4 py-3 text-placeholder hidden md:table-cell">{{ item.category.articleCount }}</td>
            <td class="px-4 py-3">{{ item.category.sortOrder }}</td>
            <td class="px-4 py-3 text-right">
              <div class="flex items-center justify-end gap-2">
                <button
                  class="text-xs text-placeholder hover:text-accent transition-colors"
                  @click="edit(item.category)"
                >
                  编辑
                </button>
                <span class="text-subtle">|</span>
                <button
                  class="text-xs text-placeholder hover:text-error transition-colors"
                  @click="confirmDelete(item.category)"
                >
                  删除
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!categoryStore.categories.length" class="py-16 text-center text-placeholder text-sm">
        暂无分类
      </div>
    </div>

    <!-- 新建/编辑弹窗 -->
    <WasModal
      v-model="modalOpen"
      :title="editingCategory ? '编辑分类' : '新建分类'"
      confirm-text="保存"
      @confirm="saveCategory"
    >
      <div class="space-y-4">
        <WasInput
          v-model="form.name"
          label="名称"
          placeholder="分类名称"
          :error="formErrors.name"
        />
        <WasInput
          v-model="form.slug"
          label="Slug"
          placeholder="url-friendly-name"
          :error="formErrors.slug"
        />
        <WasInput
          v-model="form.description"
          label="描述"
          placeholder="分类描述（可选）"
        />
        <div>
          <label class="block text-sm text-muted mb-1.5">父分类</label>
          <select
            v-model="form.parentId"
            class="w-full bg-base border border-default text-body text-sm rounded-sm px-3 py-2.5 focus:outline-none focus:border-accent"
          >
            <option value="">无（顶级分类）</option>
            <option
              v-for="cat in availableParents"
              :key="cat.id"
              :value="cat.id"
            >
              {{ cat.name }}
            </option>
          </select>
        </div>
        <WasInput
          v-model.number="form.sortOrder"
          label="排序"
          type="number"
          placeholder="0"
        />
      </div>
    </WasModal>

    <!-- 删除确认 -->
    <WasModal
      v-model="deleteModalOpen"
      title="确认删除"
      confirm-text="删除"
      confirm-variant="danger"
      @confirm="doDelete"
    >
      <p class="text-sm text-muted">
        确定要删除分类「{{ categoryToDelete?.name }}」吗？此操作不可撤销。
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

const categoryStore = useCategoryStore()
const ui = useUiStore()

// 加载分类
await categoryStore.fetchCategories()

// ---------- 表单 ----------

const modalOpen = ref(false)
const editingCategory = ref<CategoryInfo | null>(null)
const showCreateModal = ref(false)

watch(showCreateModal, (v) => {
  if (v) {
    editingCategory.value = null
    resetForm()
    modalOpen.value = true
  }
})

watch(modalOpen, (v) => {
  if (!v) showCreateModal.value = false
})

const form = reactive({
  name: '',
  slug: '',
  description: '',
  parentId: '',
  sortOrder: 0,
})

const formErrors = reactive({
  name: '',
  slug: '',
})

function resetForm() {
  form.name = ''
  form.slug = ''
  form.description = ''
  form.parentId = ''
  form.sortOrder = 0
  formErrors.name = ''
  formErrors.slug = ''
}

function edit(cat: CategoryInfo) {
  editingCategory.value = cat
  form.name = cat.name
  form.slug = cat.slug
  form.description = cat.description
  form.parentId = cat.parentId || ''
  form.sortOrder = cat.sortOrder
  modalOpen.value = true
}

const availableParents = computed(() => {
  if (!editingCategory.value) return categoryStore.flatCategories.map(c => c.category)
  // 编辑时排除自己和自己的子分类
  return categoryStore.flatCategories
    .filter(c => c.category.id !== editingCategory.value?.id)
    .map(c => c.category)
})

// ---------- CRUD ----------

function validate(): boolean {
  formErrors.name = ''
  formErrors.slug = ''
  let valid = true
  if (!form.name.trim()) {
    formErrors.name = '请输入名称'
    valid = false
  }
  if (!form.slug.trim()) {
    formErrors.slug = '请输入 Slug'
    valid = false
  }
  return valid
}

async function saveCategory() {
  if (!validate()) return

  try {
    const data = {
      name: form.name.trim(),
      slug: form.slug.trim(),
      description: form.description || undefined,
      parentId: form.parentId || undefined,
      sortOrder: form.sortOrder,
    }

    if (editingCategory.value) {
      await categoryStore.updateCategory(editingCategory.value.id, data)
      ui.toast('分类已更新', 'success')
    }
    else {
      await categoryStore.createCategory(data as CreateCategoryRequest)
      ui.toast('分类已创建', 'success')
    }

    modalOpen.value = false
    resetForm()
  }
  catch (e: any) {
    ui.toast(e?.message || '保存失败', 'error')
  }
}

const deleteModalOpen = ref(false)
const categoryToDelete = ref<CategoryInfo | null>(null)

function confirmDelete(cat: CategoryInfo) {
  categoryToDelete.value = cat
  deleteModalOpen.value = true
}

async function doDelete() {
  if (!categoryToDelete.value) return
  try {
    await categoryStore.deleteCategory(categoryToDelete.value.id)
    ui.toast('分类已删除', 'success')
  }
  catch (e: any) {
    ui.toast(e?.message || '删除失败', 'error')
  }
  finally {
    deleteModalOpen.value = false
    categoryToDelete.value = null
  }
}

useHead({
  title: '分类管理 - 管理后台',
})
</script>
