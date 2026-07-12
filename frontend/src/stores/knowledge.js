import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getCharacters, getRelations, getEvents, getEventCharacters } from '../api/index.js'
import { useReaderStore } from './reader.js'

export const useKnowledgeStore = defineStore('knowledge', () => {
  // State
  const characters = ref([])
  const relations = ref([])
  const events = ref([])
  const eventCharacters = ref([])
  const loading = ref(false)
  const polling = ref(false)

  // Getters
  const charactersByChapter = computed(() => {
    const map = {}
    for (const ch of characters.value) {
      const key = ch.revealChapter
      if (!map[key]) map[key] = []
      map[key].push(ch)
    }
    return map
  })

  const eventsByChapter = computed(() => {
    const map = {}
    for (const e of events.value) {
      const key = e.chapterNumber
      if (!map[key]) map[key] = []
      map[key].push(e)
    }
    return map
  })

  const eventCharacterMap = computed(() => {
    const map = {}
    for (const ec of eventCharacters.value) {
      if (!map[ec.eventId]) map[ec.eventId] = []
      map[ec.eventId].push(ec)
    }
    return map
  })

  // Actions
  async function fetchAll(bookId) {
    loading.value = true
    try {
      const [charRes, relRes, evtRes, ecRes] = await Promise.all([
        getCharacters(bookId).catch(() => ({ data: [] })),
        getRelations(bookId).catch(() => ({ data: [] })),
        getEvents(bookId).catch(() => ({ data: [] })),
        getEventCharacters(bookId).catch(() => ({ data: [] })),
      ])
      characters.value = charRes.data || []
      relations.value = relRes.data || []
      events.value = evtRes.data || []
      eventCharacters.value = ecRes.data || []
    } finally {
      loading.value = false
    }
  }

  async function pollForUpdates(bookId, maxAttempts = 10, interval = 3000) {
    if (polling.value) return
    polling.value = true
    const prevCount = characters.value.length

    for (let i = 0; i < maxAttempts; i++) {
      await new Promise(resolve => setTimeout(resolve, interval))

      const readerStore = useReaderStore()
      if (readerStore.currentBookId !== bookId) break

      try {
        const res = await getCharacters(bookId)
        const chars = res.data || []
        if (chars.length > prevCount) {
          // New data available — full refresh
          await fetchAll(bookId)
          break
        }
      } catch (e) {
        break
      }
    }

    polling.value = false
  }

  return {
    characters,
    relations,
    events,
    eventCharacters,
    loading,
    polling,
    charactersByChapter,
    eventsByChapter,
    eventCharacterMap,
    fetchAll,
    pollForUpdates,
  }
})
