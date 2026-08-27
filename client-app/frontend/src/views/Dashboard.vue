<script setup lang="ts">
/**
 * 运行状态视图：连接控制、连接状态、节点流量与速率统计（本地统计）。
 * 节点表格以配置的节点列表为数据源，未连接时也展示全部节点。
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  api,
  connStateText,
  formatBytes,
  formatRate,
  type NetworkStats,
  type ProxyState,
  type ProxyItem,
} from '../api'

const connState = ref('idle')
const netStats = ref<NetworkStats | null>(null)
const proxyStates = ref<ProxyState[]>([])
const proxies = ref<ProxyItem[]>([])
const loading = ref(false)

let timer: number | undefined

/** 轮询刷新连接状态、节点列表、状态与网络统计 */
async function refresh() {
  try {
    const running = await api.ServiceRunning()
    // 未运行时连接状态一律视为未连接，防止残留状态导致按钮异常
    connState.value = running ? await api.GetConnState() : 'idle'
    const cfg = await api.GetConfig()
    proxies.value = cfg.proxies || []
    proxyStates.value = await api.GetProxyStates()
    netStats.value = await api.GetNetworkStats()
  } catch (e) {
    // 静默处理轮询错误
  }
}

/** 连接服务端 */
async function start() {
  loading.value = true
  try {
    await api.StartService()
    ElMessage.success('已开始连接服务端')
  } catch (e) {
    ElMessage.error(String(e))
  } finally {
    loading.value = false
    refresh()
  }
}

/** 断开连接 */
async function stop() {
  loading.value = true
  try {
    await api.StopService()
    ElMessage.success('已断开')
  } catch (e) {
    ElMessage.error('断开失败: ' + String(e))
  } finally {
    loading.value = false
    refresh()
  }
}

/** 按节点名查运行状态 */
function stateOf(name: string): ProxyState | undefined {
  return proxyStates.value.find((s) => s.name === name)
}

/** 按节点名查本地流量统计 */
function netOf(name: string) {
  return netStats.value?.nodes.find((n) => n.name === name)
}

/** 状态标签类型 */
function stateTag(status?: string): 'success' | 'danger' | 'info' | 'warning' {
  if (status === 'running') return 'success'
  if (status === 'start error' || status === 'check failed') return 'danger'
  if (status === 'wait start' || status === 'new') return 'warning'
  return 'info'
}

/** 表格行：配置节点 + 运行状态 + 流量合并 */
const rows = computed(() =>
  proxies.value.map((p) => ({
    ...p,
    state: stateOf(p.name),
    net: netOf(p.name),
  }))
)

onMounted(() => {
  refresh()
  timer = window.setInterval(refresh, 2000)
})
onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <div class="dashboard">
    <!-- 连接控制区 -->
    <div class="control-card">
      <div class="control-left">
        <span
          class="status-dot"
          :class="{
            online: connState === 'online',
            connecting:
              connState === 'connecting' || connState === 'offline',
          }"
        ></span>
        <div>
          <div class="status-title">
            {{ connStateText[connState] || connState }}
          </div>
          <div class="status-sub server-addr">
            {{ proxies.length ? `${proxies.length} 个节点` : '暂无节点' }}
          </div>
        </div>
      </div>
      <div class="control-right">
        <el-button
          v-if="connState === 'idle'"
          type="primary"
          :loading="loading"
          @click="start"
        >
          <el-icon><VideoPlay /></el-icon>
          连接服务
        </el-button>
        <el-button v-else type="danger" :loading="loading" @click="stop">
          <el-icon><VideoPause /></el-icon>
          断开
        </el-button>
      </div>
    </div>

    <!-- 汇总统计 -->
    <el-row :gutter="16" class="stat-row">
      <el-col :span="6">
        <div class="stat-card">
          <div class="stat-value">{{ proxies.length }}</div>
          <div class="stat-label">节点总数</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card">
          <div class="stat-value small">
            {{ netStats ? formatRate(
                netStats.nodes.reduce((s, n) => s + n.rateIn, 0)
              ) : '--' }}
          </div>
          <div class="stat-label">入站速率</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card">
          <div class="stat-value small">
            {{ netStats ? formatRate(
                netStats.nodes.reduce((s, n) => s + n.rateOut, 0)
              ) : '--' }}
          </div>
          <div class="stat-label">出站速率</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card">
          <div class="stat-value small">
            {{ netStats ? formatBytes(netStats.totalIn + netStats.totalOut) : '0 B' }}
          </div>
          <div class="stat-label">累计流量</div>
        </div>
      </el-col>
    </el-row>

    <!-- 节点状态与网络 -->
    <div class="panel">
      <div class="panel-title">节点网络（本地统计）</div>
      <el-table :data="rows" style="width: 100%">
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column prop="type" label="类型" width="80">
          <template #default="{ row }">
            <el-tag size="small" effect="dark">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag v-if="connState !== 'idle'" :type="stateTag(row.state?.status)" size="small">
              {{ row.state?.status || 'closed' }}
            </el-tag>
            <el-tag v-else type="info" size="small">未连接</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="本地服务" min-width="140">
          <template #default="{ row }">
            {{ row.localIP }}:{{ row.localPort }}
          </template>
        </el-table-column>
        <el-table-column label="入站速率" width="110">
          <template #default="{ row }">
            {{ row.net ? formatRate(row.net.rateIn) : '--' }}
          </template>
        </el-table-column>
        <el-table-column label="出站速率" width="110">
          <template #default="{ row }">
            {{ row.net ? formatRate(row.net.rateOut) : '--' }}
          </template>
        </el-table-column>
        <el-table-column label="累计流量" width="130">
          <template #default="{ row }">
            {{
              row.net
                ? formatBytes(row.net.trafficIn + row.net.trafficOut)
                : '--'
            }}
          </template>
        </el-table-column>
        <template #empty>
          <el-empty description="暂无节点，请前往「节点管理」创建" :image-size="80" />
        </template>
      </el-table>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.control-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: #1e2128;
  border: 1px solid #2a2e38;
  border-radius: 10px;
}

.control-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.status-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: #909399;
}

.status-dot.online {
  background: #67c23a;
  box-shadow: 0 0 10px #67c23a;
}

.status-dot.connecting {
  background: #e6a23c;
  box-shadow: 0 0 10px #e6a23c;
  animation: blink 1s infinite;
}

@keyframes blink {
  50% {
    opacity: 0.4;
  }
}

.status-title {
  font-size: 17px;
  font-weight: 600;
}

.server-addr {
  font-size: 12px;
  color: #8b919e;
  margin-top: 2px;
}

.stat-row :deep(.el-col) {
  min-width: 180px;
}

.stat-card {
  padding: 18px 20px;
  background: #1e2128;
  border: 1px solid #2a2e38;
  border-radius: 10px;
}

.stat-value {
  font-size: 26px;
  font-weight: 700;
  color: #67c23a;
}

.stat-value.small {
  font-size: 20px;
}

.stat-label {
  margin-top: 6px;
  font-size: 12px;
  color: #8b919e;
}

.panel {
  background: #1e2128;
  border: 1px solid #2a2e38;
  border-radius: 10px;
  padding: 16px;
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 12px;
}
</style>
