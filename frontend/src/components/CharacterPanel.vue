<template>
  <div class="character-panel">
    <div v-if="!characters || characters.length === 0" class="empty-state">
      <el-empty description="阅读更多章节以解锁角色信息" />
    </div>
    <div v-else class="character-list">
      <el-card
        v-for="char in visibleCharacters"
        :key="char.id"
        class="character-card"
        shadow="hover"
      >
        <div class="character-header">
          <span class="character-name">{{ char.name }}</span>
          <el-tag size="small" type="info">第 {{ char.revealChapter +1  }} 章</el-tag>
        </div>
        <div v-if="char.aliases && char.aliases.length > 0" class="character-aliases">
          <el-tag
            v-for="alias in char.aliases"
            :key="alias"
            size="small"
            type="warning"
            effect="plain"
          >
            {{ alias }}
          </el-tag>
        </div>
        <div v-if="char.description" class="character-desc">
          {{ char.description }}
        </div>
      </el-card>
      <!-- 剧透锁定占位 -->
      <SpoilerLock
        v-if="lockedCount > 0"
        :count="lockedCount"
        label="个角色将在后续章节解锁"
      />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import SpoilerLock from './SpoilerLock.vue'

const props = defineProps({
  characters: { type: Array, default: () => [] },
  excludeIds: { type: Array, default: () => [] },
})

const visibleCharacters = computed(() => {
  return props.characters.filter(c => !props.excludeIds.includes(c.id))
})

const lockedCount = computed(() => 0)
</script>

<style scoped>
.character-panel {
  padding: 0;
}
.character-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.character-card {
  border-radius: 8px;
}
.character-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}
.character-name {
  font-weight: 600;
  font-size: 14px;
  color: #303133;
}
.character-aliases {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 6px;
}
.character-desc {
  font-size: 12px;
  color: #606266;
  line-height: 1.5;
}
.empty-state {
  padding: 20px 0;
}
</style>
