<template>
  <div class="img-container">
    <!-- Sidebar -->
    <div class="sidebar">
      <div class="sidebar-header">
        <div class="brand-icon">
          <svg width="20" height="20" viewBox="0 0 28 28" fill="none">
            <path d="M14 2L26 8.5V19.5L14 26L2 19.5V8.5L14 2Z" stroke="#3d7eff" stroke-width="1.5" fill="rgba(61,126,255,.12)"/>
            <circle cx="14" cy="14" r="4" fill="#5edfff" opacity=".9"/>
          </svg>
        </div>
        <span class="sidebar-title">图像识别</span>
      </div>
      <ul class="sidebar-list">
        <li class="sidebar-item active">
          <svg width="14" height="14" viewBox="0 0 32 32" fill="none" style="flex-shrink:0">
            <rect x="3" y="5" width="26" height="22" rx="3" stroke="currentColor" stroke-width="1.8" fill="none"/>
            <circle cx="11" cy="13" r="3" stroke="currentColor" stroke-width="1.8"/>
            <path d="M3 22L10 15L16 21L21 16L29 24" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/>
          </svg>
          多模态视觉助手
        </li>
      </ul>
    </div>

    <!-- Main -->
    <div class="main-section">
      <!-- Top bar -->
      <div class="top-bar">
        <button class="back-btn" @click="$router.push('/menu')">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none">
            <path d="M10 3L5 8L10 13" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          返回
        </button>
        <span class="top-title">AI 多模态视觉助手</span>
      </div>

      <!-- Messages -->
      <div class="messages" ref="chatContainerRef">
        <div v-if="!messages.length" class="empty-state">
          <div class="empty-icon">
            <svg width="40" height="40" viewBox="0 0 32 32" fill="none">
              <rect x="3" y="5" width="26" height="22" rx="3" stroke="#3d4a66" stroke-width="1.5" fill="none"/>
              <circle cx="11" cy="13" r="3" stroke="#3d4a66" stroke-width="1.5"/>
              <path d="M3 22L10 15L16 21L21 16L29 24" stroke="#3d4a66" stroke-width="1.5" stroke-linejoin="round"/>
            </svg>
          </div>
          <p>上传一张图片，AI 将多模态理解并描述内容</p>
          <p class="empty-hint">你也可以输入自定义问题，比如：图片中有哪些文字？</p>
        </div>

        <div
          v-for="(message, index) in messages"
          :key="index"
          :class="['message', message.role === 'user' ? 'user-msg' : 'ai-msg']"
        >
          <div class="msg-label">{{ message.role === 'user' ? '你' : 'AI' }}</div>
          <div class="msg-body">
            <img v-if="message.imageUrl" :src="message.imageUrl" alt="uploaded" class="preview-img" />
            <span>{{ message.content }}</span>
            <span v-if="message.meta && message.meta.status === 'streaming'" class="streaming-dot">▊</span>
          </div>
        </div>
      </div>

      <!-- Question input -->
      <div v-if="selectedFile" class="question-area">
        <input
          v-model="question"
          class="question-input"
          placeholder="想问 AI 关于这张图片什么？（选填，默认自动描述）"
          @keydown.enter.exact.prevent="handleSubmit"
        />
      </div>

      <!-- Input area -->
      <div class="input-area">
        <form @submit.prevent="handleSubmit" class="upload-form">
          <label class="file-label" :class="{ 'has-file': selectedFile }">
            <input
              ref="fileInputRef"
              type="file"
              accept="image/*"
              class="file-input"
              @change="handleFileSelect"
            />
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
              <path d="M12 15V3M12 3L8 7M12 3L16 7" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M3 15V18C3 19.1 3.9 20 5 20H19C20.1 20 21 19.1 21 18V15" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
            </svg>
            <span>{{ selectedFile ? selectedFile.name : '选择图片' }}</span>
          </label>
          <button type="submit" class="submit-btn" :disabled="!selectedFile || streaming">
            {{ streaming ? '识别中...' : '识 别' }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, nextTick, onUnmounted } from 'vue'

