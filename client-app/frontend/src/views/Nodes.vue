<script setup lang="ts">
/**
 * 节点管理视图：新建/编辑/删除/启停代理节点，
 * 服务运行中的变更通过热更新即时生效。
 */
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type ProxyItem, type ProxyState } from '../api'

const items = ref<ProxyItem[]>([])
const states = ref<ProxyState[]>([])
const running = ref(false)

const dialogVisible = ref(false)
const editingName = ref('') // 空表示新建
const saving = ref(false)

/** 新建/编辑表单数据 */
const form = reactive<ProxyItem>({
  name: '',
  type: 'tcp',
  localIP: '127.0.0.1',
  localPort: 80,
  remotePort: 6000,
  customDomains: '',
  subDomain: '',
  useEncryption: false,
  useCompression: false,
  enabled: true,
})

/** 加载节点配置与状态 */
async function refresh() {
  const cfg = await api.GetConfig()
  items.value = cfg.proxies || []
  running.value = await api.ServiceRunning()
  states.value = await api.GetProxyStates()
}

/** 打开新建对话框 */
function openCreate() {
  editingName.value = ''
  Object.assign(form, {
    name: '',
    type: 'tcp',
    localIP: '127.0.0.1',
    localPort: 80,
    remotePort: 6000,
    customDomains: '',
    subDomain: '',
    useEncryption: false,
    useCompression: false,
    enabled: true,
  })
  dialogVisible.value = true
}

/** 打开编辑对话框 */
function openEdit(item: ProxyItem) {
  editingName.value = item.name
  Object.assign(form, JSON.parse(JSON.stringify(item)))
  dialogVisible.value = true
}

/** 提交新建/编辑 */
async function save() {
  saving.value = true
  try {
    if (editingName.value) {
      await api.UpdateProxy(editingName.value, { ...form })
      ElMessage.success('节点已更新' + (running.value ? '（热更新生效）' : ''))
    } else {
      await api.AddProxy({ ...form })
      ElMessage.success('节点已创建' + (running.value ? '（热更新生效）' : ''))
    }
    dialogVisible.value = false
  } catch (e) {
    ElMessage.error(String(e))
  } finally {
    saving.value = false
    refresh()
  }
}

/** 删除节点 */
async function remove(item: ProxyItem) {
  try {
    await ElMessageBox.confirm(
      `确定删除节点「${item.name}」吗？`,
      '删除确认',
      { type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await api.DeleteProxy(item.name)
    ElMessage.success('节点已删除')
  } catch (e) {
    ElMessage.error(String(e))
  }
  refresh()
}

/** 切换节点启用状态 */
async function toggle(item: ProxyItem) {
  try {
    await api.ToggleProxy(item.name)
  } catch (e) {
    ElMessage.error(String(e))
  }
  refresh()
}

/** 查询节点运行状态 */
function stateOf(name: string): ProxyState | undefined {
  return states.value.find((s) => s.name === name)
}

function stateTag(status?: string): 'success' | 'danger' | 'info' | 'warning' {
  if (status === 'running') return 'success'
  if (status === 'start error' || status === 'check failed') return 'danger'
  if (status === 'wait start' || status === 'new') return 'warning'
  return 'info'
}

onMounted(refresh)
</script>

<template>
  <div class="nodes">
    <div class="panel">
      <div class="panel-head">
        <div class="panel-title">
          代理节点
          <span v-if="running" class="panel-tip">
            服务运行中，变更即时生效
          </span>
        </div>
        <el-button type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>
          新建节点
        </el-button>
      </div>

      <el-table :data="items" style="width: 100%">
        <el-table-column label="启用" width="80">
          <template #default="{ row }">
            <el-switch
              :model-value="row.enabled"
              @change="toggle(row)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column prop="type" label="类型" width="80">
          <template #default="{ row }">
            <el-tag size="small" effect="dark">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="本地服务" min-width="140">
          <template #default="{ row }">
            {{ row.localIP }}:{{ row.localPort }}
          </template>
        </el-table-column>
        <el-table-column label="暴露方式" min-width="160">
          <template #default="{ row }">
            <span v-if="row.type === 'tcp' || row.type === 'udp'">
              远程端口 {{ row.remotePort }}
            </span>
            <span v-else>
              {{ row.customDomains || row.subDomain || '-' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="运行状态" width="110">
          <template #default="{ row }">
            <el-tag
              v-if="running"
              :type="stateTag(stateOf(row.name)?.status)"
              size="small"
            >
              {{ stateOf(row.name)?.status || 'closed' }}
            </el-tag>
            <el-tag v-else type="info" size="small">未连接</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="openEdit(row)">
              编辑
            </el-button>
            <el-button size="small" text type="danger" @click="remove(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
        <template #empty>
          <el-empty description="暂无节点，点击右上角「新建节点」创建" :image-size="80" />
        </template>
      </el-table>
    </div>

    <!-- 新建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingName ? '编辑节点' : '新建节点'"
      width="560px"
    >
      <el-form label-width="110px">
        <el-form-item label="节点名称" required>
          <el-input
            v-model="form.name"
            placeholder="唯一名称，如 web-8080"
            :disabled="!!editingName"
            clearable
          />
        </el-form-item>

        <el-form-item label="类型" required>
          <el-radio-group v-model="form.type">
            <el-radio-button value="tcp">TCP</el-radio-button>
            <el-radio-button value="udp">UDP</el-radio-button>
            <el-radio-button value="http">HTTP</el-radio-button>
            <el-radio-button value="https">HTTPS</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="本地地址">
          <el-input v-model="form.localIP" placeholder="127.0.0.1" style="width: 180px" />
        </el-form-item>

        <el-form-item label="本地端口" required>
          <el-input-number v-model="form.localPort" :min="1" :max="65535" />
        </el-form-item>

        <template v-if="form.type === 'tcp' || form.type === 'udp'">
          <el-form-item label="远程端口" required>
            <el-input-number v-model="form.remotePort" :min="1" :max="65535" />
            <span class="form-tip">服务端对外暴露的端口</span>
          </el-form-item>
        </template>

        <template v-else>
          <el-form-item label="自定义域名">
            <el-input
              v-model="form.customDomains"
              placeholder="多个域名用英文逗号分隔"
              clearable
            />
          </el-form-item>
          <el-form-item label="子域名">
            <el-input
              v-model="form.subDomain"
              placeholder="需服务端配置 subDomainHost"
              clearable
            />
          </el-form-item>
          <el-alert
            type="info"
            :closable="false"
            show-icon
            title="HTTP/HTTPS 节点需要服务端开启对应穿透端口，并将域名解析到服务端"
          />
        </template>

        <el-form-item label="传输加密">
          <el-switch v-model="form.useEncryption" />
        </el-form-item>

        <el-form-item label="传输压缩">
          <el-switch v-model="form.useCompression" />
        </el-form-item>

        <el-form-item label="启用节点">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.nodes {
  display: flex;
  flex-direction: column;
}

.panel {
  background: #1e2128;
  border: 1px solid #2a2e38;
  border-radius: 10px;
  padding: 16px;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
}

.panel-tip {
  font-size: 12px;
  font-weight: 400;
  color: #67c23a;
  margin-left: 8px;
}

.form-tip {
  margin-left: 12px;
  font-size: 12px;
  color: #8b919e;
}
</style>
