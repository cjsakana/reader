import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getBookDetail, getChapterContent, updateProgress } from '../api/index.js'

export const useReaderStore = defineStore('reader', () => {
  // State
  const currentBookId = ref(null)
  const bookDetail = ref(null)         // { book, chapters[] }
  const currentChapter = ref(null)     // { id, bookId, index, title, content, nextId, prevId }
  const loading = ref(false)

  // Getters
  const progressPercent = computed(() => {
    if (!bookDetail.value || !bookDetail.value.book) return 0
    const book = bookDetail.value.book
    if (book.totalChapters <= 1) return 0
    return Math.round((book.chapterIndex / (book.totalChapters - 1)) * 100)
  })

  const currentChapterTitle = computed(() => {
    if (!currentChapter.value) return ''
    return currentChapter.value.title || `第 ${currentChapter.value.index + 1} 章`
  })

  // Actions
  async function loadBook(bookId) {
    currentBookId.value = bookId
    loading.value = true
    try {
      const res = await getBookDetail(bookId)
      bookDetail.value = res.data
      return res.data
    } finally {
      loading.value = false
    }
  }

  async function loadChapter(chapterId) {
    if (!currentBookId.value) return null
    loading.value = true
    try {
      const res = await getChapterContent(currentBookId.value, chapterId)
      currentChapter.value = res.data
      return res.data
    } finally {
      loading.value = false
    }
  }

  async function switchChapter(chapterId) {
    const ch = await loadChapter(chapterId)
    if (ch) {
      // Save progress to backend
      try {
        await updateProgress(currentBookId.value, ch.index, 0)
      } catch (e) {
        console.warn('Failed to save progress:', e)
      }
    }
    return ch
  }

  function saveCharOffset(offset) {
    if (!currentBookId.value || !currentChapter.value) return
    // Debounced save — caller is responsible for throttling
    updateProgress(currentBookId.value, currentChapter.value.index, offset).catch(() => {})
  }

  return {
    currentBookId,
    bookDetail,
    currentChapter,
    loading,
    progressPercent,
    currentChapterTitle,
    loadBook,
    loadChapter,
    switchChapter,
    saveCharOffset,
  }
})
