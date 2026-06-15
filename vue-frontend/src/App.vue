<template>
  <div id="app">
    <router-view v-slot="{ Component }">
      <transition name="page" mode="out-in">
        <component :is="Component" />
      </transition>
    </router-view>
  </div>
</template>

<script>
export default {
  name: 'App'
}
</script>

<style>
*, *::before, *::after {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  height: 100%;
  background: #0f1117;
  font-family: "Inter", "SF Pro Display", -apple-system, BlinkMacSystemFont,
    "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #e8eaf0;
}

#app { height: 100%; }

/* ── page transitions ── */
.page-enter-active,
.page-leave-active {
  transition: opacity .25s ease, transform .25s ease;
}
.page-enter-from { opacity: 0; transform: translateY(10px); }
.page-leave-to   { opacity: 0; transform: translateY(-10px); }

/* ── global scrollbar ── */
::-webkit-scrollbar { width: 4px; height: 4px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,.1);
  border-radius: 4px;
}
::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,.18); }

/* ── Element Plus dark overrides ── */
.el-message {
  background: #1e2535 !important;
  border: 1px solid rgba(255,255,255,.08) !important;
  box-shadow: 0 8px 32px rgba(0,0,0,.5) !important;
  border-radius: 10px !important;
}
.el-message__content { color: #c8cfe0 !important; }

.el-message-box {
  background: #161b27 !important;
  border: 1px solid rgba(255,255,255,.08) !important;
  border-radius: 16px !important;
  box-shadow: 0 24px 60px rgba(0,0,0,.6) !important;
}
.el-message-box__title { color: #e8eaf0 !important; }
.el-message-box__message p { color: #8892aa !important; }
.el-message-box__input .el-input__inner {
  background: rgba(255,255,255,.05) !important;
  border-color: rgba(255,255,255,.1) !important;
  color: #e8eaf0 !important;
}
.el-overlay { background: rgba(0,0,0,.6) !important; }
</style>
