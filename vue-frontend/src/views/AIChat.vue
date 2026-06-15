<template>
  <div class="ai-chat-container">
    <!-- 左侧会话列表 -->
    <div class="session-list">
      <div class="session-list-header">
        <span>会话列表</span>
        <button class="new-chat-btn" @click="createNewSession">＋ 新聊天</button>
      </div>
      <ul class="session-list-ul">
        <li
          v-for="session in sessions"
          :key="session.id"
          :class="['session-item', { active: currentSessionId === session.id }]"
          @click="switchSession(session.id)"
        >
          <span class="session-name">{{ session.name || `会话 ${session.id}` }}</span>
          <button class="session-delete-btn" @click.stop="deleteSession(session.id)" title="删除会话">×</button>
        </li>
      </ul>
    </div>

    <!-- 右侧聊天区域 -->
    <div class="chat-section">
      <div class="top-bar">
        <button class="back-btn" @click="$router.push('/menu')">← 返回</button>
        <button class="sync-btn" @click="syncHistory" :disabled="!currentSessionId || tempSession">同步历史数据</button>
        <label for="modelType">选择模型：</label>
        <select
          id="modelType"
          v-model="selectedModel"
          class="model-select"
          :disabled="modelLoading || !modelOptions.length"
          @change="handleModelChange"
        >
          <option v-if="modelLoading" value="" disabled>模型加载中...</option>
          <option v-else-if="!modelOptions.length" value="" disabled>暂无模型</option>
          <option
            v-for="model in modelOptions"
            :key="model.type"
            :value="model.type"
            :disabled="!model.available"
          >
            {{ model.available ? model.label : `${model.label}（${model.disabledReason || '当前不可用'}）` }}
          </option>
        </select>
        <span v-if="selectedModelMeta" class="model-hint">{{ selectedModelMeta.description }}</span>
        <label for="streamingMode" style="margin-left: 8px;" :title="supportsStreaming ? '' : '当前模型不支持流式响应'">
          <input type="checkbox" id="streamingMode" v-model="isStreaming" :disabled="!supportsStreaming" />
          流式响应
        </label>
        <div class="kb-toolbar">
          <label for="kbSelect">知识库：</label>
          <select id="kbSelect" v-model="selectedKBId" class="kb-select" @change="handleKBChange">
            <option value="">请选择知识库</option>
            <option v-for="kb in kbList" :key="kb.id" :value="kb.id">
              {{ kb.name }}
            </option>
          </select>
          <button class="kb-action-btn" @click="createKnowledgeBase" :disabled="kbLoading">新建知识库</button>
          <button class="kb-refresh-btn" @click="refreshKnowledgeBases" :disabled="kbLoading">刷新知识库</button>
          <button class="upload-btn" @click="triggerFileUpload" :disabled="uploading || !selectedKBId">上传文档(.md/.txt)</button>
        </div>
        <input
          ref="fileInput"
          type="file"
          accept=".md,.txt,text/markdown,text/plain"
          style="display: none"
          @change="handleFileUpload"
        />
      </div>

      <div class="kb-panel">
        <div class="kb-panel-header">
          <span>知识库管理</span>
          <span class="kb-panel-tip">{{ kbPanelTip }}</span>
        </div>
        <div v-if="!kbList.length" class="kb-empty-text">暂无知识库，请先新建知识库。</div>
        <div v-else-if="!selectedKBId" class="kb-empty-text">请选择一个知识库后再上传文档或发起依赖知识库的对话。</div>
        <template v-else>
          <div class="kb-selected-name">当前知识库：{{ selectedKBName }}</div>
          <div class="kb-file-list">
            <span v-if="kbFileLoading" class="kb-empty-text">文件加载中...</span>
            <span v-else-if="!kbFiles.length" class="kb-empty-text">当前知识库暂无文件。</span>
            <div v-else v-for="file in kbFiles" :key="file.id" class="kb-file-item">
              <div>
                <div class="kb-file-name">{{ file.origName }}</div>
                <div class="kb-file-meta">状态：{{ file.status || 'indexed' }} · 分块数：{{ file.chunkCount }}</div>
              </div>
              <button class="kb-file-delete-btn" @click="removeKBFile(file.id)">删除</button>
            </div>
          </div>
        </template>
      </div>

      <div class="chat-messages" ref="messagesRef">
        <div
          v-for="(message, index) in currentMessages"
          :key="index"
          :class="['message', message.role === 'user' ? 'user-message' : 'ai-message']"
        >
          <div class="message-header">
            <b>{{ message.role === 'user' ? '你' : 'AI' }}:</b>
            <button v-if="message.role === 'assistant'" class="tts-btn" @click="playTTS(message.content)">🔊</button>
            <span v-if="message.meta && message.meta.status === 'streaming'" class="streaming-indicator"> ··</span>
          </div>
          <div class="message-content" v-html="renderMarkdown(message.content)"></div>
        </div>
      </div>

      <div class="chat-input">
        <textarea
          v-model="inputMessage"
          placeholder="请输入你的问题..."
          @keydown.enter.exact.prevent="sendMessage"
          :disabled="loading"
          ref="messageInput"
          rows="1"
        ></textarea>
        <button
          type="button"
          :disabled="!inputMessage.trim() || loading"
          @click="sendMessage"
          class="send-btn"
        >
          {{ loading ? '发送中...' : '发送' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script>


import { ref, nextTick, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../utils/api'

export default {
  name: 'AIChat',
  setup() {

    const sessions = ref({})
    const currentSessionId = ref(null)
    const tempSession = ref(false)
    const currentMessages = ref([])
    const inputMessage = ref('')
    const loading = ref(false)
    const messagesRef = ref(null)
    const messageInput = ref(null)
    const selectedModel = ref('')
    const modelOptions = ref([])
    const modelLoading = ref(false)
    const isStreaming = ref(false)
    const uploading = ref(false)
    const fileInput = ref(null)
    const kbList = ref([])
    const selectedKBId = ref('')
    const kbFiles = ref([])
    const kbLoading = ref(false)
    const kbFileLoading = ref(false)

    const selectedModelMeta = computed(() => modelOptions.value.find(model => model.type === selectedModel.value) || null)
    const requiresKB = computed(() => Boolean(selectedModelMeta.value?.requiresKB))
    const supportsStreaming = computed(() => Boolean(selectedModelMeta.value?.supportsStream))
    const kbPanelTip = computed(() => (
      requiresKB.value
        ? '当前模型新会话会绑定当前选中的知识库'
        : '可先维护知识库，切换到知识库问答模型后再使用'
    ))
    const selectedKBName = computed(() => {
      const currentKB = kbList.value.find(kb => kb.id === selectedKBId.value)
      return currentKB ? currentKB.name : '未选择知识库'
    })

    const normalizeKB = (kb) => ({
      id: String(kb?.id || kb?.ID || ''),
      name: kb?.name || kb?.Name || '未命名知识库',
      description: kb?.description || kb?.Description || ''
    })

    const normalizeModelOption = (model) => ({
      type: String(model?.type || ''),
      key: model?.key || '',
      label: model?.label || '未命名模型',
      description: model?.description || '',
      requiresKB: Boolean(model?.requiresKB),
      supportsStream: model?.supportsStream !== false,
      available: model?.available !== false,
      disabledReason: model?.disabledReason || '',
      isDefault: Boolean(model?.isDefault),
      sort: Number(model?.sort ?? 0)
    })

    const pickModelType = (models, preferredType, defaultType) => {
      const normalizedPreferredType = String(preferredType || '')
      const normalizedDefaultType = String(defaultType || '')
      const candidates = [
        models.find(model => model.type === normalizedPreferredType && model.available),
        models.find(model => model.type === normalizedDefaultType && model.available),
        models.find(model => model.isDefault && model.available),
        models.find(model => model.available),
        models[0]
      ]
      return candidates.find(Boolean)?.type || ''
    }

    const handleModelChange = () => {
      if (!supportsStreaming.value) {
        isStreaming.value = false
      }
    }

    const loadModelOptions = async (silent = false) => {
      modelLoading.value = true
      try {
        const response = await api.get('/AI/chat/models')
        if (response.data && response.data.status_code === 1000 && Array.isArray(response.data.models)) {
          const list = response.data.models
            .map(normalizeModelOption)
            .filter(model => model.type)
            .sort((a, b) => a.sort - b.sort)

          modelOptions.value = list
          selectedModel.value = pickModelType(list, selectedModel.value, response.data.defaultModelType)
          handleModelChange()

          if (!silent && !list.some(model => model.available)) {
            ElMessage.warning('当前没有可用模型，请检查后端模型配置')
          }
          return
        }
        throw new Error(response.data?.status_msg || 'load models failed')
      } catch (error) {
        console.error('Load models error:', error)
        modelOptions.value = []
        selectedModel.value = ''
        isStreaming.value = false
        if (!silent) {
          ElMessage.error('加载模型列表失败')
        }
      } finally {
        modelLoading.value = false
      }
    }

    const normalizeKBFile = (file) => ({
      id: String(file?.id || file?.ID || ''),
      origName: file?.origName || file?.OrigName || '未命名文件',
      status: file?.status || file?.Status || '',
      chunkCount: Number(file?.chunkCount ?? file?.ChunkCount ?? 0)
    })

    const loadKBFiles = async (kbID = selectedKBId.value, silent = false) => {
      if (!kbID) {
        kbFiles.value = []
        return
      }
      kbFileLoading.value = true
      try {
        const response = await api.get(`/kb/${kbID}/files`)
        if (response.data && response.data.status_code === 1000 && Array.isArray(response.data.files)) {
          kbFiles.value = response.data.files.map(normalizeKBFile)
          return
        }
        throw new Error(response.data?.status_msg || 'load kb files failed')
      } catch (error) {
        console.error('Load KB files error:', error)
        kbFiles.value = []
        if (!silent) {
          ElMessage.error('加载知识库文件失败')
        }
      } finally {
        kbFileLoading.value = false
      }
    }

    const loadKnowledgeBases = async (silent = false) => {
      kbLoading.value = true
      try {
        const response = await api.get('/kb')
        if (response.data && response.data.status_code === 1000 && Array.isArray(response.data.kbs)) {
          const list = response.data.kbs.map(normalizeKB).filter(kb => kb.id)
          kbList.value = list

          if (!list.length) {
            selectedKBId.value = ''
            kbFiles.value = []
            return
          }

          const hasSelected = list.some(kb => kb.id === selectedKBId.value)
          selectedKBId.value = hasSelected ? selectedKBId.value : list[0].id
          await loadKBFiles(selectedKBId.value, true)
          return
        }
        throw new Error(response.data?.status_msg || 'load kb failed')
      } catch (error) {
        console.error('Load KB error:', error)
        kbList.value = []
        kbFiles.value = []
        selectedKBId.value = ''
        if (!silent) {
          ElMessage.error('加载知识库失败')
        }
      } finally {
        kbLoading.value = false
      }
    }

    const refreshKnowledgeBases = async () => {
      await loadKnowledgeBases()
      if (kbList.value.length) {
        ElMessage.success('知识库已刷新')
      }
    }

    const handleKBChange = async () => {
      await loadKBFiles(selectedKBId.value)
    }

    const createKnowledgeBase = async () => {
      try {
        const { value } = await ElMessageBox.prompt('请输入知识库名称', '新建知识库', {
          confirmButtonText: '创建',
          cancelButtonText: '取消',
          inputPattern: /\S+/,
          inputErrorMessage: '知识库名称不能为空'
        })

        const response = await api.post('/kb', {
          name: value.trim(),
          description: ''
        })

        if (response.data && response.data.status_code === 1000 && response.data.kb) {
          const createdKB = normalizeKB(response.data.kb)
          await loadKnowledgeBases(true)
          if (createdKB.id) {
            selectedKBId.value = createdKB.id
            await loadKBFiles(createdKB.id, true)
          }
          ElMessage.success('知识库创建成功')
          return
        }

        ElMessage.error(response.data?.status_msg || '知识库创建失败')
      } catch (error) {
        if (error !== 'cancel' && error !== 'close') {
          console.error('Create KB error:', error)
          ElMessage.error('知识库创建失败')
        }
      }
    }

    const removeKBFile = async (fileID) => {
      if (!selectedKBId.value || !fileID) return
      try {
        await ElMessageBox.confirm('确认删除该文件吗？', '提示', {
          confirmButtonText: '删除',
          cancelButtonText: '取消',
          type: 'warning'
        })

        const response = await api.delete(`/kb/${selectedKBId.value}/files/${fileID}`)
        if (response.data && response.data.status_code === 1000) {
          ElMessage.success('文件删除成功')
          await loadKBFiles(selectedKBId.value, true)
          return
        }

        ElMessage.error(response.data?.status_msg || '文件删除失败')
      } catch (error) {
        if (error !== 'cancel' && error !== 'close') {
          console.error('Remove KB file error:', error)
          ElMessage.error('文件删除失败')
        }
      }
    }

    const requireKBForNewRAGSession = () => {
      if (!selectedModel.value) {
        ElMessage.warning('请先选择可用模型')
        return false
      }
      if (tempSession.value && requiresKB.value && !selectedKBId.value) {
        ElMessage.warning('请先选择知识库，再发起当前模型会话')
        return false
      }
      return true
    }

    const renderMarkdown = (text) => {
      if (!text && text !== '') return ''
      return String(text)
        .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
        .replace(/\*(.*?)\*/g, '<em>$1</em>')
        .replace(/`(.*?)`/g, '<code>$1</code>')
        .replace(/\n/g, '<br>')
    }

    const playTTS = async (text) => {
      try {
        // 创建TTS任务
        const createResponse = await api.post('/AI/chat/tts', { text })
        if (createResponse.data && createResponse.data.status_code === 1000 && createResponse.data.task_id) {
          const taskId = createResponse.data.task_id
          
          // 先等待5秒钟再开始轮询
          await new Promise(resolve => setTimeout(resolve, 5000))
          
          // 轮询查询任务结果
          const maxAttempts = 30
          const pollInterval = 2000
          let attempts = 0
          
          const pollResult = async () => {
            const queryResponse = await api.get('/AI/chat/tts/query', { params: { task_id: taskId } })
            
            if (queryResponse.data && queryResponse.data.status_code === 1000) {
              const taskStatus = queryResponse.data.task_status
                
              if (taskStatus === 'Success' && queryResponse.data.task_result) {
                // 任务完成，播放音频
                // 后端返回的 task_result 是直接的 URL 字符串
                const audio = new Audio(queryResponse.data.task_result)
                audio.play()
                return true
              } else if (taskStatus === 'Running' ||taskStatus === 'Created' ) {
                // 任务进行中，继续轮询
                attempts++
                if (attempts < maxAttempts) {
                  await new Promise(resolve => setTimeout(resolve, pollInterval))
                  return await pollResult()
                } else {
                  ElMessage.error('语音合成超时')
                  return true
                }
              } else {
                // 其他状态（如失败）
                ElMessage.error('语音合成失败')
                return true
              }
            }
            
            attempts++
            if (attempts < maxAttempts) {
              await new Promise(resolve => setTimeout(resolve, pollInterval))
              return await pollResult()
            } else {
              ElMessage.error('语音合成超时')
              return true
            }
          }
          
          await pollResult()
        } else {
          ElMessage.error('无法创建语音合成任务')
        }
      } catch (error) {
        console.error('TTS error:', error)
        ElMessage.error('请求语音接口失败')
      }
    }

    const loadSessions = async () => {
      try {
        const response = await api.get('/AI/chat/sessions')
        if (response.data && response.data.status_code === 1000 && Array.isArray(response.data.sessions)) {
          const sessionMap = {}
          response.data.sessions.forEach(s => {
            const sid = String(s.sessionId)
            sessionMap[sid] = {
              id: sid,
              name: s.name || `会话 ${sid}`,
              messages: [] // lazy load
            }
          })
          sessions.value = sessionMap
        }
      } catch (error) {
        console.error('Load sessions error:', error)
      }
    }

    const createNewSession = () => {
      currentSessionId.value = 'temp'
      tempSession.value = true
      currentMessages.value = []
      // focus input
      nextTick(() => {
        if (messageInput.value) messageInput.value.focus()
      })
    }

    const switchSession = async (sessionId) => {
      if (!sessionId) return
      currentSessionId.value = String(sessionId)
      tempSession.value = false

      // lazy load history if not present
      if (!sessions.value[sessionId].messages || sessions.value[sessionId].messages.length === 0) {
        try {
          const response = await api.post('/AI/chat/history', { sessionId: currentSessionId.value })
          if (response.data && response.data.status_code === 1000 && Array.isArray(response.data.history)) {
            const messages = response.data.history.map(item => ({
              role: item.is_user ? 'user' : 'assistant',
              content: item.content
            }))
            sessions.value[sessionId].messages = messages
          }
        } catch (err) {
          console.error('Load history error:', err)
        }
      }


      currentMessages.value = [...(sessions.value[sessionId].messages || [])]
      await nextTick()
      scrollToBottom()
    }

    const syncHistory = async () => {
      if (!currentSessionId.value || tempSession.value) {
        ElMessage.warning('请选择已有会话进行同步')
        return
      }
      try {
        const response = await api.post('/AI/chat/history', { sessionId: currentSessionId.value })
        if (response.data && response.data.status_code === 1000 && Array.isArray(response.data.history)) {
          const messages = response.data.history.map(item => ({
            role: item.is_user ? 'user' : 'assistant',
            content: item.content
          }))
          sessions.value[currentSessionId.value].messages = messages
          currentMessages.value = [...messages]
          await nextTick()
          scrollToBottom()
        } else {
          ElMessage.error('无法获取历史数据')
        }
      } catch (err) {
        console.error('Sync history error:', err)
        ElMessage.error('请求历史数据失败')
      }
    }


    const sendMessage = async () => {
      if (!inputMessage.value || !inputMessage.value.trim()) {
        ElMessage.warning('请输入消息内容')
        return
      }

      // 兜底：如果没有任何活跃会话，自动创建一个
      ensureActiveSession()

      if (!requireKBForNewRAGSession()) {
        return
      }

      const userMessage = {
        role: 'user',
        content: inputMessage.value
      }
      const currentInput = inputMessage.value
      inputMessage.value = ''


      currentMessages.value.push(userMessage)
      await nextTick()
      scrollToBottom()

      try {
        loading.value = true
        if (isStreaming.value) {

          await handleStreaming(currentInput)
        } else {

          await handleNormal(currentInput)
        }
      } catch (err) {
        console.error('Send message error:', err)
        ElMessage.error('发送失败，请重试')

        if (!tempSession.value && currentSessionId.value && sessions.value[currentSessionId.value] && sessions.value[currentSessionId.value].messages) {

          const sessionArr = sessions.value[currentSessionId.value].messages
          if (sessionArr && sessionArr.length) sessionArr.pop()
        }
        currentMessages.value.pop()
      } finally {
        if (!isStreaming.value) {
          loading.value = false
        }
        await nextTick()
        scrollToBottom()
      }
    }


    async function handleStreaming(question) {

      const aiMessage = {
        role: 'assistant',
        content: '',
        meta: { status: 'streaming' } // mark streaming
      }


      const aiMessageIndex = currentMessages.value.length
      currentMessages.value.push(aiMessage)

      if (!tempSession.value && currentSessionId.value && sessions.value[currentSessionId.value]) {
        if (!sessions.value[currentSessionId.value].messages) sessions.value[currentSessionId.value].messages = []
        sessions.value[currentSessionId.value].messages.push({ role: 'assistant', content: '' })
      }


      const url = tempSession.value
        ? '/api/AI/chat/send-stream-new-session'  
        : '/api/AI/chat/send-stream'           

      const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token') || ''}`
      }

      const body = tempSession.value
        ? {
            question: question,
            modelType: selectedModel.value,
            ...(requiresKB.value ? { kbID: selectedKBId.value } : {})
          }
        : { question: question, modelType: selectedModel.value, sessionId: currentSessionId.value }

      try {
        // 创建 fetch 连接读取 SSE 流
        const response = await fetch(url, {
          method: 'POST',
          headers,
          body: JSON.stringify(body)
        })

        if (!response.ok) {
          loading.value = false
          throw new Error('Network response was not ok')
        }

        const reader = response.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''

        // 读取流数据
        // eslint-disable-next-line no-constant-condition
        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          const chunk = decoder.decode(value, { stream: true })
          buffer += chunk

          // 按行分割
          const lines = buffer.split('\n')
          buffer = lines.pop() || '' // 保留未完成的行

          for (const line of lines) {
            const trimmedLine = line.trim()
            if (!trimmedLine) continue

            // 处理 SSE 格式：data: <content>
            if (trimmedLine.startsWith('data:')) {
              const data = trimmedLine.slice(5).trim()
              console.log('[SSE] Received:', data) // 调试日志

              if (data === '[DONE]') {
                // 流结束
                console.log('[SSE] Stream done')
                loading.value = false
                currentMessages.value[aiMessageIndex].meta = { status: 'done' }
                currentMessages.value = [...currentMessages.value]
              } else if (data.startsWith('{')) {
                // 尝试解析 JSON（如 sessionId）
                try {
                  const parsed = JSON.parse(data)
                  if (parsed.sessionId) {
                    const newSid = String(parsed.sessionId)
                    console.log('[SSE] Session ID:', newSid)
                    if (tempSession.value) {
                      const initialName = question.length > 20 ? question.slice(0, 20) + '...' : question
                      sessions.value[newSid] = {
                        id: newSid,
                        name: initialName,
                        messages: [...currentMessages.value]
                      }
                      currentSessionId.value = newSid
                      tempSession.value = false
                    }
                  }
                } catch (e) {
                  // 不是 JSON，当作普通文本处理
                  currentMessages.value[aiMessageIndex].content += data
                  console.log('[SSE] Content updated:', currentMessages.value[aiMessageIndex].content.length)
                }
              } else {
                // 普通文本数据，直接追加
                // 使用数组索引直接更新，强制 Vue 响应式系统检测变化
                currentMessages.value[aiMessageIndex].content += data
                console.log('[SSE] Content updated:', currentMessages.value[aiMessageIndex].content.length)
              }

              // 每收到一条数据就立即更新 DOM
              // 强制更新整个数组以触发响应式
              currentMessages.value = [...currentMessages.value]
              
              // 使用 requestAnimationFrame 强制浏览器重排
              await new Promise(resolve => {
                requestAnimationFrame(() => {
                  scrollToBottom()
                  resolve()
                })
              })
            }
          }
        }

        // 流读取完成后的处理
        loading.value = false
        currentMessages.value[aiMessageIndex].meta = { status: 'done' }
        currentMessages.value = [...currentMessages.value]

        // 同步到 sessions 存储
        if (!tempSession.value && currentSessionId.value && sessions.value[currentSessionId.value]) {
          const sessMsgs = sessions.value[currentSessionId.value].messages
          if (Array.isArray(sessMsgs) && sessMsgs.length) {
            const lastIndex = sessMsgs.length - 1
            if (sessMsgs[lastIndex] && sessMsgs[lastIndex].role === 'assistant') {
              sessMsgs[lastIndex].content = currentMessages.value[aiMessageIndex].content
            }
          }
        }
      } catch (err) {
        console.error('Stream error:', err)
        loading.value = false
        currentMessages.value[aiMessageIndex].meta = { status: 'error' }
        currentMessages.value = [...currentMessages.value]
        ElMessage.error('流式传输出错')
      }
    }


    async function handleNormal(question) {
      if (tempSession.value) {
        const response = await api.post('/AI/chat/send-new-session', {
          question: question,
          modelType: selectedModel.value,
          ...(requiresKB.value ? { kbID: selectedKBId.value } : {})
        })
        if (response.data && response.data.status_code === 1000) {
          const sessionId = String(response.data.sessionId)
          const aiMessage = {
            role: 'assistant',
            content: response.data.Information || ''
          }

          sessions.value[sessionId] = {
            id: sessionId,
            name: question.length > 20 ? question.slice(0, 20) + '...' : question,
            messages: [{ role: 'user', content: question }, aiMessage]
          }
          currentSessionId.value = sessionId
          tempSession.value = false
          currentMessages.value = [...sessions.value[sessionId].messages]
        } else {
          ElMessage.error(response.data?.status_msg || '发送失败')

          currentMessages.value.pop()
        }
      } else {

        const sessionMsgs = sessions.value[currentSessionId.value].messages

        sessionMsgs.push({ role: 'user', content: question })

        const response = await api.post('/AI/chat/send', {
          question: question,
          modelType: selectedModel.value,
          sessionId: currentSessionId.value
        })
        if (response.data && response.data.status_code === 1000) {
          const aiMessage = { role: 'assistant', content: response.data.Information || '' }
          sessionMsgs.push(aiMessage)
          currentMessages.value = [...sessionMsgs]
        } else {
          ElMessage.error(response.data?.status_msg || '发送失败')
          sessionMsgs.pop() // rollback
          currentMessages.value.pop()
        }
      }
    }


    const scrollToBottom = () => {
      if (messagesRef.value) {
        try {
          messagesRef.value.scrollTop = messagesRef.value.scrollHeight
        } catch (e) {
          // ignore
        }
      }
    }

    const triggerFileUpload = () => {
      if (!selectedKBId.value) {
        ElMessage.warning('请先选择知识库')
        return
      }
      if (fileInput.value) {
        fileInput.value.click()
      }
    }

    const handleFileUpload = async (event) => {
      const file = event.target.files[0]
      if (!file) return

      const resetFileInput = () => {
        if (fileInput.value) {
          fileInput.value.value = ''
        }
      }

      const fileName = file.name.toLowerCase()
      if (!fileName.endsWith('.md') && !fileName.endsWith('.txt')) {
        ElMessage.error('只允许上传 .md 或 .txt 文件')
        resetFileInput()
        return
      }

      if (!selectedKBId.value) {
        ElMessage.warning('请先选择知识库')
        resetFileInput()
        return
      }

      try {
        uploading.value = true
        const formData = new FormData()
        formData.append('file', file)

        const response = await api.post(`/kb/${selectedKBId.value}/files`, formData, {
          headers: {
            'Content-Type': 'multipart/form-data'
          }
        })

        if (response.data && response.data.status_code === 1000) {
          ElMessage.success('文件上传成功')
          await loadKBFiles(selectedKBId.value, true)
        } else {
          ElMessage.error(response.data?.status_msg || '上传失败')
        }
      } catch (error) {
        console.error('File upload error:', error)
        ElMessage.error('文件上传失败')
      } finally {
        uploading.value = false
        resetFileInput()
      }
    }

    const deleteSession = async (sessionId) => {
      try {
        await ElMessageBox.confirm('确定要删除该会话吗？删除后不可恢复。', '删除确认', {
          confirmButtonText: '删除',
          cancelButtonText: '取消',
          type: 'warning'
        })

        const response = await api.delete('/AI/chat/session', {
          data: { sessionId: String(sessionId) }
        })

        if (response.data && response.data.status_code === 1000) {
          ElMessage.success('会话已删除')
          delete sessions.value[sessionId]

          // 如果删除的是当前会话，切换到其他会话或新建
          if (currentSessionId.value === sessionId) {
            const remaining = Object.keys(sessions.value)
            if (remaining.length > 0) {
              switchSession(remaining[0])
            } else {
              currentSessionId.value = null
              currentMessages.value = []
            }
          }
        } else {
          ElMessage.error(response.data?.status_msg || '删除失败')
        }
      } catch (error) {
        if (error !== 'cancel' && error !== 'close') {
          console.error('Delete session error:', error)
          ElMessage.error('删除会话失败')
        }
      }
    }

    const ensureActiveSession = () => {
      // 如果没有任何会话，自动创建一个临时会话，让用户可以直接开始对话
      if (!currentSessionId.value || !sessions.value[currentSessionId.value]) {
        createNewSession()
      }
    }

    onMounted(async () => {
      await loadSessions()
      const hasSessions = Object.keys(sessions.value).length > 0
      await Promise.all([
        loadModelOptions(),
        loadKnowledgeBases(true)
      ])

      // 没有历史会话时自动进入临时会话模式
      if (!hasSessions) {
        ensureActiveSession()
      }
    })

    return {
      sessions: computed(() => Object.values(sessions.value)),
      currentSessionId,
      tempSession,
      currentMessages,
      inputMessage,
      loading,
      messagesRef,
      messageInput,
      selectedModel,
      selectedModelMeta,
      modelOptions,
      modelLoading,
      isStreaming,
      uploading,
      fileInput,
      kbList,
      selectedKBId,
      selectedKBName,
      kbFiles,
      kbLoading,
      kbFileLoading,
      supportsStreaming,
      kbPanelTip,
      renderMarkdown,
      playTTS,
      createNewSession,
      switchSession,
      syncHistory,
      sendMessage,
      handleModelChange,
      triggerFileUpload,
      handleFileUpload,
      handleKBChange,
      refreshKnowledgeBases,
      createKnowledgeBase,
      removeKBFile,
      deleteSession,
      ensureActiveSession
    }
  }
}
</script>

<style scoped>
/* ─── Design tokens ─────────────────────────────────────────── */
/* bg-base: #0f1117  bg-surface: #161b27  bg-raised: #1e2535   */
/* accent: #3d7eff   accent-dim: rgba(61,126,255,.15)           */
/* neon: #5edfff     border: rgba(255,255,255,.06)              */
/* text-primary: #e8eaf0  text-muted: #5a6480                  */

/* ─── Root ──────────────────────────────────────────────────── */
.ai-chat-container {
  height: 100vh;
  display: flex;
  background: #0f1117;
  font-family: "Inter", "SF Pro Display", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  color: #e8eaf0;
  position: relative;
  overflow: hidden;
}

/* subtle grid overlay */
.ai-chat-container::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(61,126,255,.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(61,126,255,.03) 1px, transparent 1px);
  background-size: 48px 48px;
  pointer-events: none;
  z-index: 0;
}

