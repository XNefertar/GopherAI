#!/usr/bin/env bash
# ============================================================
# GopherAI 环境变量加载脚本
# 用法: source load_env.sh        # 加载本地 .env.local
# 用法: source load_env.sh example  # 加载模板让你看到需要填哪些
# ============================================================

MODE="${1:-local}"

if [ "$MODE" = "local" ]; then
  ENV_FILE=".env.local"
elif [ "$MODE" = "example" ]; then
  ENV_FILE=".env.example"
else
  echo "Usage: source load_env.sh        (load .env.local)"
  echo "       source load_env.sh example (view .env.example)"
  return 1
fi

if [ ! -f "$ENV_FILE" ]; then
  echo "[load_env] ERROR: $ENV_FILE not found!"
  echo "[load_env] Run: cp .env.example .env.local  then edit .env.local"
  return 1
fi

set -a  # 自动 export 所有变量
source "$ENV_FILE"
set +a

echo "[load_env] ✅  loaded variables from $ENV_FILE"
echo "[load_env]     OPENAI_API_KEY=$(echo ${OPENAI_API_KEY:0:8}...${OPENAI_API_KEY: -4})"
echo "[load_env]     OPENAI_MODEL_NAME=$OPENAI_MODEL_NAME"
echo "[load_env]     OPENAI_BASE_URL=$OPENAI_BASE_URL"
echo "[load_env]     TITLE_MODEL_NAME=$TITLE_MODEL_NAME"
echo "[load_env]     RAG_EMBEDDING_API_KEY=$(echo ${RAG_EMBEDDING_API_KEY:0:8}...${RAG_EMBEDDING_API_KEY: -4})"
