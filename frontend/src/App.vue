<script setup lang="ts">
import { Plus, Tags } from 'lucide-vue-next'
import { onMounted, ref, watch } from 'vue'

import AccountDialog from '@/components/AccountDialog.vue'
import AppShell from '@/components/AppShell.vue'
import CategoryDialog from '@/components/CategoryDialog.vue'
import CategoryTabs from '@/components/CategoryTabs.vue'
import HeroSection from '@/components/HeroSection.vue'
import LoginDialog from '@/components/LoginDialog.vue'
import NoteWorkspace from '@/components/NoteWorkspace.vue'
import SettingsDialog from '@/components/SettingsDialog.vue'
import SiteDialog from '@/components/SiteDialog.vue'
import SiteGrid from '@/components/SiteGrid.vue'
import UiButton from '@/components/ui/Button.vue'
import { useAuth } from '@/composables/useAuth'
import { useNotes } from '@/composables/useNotes'
import { useSettings } from '@/composables/useSettings'
import { useSites } from '@/composables/useSites'
import { useTheme } from '@/composables/useTheme'
import { debounce } from '@/lib/utils'
import type { AccountInput, AppSettings, CategoryStat, Note, Site, SiteInput } from '@/types/api'

const auth = useAuth()
const sites = useSites()
const notes = useNotes()
const settingsStore = useSettings()
const { applyDefaultTheme } = useTheme()

const activeModule = ref<'sites' | 'notes'>('sites')
const siteDialogOpen = ref(false)
const editingSite = ref<Site | null>(null)
const siteError = ref('')
const categoryDialogOpen = ref(false)
const categoryError = ref('')
const accountDialogOpen = ref(false)
const accountError = ref('')
const settingsDialogOpen = ref(false)
const settingsError = ref('')
const bootError = ref('')

const debouncedLoadSites = debounce(async () => {
  try {
    await sites.loadSitesOnly()
  } catch (error) {
    bootError.value = error instanceof Error ? error.message : '读取站点数据失败'
  }
}, 250)

watch(sites.query, () => {
  debouncedLoadSites()
})

const debouncedLoadNotes = debounce(async () => {
  if (!auth.user.value) return
  try {
    await notes.loadNotes()
  } catch (error) {
    if (auth.handleAuthError(error)) return
    notes.error.value = error instanceof Error ? error.message : '读取笔记失败'
  }
}, 250)

watch(notes.query, () => {
  debouncedLoadNotes()
})

watch(auth.user, async (user) => {
  if (user && activeModule.value === 'notes') {
    await loadNotes()
  }
})

async function bootstrap() {
  await auth.refreshSession()
  try {
    const settings = await settingsStore.loadSettings()
    applyDefaultTheme(settings.theme)
    await sites.loadAll()
  } catch (error) {
    bootError.value = error instanceof Error ? error.message : '页面加载失败'
  }
}

function openSiteDialog(site: Site | null = null) {
  if (!auth.requireLogin()) return
  siteError.value = ''
  editingSite.value = site
  siteDialogOpen.value = true
}

async function saveSite(input: SiteInput, id?: string) {
  siteError.value = ''
  try {
    await sites.saveSite(input, id)
    siteDialogOpen.value = false
  } catch (error) {
    if (auth.handleAuthError(error)) return
    siteError.value = error instanceof Error ? error.message : '保存站点失败'
  }
}

async function removeSite(site: Site) {
  if (!auth.requireLogin()) return
  if (!window.confirm(`确定删除「${site.name}」吗？`)) return
  try {
    await sites.removeSite(site)
  } catch (error) {
    if (!auth.handleAuthError(error)) window.alert(error instanceof Error ? error.message : '删除站点失败')
  }
}

async function openCategoryDialog() {
  if (!auth.requireLogin()) return
  categoryError.value = ''
  categoryDialogOpen.value = true
  try {
    await sites.loadCategoryStats()
  } catch (error) {
    if (auth.handleAuthError(error)) return
    categoryError.value = error instanceof Error ? error.message : '读取分类失败'
  }
}

async function removeCategory(category: CategoryStat) {
  if (!auth.requireLogin()) return
  const message = `确定删除「${category.name}」分类吗？该分类下的 ${category.count} 个站点会保留，但分类会被清空。`
  if (!window.confirm(message)) return
  try {
    await sites.removeCategory(category.name)
  } catch (error) {
    if (!auth.handleAuthError(error)) window.alert(error instanceof Error ? error.message : '删除分类失败')
  }
}

async function renameCategory(category: CategoryStat) {
  if (!auth.requireLogin()) return
  const nextName = window.prompt('请输入新的分类名称', category.name)
  if (nextName === null) return
  const normalizedName = nextName.trim()
  if (!normalizedName || normalizedName === category.name) return
  try {
    await sites.renameCategory(category.name, normalizedName)
  } catch (error) {
    if (!auth.handleAuthError(error)) window.alert(error instanceof Error ? error.message : '重命名分类失败')
  }
}