/* ─── Sidebar ───────────────────────────────────────────────── */
.session-list {
  width: 260px;
  flex-shrink: 0;
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #161b27;
  border-right: 1px solid rgba(255,255,255,.06);
  position: relative;
  z-index: 10;
}

.session-list-header {
  padding: 20px 16px 16px;
  border-bottom: 1px solid rgba(255,255,255,.05);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.session-list-header > span {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: .1em;
  text-transform: uppercase;
  color: #5a6480;
}

.new-chat-btn {
  width: 100%;
  padding: 10px 0;
  cursor: pointer;
  background: rgba(61,126,255,.14);
  color: #6aa3ff;
  border: 1px solid rgba(61,126,255,.28);
  border-radius: 10px;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: .02em;
  transition: background .18s, border-color .18s, color .18s;
}

.new-chat-btn:hover {
  background: rgba(61,126,255,.24);
  border-color: rgba(61,126,255,.5);
  color: #90bcff;
}

.session-list-ul {
  list-style: none;
  padding: 8px 8px 0;
  margin: 0;
  flex: 1;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: rgba(255,255,255,.08) transparent;
}

.session-list-ul::-webkit-scrollbar { width: 4px; }
.session-list-ul::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,.08);
  border-radius: 4px;
}

.session-item {
  padding: 10px 12px;
  cursor: pointer;
  border-radius: 8px;
  margin-bottom: 2px;
  transition: background .15s;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #8892aa;
  font-size: 13.5px;
}

