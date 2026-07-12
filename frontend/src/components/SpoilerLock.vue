<template>
  <div class="spoiler-lock">
    <template v-if="count">
      <el-icon :size="16" color="#c0c4cc"><Lock /></el-icon>
      <span>{{ count }} {{ label }}</span>
    </template>
    <template v-else>
      <el-icon :size="16" color="#c0c4cc"><Lock /></el-icon>
      <span>{{ label }}</span>
    </template>
  </div>
</template>

<script setup>
import { Lock } from '@element-plus/icons-vue'

defineProps({
  count: { type: Number, default: 0 },
  label: { type: String, default: '阅读到对应章节后解锁' },
})
</script>

<style scoped>
.spoiler-lock {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 12px;
  color: #c0c4cc;
  font-size: 12px;
}
</style>
