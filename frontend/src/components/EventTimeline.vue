<template>
  <div class="timeline-panel">
    <div v-if="!events || events.length === 0" class="empty-state">
      <el-empty description="阅读更多章节以解锁事件时间线" />
    </div>
    <el-timeline v-else>
      <el-timeline-item
        v-for="evt in sortedEvents"
        :key="evt.id"
        :timestamp="'第 ' + (evt.chapterNumber + 1) + ' 章'"
        placement="top"
        :color="importanceColor(evt.importance)"
      >
        <el-card shadow="hover" class="event-card" @click="$emit('jumpToChapter', evt.chapterNumber)">
          <div class="event-summary">{{ evt.summary }}</div>
          <div class="event-meta">
            <el-rate
              :model-value="evt.importance"
              disabled
              show-score
              size="small"
              :max="5"
            />
            <el-tag
              v-for="char in getEventChars(evt.id)"
              :key="char.character?.id || char.characterId"
              size="small"
              effect="plain"
            >
              {{ char.character?.name || '未知' }}
            </el-tag>
          </div>
        </el-card>
      </el-timeline-item>
    </el-timeline>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  events: { type: Array, default: () => [] },
  eventCharacterMap: { type: Object, default: () => ({}) },
})

defineEmits(['jumpToChapter'])

const sortedEvents = computed(() => {
  return [...props.events].sort((a, b) => a.chapterNumber - b.chapterNumber)
})

function getEventChars(eventId) {
  return props.eventCharacterMap[eventId] || []
}

function importanceColor(imp) {
  const colors = { 1: '#909399', 2: '#b3d8ff', 3: '#67c23a', 4: '#e6a23c', 5: '#f56c6c' }
  return colors[imp] || '#909399'
}
</script>

<style scoped>
.timeline-panel {
  padding: 0;
}
.empty-state {
  padding: 20px 0;
}
.event-card {
  cursor: pointer;
  border-radius: 8px;
}
.event-summary {
  font-size: 13px;
  color: #303133;
  line-height: 1.5;
  margin-bottom: 6px;
}
.event-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