.session-item:hover {
  background: rgba(255,255,255,.05);
  color: #c8cfe0;
}

.session-item.active {
  background: rgba(61,126,255,.16);
  color: #90bcff;
  font-weight: 500;
}

.session-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-delete-btn {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: #3d4a66;
  font-size: 15px;
  line-height: 1;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity .15s, background .15s, color .15s;
}

.session-item:hover .session-delete-btn { opacity: 1; }
.session-item.active .session-delete-btn { color: rgba(144,188,255,.4); }

.session-delete-btn:hover {
  background: rgba(255,80,80,.15);
  color: #ff6b6b;
}

/* ─── Main area ─────────────────────────────────────────────── */
.chat-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  position: relative;
  z-index: 1;
  background: #0f1117;
}

/* ─── Top toolbar ───────────────────────────────────────────── */
.top-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 10px 20px;
  background: rgba(22,27,39,.9);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(255,255,255,.06);
  color: #8892aa;
  font-size: 13px;
}

.top-bar label { color: #5a6480; font-size: 12px; }

.back-btn {
  background: transparent;
  border: 1px solid rgba(255,255,255,.1);
  color: #8892aa;
  padding: 6px 12px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  transition: all .15s;
}
.back-btn:hover {
  border-color: rgba(255,255,255,.2);
  color: #c8cfe0;
  background: rgba(255,255,255,.05);
}

.sync-btn {
  background: rgba(255,255,255,.05);
  color: #6aa3ff;
  padding: 6px 12px;
  border: 1px solid rgba(61,126,255,.2);
  border-radius: 8px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  transition: all .15s;
}
.sync-btn:hover:not(:disabled) {
  background: rgba(61,126,255,.14);
  border-color: rgba(61,126,255,.4);
}
.sync-btn:disabled {
  opacity: .35;
  cursor: not-allowed;
}

.model-select,
.kb-select {
  padding: 5px 9px;
  border: 1px solid rgba(255,255,255,.08);
  border-radius: 8px;
  background: rgba(255,255,255,.04);
  color: #c8cfe0;
  font-size: 12.5px;
  font-weight: 500;
  cursor: pointer;
  transition: border-color .15s;
  outline: none;
}
.model-select:focus, .kb-select:focus {
  border-color: rgba(61,126,255,.5);
}
.model-select option, .kb-select option {
  background: #1e2535;
  color: #c8cfe0;
}
.model-select:disabled { opacity: .4; cursor: not-allowed; }

.model-hint {
  font-size: 11.5px;
  color: #3d4a66;
  max-width: 220px;
  line-height: 1.4;
}

/* streaming checkbox */
#streamingMode { accent-color: #3d7eff; }

.kb-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.kb-action-btn,
.kb-refresh-btn,
.upload-btn,
.kb-file-delete-btn {
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  padding: 6px 11px;
  transition: all .15s;
}

.kb-action-btn {
  background: rgba(94,223,255,.1);
  color: #5edfff;
  border: 1px solid rgba(94,223,255,.2);
}
.kb-action-btn:hover:not(:disabled) {
  background: rgba(94,223,255,.18);
  border-color: rgba(94,223,255,.4);
}

.kb-refresh-btn {
  background: rgba(255,255,255,.04);
  color: #5a6480;
  border: 1px solid rgba(255,255,255,.07);
}
.kb-refresh-btn:hover:not(:disabled) {
  background: rgba(255,255,255,.08);
  color: #8892aa;
}

.upload-btn {
  background: rgba(61,126,255,.12);
  color: #6aa3ff;
  border: 1px solid rgba(61,126,255,.22);
}
.upload-btn:hover:not(:disabled) {
  background: rgba(61,126,255,.22);
  border-color: rgba(61,126,255,.45);
}
.upload-btn:disabled,
.kb-action-btn:disabled,
.kb-refresh-btn:disabled {
  opacity: .35;
  cursor: not-allowed;
}

/* ─── KB panel ──────────────────────────────────────────────── */
.kb-panel {
  margin: 12px 20px 0;
  padding: 14px 16px;
  border-radius: 12px;
  background: #161b27;
  border: 1px solid rgba(255,255,255,.06);
  flex-shrink: 0;
}

.kb-panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: .06em;
  text-transform: uppercase;
  color: #5a6480;
}

