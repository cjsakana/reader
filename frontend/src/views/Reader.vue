<template>
  <div class="reader-page">
    <!-- 顶部栏 -->
    <el-header class="reader-header">
      <el-button text @click="$router.push('/')">
        <el-icon><ArrowLeft /></el-icon> 返回书架
      </el-button>
      <span class="book-name">{{ readerStore.bookDetail?.book?.title || '加载中...' }}</span>
      <span class="progress-info" v-if="readerStore.bookDetail">
        第{{ readerStore.bookDetail.book.chapterIndex + 1 }}/{{ readerStore.bookDetail.book.totalChapters }}章
      </span>
    </el-header>

    <el-container class="reader-body">
      <!-- 左侧侧边栏（标签页） -->
      <el-aside class="reader-sidebar" :style="{ width: sidebarCollapsed ? '0px' : '300px' }">
        <template v-if="!sidebarCollapsed">
          <el-tabs v-model="activeTab" class="sidebar-tabs">
            <!-- 目录 Tab -->
            <el-tab-pane label="目录" name="toc">
              <div class="chapter-list">
                <div
                  v-for="(ch, i) in readerStore.bookDetail?.chapters || []"
                  :key="ch.id"
                  class="chapter-item"
                  :class="{ active: i === readerStore.bookDetail?.book?.chapterIndex }"
                  @click="switchChapter(ch)"
                >
                  <span class="chapter-index">{{ ch.index + 1 }}</span>
                  <span class="chapter-title-aside">{{ ch.title }}</span>
                </div>
              </div>
            </el-tab-pane>

            <!-- 角色 Tab —— 仅为头部关系图 + 角色列表 -->
            <el-tab-pane label="角色" name="characters">
              <div class="tab-characters-wrapper">
                <RelationshipGraph
                  :characters="activeCharacters"
                  :relations="activeRelations"
                  @click-character="openCharDetail"
                />
                <el-button
                  size="small"
                  text
                  class="fullscreen-btn"
                  @click="showFullscreenGraph = true"
                >
                  <el-icon><FullScreen /></el-icon>
                </el-button>
                <div v-if="activeCharacters.length > 0" class="character-list-mini">
                  <el-card
                    v-for="char in activeCharacters"
                    :key="char.id"
                    class="character-card-mini"
                    shadow="hover"
                    @click="openCharDetail(char)"
                  >
                    <div class="c-mini-name">{{ char.name }}</div>
                    <div class="c-mini-meta">
                      <el-tag size="small" type="info">登场第{{ char.revealChapter + 1 }}章</el-tag>
                      <el-tag size="small" type="success" effect="plain">出现{{ char.frequency || 1 }}次</el-tag>
                      <el-tag
                        v-for="alias in (char.aliases || [])"
                        :key="alias"
                        size="small"
                        type="warning"
                        effect="plain"
                      >{{ alias }}</el-tag>
                      <el-tag
                        v-if="char.lastAppearChapter !== undefined && char.lastAppearChapter !== char.revealChapter"
                        size="small"
                        type="success"
                        effect="plain"
                      >至第{{ char.lastAppearChapter + 1 }}章</el-tag>
                      <el-tag
                        v-if="isStale(char)"
                        size="small"
                        type="danger"
                        effect="dark"
                      >已下线</el-tag>
                    </div>
                    <div v-if="char.description" class="c-mini-desc">{{ char.description }}</div>
                  </el-card>
                </div>
              </div>
            </el-tab-pane>

            <!-- 事件 Tab -->
            <el-tab-pane label="事件" name="events">
              <EventTimeline
                :events="knowledgeStore.events"
                :event-character-map="knowledgeStore.eventCharacterMap"
                @jump-to-chapter="jumpToChapter"
              />
            </el-tab-pane>

            <!-- 问答 Tab -->
            <el-tab-pane label="问答" name="chat">
              <ChatPanel :book-id="props.id" />
            </el-tab-pane>
          </el-tabs>
        </template>
      </el-aside>

      <div class="sidebar-toggle" @click="sidebarCollapsed = !sidebarCollapsed">
        <el-icon><List /></el-icon>
      </div>

      <!-- 中央阅读区 -->
      <el-main class="content-area" ref="contentAreaRef" @scroll="handleScroll">
        <div v-if="readerStore.loading" class="content-loading">
          <el-skeleton :rows="10" animated />
        </div>
        <div v-else class="chapter-content">
          <h2 class="chapter-title">{{ readerStore.currentChapter?.title || '' }}</h2>
          <div class="chapter-text" v-html="renderedContent"></div>

          <div class="chapter-nav">
            <el-button
              v-if="readerStore.currentChapter?.prevId"
              @click="switchChapterById(readerStore.currentChapter.prevId)"
            >
              <el-icon><ArrowLeft /></el-icon> 上一章
            </el-button>
            <el-button
              v-if="readerStore.currentChapter?.nextId"
              type="primary"
              @click="switchChapterById(readerStore.currentChapter.nextId)"
            >
              下一章 <el-icon><ArrowRight /></el-icon>
            </el-button>
          </div>
          <p v-if="!readerStore.currentChapter?.nextId" class="end-mark">— 全书完 —</p>
        </div>
      </el-main>

      <!-- 右侧关系图面板 -->
      <el-aside class="graph-panel" :style="{ width: graphCollapsed ? '0px' : '420px' }">
        <template v-if="!graphCollapsed">
          <div class="graph-header">
            <span>关系图谱</span>
            <div class="graph-header-actions">
              <el-button size="small" text @click="showFullscreenGraph = true">
                <el-icon><FullScreen /></el-icon>
              </el-button>
              <el-button text size="small" @click="graphCollapsed = true">
                <el-icon><Close /></el-icon>
              </el-button>
            </div>
          </div>
          <RelationshipGraph
            :characters="activeCharacters"
            :relations="activeRelations"
            @click-character="openCharDetail"
          />
        </template>
      </el-aside>

      <div class="graph-toggle" @click="graphCollapsed = !graphCollapsed">
        <el-icon><Share /></el-icon>
      </div>
    </el-container>

    <!-- 全屏关系图对话框 -->
    <el-dialog
      v-model="showFullscreenGraph"
      title="关系图谱"
      fullscreen
      :destroy-on-close="false"
      :append-to-body="true"
      @opened="onFullscreenOpened"
    >
      <RelationshipGraph
        ref="fullscreenGraphRef"
        v-if="showFullscreenGraph"
        style="height: calc(100vh - 160px); width: 100%"
        :characters="activeCharacters"
        :relations="activeRelations"
        @click-character="openCharDetail"
      />
    </el-dialog>

    <!-- 角色详情对话框 -->
    <el-dialog
      v-model="showCharDetail"
      :title="detailChar?.name || '角色详情'"
      width="520px"
    >
      <template v-if="detailChar">
        <div class="detail-section">
          <div class="detail-label">名称</div>
          <div class="detail-value">{{ detailChar.name }}</div>
        </div>
        <div class="detail-section" v-if="detailChar.aliases && detailChar.aliases.length">
          <div class="detail-label">别名</div>
          <div class="detail-value">
            <el-tag
              v-for="a in detailChar.aliases"
              :key="a"
              size="small"
              style="margin-right:6px;margin-bottom:4px"
            >{{ a }}</el-tag>
          </div>
        </div>
        <div class="detail-section">
          <div class="detail-label">首次登场</div>
          <div class="detail-value">第{{ detailChar.revealChapter + 1 }}章</div>
        </div>
        <div class="detail-section" v-if="detailChar.lastAppearChapter !== undefined">
          <div class="detail-label">最后出现</div>
          <div class="detail-value">第{{ detailChar.lastAppearChapter + 1 }}章</div>
        </div>
        <div class="detail-section" v-if="detailChar.description">
          <div class="detail-label">简介</div>
          <div class="detail-value detail-desc">{{ detailChar.description }}</div>
        </div>

        <!-- 关联关系 -->
        <div class="detail-section" v-if="charRelations.length">
          <div class="detail-label">关联关系</div>
          <div class="detail-value">
            <div v-for="r in charRelations" :key="r.id" class="rel-item">
              <div class="rel-line1">
                <span class="rel-target-name">{{ getOtherCharName(r) }}</span>
                <el-tag size="small" :type="relTagType(r.relationType)">{{ r.relationType }}</el-tag>
              </div>
              <div v-if="r.description" class="rel-desc">{{ r.description }}</div>
            </div>
          </div>
        </div>

        <!-- 参与事件 -->
        <div class="detail-section" v-if="charEvents.length">
          <div class="detail-label">参与事件</div>
          <div class="detail-value">
            <div v-for="evt in charEvents" :key="evt.id" class="event-item"
                 @click="jumpToChapter(evt.chapterNumber); showCharDetail = false">
              <div class="evt-header">
                <el-tag size="small" type="primary">第{{ evt.chapterNumber + 1 }}章</el-tag>
                <el-rate :model-value="evt.importance" disabled size="small" :max="5" style="display:inline-flex;margin-left:6px" />
              </div>
              <div class="evt-summary">{{ evt.summary }}</div>
            </div>
          </div>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { ArrowLeft, ArrowRight, List, Close, Share, FullScreen } from '@element-plus/icons-vue'
