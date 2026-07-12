<template>
  <div class="graph-container" ref="container">
    <div class="graph-controls">
      <span class="control-label">章节范围：</span>
      <el-slider
        v-model="currentFilterChapter"
        :min="minChapter"
        :max="maxChapter"
        :disabled="!nodes.length"
        style="flex: 1"
        @change="restartSimulation"
      />
      <span class="control-chapter-label">第{{ currentFilterChapter + 1 }}章</span>
    </div>
    <div class="graph-legend">
      <span v-for="item in legendItems" :key="item.type" class="legend-item">
        <span class="legend-color" :style="{ background: item.color }"></span>
        {{ item.label }}
      </span>
    </div>
    <div v-if="!nodes.length" class="empty-state">
      <el-empty description="暂无关系数据" />
    </div>
    <svg ref="svg" class="graph-svg" v-show="nodes.length"></svg>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted, nextTick, defineExpose } from 'vue'
import {
  forceSimulation,
  forceLink,
  forceManyBody,
  forceCenter,
  forceCollide,
} from 'd3-force'
import { zoom } from 'd3-zoom'
import { select } from 'd3-selection'
import { drag } from 'd3-drag'

const props = defineProps({
  characters: { type: Array, default: () => [] },
  relations: { type: Array, default: () => [] },
})

const emit = defineEmits(['clickCharacter'])

const container = ref(null)
const svg = ref(null)
const chapterRange = ref([0, 0])
const minChapter = ref(0)
const maxChapter = ref(0)
const currentFilterChapter = ref(0)
const nodes = ref([])
const links = ref([])

const relationColors = {
  '朋友': '#67c23a',
  '情侣': '#f56c6c',
  '家人': '#e6a23c',
  '敌对': '#f56c6c',
  '师徒': '#409eff',
  '同事': '#909399',
  '主仆': '#b37feb',
  '盟友': '#409eff',
  '其他': '#909399',
}
const legendItems = [
  { type: '朋友', label: '朋友', color: '#67c23a' },
  { type: '情侣', label: '情侣', color: '#f56c6c' },
  { type: '家人', label: '家人', color: '#e6a23c' },
  { type: '敌对', label: '敌对', color: '#f56c6c' },
  { type: '盟友', label: '盟友', color: '#409eff' },
]

// 颜色调色板（更和谐的配色）
const nodePalette = [
  '#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de',
  '#3ba272', '#fc8452', '#9a60b4', '#ea7ccc', '#48b8d0',
  '#f56c6c', '#67c23a', '#e6a23c', '#409eff', '#b37feb',
]

let simulation = null
let resizeObserver = null
let zoomBehavior = null

onMounted(() => {
  resizeObserver = new ResizeObserver(() => updateSimulation())
  if (container.value) resizeObserver.observe(container.value)
  // 立即初始化渲染（全屏对话框重新挂载时 props 不会变化，watch 不会触发）
  updateRange()
  nextTick(() => restartSimulation())
})

onUnmounted(() => {
  if (resizeObserver) resizeObserver.disconnect()
  if (simulation) simulation.stop()
  if (zoomBehavior) zoomBehavior.on('zoom', null)
})

watch([() => props.characters, () => props.relations], () => {
  updateRange()
  restartSimulation()
}, { deep: true })

// 初始时 currentFilterChapter 设为当前最大章节
watch(() => props.characters, () => {
  if (props.characters.length) {
    const chapters = props.characters.map(c => c.revealChapter)
    if (chapters.length) currentFilterChapter.value = Math.max(...chapters)
  }
})

function updateRange() {
  if (!props.characters.length) {
    minChapter.value = 0
    maxChapter.value = 0
    return
  }
  const chapters = props.characters.map(c => c.revealChapter)
  minChapter.value = Math.min(...chapters)
  maxChapter.value = Math.max(...chapters)
  if (currentFilterChapter.value < minChapter.value || currentFilterChapter.value > maxChapter.value) {
    currentFilterChapter.value = maxChapter.value
  }
}

function restartSimulation() {
  if (!props.characters.length) {
    nodes.value = []
    links.value = []
    return
  }

  const upToChapter = currentFilterChapter.value
  // 仅显示 reveal_chapter <= upToChapter 的角色（过滤别名：角色名本身已在列表中）
  const filteredChars = props.characters.filter(
    c => c.revealChapter <= upToChapter
  )
  const charIds = new Set(filteredChars.map(c => c.id))

  nodes.value = filteredChars.map(c => {
    // 显示角色名本身，不显示别名（别名在角色详情弹窗中展示）
    return {
      id: c.id,
      name: c.name,
      chapter: c.revealChapter,
    }
  })

  links.value = []
  
  for (const r of props.relations) {
    const c1Id = r.char1?.id
    const c2Id = r.char2?.id
    if (!c1Id || !c2Id) continue
    if (!charIds.has(c1Id) || !charIds.has(c2Id)) continue

    const exists = links.value.some(
      l => (l.source === c1Id && l.target === c2Id) ||
           (l.source === c2Id && l.target === c1Id)
    )
    if (!exists) {
      links.value.push({
        source: c1Id,
        target: c2Id,
        type: r.relationType || '其他',
      })
    }
  }

  nextTick(() => updateSimulation())
}

