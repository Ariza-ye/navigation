import { ref } from 'vue'

import {
  createNote,
  deleteNote as deleteNoteRequest,
  getNote,
  listNotes,
  updateNote,
} from '@/lib/api'
import type { Note, NoteContent, NoteInput } from '@/types/api'

const emptyDraft: NoteInput = {
  title: '',
  content: '',
  tags: [],
  status: 'active',
  pinned: false,
}

export function useNotes() {
  const notes = ref<Note[]>([])
  const selected = ref<NoteContent | null>(null)
  const draft = ref<NoteInput>({ ...emptyDraft })
  const query = ref('')
  const loading = ref(false)
  const saving = ref(false)
  const error = ref('')

  function resetDraft() {
    selected.value = null
    draft.value = { ...emptyDraft }
  }

  async function loadNotes() {
    loading.value = true
    error.value = ''
    try {
      notes.value = await listNotes({ q: query.value })
    } finally {
      loading.value = false
    }
  }

  async function selectNote(note: Note) {
    loading.value = true
    error.value = ''
    try {
      const detail = await getNote(note.id)
      selected.value = detail
      draft.value = {
        title: detail.title,
        content: detail.content,
        tags: [...detail.tags],
        status: detail.status === 'deleted' ? 'active' : detail.status,
        pinned: detail.pinned,
      }
    } finally {
      loading.value = false
    }
  }

  async function saveDraft() {
    saving.value = true
    error.value = ''
    try {
      const saved = selected.value
        ? await updateNote(selected.value.id, draft.value)
        : await createNote(draft.value)
      selected.value = saved
      draft.value = {
        title: saved.title,
        content: saved.content,
        tags: [...saved.tags],
        status: saved.status,
        pinned: saved.pinned,
      }
      await loadNotes()
      const next = notes.value.find((note) => note.id === saved.id)
      if (next) selected.value = saved
      return saved
    } finally {
      saving.value = false
    }
  }

  async function removeSelected() {
    if (!selected.value) return
    const id = selected.value.id
    await deleteNoteRequest(id)
    resetDraft()
    await loadNotes()
  }

  return {
    notes,
    selected,
    draft,
    query,
    loading,
    saving,
    error,
    loadNotes,
    selectNote,
    saveDraft,
    removeSelected,
    resetDraft,
  }
}