import EventTimeline from '../components/EventTimeline.vue'
import ChatPanel from '../components/ChatPanel.vue'
import RelationshipGraph from '../components/RelationshipGraph.vue'
import { useReaderStore } from '../stores/reader.js'
import { useKnowledgeStore } from '../stores/knowledge.js'

const STALE_THRESHOLD = 50  // 超过50章未再出现则视为过时

const props = defineProps({
  id: { type: [String, Number], required: true },
})

const readerStore = useReaderStore()
const knowledgeStore = useKnowledgeStore()

const activeTab = ref('toc')
const sidebarCollapsed = ref(false)
const graphCollapsed = ref(true)
const contentAreaRef = ref(null)

// 全屏 & 详情
const showFullscreenGraph = ref(false)
const fullscreenGraphRef = ref(null)
const showCharDetail = ref(false)
const detailChar = ref(null)

function onFullscreenOpened() {
  // 全屏对话框打开后强制图组件重绘
  setTimeout(() => {
    if (fullscreenGraphRef.value?.restartSimulation) {
      fullscreenGraphRef.value.restartSimulation()
    }
  }, 300)
}

let saveTimer = null

// 当前进度
const currentChapter = computed(() => readerStore.bookDetail?.book?.chapterIndex || 0)

// 过滤过时角色：lastAppearChapter 与当前进度差距 > STALE_THRESHOLD
const activeCharacters = computed(() => {
  const cp = currentChapter.value
  return (knowledgeStore.characters || []).filter(c => {
    const last = c.lastAppearChapter ?? c.revealChapter
    return (cp - last) <= STALE_THRESHOLD
  })
})

