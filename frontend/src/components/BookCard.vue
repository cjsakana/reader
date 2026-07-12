<template>
  <el-card class="book-card" shadow="hover" @click="$emit('click')">
    <div class="book-info">
      <h3 class="book-title">{{ book.title }}</h3>
      <p class="book-meta">
        <span>{{ book.author }}</span>
        <span class="divider">|</span>
        <span>{{ formatSize(book.fileSize) }}</span>
        <span class="divider">|</span>
        <span>{{ book.totalChapters }}章</span>
      </p>
      <div class="book-progress" v-if="book.totalChapters > 0">
        <el-progress
          :percentage="progressPercent"
          :stroke-width="6"
          :show-text="false"
        />
        <p class="progress-text">
          第{{ book.chapterIndex + 1 }}章 · {{ Math.round(progressPercent) }}%
        </p>
      </div>
    </div>
    <div class="book-actions" @click.stop>
      <el-button type="danger" size="small" @click="handleDelete" :icon="Delete" text>
        删除
      </el-button>
    </div>
  </el-card>
</template>

<script setup>
import { computed } from 'vue'
import { Delete } from '@element-plus/icons-vue'

const props = defineProps({
  book: { type: Object, required: true },
})

const emit = defineEmits(['click', 'delete'])

const progressPercent = computed(() => {
  if (!props.book.totalChapters) return 0
  return Math.round((props.book.chapterIndex / Math.max(props.book.totalChapters - 1 || 1, 1)) * 100)
})

function formatSize(bytes) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return size.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

function handleDelete() {
  emit('delete', props.book.id)
}
</script>

<style scoped>
.book-card {
  cursor: pointer;
  transition: transform 0.2s;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}
.book-card:hover {
  transform: translateY(-2px);
}
.book-info {
  flex: 1;
}
.book-title {
  font-size: 18px;
  margin-bottom: 8px;
  color: #303133;
}
.book-meta {
  font-size: 13px;
  color: #909399;
  margin-bottom: 12px;
}
.divider {
  margin: 0 8px;
}
.book-progress {
  max-width: 200px;
}
.progress-text {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
.book-actions {
  margin-left: 16px;
}
</style>
