<script setup lang="ts">
/**
 * 登录 / 注册页：同一个路由下用标签页切换，避免为两个表单各开一条路由
 */
import { reactive, ref } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { login, register } from '@/api/auth'
import { ApiRequestError } from '@/api/request'

const router = useRouter()
const route = useRoute()

const mode = ref<'login' | 'register'>('login')
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive({ username: '', password: '' })

const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' }
  ]
}

/** 把后端错误翻译成人能看懂的话 */
function describeError(err: unknown): string {
  if (err instanceof ApiRequestError) {
    if (mode.value === 'register') {
      if (err.status === 409) return '用户名已存在'
      if (err.status === 404) return '后端还没有实现 POST /register'
    }
    if (err.status === 401) return '用户名或密码错误'
    // 登录成功但接口全 403 —— 后端 authorizator 目前只放行 role=admin
    if (err.status === 403) return '当前账号没有接口权限（后端只放行 admin）'
    return err.message
  }
  return '请求失败，请确认后端服务已启动'
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (mode.value === 'login') {
      await login(form.username, form.password)
      ElMessage.success('登录成功')
      const redirect = route.query.redirect
      router.replace(typeof redirect === 'string' ? redirect : '/')
    } else {
      await register(form.username, form.password)
      // 注册不返回 token，切回登录页并保留用户名，只清密码
      ElMessage.success('注册成功，请登录')
      mode.value = 'login'
      form.password = ''
    }
  } catch (err) {
    ElMessage.error(describeError(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <el-card class="login-card">
      <h1 class="login-title">行情看板</h1>

      <el-tabs v-model="mode" stretch>
        <el-tab-pane label="登录" name="login" />
        <el-tab-pane label="注册" name="register" />
      </el-tabs>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        @submit.prevent="handleSubmit"
      >
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="form.username"
            placeholder="请输入用户名"
            clearable
            @keyup.enter="handleSubmit"
          />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="至少 6 位"
            show-password
            @keyup.enter="handleSubmit"
          />
        </el-form-item>

        <el-button
          type="primary"
          round
          class="submit-btn"
          :loading="submitting"
          @click="handleSubmit"
        >
          {{ mode === 'login' ? '登录' : '注册' }}
        </el-button>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.login-page {
  /* #app 是 flex column，用 flex:1 撑满剩下的高度把卡片居中 */
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-page);
  padding: 24px;
  box-sizing: border-box;
}

.login-card {
  width: 100%;
  max-width: 380px;
  /* 全局 #app 有 text-align: center，表单内容要拉回左对齐 */
  text-align: left;
}

.login-title {
  margin: 0 0 16px;
  font-size: 20px;
  font-weight: 600;
  text-align: center;
  color: var(--text-primary);
}

.submit-btn {
  width: 100%;
  margin-top: 4px;
}
</style>
