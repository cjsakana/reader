import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

// Request interceptor (reserved for future use: auth tokens, loading state, etc.)
api.interceptors.request.use(
  (config) => config,
  (error) => Promise.reject(error),
)

// Response interceptor: unwrap standardized {code, msg, data} envelope
api.interceptors.response.use(
  (response) => {
    const body = response.data
    if (body && body.code === 0) {
      // Replace response.data with the inner data so callers see it transparently
      response.data = body.data
      return response
    }
    // Handle responses that don't have {code, msg, data} format (e.g. static files)
    if (body && body.code !== undefined) {
      // Application-level error
      return Promise.reject(new Error(body.msg || '未知错误'))
    }
    // Legacy format or non-API response — pass through unchanged
    return response
  },
  (error) => {
    // Network errors, CORS errors, HTTP 5xx from reverse proxy, etc.
    if (error.response) {
      const body = error.response.data
      if (body && body.msg) {
        return Promise.reject(new Error(body.msg))
      }
    }
    return Promise.reject(new Error(error.message || '网络错误'))
  },
)

// 上传图书
export function uploadBook(file) {
  const formData = new FormData()
  formData.append('file', file)
  return api.post('/books', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

// 获取图书列表
export function getBooks() {
  return api.get('/books')
}

// 获取图书详情及章节目录
export function getBookDetail(id) {
  return api.get(`/books/${id}`)
}

// 获取章节内容
export function getChapterContent(bookId, chapterId) {
  return api.get(`/books/${bookId}/chapters/${chapterId}`)
}

// 删除图书
export function deleteBook(id) {
  return api.delete(`/books/${id}`)
}

// 更新阅读进度
export function updateProgress(id, chapterIndex, charOffset) {
  return api.put(`/books/${id}/progress`, { chapterIndex, charOffset })
}

// 获取已揭示的人物及别名
export function getCharacters(bookId) {
  return api.get(`/books/${bookId}/characters`)
}

// 获取已揭示的人物关系
export function getRelations(bookId) {
  return api.get(`/books/${bookId}/relations`)
}

// 获取已揭示的事件时间线
export function getEvents(bookId) {
  return api.get(`/books/${bookId}/events`)
}

// 获取事件与人物关联
export function getEventCharacters(bookId) {
  return api.get(`/books/${bookId}/event-characters`)
}

// 发送问题并获取回答
export function askQuestion(bookId, conversationId, question) {
  return api.post(`/books/${bookId}/ask`, { conversation_id: conversationId, question })
}

// 获取对话列表
export function getConversations(bookId) {
  return api.get(`/books/${bookId}/conversations`)
}

// 创建新对话
export function createConversation(bookId, title) {
  return api.post(`/books/${bookId}/conversations`, { title })
}

// 获取对话消息历史
export function getMessages(conversationId) {
  return api.get(`/conversations/${conversationId}/messages`)
}

// 删除对话
export function deleteConversation(conversationId) {
  return api.delete(`/conversations/${conversationId}`)
}

export default api