export default {
  name: 'ImageRecognition',
  setup() {
    const messages = ref([])
    const selectedFile = ref(null)
    const question = ref('')
    const streaming = ref(false)
    const fileInputRef = ref(null)
    const chatContainerRef = ref(null)
    const abortController = ref(null)

    const handleFileSelect = (event) => {
      selectedFile.value = event.target.files[0] || null
    }

    const handleSubmit = async () => {
      if (!selectedFile.value || streaming.value) return

      const file = selectedFile.value
      const imageUrl = URL.createObjectURL(file)
      const currentQuestion = question.value.trim()

      messages.value.push({
        role: 'user',
        content: currentQuestion ? `问题：${currentQuestion}` : file.name,
        imageUrl
      })
      await nextTick()
      scrollToBottom()

      streaming.value = true
      const aiMsgIdx = messages.value.length
      messages.value.push({
        role: 'assistant',
        content: '',
        meta: { status: 'streaming' }
      })

      const formData = new FormData()
      formData.append('image', file)
      formData.append('question', currentQuestion)

      const controller = new AbortController()
      abortController.value = controller
      const token = localStorage.getItem('token') || ''

      try {
        const res = await fetch('/api/image/recognize-stream', {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
          body: formData,
          signal: controller.signal
        })
        if (!res.ok) throw new Error(`HTTP ${res.status}`)

        const reader = res.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''

        // eslint-disable-next-line no-constant-condition
        while (true) {
          const { done, value } = await reader.read()
          if (done) break
          buffer += decoder.decode(value, { stream: true })
          const lines = buffer.split('\n')
          buffer = lines.pop() || ''

          for (const line of lines) {
            const trimmed = line.trim()
            if (!trimmed) continue
            if (trimmed.startsWith('data:')) {
              const data = trimmed.slice(5).trim()
              if (data === '[DONE]') {
                messages.value[aiMsgIdx].meta = { status: 'done' }
              } else {
                messages.value[aiMsgIdx].content += data
              }
              messages.value = [...messages.value]
              await nextTick()
              scrollToBottom()
            }
          }
        }
      } catch (err) {
        if (err.name !== 'AbortError') {
          messages.value[aiMsgIdx].content += `\n\n连接失败：${err.message}`
        }
        messages.value[aiMsgIdx].meta = { status: 'error' }
      } finally {
        streaming.value = false
        abortController.value = null
        URL.revokeObjectURL(imageUrl)
        await nextTick()
        scrollToBottom()
        selectedFile.value = null
        question.value = ''
        if (fileInputRef.value) fileInputRef.value.value = ''
      }
    }

    const scrollToBottom = () => {
      if (chatContainerRef.value) {
        chatContainerRef.value.scrollTop = chatContainerRef.value.scrollHeight
      }
    }

    onUnmounted(() => {
      if (abortController.value) abortController.value.abort()
    })

    return { messages, selectedFile, question, streaming, fileInputRef, chatContainerRef, handleFileSelect, handleSubmit }
  }
}
</script>

