<template>
  <el-dialog
    v-model="visible"
    title="导入图书"
    width="460px"
    :close-on-click-modal="false"
    @close="resetForm"
  >
    <el-upload
      ref="uploadRef"
      class="upload-area"
      drag
      :auto-upload="false"
      :limit="1"
      accept=".txt"
      :on-change="handleFileChange"
      :on-remove="handleRemove"
    >
      <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
      <div class="el-upload__text">
        将TXT文件拖到此处，或<em>点击上传</em>
      </div>
      <template #tip>
        <div class="el-upload__tip">仅支持 .txt 格式的文本文件</div>
      </template>
    </el-upload>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="uploading" :disabled="!file" @click="submitUpload">
        开始导入
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'
import { uploadBook } from '../api/index.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'success'])

const visible = ref(false)
const uploading = ref(false)
const file = ref(null)

watch(() => props.modelValue, (val) => {
  visible.value = val
})
watch(visible, (val) => {
  emit('update:modelValue', val)
})

function handleFileChange(uploadFile) {
  file.value = uploadFile.raw
}

function handleRemove() {
  file.value = null
}

function resetForm() {
  file.value = null
}

async function submitUpload() {
  if (!file.value) return
  uploading.value = true
  try {
    await uploadBook(file.value)
    ElMessage.success('导入成功！')
    visible.value = false
    emit('success')
  } catch (err) {
    const msg = err.message || '导入失败'
    ElMessage.error(msg)
  } finally {
    uploading.value = false
  }
}
</script>

<style scoped>
.upload-area {
  width: 100%;
}
</style>
