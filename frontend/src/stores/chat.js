import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  getConversations,
  createConversation,
  getMessages,
  deleteConversation,
  askQuestion,
} from '../api/index.js'

export const useChatStore = defineStore('chat', () => {
  // State
  const conversations = ref([])
  const activeConversationId = ref(null)
  const messages = ref([])
  const sending = ref(false)

  // Getters
  const activeConversation = computed(() => {
    return conversations.value.find(c => c.id === activeConversationId.value)
  })

  const sortedMessages = computed(() => {
    return [...messages.value].sort((a, b) =>
      new Date(a.createdAt) - new Date(b.createdAt)
    )
  })

  // Actions
  async function loadConversations(bookId) {
    try {
      const res = await getConversations(bookId)
      conversations.value = res.data || []
    } catch (e) {
      console.warn('Failed to load conversations:', e)
    }
  }

  async function createNewConversation(bookId, title) {
    try {
      const res = await createConversation(bookId, title || '新对话')
      const conv = res.data
      conversations.value.unshift(conv)
      activeConversationId.value = conv.id
      messages.value = []
      return conv
    } catch (e) {
      console.warn('Failed to create conversation:', e)
      return null
    }
  }

  async function switchConversation(convId) {
    activeConversationId.value = convId
    messages.value = []
    try {
      const res = await getMessages(convId)
      messages.value = res.data || []
    } catch (e) {
      console.warn('Failed to load messages:', e)
    }
  }

  async function sendMessage(bookId, question) {
    if (sending.value || !question.trim()) return null
    sending.value = true

    try {
      // Auto-create conversation if needed
      let convId = activeConversationId.value
      if (!convId) {
        const conv = await createNewConversation(bookId, '新对话')
        if (conv) convId = conv.id
      }

      const res = await askQuestion(bookId, convId, question)
      const result = res.data

      // Reload messages to get the full history
      if (convId) {
        const msgsRes = await getMessages(convId)
        messages.value = msgsRes.data || []
      }

      return result
    } catch (e) {
      console.warn('Failed to send message:', e)
      return null
    } finally {
      sending.value = false
    }
  }

  async function removeConversation(convId) {
    try {
      await deleteConversation(convId)
      conversations.value = conversations.value.filter(c => c.id !== convId)
      if (activeConversationId.value === convId) {
        activeConversationId.value = null
        messages.value = []
      }
    } catch (e) {
      console.warn('Failed to delete conversation:', e)
    }
  }

  return {
    conversations,
    activeConversationId,
    messages,
    sending,
    activeConversation,
    sortedMessages,
    loadConversations,
    createNewConversation,
    switchConversation,
    sendMessage,
    removeConversation,
  }
})
