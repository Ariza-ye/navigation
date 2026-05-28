<script setup lang="ts">
import { BookOpen, Compass } from 'lucide-vue-next'

import ThemeSwitcher from '@/components/ThemeSwitcher.vue'
import UserMenu from '@/components/UserMenu.vue'
import type { UserSession } from '@/types/api'

defineProps<{
  user: UserSession | null
  activeModule: 'sites' | 'notes'
}>()

const emit = defineEmits<{
  login: []
  account: []
  settings: []
  logout: []
  module: [module: 'sites' | 'notes']
}>()
</script>

<template>
  <main class="relative mx-auto min-h-screen w-[min(1180px,calc(100%-24px))] py-7 sm:w-[min(1180px,calc(100%-40px))] sm:py-14">
    <div class="relative z-10 mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <nav class="flex w-fit rounded-xl border border-[var(--border-soft)] bg-[var(--surface)] p-1">
        <button
          type="button"
          class="module-tab"
          :class="{ 'module-tab-active': activeModule === 'sites' }"
          @click="emit('module', 'sites')"
        >
          <Compass class="h-4 w-4" /> 导航
        </button>
        <button
          type="button"
          class="module-tab"
          :class="{ 'module-tab-active': activeModule === 'notes' }"
          @click="emit('module', 'notes')"
        >
          <BookOpen class="h-4 w-4" /> 笔记
        </button>
      </nav>
      <UserMenu
        :user="user"
        @login="emit('login')"
        @account="emit('account')"
        @settings="emit('settings')"
        @logout="emit('logout')"
      />
    </div>
    <slot />
  </main>
  <ThemeSwitcher />
</template>

<style scoped>
.module-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  border-radius: 8px;
  padding: 0 12px;
  color: var(--page-muted);
  font-size: 14px;
  font-weight: 700;
  transition: background .2s ease, color .2s ease;
}

.module-tab:hover,
.module-tab-active {
  background: var(--surface-hover);
  color: var(--page-text);
}
</style>