function updateSimulation() {
  if (!svg.value) return

  if (simulation) simulation.stop()

  const el = svg.value
  const width = el.clientWidth || 300
  const height = el.clientHeight || 200

  select(el).selectAll('*').remove()

  if (!nodes.value.length) return

  select(el).selectAll('*').remove()

  const svgSel = select(el)

  // 缩放行为
  zoomBehavior = zoom()
    .scaleExtent([0.3, 4])
    .on('zoom', (event) => {
      g.attr('transform', event.transform)
    })
  svgSel.call(zoomBehavior)

  // 箭头标记
  svgSel.append('defs').selectAll('marker')
    .data(['end'])
    .join('marker')
    .attr('id', 'arrow')
    .attr('viewBox', '0 -5 10 10')
    .attr('refX', 20)
    .attr('refY', 0)
    .attr('markerWidth', 6)
    .attr('markerHeight', 6)
    .attr('orient', 'auto')
    .append('path')
    .attr('d', 'M0,-5L10,0L0,5')
    .attr('fill', '#999')

  const g = svgSel.append('g')

  // 节点颜色
  const colorMap = {}
  nodes.value.forEach((n, i) => {
    colorMap[n.id] = nodePalette[i % nodePalette.length]
  })

  const simNodes = nodes.value.map(n => ({ ...n }))
  const simLinks = links.value.map(l => ({
    source: simNodes.findIndex(n => n.id === (typeof l.source === 'object' ? l.source.id : l.source)),
    target: simNodes.findIndex(n => n.id === (typeof l.target === 'object' ? l.target.id : l.target)),
    type: l.type,
  })).filter(l => l.source >= 0 && l.target >= 0)

  simulation = forceSimulation(simNodes)
    .force('link', forceLink(simLinks).distance(120))
    .force('charge', forceManyBody().strength(-400))
    .force('center', forceCenter(width / 2, height / 2))
    .force('collide', forceCollide(40))
    .on('tick', () => {
      // 连线
      const linkG = g.selectAll('g.link-group')
        .data(simLinks)
        .join('g')
        .attr('class', 'link-group')

      linkG.selectAll('line')
        .data(d => [d])
        .join('line')
        .attr('stroke', d => relationColors[d.type] || '#909399')
        .attr('stroke-width', 2)
        .attr('stroke-opacity', 0.6)
        .attr('x1', d => d.source.x)
        .attr('y1', d => d.source.y)
        .attr('x2', d => d.target.x)
        .attr('y2', d => d.target.y)

      // 关系标签
      linkG.selectAll('text.edge-label')
        .data(d => [d])
        .join('text')
        .attr('class', 'edge-label')
        .text(d => d.type)
        .attr('x', d => (d.source.x + d.target.x) / 2)
        .attr('y', d => (d.source.y + d.target.y) / 2 - 5)
        .attr('text-anchor', 'middle')
        .attr('font-size', 10)
        .attr('fill', '#666')
        .attr('font-weight', 500)

      // 节点组
      const nodeG = g.selectAll('g.node')
        .data(simNodes)
        .join('g')
        .attr('class', 'node')
        .style('cursor', 'pointer')
        .attr('transform', d => `translate(${d.x},${d.y})`)
        .on('click', (e, d) => emit('clickCharacter', d))
        .call(drag()
          .on('start', (e, d) => {
            if (!e.active) simulation.alphaTarget(0.3).restart()
            d.fx = d.x; d.fy = d.y
          })
          .on('drag', (e, d) => {
            d.fx = e.x; d.fy = e.y
          })
          .on('end', (e, d) => {
            if (!e.active) simulation.alphaTarget(0)
            d.fx = null; d.fy = null
          })
        )

      // 圆形节点
      nodeG.selectAll('circle')
        .data(d => [d])
        .join('circle')
        .attr('r', 18)
        .attr('fill', d => colorMap[d.id])
        .attr('stroke', '#fff')
        .attr('stroke-width', 3)
        .attr('filter', 'drop-shadow(0 2px 3px rgba(0,0,0,0.15))')

      // 名字标签
      nodeG.selectAll('text.node-label')
        .data(d => [d])
        .join('text')
        .attr('class', 'node-label')
        .text(d => d.name.length > 5 ? d.name.slice(0, 5) + '…' : d.name)
        .attr('text-anchor', 'middle')
        .attr('dy', 34)
        .attr('font-size', 12)
        .attr('fill', '#303133')
        .attr('font-weight', 600)
        .attr('paint-order', 'stroke')
        .attr('stroke', '#fff')
        .attr('stroke-width', 2)
    })
}

updateRange()

defineExpose({ restartSimulation, updateSimulation })
</script>

<style scoped>
.graph-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 200px;
}
.graph-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0 2px 0;
}
.control-label {
  font-size: 12px;
  color: #909399;
  white-space: nowrap;
}
.control-chapter-label {
  font-size: 12px;
  color: #409eff;
  white-space: nowrap;
  min-width: 50px;
  text-align: right;
  font-weight: 500;
}
.graph-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  padding: 4px 0 6px 0;
}
.legend-item {
  font-size: 11px;
  color: #606266;
  display: flex;
  align-items: center;
  gap: 4px;
}
.legend-color {
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 3px;
}
.graph-svg {
  flex: 1;
  min-height: 0;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: linear-gradient(135deg, #f8f9fc 0%, #f0f2f5 100%);
  cursor: grab;
}
.graph-svg:active {
  cursor: grabbing;
}
.empty-state {
  padding: 20px 0;
}
</style>
