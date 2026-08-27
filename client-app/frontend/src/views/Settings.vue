<script setup lang="ts">
/**
 * 设置视图：服务端连接、日志级别、自动连接等配置表单。
 */
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api, type ClientAppConfig } from '../api'

/** 表单数据（深拷贝避免直接修改后端对象） */
const form = reactive<ClientAppConfig>({
  serverAddr: '',
  serverPort: 7000,
  token: '',
  user: '',
  logLevel: 'info',
  autoStart: false,
  proxies: [],
})

const saving = ref(false)

onMounted(async () => {
  Object.assign(form, await api.GetConfig())
})

/** 保存配置 */
async function save() {
  saving.value = true
  try {
    await api.SaveConfig({ ...form })
    ElMessage.success('配置已保存')
  } catch (e) {
    ElMessage.error(String(e))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="settings">
    <el-form label-width="170px" class="settings-form">
      <div class="form-section">服务端连接</div>

      <el-form-item label="服务端地址" required>
        <el-input
          v-model="form.serverAddr"
          placeholder="服务器 IP 或域名"
          clearable
        />
      </el-form-item>

      <el-form-item label="服务端端口">
        <el-input-number v-model="form.serverPort" :min="1" :max="65535" />
      </el-form-item>

      <el-form-item label="认证口令 (Token)">
        <el-input
          v-model="form.token"
          placeholder="与服务端设置一致"
          show-password
          clearable
        />
      </el-form-item>

      <el-form-item label="用户前缀 (User)">
        <el-input
          v-model="form.user"
          placeholder="可选，多客户端区分用"
          clearable
        />
        <span class="form-tip">节点将以 前缀.名称 注册到服务端</span>
      </el-form-item>

      <div class="form-section">其他</div>

      <el-form-item label="日志级别">
        <el-select v-model="form.logLevel" style="width: 140px">
          <el-option label="trace" value="trace" />
          <el-option label="debug" value="debug" />
          <el-option label="info" value="info" />
          <el-option label="warn" value="warn" />
          <el-option label="error" value="error" />
        </el-select>
      </el-form-item>

      <el-form-item label="程序启动时自动连接">
        <el-switch v-model="form.autoStart" />
      </el-form-item>

      <el-form-item>
        <el-button type="primary" :loading="saving" @click="save">
          保存配置
        </el-button>
        <span class="form-tip">连接参数修改后需重新连接生效</span>
      </el-form-item>
    </el-form>
  </div>
</template>

<style scoped>
.settings {
  background: #1e2128;
  border: 1px solid #2a2e38;
  border-radius: 10px;
  padding: 24px;
  max-width: 760px;
}

.form-section {
  font-size: 13px;
  font-weight: 600;
  color: #67c23a;
  margin: 18px 0 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #2a2e38;
}

.form-section:first-child {
  margin-top: 0;
}

.form-tip {
  margin-left: 12px;
  font-size: 12px;
  color: #8b919e;
}
</style>
