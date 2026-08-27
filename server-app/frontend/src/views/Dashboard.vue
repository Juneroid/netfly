<script setup lang="ts">
/**
 * 运行状态视图：服务启停控制、实时统计、在线代理列表。
 */
import { onMounted, onUnmounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api, formatBytes, type ServerStats, type ProxyBrief } from '../api'

const stats = ref<ServerStats | null>(null)
const proxies = ref<ProxyBrief[]>([])
const loading = ref(false)

let timer: number | undefined

/** 拉取服务统计与代理列表 */
async function refresh() {
  try {
    stats.value = await api.GetStats()
    proxies.value = await api.GetProxies()
  } catch (e) {
    // 静默处理轮询错误，避免打扰
  }
}

/** 启动服务 */
async function start() {
  loading.value = true
  try {
    await api.StartService()
    ElMessage.success('服务已启动')
  } catch (e) {
    ElMessage.error(String(e))
  } finally {
    loading.value = false
    refresh()
  }
}

/** 停止服务 */
async function stop() {
  loading.value = true
  try {
    await api.StopService()
    ElMessage.success('服务已停止')
  } finally {
    loading.value = false
    refresh()
  }
}

/** 打开 frps 管理网页 */
function openDashboard() {
  api.OpenDashboard()
}

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
    <!-- 服务控制区 -->
    <div class="control-card">
      <div class="control-left">
        <span class="status-dot" :class="{ running: stats?.running }"></span>
        <div>
          <div class="status-title">
            {{ stats?.running ? '服务运行中' : '服务已停止' }}
          </div>
          <div class="status-sub">监听端口 {{ stats ? '' : '' }}</div>
        </div>
      </div>
      <div class="control-right">
        <el-button
          v-if="!stats?.running"
          type="primary"
          :loading="loading"
          @click="start"
        >
          <el-icon><VideoPlay /></el-icon>
          启动服务
        </el-button>
        <el-button v-else type="danger" :loading="loading" @click="stop">
          <el-icon><VideoPause /></el-icon>
          停止服务
        </el-button>
        <el-button
          v-if="stats?.running"
          plain
          @click="openDashboard"
        >
          <el-icon><Link /></el-icon>
          打开管理网页
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stat-row">
      <el-col :span="6">
        <div class="stat-card">
          <div class="stat-value">{{ stats?.clientCount ?? 0 }}</div>
          <div class="stat-label">在线客户端</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card">
          <div class="stat-value">{{ stats?.proxyCount ?? 0 }}</div>
          <div class="stat-label">在线代理</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card">
          <div class="stat-value">{{ stats?.curConns ?? 0 }}</div>
          <div class="stat-label">当前连接数</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card">
          <div class="stat-value small">
            {{ formatBytes(stats?.sessionTrafficIn ?? 0) }}
          </div>
          <div class="stat-label">本次流入 / 流出 {{ formatBytes(stats?.sessionTrafficOut ?? 0) }}</div>
        </div>
      </el-col>
    </el-row>

    <!-- 在线代理 -->
    <div class="panel">
      <div class="panel-title">在线代理</div>
      <el-table :data="proxies" style="width: 100%" size="default">
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="type" label="类型" width="90">
          <template #default="{ row }">
            <el-tag size="small" effect="dark">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="user" label="所属用户" width="120" />
        <el-table-column prop="curConns" label="连接数" width="90" />
        <el-table-column label="今日流入" width="110">
          <template #default="{ row }">
            {{ formatBytes(row.todayTrafficIn) }}
          </template>
        </el-table-column>
        <el-table-column label="今日流出" width="110">
          <template #default="{ row }">
            {{ formatBytes(row.todayTrafficOut) }}
          </template>
        </el-table-column>
        <el-table-column v-if="!proxies.length" label="暂无在线代理" />
        <template #empty>
          <el-empty description="暂无在线代理，等待客户端接入" :image-size="80" />
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

.status-dot.running {
  background: #67c23a;
  box-shadow: 0 0 10px #67c23a;
}

.status-title {
  font-size: 17px;
  font-weight: 600;
}

.status-sub {
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
  color: #409eff;
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
