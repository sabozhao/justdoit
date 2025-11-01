<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <h1>{{ isLogin ? '用户登录' : '用户注册' }}</h1>
        <p>{{ isLogin ? '欢迎回到智能刷题平台' : '加入智能刷题平台' }}</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="0"
        size="large"
        @submit.prevent="handleSubmit"
      >
        <el-form-item prop="username">
          <el-input
            v-model="form.username"
            placeholder="用户名"
            prefix-icon="User"
            clearable
          />
        </el-form-item>

        <el-form-item prop="email" v-if="!isLogin">
          <el-input
            v-model="form.email"
            placeholder="邮箱（可选）"
            prefix-icon="Message"
            clearable
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="密码"
            prefix-icon="Lock"
            show-password
            clearable
          />
        </el-form-item>

        <el-form-item prop="confirmPassword" v-if="!isLogin">
          <el-input
            v-model="form.confirmPassword"
            type="password"
            placeholder="确认密码"
            prefix-icon="Lock"
            show-password
            clearable
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="authStore.loading"
            @click="handleSubmit"
            class="login-btn"
          >
            {{ isLogin ? '登录' : '注册' }}
          </el-button>
        </el-form-item>
      </el-form>

      <div class="login-footer">
        <span>{{ isLogin ? '还没有账号？' : '已有账号？' }}</span>
        <el-button type="text" @click="toggleMode">
          {{ isLogin ? '立即注册' : '立即登录' }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'

export default {
  name: 'Login',
  setup() {
    const router = useRouter()
    const authStore = useAuthStore()
    const formRef = ref()
    
    const isLogin = ref(true)
    const form = reactive({
      username: '',
      email: '',
      password: '',
      confirmPassword: ''
    })

    const rules = computed(() => ({
      username: [
        { required: true, message: '请输入用户名', trigger: 'blur' },
        { min: 3, max: 20, message: '用户名长度在 3 到 20 个字符', trigger: 'blur' }
      ],
      email: [
        { type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' }
      ],
      password: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于 6 个字符', trigger: 'blur' }
      ],
      confirmPassword: isLogin.value ? [] : [
        { required: true, message: '请确认密码', trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            if (value !== form.password) {
              callback(new Error('两次输入密码不一致'))
            } else {
              callback()
            }
          },
          trigger: 'blur'
        }
      ]
    }))

    const toggleMode = () => {
      isLogin.value = !isLogin.value
      resetForm()
    }

    const resetForm = () => {
      form.username = ''
      form.email = ''
      form.password = ''
      form.confirmPassword = ''
      if (formRef.value) {
        formRef.value.clearValidate()
      }
    }

    const handleSubmit = async () => {
      if (!formRef.value) return

      try {
        await formRef.value.validate()
        
        if (isLogin.value) {
          await authStore.login({
            username: form.username,
            password: form.password
          })
        } else {
          await authStore.register({
            username: form.username,
            email: form.email || undefined,
            password: form.password
          })
        }

        // 登录/注册成功后跳转到首页
        router.push('/')
      } catch (error) {
        console.error('认证失败:', error)
      }
    }

    return {
      formRef,
      isLogin,
      form,
      rules,
      authStore,
      toggleMode,
      handleSubmit
    }
  }
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.login-card {
  width: 100%;
  max-width: 400px;
  background: white;
  border-radius: 20px;
  padding: 40px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
}

.login-header {
  text-align: center;
  margin-bottom: 40px;
}

.login-header h1 {
  font-size: 28px;
  color: #303133;
  margin-bottom: 10px;
}

.login-header p {
  color: #606266;
  font-size: 14px;
}

.login-btn {
  width: 100%;
  height: 50px;
  font-size: 16px;
  border-radius: 10px;
}

.login-footer {
  text-align: center;
  margin-top: 20px;
  color: #909399;
  font-size: 14px;
}

.login-footer .el-button {
  padding: 0;
  margin-left: 5px;
}

:deep(.el-form-item) {
  margin-bottom: 25px;
}

:deep(.el-input__wrapper) {
  border-radius: 10px;
  height: 50px;
}

:deep(.el-input__inner) {
  font-size: 16px;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .login-container {
    padding: 15px;
  }
  
  .login-card {
    padding: 30px 25px;
    border-radius: 15px;
  }
  
  .login-header {
    margin-bottom: 30px;
  }
  
  .login-header h1 {
    font-size: 24px;
    margin-bottom: 8px;
  }
  
  .login-header p {
    font-size: 13px;
  }
  
  .login-btn {
    height: 45px;
    font-size: 15px;
  }
  
  :deep(.el-form-item) {
    margin-bottom: 20px;
  }
  
  :deep(.el-input__wrapper) {
    height: 45px;
    border-radius: 8px;
  }
  
  :deep(.el-input__inner) {
    font-size: 15px;
  }
}

@media (max-width: 480px) {
  .login-container {
    padding: 10px;
  }
  
  .login-card {
    padding: 25px 20px;
    border-radius: 12px;
  }
  
  .login-header {
    margin-bottom: 25px;
  }
  
  .login-header h1 {
    font-size: 20px;
    margin-bottom: 6px;
  }
  
  .login-header p {
    font-size: 12px;
  }
  
  .login-btn {
    height: 42px;
    font-size: 14px;
  }
  
  :deep(.el-form-item) {
    margin-bottom: 18px;
  }
  
  :deep(.el-input__wrapper) {
    height: 42px;
    border-radius: 6px;
  }
  
  :deep(.el-input__inner) {
    font-size: 14px;
  }
  
  .login-footer {
    font-size: 13px;
  }
}
</style>