const activeCharIds = computed(() => new Set(activeCharacters.value.map(c => c.id)))

const activeRelations = computed(() => {
  const ids = activeCharIds.value
  return (knowledgeStore.relations || []).filter(r => {
    const c1 = r.char1?.id
    const c2 = r.char2?.id
    return ids.has(c1) && ids.has(c2)
  })
})

function isStale(c) {
  const last = c.lastAppearChapter ?? c.revealChapter
  return (currentChapter.value - last) > STALE_THRESHOLD
}

const renderedContent = computed(() => {
  if (!readerStore.currentChapter?.content) return ''
  return readerStore.currentChapter.content
    .split('\n')
    .map(line => `<p>${line || '&nbsp;'}</p>`)
    .join('')
})

onMounted(async () => {
  await readerStore.loadBook(props.id)
  const chIdx = readerStore.bookDetail?.book?.chapterIndex || 0
  const ch = readerStore.bookDetail?.chapters?.[chIdx]
  if (ch) {
    await readerStore.loadChapter(ch.id)
    await nextTick()
    if (contentAreaRef.value) {
      contentAreaRef.value.scrollTop = readerStore.bookDetail.book.charOffset || 0
    }
  }
  await knowledgeStore.fetchAll(props.id)
})

watch(() => props.id, async () => {
  await readerStore.loadBook(props.id)
  const chIdx = readerStore.bookDetail?.book?.chapterIndex || 0
  const ch = readerStore.bookDetail?.chapters?.[chIdx]
  if (ch) await readerStore.loadChapter(ch.id)
  await knowledgeStore.fetchAll(props.id)
  sidebarCollapsed.value = false
  graphCollapsed.value = true
})

