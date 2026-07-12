<template>
  <div class="chat-panel">
    <!-- 对话选择器 -->
    <div class="chat-header" v-if="chatStore.conversations.length > 0">
      <el-select
        v-model="selectedConvId"
        placeholder="选择对话"
        size="small"
        clearable
        style="flex: 1"
        @change="handleConvChange"
      >
        <el-option
          v-for="conv in chatStore.conversations"
          :key="conv.id"
          :label="conv.title"
          :value="conv.id"
        />
      </el-select>
      <el-button
        v-if="selectedConvId"
        size="small"
        type="danger"
        text
        @click="handleDeleteConv"
      >
        <el-icon><Delete /></el-icon>
      </el-button>
    </div>

    <!-- 消息列表 -->
    <div class="chat-messages" ref="msgContainer">
      <div v-if="localMessages.length === 0 && !sending" class="empty-state">
        <el-empty description="向AI助手提问已读内容" :image-size="60" />
      </div>
      <div
        v-for="(msg, idx) in localMessages"
        :key="msg._key || msg.id || idx"
        class="msg-item"
        :class="{ 'msg-user': msg.role === 'user', 'msg-assistant': msg.role === 'assistant' }"
      >
        <div class="msg-role">{{ msg.role === 'user' ? '我' : 'AI助手' }}</div>
        <div
          class="msg-content"
          :class="{ 'markdown-body': msg.role === 'assistant' }"
          v-html="renderContent(msg)"
        ></div>
      </div>
      <div v-if="sending" class="msg-item msg-assistant">
        <div class="msg-role">AI助手</div>
        <div class="msg-content thinking-text">思考中……</div>
      </div>
    </div>

    <!-- 输入区域（固定在底部） -->
    <div class="chat-input">
      <el-input
        v-model="inputText"
        placeholder="只能询问已读内容"
        :disabled="sending"
        @keyup.enter="handleSend"
      />
      <el-button
        type="primary"
        :disabled="!inputText.trim() || sending"
        size="small"
        @click="handleSend"
      >
        发送
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick, onMounted } from 'vue'
import { Delete } from '@element-plus/icons-vue'
import { useChatStore } from '../stores/chat.js'

const props = defineProps({
  bookId: { type: [Number, String], required: true },
})

const chatStore = useChatStore()
const selectedConvId = ref(null)
const inputText = ref('')
const msgContainer = ref(null)
const sending = ref(false)
const localMessages = ref([])

// 从 store 同步消息
watch(() => chatStore.messages, (msgs) => {
  localMessages.value = msgs || []
  nextTick(scrollToBottom)
}, { deep: true })

// 监听 store 中活跃对话的变化
watch(() => chatStore.activeConversationId, (id) => {
  selectedConvId.value = id
})

// 监听 conversations 列表更新
watch(() => chatStore.conversations, () => {
  if (!selectedConvId.value && chatStore.activeConversationId) {
    selectedConvId.value = chatStore.activeConversationId
  }
}, { deep: true })

onMounted(async () => {
  await chatStore.loadConversations(props.bookId)
})

// Markdown → HTML
function renderContent(msg) {
  if (msg.role === 'user') return escapeHtml(msg.content)
  return markdownToHtml(msg.content)
}

function escapeHtml(text) {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

function markdownToHtml(text) {
  let html = escapeHtml(text)
  // code blocks
  html = html.replace(/```(\w*)\n([\s\S]*?)```/g, '<pre><code>$2</code></pre>')
  // inline code
  html = html.replace(/`([^`]+)`/g, '<code>$1</code>')
  // headers
  html = html.replace(/^### (.+)$/gm, '<h4>$1</h4>')
  html = html.replace(/^## (.+)$/gm, '<h3>$1</h3>')
  html = html.replace(/^# (.+)$/gm, '<h2>$1</h2>')
  // bold / italic
  html = html.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  html = html.replace(/\*([^*]+)\*/g, '<em>$1</em>')
  // lists
  html = html.replace(/^- (.+)$/gm, '<li>$1</li>')
  html = html.replace(/^\d+\.\s+(.+)$/gm, '<li>$1</li>')
  // newlines
  html = html.replace(/\n/g, '<br>')
  return html
}

function scrollToBottom() {
  if (msgContainer.value) {
    msgContainer.value.scrollTop = msgContainer.value.scrollHeight
  }
}

async function handleConvChange(val) {
  if (val) {
    await chatStore.switchConversation(val)
  } else {
    chatStore.activeConversationId = null
    localMessages.value = []
  }
}

async function handleDeleteConv() {
  if (!selectedConvId.value) return
  await chatStore.removeConversation(selectedConvId.value)
  selectedConvId.value = null
  localMessages.value = []
}

async function handleSend() {
  const text = inputText.value.trim()
  if (!text || sending.value) return

  inputText.value = ''

  // 1. 用户消息立刻显示
  localMessages.value.push({
    _key: 'user-' + Date.now(),
    role: 'user',
    content: text,
  })
  nextTick(scrollToBottom)

  // 2. 思考中
  sending.value = true
  nextTick(scrollToBottom)

  try {
    const result = await chatStore.sendMessage(props.bookId, text)
    // 刷新对话列表以显示新标题
    if (result && chatStore.activeConversationId) {
      await chatStore.loadConversations(props.bookId)
    }
  } catch (e) {
    console.warn('发送消息失败:', e)
  } finally {
    sending.value = false
  }
}
</script>

<style scoped>
.chat-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.chat-header {
  display: flex;
  gap: 6px;
  margin-bottom: 8px;
  align-items: center;
  flex-shrink: 0;
}
.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
  min-height: 0;
}
.chat-input {
  display: flex;
  gap: 6px;
  padding-top: 8px;
  border-top: 1px solid #ebeef5;
  flex-shrink: 0;
}
.msg-item {
  margin-bottom: 10px;
}
.msg-role {
  font-size: 11px;
  color: #909399;
  margin-bottom: 2px;
}
.msg-user .msg-role {
  text-align: right;
}
.msg-content {
  font-size: 13px;
  line-height: 1.6;
  padding: 8px 12px;
  border-radius: 8px;
  max-width: 85%;
  display: inline-block;
  word-break: break-word;
}
.msg-user .msg-content {
  background: #ecf5ff;
  color: #303133;
  float: right;
}
.msg-assistant .msg-content {
  background: #f5f7fa;
  color: #303133;
}
.thinking-text {
  font-size: 12px;
  color: #909399;
  font-style: italic;
}
/* Markdown */
.markdown-body :deep(h2) { font-size: 15px; margin: 8px 0 4px; }
.markdown-body :deep(h3) { font-size: 14px; margin: 6px 0 3px; }
.markdown-body :deep(h4) { font-size: 13px; margin: 4px 0 2px; }
.markdown-body :deep(strong) { font-weight: 600; }
.markdown-body :deep(em) { font-style: italic; }
.markdown-body :deep(code) { background: #f0f0f0; padding: 1px 4px; border-radius: 3px; font-size: 12px; }
.markdown-body :deep(pre) { background: #f5f5f5; padding: 8px; border-radius: 4px; overflow-x: auto; font-size: 12px; margin: 4px 0; }
.markdown-body :deep(pre code) { background: none; padding: 0; }
.markdown-body :deep(li) { margin-left: 16px; }
.empty-state { padding: 20px 0; }
</style>
