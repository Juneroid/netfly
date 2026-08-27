<script setup lang="ts">
/**
 * 设置视图：服务端口、Token、管理网页、穿透端口等配置表单。
 */
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api, type ServerAppConfig } from '../api'

/** 表单数据（深拷贝避免直接修改后端对象） */
const form = reactive<ServerAppConfig>({
  bindAddr: '0.0.0.0',
  bindPort: 7000,
  token: '',
  vhostHTTPPort: 0,
  vhostHTTPSPort: 0,
  dashboardEnabled: true,
  dashboardPort: 7500,
  dashboardUser: 'admin',
  dashboardPassword: 'admin',
  logLevel: 'info',
  autoStart: false,
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
    <el-form label-width="150px" class="settings-form">
      <div class="form-section">基础服务</div>

      <el-form-item label="服务端口">
        <el-input-number v-model="form.bindPort" :min="1" :max="65535" />
        <span class="form-tip">客户端连接此端口</span>
      </el-form-item>

      <el-form-item label="认证口令 (Token)">
        <el-input
          v-model="form.token"
          placeholder="留空表示不启用认证（不推荐）"
          show-password
          clearable
        />
        <span class="form-tip">客户端需填写相同的口令</span>
      </el-form-item>

      <div class="form-section">HTTP / HTTPS 穿透（可选）</div>

      <el-form-item label="HTTP 穿透端口">
        <el-input-number v-model="form.vhostHTTPPort" :min="0" :max="65535" />
        <span class="form-tip">0 表示关闭</span>
      </el-form-item>

      <el-form-item label="HTTPS 穿透端口">
        <el-input-number v-model="form.vhostHTTPSPort" :min="0" :max="65535" />
        <span class="form-tip">0 表示关闭</span>
      </el-form-item>

      <div class="form-section">管理网页</div>

      <el-form-item label="开启管理网页">
        <el-switch v-model="form.dashboardEnabled" />
      </el-form-item>

      <template v-if="form.dashboardEnabled">
        <el-form-item label="管理网页端口">
          <el-input-number v-model="form.dashboardPort" :min="1" :max="65535" />
        </el-form-item>

        <el-form-item label="登录用户名">
          <el-input v-model="form.dashboardUser" clearable />
        </el-form-item>

        <el-form-item label="登录密码">
          <el-input v-model="form.dashboardPassword" show-password clearable />
        </el-form-item>
      </template>

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

      <el-form-item label="程序启动时自动开启服务">
        <el-switch v-model="form.autoStart" />
      </el-form-item>

      <el-form-item>
        <el-button type="primary" :loading="saving" @click="save">
          保存配置
        </el-button>
        <span class="form-tip">服务运行中的修改需重启服务后生效</span>
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
  max-width: 720px;
}

.form-section {
  font-size: 13px;
  font-weight: 600;
  color: #409eff;
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