.kb-panel-tip { color: #3d4a66; font-size: 11.5px; font-weight: 400; }
.kb-empty-text { color: #3d4a66; font-size: 12.5px; }
.kb-selected-name { color: #8892aa; font-size: 12.5px; margin-bottom: 10px; }

.kb-file-list { display: flex; flex-direction: column; gap: 6px; }

.kb-file-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(255,255,255,.03);
  border: 1px solid rgba(255,255,255,.05);
}

.kb-file-name {
  font-size: 13px;
  font-weight: 500;
  color: #c8cfe0;
  margin-bottom: 2px;
}

.kb-file-meta { color: #3d4a66; font-size: 11.5px; }

.kb-file-delete-btn {
  background: rgba(255,80,80,.1);
  color: #ff7070;
  border: 1px solid rgba(255,80,80,.18);
}
.kb-file-delete-btn:hover {
  background: rgba(255,80,80,.2);
  border-color: rgba(255,80,80,.4);
}

@media (max-width: 960px) {
  .kb-panel-header, .kb-file-item {
    flex-direction: column;
    align-items: flex-start;
  }
}

/* ─── Messages ──────────────────────────────────────────────── */
.chat-messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 28px 32px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  scrollbar-width: thin;
  scrollbar-color: rgba(255,255,255,.06) transparent;
}

.chat-messages::-webkit-scrollbar { width: 4px; }
.chat-messages::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,.07);
  border-radius: 4px;
}