<style scoped>
.img-container {
  height: 100vh; display: flex; background: #0f1117;
  font-family: "Inter", "SF Pro Display", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  color: #e8eaf0; overflow: hidden;
}
/* Sidebar */
.sidebar { width: 220px; flex-shrink: 0; height: 100vh; display: flex; flex-direction: column; background: #161b27; border-right: 1px solid rgba(255,255,255,.06); }
.sidebar-header { display: flex; align-items: center; gap: 10px; padding: 20px 16px 16px; border-bottom: 1px solid rgba(255,255,255,.05); }
.brand-icon { display: flex; align-items: center; justify-content: center; width: 32px; height: 32px; background: rgba(61,126,255,.1); border: 1px solid rgba(61,126,255,.18); border-radius: 8px; }
.sidebar-title { font-size: 13px; font-weight: 600; letter-spacing: .06em; text-transform: uppercase; color: #5a6480; }
.sidebar-list { list-style: none; padding: 8px 8px 0; margin: 0; }
.sidebar-item { display: flex; align-items: center; gap: 8px; padding: 10px 12px; border-radius: 8px; font-size: 13.5px; color: #8892aa; cursor: pointer; transition: background .15s; }
.sidebar-item.active { background: rgba(94,223,255,.1); color: #5edfff; font-weight: 500; }
/* Main */
.main-section { flex: 1; display: flex; flex-direction: column; min-width: 0; min-height: 0; overflow: hidden; }
.top-bar { display: flex; align-items: center; gap: 14px; padding: 12px 24px; background: rgba(22,27,39,.9); backdrop-filter: blur(12px); border-bottom: 1px solid rgba(255,255,255,.06); flex-shrink: 0; }
.back-btn { display: flex; align-items: center; gap: 6px; padding: 6px 12px; background: transparent; border: 1px solid rgba(255,255,255,.1); color: #8892aa; border-radius: 8px; font-size: 12px; font-weight: 500; font-family: inherit; cursor: pointer; transition: all .15s; }
.back-btn:hover { border-color: rgba(255,255,255,.2); color: #c8cfe0; background: rgba(255,255,255,.05); }
.top-title { font-size: 14px; font-weight: 600; color: #8892aa; }
/* Messages */
.messages { flex: 1; min-height: 0; overflow-y: auto; padding: 28px 32px; display: flex; flex-direction: column; gap: 16px; scrollbar-width: thin; scrollbar-color: rgba(255,255,255,.06) transparent; }
.messages::-webkit-scrollbar { width: 4px; }
.messages::-webkit-scrollbar-thumb { background: rgba(255,255,255,.07); border-radius: 4px; }
.empty-state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: #3d4a66; font-size: 13.5px; }
.empty-hint { font-size: 12px; color: #3d4a66; opacity: .7; }
.empty-icon { width: 72px; height: 72px; background: rgba(255,255,255,.03); border: 1px solid rgba(255,255,255,.06); border-radius: 18px; display: flex; align-items: center; justify-content: center; }
@keyframes msgIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }
.message { max-width: 72%; animation: msgIn .22s ease-out; display: flex; flex-direction: column; gap: 6px; }
.msg-label { font-size: 11px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; color: #3d4a66; }
.user-msg { align-self: flex-end; }
.ai-msg { align-self: flex-start; }
.ai-msg .msg-label { color: rgba(94,223,255,.5); }
.msg-body { padding: 12px 16px; border-radius: 14px; font-size: 14.5px; line-height: 1.6; word-wrap: break-word; white-space: pre-wrap; }
.user-msg .msg-body { background: rgba(61,126,255,.2); color: #b8d0ff; border: 1px solid rgba(61,126,255,.3); display: flex; flex-direction: column; gap: 10px; }
.ai-msg .msg-body { background: #161b27; color: #c8cfe0; border: 1px solid rgba(255,255,255,.07); box-shadow: 0 2px 16px rgba(0,0,0,.3); position: relative; padding-left: 20px; }
.ai-msg .msg-body::before { content: ''; position: absolute; left: 0; top: 12px; bottom: 12px; width: 2px; border-radius: 2px; background: linear-gradient(180deg, #5edfff 0%, #3d7eff 100%); }
.preview-img { max-width: 200px; border-radius: 10px; display: block; border: 1px solid rgba(255,255,255,.1); }
.streaming-dot { display: inline; color: #3d7eff; animation: blink .8s infinite; }
@keyframes blink { 0%,100%{opacity:1} 50%{opacity:.2} }
/* Question */
.question-area { padding: 0 24px; flex-shrink: 0; }
.question-input { width: 100%; padding: 10px 14px; border: 1px solid rgba(255,255,255,.09); border-radius: 10px; background: rgba(255,255,255,.04); color: #e8eaf0; font-size: 13.5px; font-family: inherit; outline: none; transition: border-color .18s, box-shadow .18s; box-sizing: border-box; }
.question-input::placeholder { color: #3d4a66; }
.question-input:focus { border-color: rgba(61,126,255,.5); box-shadow: 0 0 0 3px rgba(61,126,255,.08); }
/* Input */
.input-area { padding: 16px 24px; background: #161b27; border-top: 1px solid rgba(255,255,255,.06); flex-shrink: 0; }
.upload-form { display: flex; gap: 10px; align-items: center; }
.file-label { flex: 1; display: flex; align-items: center; gap: 10px; padding: 11px 16px; background: rgba(255,255,255,.04); border: 1px dashed rgba(255,255,255,.12); border-radius: 10px; color: #3d4a66; font-size: 13.5px; cursor: pointer; transition: all .18s; overflow: hidden; }
.file-label:hover { border-color: rgba(61,126,255,.4); color: #6aa3ff; background: rgba(61,126,255,.05); }
.file-label.has-file { border-color: rgba(94,223,255,.3); color: #5edfff; border-style: solid; background: rgba(94,223,255,.05); }
.file-label span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; min-width: 0; }
.file-input { position: absolute; width: 1px; height: 1px; opacity: 0; pointer-events: none; }
.submit-btn { flex-shrink: 0; padding: 11px 22px; background: #3d7eff; color: #fff; font-size: 14px; font-weight: 700; letter-spacing: .05em; border: none; border-radius: 10px; cursor: pointer; font-family: inherit; box-shadow: 0 4px 16px rgba(61,126,255,.3); transition: background .15s, transform .15s, box-shadow .15s; }
.submit-btn:hover:not(:disabled) { background: #5590ff; box-shadow: 0 6px 20px rgba(61,126,255,.4); transform: translateY(-1px); }
.submit-btn:disabled { background: rgba(255,255,255,.07); color: #3d4a66; box-shadow: none; cursor: not-allowed; }
</style>
