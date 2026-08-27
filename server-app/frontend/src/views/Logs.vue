<script setup lang="ts">
/**
 * 日志视图：实时滚动显示 frp 运行日志。
 */
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import { EventsOff, EventsOn } from '../../wailsjs/runtime/runtime'
import { api } from '../api'

const lines = ref<string[]>([])
const autoScroll = ref(true)
const logBox = ref<HTMLElement | null>(null)

/** 追加一行日志并按需滚动到底部 */
async function append(line: string) {
  lines.value.push(line)
  // 保留最近 3000 行
  if (lines.value.length > 3000) {
    lines.value.splice(0, lines.value.length - 3000)
  }
  if (autoScroll.value) {
    await nextTick()
    if (logBox.value) {
      logBox.value.scrollTop = logBox.value.scrollHeight
    }
  }
}

/** 根据级别给日志行上色 */
function lineClass(line: string): string {
  if (line.includes('[ERRO]') || line.includes('error')) return 'log-error'
  if (line.includes('[WARN]') || line.includes('warn')) return 'log-warn'
  return 'log-info'
}

onMounted(async () => {
  // 先加载历史缓冲，再订阅实时推送
  lines.value = await api.GetLogs()
  EventsOn('log', (line: string) => append(line))
})

onUnmounted(() => {
  EventsOff('log')
})
</script>

<template>
  <div class="logs-panel">
    <div class="logs-toolbar">
      <span class="logs-title">运行日志</span>
      <div class="logs-actions">
        <el-switch
          v-model="autoScroll"
          active-text="自动滚动"
          size="small"
        />
        <el-button size="small" text @click="lines = []">清空</el-button>
      </div>
    </div>
    <div ref="logBox" class="log-box">
      <div
        v-for="(line, i) in lines"
        :key="i"
        class="log-line"
        :class="lineClass(line)"
      >
        {{ line }}
      </div>
      <div v-if="!lines.length" class="log-empty">暂无日志</div>
    </div>
  </div>
</template>

<style scoped>
.logs-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1e2128;
  border: 1px solid #2a2e38;
  border-radius: 10px;
  padding: 12px;
}

.logs-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.logs-title {
  font-size: 14px;
  font-weight: 600;
}

.logs-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.log-box {
  flex: 1;
  overflow: auto;
  background: #101216;
  border-radius: 6px;
  padding: 10px 12px;
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.7;
}

.log-line {
  white-space: pre-wrap;
  word-break: break-all;
}

.log-info {
  color: #c9cdd4;
}

.log-warn {
  color: #e6a23c;
}

.log-error {
  color: #f56c6c;
}

.log-empty {
  color: #6b7280;
  text-align: center;
  padding: 40px 0;
}
</style>