@keyframes msgIn {
  from { opacity: 0; transform: translateY(8px); }
  to   { opacity: 1; transform: translateY(0); }
}

.message {
  max-width: 72%;
  padding: 13px 16px;
  border-radius: 14px;
  line-height: 1.65;
  word-wrap: break-word;
  font-size: 14.5px;
  box-sizing: border-box;
  animation: msgIn .22s ease-out;
}

/* user bubble */
.user-message {
  align-self: flex-end;
  background: rgba(61,126,255,.2);
  color: #b8d0ff;
  border: 1px solid rgba(61,126,255,.3);
}

/* AI bubble */
.ai-message {
  align-self: flex-start;
  background: #161b27;
  color: #c8cfe0;
  border: 1px solid rgba(255,255,255,.07);
  box-shadow: 0 2px 16px rgba(0,0,0,.3);
}

/* neon left accent for AI */
.ai-message::before {
  content: '';
  position: absolute;
  left: 0;
  top: 14px;
  bottom: 14px;
  width: 2px;
  border-radius: 2px;
  background: linear-gradient(180deg, #5edfff 0%, #3d7eff 100%);
}
.ai-message { position: relative; padding-left: 20px; }

.message-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.message-header b {
  font-size: 11.5px;
  font-weight: 700;
  letter-spacing: .06em;
  text-transform: uppercase;
  color: #3d4a66;
}

.user-message .message-header b { color: rgba(144,188,255,.5); }

.tts-btn {
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 11px;
  cursor: pointer;
  background: rgba(94,223,255,.1);
  color: #5edfff;
  border: 1px solid rgba(94,223,255,.2);
  transition: all .15s;
}
.tts-btn:hover {
  background: rgba(94,223,255,.18);
  border-color: rgba(94,223,255,.4);
}

.streaming-indicator {
  display: inline-block;
  color: #3d7eff;
  font-weight: 700;
  letter-spacing: .15em;
  animation: blink 1s infinite;
}
@keyframes blink {
  0%,100% { opacity: 1; }
  50%      { opacity: .3; }
}

.message-content {
  white-space: pre-wrap;
  word-break: break-word;
}

.message-content code {
  background: rgba(94,223,255,.08);
  color: #5edfff;
  padding: 1px 5px;
  border-radius: 4px;
  font-size: 13px;
  font-family: "JetBrains Mono", "Fira Code", monospace;
}

/* ─── Input ─────────────────────────────────────────────────── */
.chat-input {
  padding: 16px 20px;
  background: #161b27;
  border-top: 1px solid rgba(255,255,255,.06);
  position: relative;
  z-index: 1;
  display: flex;
  gap: 10px;
  align-items: flex-end;
}

.chat-input textarea {
  flex: 1;
  resize: none;
  border: 1px solid rgba(255,255,255,.09);
  border-radius: 12px;
  padding: 12px 14px;
  font-size: 14px;
  outline: none;
  background: rgba(255,255,255,.04);
  color: #e8eaf0;
  transition: border-color .18s, box-shadow .18s;
  min-height: 44px;
  max-height: 140px;
  font-family: inherit;
  line-height: 1.6;
}
.chat-input textarea::placeholder { color: #3d4a66; }
.chat-input textarea:focus {
  border-color: rgba(61,126,255,.5);
  box-shadow: 0 0 0 3px rgba(61,126,255,.08);
}

.send-btn {
  flex-shrink: 0;
  padding: 11px 20px;
  border: none;
  border-radius: 12px;
  background: #3d7eff;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background .15s, transform .15s, box-shadow .15s;
  box-shadow: 0 4px 16px rgba(61,126,255,.3);
  white-space: nowrap;
}
.send-btn:hover:not(:disabled) {
  background: #5a93ff;
  box-shadow: 0 6px 20px rgba(61,126,255,.4);
  transform: translateY(-1px);
}
.send-btn:disabled {
  background: rgba(255,255,255,.07);
  color: #3d4a66;
  box-shadow: none;
  cursor: not-allowed;
}
</style>
