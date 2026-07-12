<template>
  <div class="book-list-page">
    <el-container>
      <el-header class="page-header">
        <h1>📚 我的书架</h1>
        <el-button type="primary" @click="showUpload = true">
          <el-icon><Plus /></el-icon> 导入图书
        </el-button>
      </el-header>

      <el-main>
        <div v-if="loading" class="loading-area">
          <el-skeleton :rows="3" animated />
        </div>

        <div v-else-if="books.length === 0" class="empty-area">
          <el-empty description="还没有图书，快来导入第一本吧！">
            <el-button type="primary" @click="showUpload = true">导入图书</el-button>
          </el-empty>
        </div>

        <div v-else class="book-grid">
          <BookCard
            v-for="book in books"
            :key="book.id"
            :book="book"
            @click="$router.push(`/read/${book.id}`)"
            @delete="handleDelete"
          />
        </div>
      </el-main>
    </el-container>

    <UploadDialog v-model="showUpload" @success="fetchBooks" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { getBooks, deleteBook } from '../api/index.js'
import BookCard from '../components/BookCard.vue'
import UploadDialog from '../components/UploadDialog.vue'

const books = ref([])
const loading = ref(true)
const showUpload = ref(false)

onMounted(() => {
  fetchBooks()
})

async function fetchBooks() {
  loading.value = true
  try {
    const res = await getBooks()
    books.value = res.data || []
  } catch (err) {
    ElMessage.error('获取图书列表失败')
  } finally {
    loading.value = false
  }
}

async function handleDelete(id) {
  try {
    await ElMessageBox.confirm('确定要删除这本书吗？所有章节数据也会被删除。', '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteBook(id)
    ElMessage.success('删除成功')
    fetchBooks()
  } catch (err) {
    if (err !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}
</script>

<style scoped>
.book-list-page {
  min-height: 100vh;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 40px;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.page-header h1 {
  font-size: 24px;
  color: #303133;
}

.loading-area {
  padding: 40px;
  max-width: 600px;
  margin: 0 auto;
}

.empty-area {
  display: flex;
  justify-content: center;
  padding-top: 120px;
}

.book-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 20px;
  padding: 20px 40px;
  max-width: 1200px;
  margin: 0 auto;
}
</style>
