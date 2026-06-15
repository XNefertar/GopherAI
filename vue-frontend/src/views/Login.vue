<template>
  <div class="login-container">
    <!-- background grid -->
    <div class="bg-grid"></div>
    <!-- glow blobs -->
    <div class="blob blob-1"></div>
    <div class="blob blob-2"></div>

    <div class="login-card">
      <!-- logo area -->
      <div class="brand">
        <div class="brand-icon">
          <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
            <path d="M14 2L26 8.5V19.5L14 26L2 19.5V8.5L14 2Z" stroke="#3d7eff" stroke-width="1.5" fill="rgba(61,126,255,.12)"/>
            <circle cx="14" cy="14" r="4" fill="#5edfff" opacity=".9"/>
          </svg>
        </div>
        <span class="brand-name">GopherAI</span>
      </div>

      <h1 class="title">欢迎回来</h1>
      <p class="subtitle">登录你的 AI 工作台</p>

      <form class="form" @submit.prevent="handleLogin">
        <div class="field">
          <label class="field-label">用户名</label>
          <input
            v-model="loginForm.username"
            class="field-input"
            type="text"
            placeholder="请输入用户名"
            autocomplete="username"
          />
        </div>

        <div class="field">
          <label class="field-label">密码</label>
          <input
            v-model="loginForm.password"
            class="field-input"
            type="password"
            placeholder="请输入密码（至少 6 位）"
            autocomplete="current-password"
          />
        </div>

        <button class="submit-btn" type="submit" :disabled="loading">
          <span v-if="!loading">登 录</span>
          <span v-else class="loading-dots">
            <span></span><span></span><span></span>
          </span>
        </button>
      </form>

      <p class="register-link">
        还没有账号？
        <a @click.prevent="$router.push('/register')">立即注册</a>
      </p>
    </div>
  </div>
</template>

<script>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '../utils/api'

export default {
  name: 'LoginView',
  setup() {
    const router = useRouter()
    const loginFormRef = ref()
    const loading = ref(false)
    const loginForm = ref({
      username: '',
      password: ''
    })

    const loginRules = {
      username: [
        { required: true, message: '请输入用户名', trigger: 'blur' }
      ],
      password: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
      ]
    }

    const handleLogin = async () => {
      if (!loginForm.value.username) {
        ElMessage.warning('请输入用户名')
        return
      }
      if (!loginForm.value.password || loginForm.value.password.length < 6) {
        ElMessage.warning('密码长度不能少于6位')
        return
      }
      try {
        loading.value = true
        const response = await api.post('/user/login', {
          username: loginForm.value.username,
          password: loginForm.value.password
        })
        if (response.data.status_code === 1000) {
          localStorage.setItem('token', response.data.token)
          ElMessage.success('登录成功')
          router.push('/menu')
        } else {
          ElMessage.error(response.data.status_msg || '登录失败')
        }
      } catch (error) {
        console.error('Login error:', error)
        ElMessage.error('登录失败，请重试')
      } finally {
        loading.value = false
      }
    }

    return {
      loginFormRef,
      loading,
      loginForm,
      loginRules,
      handleLogin
    }
  }
}
</script>

<style scoped>
/* ─── Base ──────────────────────────────────────────────────── */
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0f1117;
  font-family: "Inter", "SF Pro Display", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  position: relative;
  overflow: hidden;
}

/* ─── Background grid ───────────────────────────────────────── */
.bg-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(61,126,255,.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(61,126,255,.04) 1px, transparent 1px);
  background-size: 52px 52px;
  pointer-events: none;
}

/* ─── Ambient blobs ─────────────────────────────────────────── */
.blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  pointer-events: none;
  opacity: .18;
}
.blob-1 {
  width: 520px;
  height: 520px;
  background: radial-gradient(circle, #3d7eff 0%, transparent 70%);
  top: -160px;
  right: -120px;
  animation: blobDrift 18s ease-in-out infinite alternate;
}
.blob-2 {
  width: 380px;
  height: 380px;
  background: radial-gradient(circle, #5edfff 0%, transparent 70%);
  bottom: -100px;
  left: -80px;
  animation: blobDrift 24s ease-in-out infinite alternate-reverse;
}
@keyframes blobDrift {
  from { transform: translate(0, 0) scale(1); }
  to   { transform: translate(30px, 20px) scale(1.08); }
}

/* ─── Card ──────────────────────────────────────────────────── */
.login-card {
  position: relative;
  z-index: 10;
  width: 400px;
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

/* neon top line */
.login-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 60%;
  height: 1px;
  background: linear-gradient(90deg, transparent, #3d7eff, #5edfff, #3d7eff, transparent);
  border-radius: 1px;
}

/* ─── Brand ─────────────────────────────────────────────────── */
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
  width: 40px;
  height: 40px;
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

/* ─── Headings ──────────────────────────────────────────────── */
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

/* ─── Form ──────────────────────────────────────────────────── */
.form { display: flex; flex-direction: column; gap: 18px; }

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
}
.field-input::placeholder { color: #3d4a66; }
.field-input:focus {
  border-color: rgba(61,126,255,.55);
  background: rgba(61,126,255,.05);
  box-shadow: 0 0 0 3px rgba(61,126,255,.1);
}

/* ─── Submit button ─────────────────────────────────────────── */
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
.submit-btn:active:not(:disabled) { transform: translateY(0); }
.submit-btn:disabled {
  background: rgba(255,255,255,.07);
  color: #3d4a66;
  box-shadow: none;
  cursor: not-allowed;
}

/* loading dots */
.loading-dots {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.loading-dots span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(255,255,255,.7);
  animation: dot .9s infinite;
}
.loading-dots span:nth-child(2) { animation-delay: .15s; }
.loading-dots span:nth-child(3) { animation-delay: .3s; }
@keyframes dot {
  0%,80%,100% { transform: scale(.6); opacity: .4; }
  40%          { transform: scale(1);  opacity: 1; }
}

/* ─── Register link ─────────────────────────────────────────── */
.register-link {
  margin-top: 22px;
  text-align: center;
  font-size: 13px;
  color: #3d4a66;
}
.register-link a {
  color: #6aa3ff;
  cursor: pointer;
  font-weight: 500;
  transition: color .15s;
  text-decoration: none;
}
.register-link a:hover { color: #90bcff; }
</style>