async function saveAccount(input: AccountInput) {
  accountError.value = ''
  try {
    await auth.saveAccount(input)
    accountDialogOpen.value = false
  } catch (error) {
    if (auth.handleAuthError(error)) return
    accountError.value = error instanceof Error ? error.message : '保存账号失败'
  }
}

async function saveSettings(input: AppSettings) {
  settingsError.value = ''
  try {
    const settings = await settingsStore.saveSettings(input)
    applyDefaultTheme(settings.theme)
    settingsDialogOpen.value = false
  } catch (error) {
    if (auth.handleAuthError(error)) return
    settingsError.value = error instanceof Error ? error.message : '保存设置失败'
  }
}

async function changeCategory(category: string) {
  sites.category.value = category
  await sites.loadSitesOnly()
}

async function switchModule(module: 'sites' | 'notes') {
  if (module === 'notes' && !auth.requireLogin()) return
  activeModule.value = module
  if (module === 'notes') {
    await loadNotes()
  }
}

async function loadNotes() {
  try {
    await notes.loadNotes()
  } catch (error) {
    if (auth.handleAuthError(error)) return
    notes.error.value = error instanceof Error ? error.message : '读取笔记失败'
  }
}

async function selectNote(note: Note) {
  try {
    await notes.selectNote(note)
  } catch (error) {
    if (auth.handleAuthError(error)) return
    notes.error.value = error instanceof Error ? error.message : '读取笔记失败'
  }
}

async function saveNote() {
  try {
    await notes.saveDraft()
  } catch (error) {
    if (auth.handleAuthError(error)) return
    notes.error.value = error instanceof Error ? error.message : '保存笔记失败'
  }
}

async function deleteNote() {
  if (!notes.selected.value) return
  if (!window.confirm(`确定删除「${notes.selected.value.title}」吗？文件会保留，笔记会被软删除。`)) return
  try {
    await notes.removeSelected()
  } catch (error) {
    if (auth.handleAuthError(error)) return
    notes.error.value = error instanceof Error ? error.message : '删除笔记失败'
  }
}

function updateNoteQuery(query: string) {
  notes.query.value = query
}

onMounted(() => {
  bootstrap()
})
</script>

<template>
  <AppShell
    :user="auth.user.value"
    :active-module="activeModule"
    @login="auth.loginOpen.value = true"
    @account="accountDialogOpen = true"
    @settings="settingsDialogOpen = true"
    @logout="auth.logout"
    @module="switchModule"
  >
    <template v-if="activeModule === 'sites'">
      <HeroSection v-model:query="sites.query.value" :settings="settingsStore.settings.value" :stats="sites.stats.value" />
      <CategoryTabs :categories="sites.categories.value" :active="sites.category.value" @change="changeCategory" />

      <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 class="text-2xl font-semibold text-[var(--page-text)]">常用入口</h2>
          <p class="mt-1 text-sm text-[var(--page-soft)]">点击卡片快速访问</p>
        </div>
        <div v-if="auth.user.value" class="flex flex-wrap gap-2">
          <UiButton variant="outline" @click="openCategoryDialog">
            <Tags class="h-4 w-4" /> 分类管理
          </UiButton>
          <UiButton @click="openSiteDialog()">
            <Plus class="h-4 w-4" /> 新增站点
          </UiButton>
        </div>
      </div>

      <div v-if="bootError" class="mb-4 rounded-[16px] border border-[var(--danger-border)] bg-[var(--danger-bg)] px-4 py-3 text-sm text-[var(--danger-text)]">
        {{ bootError }}
      </div>
      <SiteGrid :sites="sites.sites.value" :user="auth.user.value" @edit="openSiteDialog" @delete="removeSite" />
    </template>

    <NoteWorkspace
      v-else
      :notes="notes.notes.value"
      :selected="notes.selected.value"
      :draft="notes.draft.value"
      :loading="notes.loading.value"
      :saving="notes.saving.value"
      :error="notes.error.value"
      :query="notes.query.value"
      @new="notes.resetDraft"
      @select="selectNote"
      @save="saveNote"
      @delete="deleteNote"
      @search="updateNoteQuery"
      @update:draft="notes.draft.value = $event"
    />
  </AppShell>

  <LoginDialog
    :open="auth.loginOpen.value"
    :error="auth.loginError.value"
    @close="auth.setAnonymous"
    @login="auth.login"
  />
  <SiteDialog
    :open="siteDialogOpen"
    :site="editingSite"
    :categories="sites.categories.value"
    :error="siteError"
    @close="siteDialogOpen = false"
    @save="saveSite"
  />
  <CategoryDialog
    :open="categoryDialogOpen"
    :categories="sites.categoryStats.value"
    :error="categoryError"
    @close="categoryDialogOpen = false"
    @rename="renameCategory"
    @delete="removeCategory"
  />
  <AccountDialog
    :open="accountDialogOpen"
    :user="auth.user.value"
    :error="accountError"
    @close="accountDialogOpen = false"
    @save="saveAccount"
  />
  <SettingsDialog
    :open="settingsDialogOpen"
    :settings="settingsStore.settings.value"
    :error="settingsError"
    @close="settingsDialogOpen = false"
    @save="saveSettings"
  />
</template>
