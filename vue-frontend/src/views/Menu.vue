<template>
  <div class="menu-container">
    <div class="bg-grid"></div>
    <div class="blob blob-1"></div>
    <div class="blob blob-2"></div>

    <!-- Header -->
    <header class="header">
      <div class="header-brand">
        <div class="brand-icon">
          <svg width="24" height="24" viewBox="0 0 28 28" fill="none">
            <path d="M14 2L26 8.5V19.5L14 26L2 19.5V8.5L14 2Z" stroke="#3d7eff" stroke-width="1.5" fill="rgba(61,126,255,.12)"/>
            <circle cx="14" cy="14" r="4" fill="#5edfff" opacity=".9"/>
          </svg>
        </div>
        <span class="brand-name">GopherAI</span>
      </div>
      <button class="logout-btn" @click="handleLogout">退出登录</button>
    </header>

    <!-- Main content -->
    <main class="main">
      <div class="page-title">
        <h1>选择功能</h1>
        <p>选择下方功能模块开始使用</p>
      </div>
      <div class="menu-grid">
        <div class="menu-card" @click="$router.push('/ai-chat')">
          <div class="card-icon chat-icon">
            <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
              <path d="M4 6C4 4.9 4.9 4 6 4H26C27.1 4 28 4.9 28 6V20C28 21.1 27.1 22 26 22H18L12 28V22H6C4.9 22 4 21.1 4 20V6Z"
                stroke="#3d7eff" stroke-width="1.5" fill="rgba(61,126,255,.1)" stroke-linejoin="round"/>
              <circle cx="11" cy="13" r="1.5" fill="#6aa3ff"/>
              <circle cx="16" cy="13" r="1.5" fill="#6aa3ff"/>
              <circle cx="21" cy="13" r="1.5" fill="#6aa3ff"/>
            </svg>
          </div>
          <h3>AI 聊天</h3>
          <p>与 AI 进行智能对话，支持知识库问答与多模型切换</p>
          <div class="card-arrow">→</div>
        </div>

        <div class="menu-card" @click="$router.push('/image-recognition')">
          <div class="card-icon img-icon">
            <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
              <rect x="3" y="5" width="26" height="22" rx="3" stroke="#5edfff" stroke-width="1.5" fill="rgba(94,223,255,.08)"/>
              <circle cx="11" cy="13" r="3" stroke="#5edfff" stroke-width="1.5"/>
              <path d="M3 22L10 15L16 21L21 16L29 24" stroke="#5edfff" stroke-width="1.5" stroke-linejoin="round"/>
            </svg>
          </div>
          <h3>图像识别</h3>
          <p>上传图片，AI 自动识别内容并返回分类结果</p>
          <div class="card-arrow">→</div>
        </div>
      </div>
    </main>
  </div>
</template>

<script>
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'

export default {
  name: 'MenuView',
  setup() {
    const router = useRouter()

    const handleLogout = async () => {
      try {
        await ElMessageBox.confirm('确定要退出登录吗？', '退出确认', {
          confirmButtonText: '退出',
          cancelButtonText: '取消',
          type: 'warning'
        })
        localStorage.removeItem('token')
        ElMessage.success('已退出登录')
        router.push('/login')
      } catch {
        // cancelled
      }
    }

    return { handleLogout }
  }
}
</script>

<style scoped>
.menu-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
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
  filter: blur(100px);
  pointer-events: none;
  opacity: .14;
}
.blob-1 {
  width: 600px; height: 600px;
  background: radial-gradient(circle, #3d7eff 0%, transparent 70%);
  top: -200px; right: -150px;
  animation: blobDrift 20s ease-in-out infinite alternate;
}
.blob-2 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, #5edfff 0%, transparent 70%);
  bottom: -150px; left: -100px;
  animation: blobDrift 26s ease-in-out infinite alternate-reverse;
}
@keyframes blobDrift {
  from { transform: translate(0,0) scale(1); }
  to   { transform: translate(40px,30px) scale(1.1); }
}

/* ── Header ── */
.header {
  position: relative;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 40px;
  height: 60px;
  background: rgba(22,27,39,.8);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(255,255,255,.06);
}

.header-brand {
  display: flex;
  align-items: center;
  gap: 10px;
}
.brand-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px; height: 36px;
  background: rgba(61,126,255,.1);
  border: 1px solid rgba(61,126,255,.2);
  border-radius: 9px;
}
.brand-name {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: .04em;
  background: linear-gradient(90deg, #90bcff, #5edfff);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.logout-btn {
  padding: 7px 16px;
  background: rgba(255,80,80,.1);
  color: #ff7070;
  border: 1px solid rgba(255,80,80,.2);
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  transition: all .15s;
}
.logout-btn:hover {
  background: rgba(255,80,80,.2);
  border-color: rgba(255,80,80,.4);
}

/* ── Main ── */
.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 40px;
  position: relative;
  z-index: 1;
}

.page-title {
  text-align: center;
  margin-bottom: 48px;
  animation: fadeUp .6s ease-out;
}
.page-title h1 {
  font-size: 32px;
  font-weight: 700;
  color: #e8eaf0;
  letter-spacing: -.02em;
  margin-bottom: 8px;
}
.page-title p {
  font-size: 15px;
  color: #3d4a66;
}

.menu-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 24px;
  max-width: 760px;
  width: 100%;
}

@keyframes fadeUp {
  from { opacity: 0; transform: translateY(20px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* ── Cards ── */
.menu-card {
  position: relative;
  padding: 36px 32px;
  background: #161b27;
  border: 1px solid rgba(255,255,255,.07);
  border-radius: 18px;
  cursor: pointer;
  transition: transform .22s, border-color .22s, box-shadow .22s;
  overflow: hidden;
  animation: fadeUp .6s ease-out both;
}
.menu-card:nth-child(1) { animation-delay: .08s; }
.menu-card:nth-child(2) { animation-delay: .16s; }

/* shimmer on hover */
.menu-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(255,255,255,.03) 0%, transparent 60%);
  opacity: 0;
  transition: opacity .22s;
}
.menu-card:hover::before { opacity: 1; }

.menu-card:hover {
  transform: translateY(-6px);
  border-color: rgba(61,126,255,.25);
  box-shadow: 0 20px 48px rgba(0,0,0,.4), 0 0 0 1px rgba(61,126,255,.1);
}

.card-icon {
  width: 56px; height: 56px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
}
.chat-icon {
  background: rgba(61,126,255,.1);
  border: 1px solid rgba(61,126,255,.18);
}
.img-icon {
  background: rgba(94,223,255,.08);
  border: 1px solid rgba(94,223,255,.16);
}

.menu-card h3 {
  font-size: 18px;
  font-weight: 700;
  color: #e8eaf0;
  margin-bottom: 10px;
  letter-spacing: -.01em;
}
.menu-card p {
  font-size: 13.5px;
  color: #5a6480;
  line-height: 1.6;
  margin-bottom: 24px;
}

.card-arrow {
  font-size: 18px;
  color: #3d4a66;
  transition: transform .2s, color .2s;
}
.menu-card:hover .card-arrow {
  transform: translateX(4px);
  color: #6aa3ff;
}
</style>