async function switchChapter(ch) {
  if (!ch) return
  await readerStore.switchChapter(ch.id)
  if (contentAreaRef.value) contentAreaRef.value.scrollTop = 0
  knowledgeStore.pollForUpdates(props.id)
}

async function switchChapterById(chapterId) {
  await readerStore.switchChapter(chapterId)
  if (contentAreaRef.value) contentAreaRef.value.scrollTop = 0
  knowledgeStore.pollForUpdates(props.id)
}

function handleScroll() {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    readerStore.saveCharOffset(contentAreaRef.value?.scrollTop || 0)
  }, 1000)
}

function jumpToChapter(chapterIndex) {
  const ch = readerStore.bookDetail?.chapters?.find(c => c.index === chapterIndex)
  if (ch) {
    switchChapter(ch)
    activeTab.value = 'toc'
  }
}

// --- 角色详情 ---

const charRelations = computed(() => {
  if (!detailChar.value) return []
  const cid = detailChar.value.id
  return (knowledgeStore.relations || []).filter(r => {
    const c1 = r.char1?.id
    const c2 = r.char2?.id
    return (c1 === cid || c2 === cid)
  })
})

const charEvents = computed(() => {
  if (!detailChar.value) return []
  const cid = detailChar.value.id
  const ecMap = knowledgeStore.eventCharacterMap
  const evts = (knowledgeStore.events || []).filter(e => {
    const ecs = ecMap[e.id] || []
    return ecs.some(ec => ec.characterId === cid || ec.character?.id === cid)
  })
  return evts.sort((a, b) => a.chapterNumber - b.chapterNumber)
})

function openCharDetail(char) {
  // char 来自图组件，只含 {id, name, chapter}。从 knowledgeStore 中查找完整角色数据
  const full = (knowledgeStore.characters || []).find(c => c.id === char.id)
  detailChar.value = full || char
  showCharDetail.value = true
}

function getOtherCharName(rel) {
  if (!detailChar.value) return ''
  if (rel.char1?.id === detailChar.value.id) return rel.char2?.name || '未知'
  return rel.char1?.name || '未知'
}

function relTagType(type) {
  const m = { '朋友': 'success', '情侣': 'danger', '家人': 'warning', '敌对': 'danger', '盟友': '' }
  return m[type] || 'info'
}
</script>

