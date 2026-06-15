<template>
  <div class="register-container">
    <div class="bg-grid"></div>
    <div class="blob blob-1"></div>
    <div class="blob blob-2"></div>

    <div class="register-card">
      <div class="brand">
        <div class="brand-icon">
          <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
            <path d="M14 2L26 8.5V19.5L14 26L2 19.5V8.5L14 2Z" stroke="#3d7eff" stroke-width="1.5" fill="rgba(61,126,255,.12)"/>
            <circle cx="14" cy="14" r="4" fill="#5edfff" opacity=".9"/>
          </svg>
        </div>
        <span class="brand-name">GopherAI</span>
      </div>

      <h1 class="title">创建账号</h1>
      <p class="subtitle">注册后即可使用全部 AI 功能</p>

      <form class="form" @submit.prevent="handleRegister">
        <div class="field">
          <label class="field-label">邮箱</label>
          <input
            v-model="registerForm.email"
            class="field-input"
            type="email"
            placeholder="请输入邮箱地址"
            autocomplete="email"
          />
        </div>

        <div class="field">
          <label class="field-label">验证码</label>
          <div class="captcha-row">
            <input
              v-model="registerForm.captcha"
              class="field-input"
              type="text"
              placeholder="请输入验证码"
              autocomplete="one-time-code"
            />
            <button
              type="button"
              class="captcha-btn"
              :disabled="codeLoading || countdown > 0"
              @click="sendCode"
            >
              <span v-if="codeLoading" class="loading-dots">
                <span></span><span></span><span></span>
              </span>
              <span v-else-if="countdown > 0">{{ countdown }}s</span>
              <span v-else>发送验证码</span>
            </button>
          </div>
        </div>

        <div class="field">
          <label class="field-label">密码</label>
          <input
            v-model="registerForm.password"
            class="field-input"
            type="password"
            placeholder="至少 6 位"
            autocomplete="new-password"
          />
        </div>

        <div class="field">
          <label class="field-label">确认密码</label>
          <input
            v-model="registerForm.confirmPassword"
            class="field-input"
            type="password"
            placeholder="请再次输入密码"
            autocomplete="new-password"
          />
        </div>

        <button class="submit-btn" type="submit" :disabled="loading">
          <span v-if="!loading">注 册</span>
          <span v-else class="loading-dots">
            <span></span><span></span><span></span>
          </span>
        </button>
      </form>

      <p class="login-link">
        已有账号？
        <a @click.prevent="$router.push('/login')">立即登录</a>
      </p>
    </div>
  </div>
</template>

<script>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '../utils/api'

export default {
  name: 'RegisterView',
  setup() {
    const router = useRouter()
    const registerFormRef = ref()
    const loading = ref(false)
    const codeLoading = ref(false)
    const countdown = ref(0)

    const registerForm = reactive({
      email: '',
      captcha: '',
      password: '',
      confirmPassword: ''
    })

    const validateConfirmPassword = (rule, value, callback) => {
      if (value !== registerForm.password) {
        callback(new Error('两次输入密码不一致'))
      } else {
        callback()
      }
    }

    const registerRules = {
      email: [
        { required: true, message: '请输入邮箱', trigger: 'blur' },
        { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
      ],
      captcha: [
        { required: true, message: '请输入验证码', trigger: 'blur' }
      ],
      password: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
      ],
      confirmPassword: [
        { required: true, message: '请确认密码', trigger: 'blur' },
        { validator: validateConfirmPassword, trigger: 'blur' }
      ]
    }

    const sendCode = async () => {
      if (!registerForm.email) {
        ElMessage.warning('请先输入邮箱')
        return
      }
      try {
        codeLoading.value = true
        const response = await api.post('/user/captcha', { email: registerForm.email })
        if (response.data.status_code === 1000) {
          ElMessage.success('验证码发送成功')
          countdown.value = 60
          const timer = setInterval(() => {
            countdown.value--
            if (countdown.value <= 0) clearInterval(timer)
          }, 1000)
        } else {
          ElMessage.error(response.data.status_msg || '验证码发送失败')
        }
      } catch (error) {
        console.error('Send code error:', error)
        ElMessage.error('验证码发送失败，请重试')
      } finally {
        codeLoading.value = false
      }
    }

    const handleRegister = async () => {
      if (!registerForm.email) { ElMessage.warning('请输入邮箱'); return }
      if (!registerForm.captcha) { ElMessage.warning('请输入验证码'); return }
      if (!registerForm.password || registerForm.password.length < 6) { ElMessage.warning('密码不能少于6位'); return }
      if (registerForm.password !== registerForm.confirmPassword) { ElMessage.warning('两次密码不一致'); return }
      try {
        loading.value = true
        const response = await api.post('/user/register', {
          email: registerForm.email,
          captcha: registerForm.captcha,
          password: registerForm.password
        })
        if (response.data.status_code === 1000) {
          ElMessage.success('注册成功，请登录')
          router.push('/login')
        } else {
          ElMessage.error(response.data.status_msg || '注册失败')
        }
      } catch (error) {
        console.error('Register error:', error)
        ElMessage.error('注册失败，请重试')
      } finally {
        loading.value = false
      }
    }

    return {
      registerFormRef,
      loading,
      codeLoading,
      countdown,
      registerForm,
      registerRules,
      sendCode,
      handleRegister
    }
  }
}
</script>

<style scoped>
.register-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0f1117;
  font-family: "Inter", "SF Pro Display", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  position: relative;
  overflow: hidden;
}

.bg-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(61,126,255,.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(61,126,255,.04) 1px, transparent 1px);
  background-size: 52px 52px;
  pointer-events: none;
}

.blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  pointer-events: none;
  opacity: .16;
}
.blob-1 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, #3d7eff 0%, transparent 70%);
  top: -140px; right: -100px;
  animation: blobDrift 18s ease-in-out infinite alternate;
}
.blob-2 {
  width: 360px; height: 360px;
  background: radial-gradient(circle, #5edfff 0%, transparent 70%);
  bottom: -80px; left: -60px;
  animation: blobDrift 24s ease-in-out infinite alternate-reverse;
}
@keyframes blobDrift {
  from { transform: translate(0,0) scale(1); }
  to   { transform: translate(30px,20px) scale(1.08); }
}

.register-card {
  position: relative;
  z-index: 10;
  width: 420px;
  padding: 40px 36px 36px;
  background: #161b27;
  border: 1px solid rgba(255,255,255,.07);
  border-radius: 20px;
  box-shadow:
    0 0 0 1px rgba(61,126,255,.06),
    0 24px 60px rgba(0,0,0,.5);
  animation: cardIn .5s cubic-bezier(.22,1,.36,1);
}
@keyframes cardIn {
  from { opacity: 0; transform: translateY(24px) scale(.97); }
  to   { opacity: 1; transform: translateY(0) scale(1); }
}
.register-card::before {
  content: '';
  position: absolute;
  top: 0; left: 50%;
  transform: translateX(-50%);
  width: 60%; height: 1px;
  background: linear-gradient(90deg, transparent, #3d7eff, #5edfff, #3d7eff, transparent);
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 28px;
}
.brand-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px; height: 40px;
  background: rgba(61,126,255,.1);
  border: 1px solid rgba(61,126,255,.2);
  border-radius: 10px;
}
.brand-name {
  font-size: 17px;
  font-weight: 700;
  letter-spacing: .04em;
  background: linear-gradient(90deg, #90bcff, #5edfff);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.title {
  margin: 0 0 6px;
  font-size: 22px;
  font-weight: 700;
  color: #e8eaf0;
  letter-spacing: -.01em;
}
.subtitle {
  margin: 0 0 28px;
  font-size: 13.5px;
  color: #3d4a66;
}

.form { display: flex; flex-direction: column; gap: 16px; }

.field { display: flex; flex-direction: column; gap: 6px; }

.field-label {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: .06em;
  text-transform: uppercase;
  color: #5a6480;
}

.field-input {
  padding: 11px 14px;
  background: rgba(255,255,255,.04);
  border: 1px solid rgba(255,255,255,.09);
  border-radius: 10px;
  color: #e8eaf0;
  font-size: 14px;
  font-family: inherit;
  outline: none;
  transition: border-color .18s, box-shadow .18s, background .18s;
  width: 100%;
}
.field-input::placeholder { color: #3d4a66; }
.field-input:focus {
  border-color: rgba(61,126,255,.55);
  background: rgba(61,126,255,.05);
  box-shadow: 0 0 0 3px rgba(61,126,255,.1);
}

.captcha-row {
  display: flex;
  gap: 10px;
}
.captcha-row .field-input { flex: 1; }

.captcha-btn {
  flex-shrink: 0;
  padding: 0 14px;
  height: 44px;
  background: rgba(61,126,255,.12);
  color: #6aa3ff;
  border: 1px solid rgba(61,126,255,.22);
  border-radius: 10px;
  font-size: 12.5px;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  white-space: nowrap;
  transition: all .15s;
}
.captcha-btn:hover:not(:disabled) {
  background: rgba(61,126,255,.22);
  border-color: rgba(61,126,255,.45);
  color: #90bcff;
}
.captcha-btn:disabled { opacity: .45; cursor: not-allowed; }

.submit-btn {
  margin-top: 6px;
  width: 100%;
  padding: 13px 0;
  background: #3d7eff;
  color: #fff;
  font-size: 14.5px;
  font-weight: 700;
  letter-spacing: .05em;
  border: none;
  border-radius: 11px;
  cursor: pointer;
  box-shadow: 0 4px 20px rgba(61,126,255,.35);
  transition: background .18s, transform .15s, box-shadow .18s;
  font-family: inherit;
  position: relative;
  overflow: hidden;
}
.submit-btn::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,.08), transparent);
  transform: translateX(-100%);
  transition: transform .5s;
}
.submit-btn:hover:not(:disabled)::before { transform: translateX(100%); }
.submit-btn:hover:not(:disabled) {
  background: #5590ff;
  box-shadow: 0 6px 28px rgba(61,126,255,.45);
  transform: translateY(-1px);
}
.submit-btn:disabled {
  background: rgba(255,255,255,.07);
  color: #3d4a66;
  box-shadow: none;
  cursor: not-allowed;
}

.loading-dots {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.loading-dots span {
  width: 5px; height: 5px;
  border-radius: 50%;
  background: rgba(255,255,255,.7);
  animation: dot .9s infinite;
}
.loading-dots span:nth-child(2) { animation-delay: .15s; }
.loading-dots span:nth-child(3) { animation-delay: .3s; }
@keyframes dot {
  0%,80%,100% { transform: scale(.6); opacity: .4; }
  40%          { transform: scale(1); opacity: 1; }
}

.login-link {
  margin-top: 22px;
  text-align: center;
  font-size: 13px;
  color: #3d4a66;
}
.login-link a {
  color: #6aa3ff;
  cursor: pointer;
  font-weight: 500;
  transition: color .15s;
  text-decoration: none;
}
.login-link a:hover { color: #90bcff; }
</style>
