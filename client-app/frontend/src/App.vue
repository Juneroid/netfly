<script setup lang="ts">
/**
 * 应用根组件：左侧导航 + 主内容区布局。
 */
import { ref } from 'vue'
import Dashboard from './views/Dashboard.vue'
import Nodes from './views/Nodes.vue'
import Logs from './views/Logs.vue'
import Settings from './views/Settings.vue'

const activeView = ref<'dashboard' | 'nodes' | 'logs' | 'settings'>('dashboard')
</script>

<template>
  <el-container class="layout">
    <!-- 左侧导航 -->
    <el-aside width="200px" class="sidebar">
      <div class="brand">
        <span class="brand-dot"></span>
        <span class="brand-name">NetFly 客户端</span>
      </div>
      <el-menu
        :default-active="activeView"
        class="sidebar-menu"
        @select="(k: any) => (activeView = k)"
      >
        <el-menu-item index="dashboard">
          <el-icon><Monitor /></el-icon>
          <span>运行状态</span>
        </el-menu-item>
        <el-menu-item index="nodes">
          <el-icon><Connection /></el-icon>
          <span>节点管理</span>
        </el-menu-item>
        <el-menu-item index="logs">
          <el-icon><Document /></el-icon>
          <span>日志</span>
        </el-menu-item>
        <el-menu-item index="settings">
          <el-icon><Setting /></el-icon>
          <span>设置</span>
        </el-menu-item>
      </el-menu>
      <div class="sidebar-footer">基于 frp v0.71.0</div>
    </el-aside>

    <!-- 主内容区 -->
    <el-main class="main">
      <Dashboard v-show="activeView === 'dashboard'" />
      <Nodes v-show="activeView === 'nodes'" />
      <Logs v-show="activeView === 'logs'" />
      <Settings v-show="activeView === 'settings'" />
    </el-main>
  </el-container>
</template>

<style scoped>
.layout {
  height: 100%;
}

.sidebar {
  display: flex;
  flex-direction: column;
  background: #14161c;
  border-right: 1px solid #2a2e38;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 20px 16px;
  font-size: 16px;
  font-weight: 600;
}

.brand-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #67c23a;
  box-shadow: 0 0 8px #67c23a;
}

.sidebar-menu {
  flex: 1;
  border-right: none;
  background: transparent;
}

.sidebar-footer {
  padding: 14px 16px;
  font-size: 12px;
  color: #6b7280;
}

.main {
  padding: 20px;
  overflow: auto;
}
</style>
