<script setup lang="ts">
import { Archive, FileText, Loader2, Pin, Plus, Save, Search, Trash2 } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import UiButton from '@/components/ui/Button.vue'
import { debounce } from '@/lib/utils'
import type { Note, NoteContent, NoteInput } from '@/types/api'

const props = defineProps<{
  notes: Note[]
  selected: NoteContent | null
  draft: NoteInput
  loading: boolean
  saving: boolean
  error: string
  query: string
}>()

const emit = defineEmits<{
  new: []
  select: [note: Note]
  save: []
  delete: []
  search: [query: string]
  'update:draft': [draft: NoteInput]
}>()

const tagText = ref('')

const debouncedSearch = debounce((value: string) => {
  emit('search', value)
}, 250)

const activeID = computed(() => props.selected?.id || '')

watch(() => props.query, (value) => {
  debouncedSearch(value)
})

watch(() => props.draft.tags, (tags) => {
  tagText.value = tags.join(', ')
}, { immediate: true })

function patchDraft(patch: Partial<NoteInput>) {
  emit('update:draft', { ...props.draft, ...patch })
}

function updateTags(value: string) {
  tagText.value = value
  const tags = value.split(',').map((tag) => tag.trim()).filter(Boolean)
  patchDraft({ tags })
}

function formatDate(value: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-SG', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}
</script>

<template>
  <section class="notes-workspace">
    <div class="notes-panel notes-sidebar">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h2 class="text-2xl font-semibold text-[var(--page-text)]">笔记</h2>
          <p class="mt-1 text-sm text-[var(--page-soft)]">Markdown 草稿与长期资料</p>
        </div>
        <UiButton size="icon" title="新建笔记" @click="emit('new')">
          <Plus class="h-4 w-4" />
        </UiButton>
      </div>

      <label class="mt-5 flex h-11 items-center gap-2 rounded-lg border border-[var(--border-soft)] bg-[var(--surface-input)] px-3 text-[var(--page-muted)]">
        <Search class="h-4 w-4 shrink-0" />
        <input
          :value="query"
          class="min-w-0 flex-1 bg-transparent text-sm text-[var(--page-text)] outline-none placeholder:text-[var(--page-soft)]"
          placeholder="搜索标题或摘要"
          @input="emit('search', ($event.target as HTMLInputElement).value)"
        />
      </label>

      <div class="mt-5 grid gap-2">
        <button
          v-for="note in notes"
          :key="note.id"
          type="button"
          class="note-row"
          :class="{ 'note-row-active': note.id === activeID }"
          @click="emit('select', note)"
        >
          <span class="flex min-w-0 items-center gap-2">
            <Pin v-if="note.pinned" class="h-3.5 w-3.5 shrink-0 text-[var(--accent)]" />
            <FileText v-else class="h-3.5 w-3.5 shrink-0 text-[var(--page-soft)]" />
            <span class="truncate font-semibold">{{ note.title }}</span>
          </span>
          <span class="line-clamp-2 text-left text-xs leading-5 text-[var(--page-soft)]">{{ note.summary || '暂无摘要' }}</span>
          <span class="flex items-center justify-between gap-3 text-[11px] text-[var(--page-soft)]">
            <span class="truncate">{{ note.tags.join(' / ') || '未标记' }}</span>
            <span class="shrink-0">{{ formatDate(note.updatedAt) }}</span>
          </span>
        </button>

        <div v-if="!notes.length && !loading" class="rounded-lg border border-dashed border-[var(--border)] px-4 py-8 text-center text-sm text-[var(--page-soft)]">
          暂无笔记
        </div>
        <div v-if="loading" class="flex items-center justify-center gap-2 py-8 text-sm text-[var(--page-muted)]">
          <Loader2 class="h-4 w-4 animate-spin" /> 读取中
        </div>
      </div>
    </div>

    <form class="notes-panel notes-editor" @submit.prevent="emit('save')">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="min-w-0">
          <p class="text-xs font-semibold uppercase tracking-[.16em] text-[var(--page-soft)]">
            {{ selected ? '正在编辑' : '新建笔记' }}
          </p>
          <h3 class="mt-1 truncate text-xl font-semibold text-[var(--page-text)]">
            {{ draft.title || '未命名笔记' }}
          </h3>
        </div>
        <div class="flex flex-wrap gap-2">
          <UiButton type="button" variant="outline" @click="patchDraft({ pinned: !draft.pinned })">
            <Pin class="h-4 w-4" /> {{ draft.pinned ? '取消置顶' : '置顶' }}
          </UiButton>
          <UiButton type="submit" :disabled="saving">
            <Save class="h-4 w-4" /> {{ saving ? '保存中' : '保存' }}
          </UiButton>
          <UiButton v-if="selected" type="button" variant="danger" @click="emit('delete')">
            <Trash2 class="h-4 w-4" /> 删除
          </UiButton>
        </div>
      </div>

      <div v-if="error" class="rounded-lg border border-[var(--danger-border)] bg-[var(--danger-bg)] px-4 py-3 text-sm text-[var(--danger-text)]">
        {{ error }}
      </div>

      <div class="grid gap-4 md:grid-cols-[1fr_180px]">
        <label class="grid gap-2 text-sm text-[var(--page-muted)]">
          <span>标题</span>
          <input
            :value="draft.title"
            class="h-11 rounded-lg border border-[var(--border-soft)] bg-[var(--surface-input)] px-3 text-[15px] text-[var(--page-text)] outline-none transition placeholder:text-[var(--page-soft)] focus:border-[var(--focus)] focus:ring-4 focus:ring-[var(--focus-ring)]"
            placeholder="笔记标题"
            @input="patchDraft({ title: ($event.target as HTMLInputElement).value })"
          />
        </label>
        <label class="grid gap-2 text-sm text-[var(--page-muted)]">
          <span>状态</span>
          <select
            :value="draft.status"
            class="h-11 rounded-lg border border-[var(--border-soft)] bg-[var(--surface-input)] px-3 text-[15px] text-[var(--page-text)] outline-none transition focus:border-[var(--focus)] focus:ring-4 focus:ring-[var(--focus-ring)]"
            @change="patchDraft({ status: ($event.target as HTMLSelectElement).value as NoteInput['status'] })"
          >
            <option value="active">活动</option>
            <option value="archived">归档</option>
          </select>
        </label>
      </div>

      <label class="grid gap-2 text-sm text-[var(--page-muted)]">
        <span>标签</span>
        <input
          :value="tagText"
          class="h-11 rounded-lg border border-[var(--border-soft)] bg-[var(--surface-input)] px-3 text-[15px] text-[var(--page-text)] outline-none transition placeholder:text-[var(--page-soft)] focus:border-[var(--focus)] focus:ring-4 focus:ring-[var(--focus-ring)]"
          placeholder="idea, work"
          @input="updateTags(($event.target as HTMLInputElement).value)"
        />
      </label>

      <label class="grid min-h-[420px] flex-1 gap-2 text-sm text-[var(--page-muted)]">
        <span class="flex items-center gap-2"><Archive class="h-4 w-4" /> Markdown</span>
        <textarea
          :value="draft.content"
          class="min-h-[420px] flex-1 resize-y rounded-lg border border-[var(--border-soft)] bg-[var(--surface-input)] px-4 py-3 text-[15px] leading-7 text-[var(--page-text)] outline-none transition placeholder:text-[var(--page-soft)] focus:border-[var(--focus)] focus:ring-4 focus:ring-[var(--focus-ring)]"
          placeholder="# 标题&#10;&#10;写下正文..."
          @input="patchDraft({ content: ($event.target as HTMLTextAreaElement).value })"
        />
      </label>
    </form>
  </section>
</template>

<style scoped>
.notes-workspace {
  display: grid;
  grid-template-columns: minmax(280px, 360px) minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

.notes-panel {
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--card-bg);
  box-shadow: 0 18px 60px var(--shadow);
}

.notes-sidebar {
  padding: 20px;
}

.notes-editor {
  display: flex;
  min-height: 680px;
  flex-direction: column;
  gap: 18px;
  padding: 22px;
}

.note-row {
  display: grid;
  gap: 8px;
  width: 100%;
  min-height: 104px;
  border: 1px solid var(--border-soft);
  border-radius: 12px;
  background: var(--surface);
  padding: 12px;
  color: var(--page-text);
  transition: border-color .2s ease, background .2s ease, transform .2s ease;
}

.note-row:hover,
.note-row-active {
  border-color: var(--border-hover);
  background: var(--surface-hover);
}

.note-row:hover {
  transform: translateY(-1px);
}

@media (max-width: 860px) {
  .notes-workspace {
    grid-template-columns: 1fr;
  }

  .notes-editor {
    min-height: auto;
  }
}
</style>