<style scoped>
.reader-page { height: 100vh; display: flex; flex-direction: column; background: #fff; }
.reader-header { display: flex; align-items: center; gap: 20px; height: 56px; padding: 0 20px; background: #fff; border-bottom: 1px solid #ebeef5; box-shadow: 0 1px 4px rgba(0,0,0,0.05); flex-shrink: 0; }
.book-name { font-size: 16px; font-weight: 500; color: #303133; flex: 1; }
.progress-info { font-size: 13px; color: #909399; }
.reader-body { flex: 1; overflow: hidden; position: relative; }
.reader-sidebar { border-right: 1px solid #ebeef5; background: #fafafa; overflow: hidden; transition: width 0.3s; flex-shrink: 0; display: flex; flex-direction: column; }
.sidebar-tabs { display: flex; flex-direction: column; height: 100%; }
.sidebar-tabs :deep(.el-tabs__content) { flex: 1; overflow-y: auto; padding: 0 12px; }
.sidebar-tabs :deep(.el-tabs__header) { margin-bottom: 0; padding: 0 12px; }

/* 角色 Tab 布局 */
.tab-characters-wrapper { display: flex; flex-direction: column; height: 100%; position: relative; }
.tab-characters-wrapper .graph-container { flex-shrink: 0; height: 220px; }
.fullscreen-btn { position: absolute; top: 4px; right: 4px; z-index: 2; }

.chapter-list { overflow-y: auto; padding: 4px 0; }
.chapter-item { display: flex; gap: 10px; padding: 8px 12px; cursor: pointer; font-size: 14px; color: #606266; transition: background 0.15s; border-radius: 6px; }
.chapter-item:hover { background: #ecf5ff; }
.chapter-item.active { background: #ecf5ff; color: #409eff; font-weight: 500; }
.chapter-index { color: #c0c4cc; min-width: 24px; text-align: right; }
.chapter-title-aside { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sidebar-toggle { position: absolute; left: 0; top: 50%; transform: translateY(-50%); width: 28px; height: 56px; background: #fff; border: 1px solid #ebeef5; border-left: none; border-radius: 0 6px 6px 0; display: flex; align-items: center; justify-content: center; cursor: pointer; z-index: 10; color: #909399; transition: color 0.2s; }
.sidebar-toggle:hover { color: #409eff; }
.content-area { padding: 40px 80px; overflow-y: auto; flex: 1; }
@media (max-width:768px) { .content-area { padding: 24px 20px; } }
.content-loading { max-width: 700px; margin: 0 auto; }
.chapter-content { max-width: 700px; margin: 0 auto; }
.chapter-title { font-size: 28px; font-weight: 600; color: #303133; text-align: center; margin-bottom: 32px; padding-bottom: 20px; border-bottom: 1px solid #ebeef5; }
.chapter-text { font-size: 17px; line-height: 2; color: #303133; }
.chapter-text :deep(p) { margin-bottom: 0; text-indent: 2em; }
.chapter-nav { display: flex; justify-content: space-between; margin-top: 48px; padding-top: 24px; border-top: 1px solid #ebeef5; }
.end-mark { text-align: center; color: #909399; margin-top: 48px; font-size: 14px; }

.graph-panel { border-left: 1px solid #ebeef5; background: #fafafa; overflow: hidden; transition: width 0.3s; flex-shrink: 0; display: flex; flex-direction: column; }
.graph-header { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; font-weight: 500; font-size: 14px; border-bottom: 1px solid #ebeef5; }
.graph-header-actions { display: flex; gap: 4px; align-items: center; }
.graph-toggle { position: absolute; right: 0; top: 50%; transform: translateY(-50%); width: 28px; height: 56px; background: #fff; border: 1px solid #ebeef5; border-right: none; border-radius: 6px 0 0 6px; display: flex; align-items: center; justify-content: center; cursor: pointer; z-index: 10; color: #909399; transition: color 0.2s; }
.graph-toggle:hover { color: #409eff; }

/* 角色卡片列表 */
.character-list-mini { overflow-y: auto; padding: 8px 0 0 0; flex: 1; min-height: 0; }
.character-card-mini { cursor: pointer; border-radius: 6px; margin-bottom: 6px; }
.character-card-mini :deep(.el-card__body) { padding: 8px 12px; }
.c-mini-name { font-weight: 600; font-size: 13px; color: #303133; }
.c-mini-meta { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 4px; }
.c-mini-desc { font-size: 11px; color: #909399; margin-top: 4px; line-height: 1.5; max-height: 44px; overflow: hidden; text-overflow: ellipsis; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; white-space: pre-line; }

/* 角色详情 */
.detail-section { margin-bottom: 14px; }
.detail-label { font-size: 12px; color: #909399; margin-bottom: 4px; }
.detail-value { font-size: 14px; color: #303133; line-height: 1.5; }
.detail-desc { white-space: pre-line; }
.rel-item { margin-bottom: 10px; padding: 8px 10px; background: #f8f9fc; border-radius: 6px; }
.rel-line1 { display: flex; align-items: center; gap: 8px; }
.rel-target-name { font-weight: 600; color: #303133; }
.rel-desc { font-size: 12px; color: #909399; margin-top: 4px; line-height: 1.5; }
.event-item { padding: 8px 0; border-bottom: 1px solid #f0f0f0; cursor: pointer; }
.event-item:last-child { border-bottom: none; }
.evt-header { display: flex; align-items: center; margin-bottom: 4px; }
.evt-summary { font-size: 13px; color: #303133; line-height: 1.5; }
</style>
