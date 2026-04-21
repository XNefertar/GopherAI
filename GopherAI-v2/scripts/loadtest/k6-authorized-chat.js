import http from 'k6/http';
import { check, fail, sleep } from 'k6';

const BASE_URL = __ENV.K6_BASE_URL || 'http://localhost:9090/api/v1';
const USERNAME = __ENV.K6_USERNAME || '';
const PASSWORD = __ENV.K6_PASSWORD || '';
const MODEL_TYPE = __ENV.K6_MODEL_TYPE || '1';
const THINK_TIME_MS = Number(__ENV.K6_THINK_TIME_MS || '300');

const LIST_VUS = Number(__ENV.K6_LIST_VUS || '5');
const LIST_DURATION = __ENV.K6_LIST_DURATION || '30s';
const CHAT_VUS = Number(__ENV.K6_CHAT_VUS || '5');
const CHAT_DURATION = __ENV.K6_CHAT_DURATION || '30s';
const STREAM_VUS = Number(__ENV.K6_STREAM_VUS || '3');
const STREAM_DURATION = __ENV.K6_STREAM_DURATION || '30s';

export const options = {
  scenarios: {
    sessions_list: {
      executor: 'constant-vus',
      exec: 'sessionsList',
      vus: LIST_VUS,
      duration: LIST_DURATION,
    },
    normal_chat_new_session: {
      executor: 'constant-vus',
      exec: 'normalChat',
      vus: CHAT_VUS,
      duration: CHAT_DURATION,
      startTime: '2s',
    },
    stream_chat_new_session: {
      executor: 'constant-vus',
      exec: 'streamChat',
      vus: STREAM_VUS,
      duration: STREAM_DURATION,
      startTime: '4s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<5000'],
    checks: ['rate>0.95'],
  },
};

function jsonHeaders(token) {
  return {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    timeout: '120s',
  };
}

function sleepThinkTime() {
  sleep(THINK_TIME_MS / 1000);
}

export function setup() {
  if (!USERNAME || !PASSWORD) {
    fail('请设置环境变量 K6_USERNAME 和 K6_PASSWORD');
  }

  const payload = JSON.stringify({
    username: USERNAME,
    password: PASSWORD,
  });

  const res = http.post(`${BASE_URL}/user/login`, payload, {
    headers: { 'Content-Type': 'application/json' },
    timeout: '30s',
  });

  const loginOk = check(res, {
    'login http status is 200': (r) => r.status === 200,
    'login business success': (r) => {
      const body = safeJSON(r.body);
      return body && body.status_code === 1000 && !!body.token;
    },
  });

  if (!loginOk) {
    fail(`登录失败，响应体: ${res.body}`);
  }

  const body = JSON.parse(res.body);
  return {
    token: body.token,
  };
}

export function sessionsList(data) {
  const res = http.get(`${BASE_URL}/AI/chat/sessions`, {
    headers: {
      Authorization: `Bearer ${data.token}`,
    },
    timeout: '30s',
  });

  check(res, {
    'sessions status is 200': (r) => r.status === 200,
    'sessions returns success or limiter signal': (r) => {
      const body = safeJSON(r.body);
      return isAcceptableProtectedResponse(r.status, body);
    },
  });

  sleepThinkTime();
}

export function normalChat(data) {
  const question = `压测普通聊天请求，vu=${__VU}, iter=${__ITER}, ts=${Date.now()}`;
  const res = http.post(
    `${BASE_URL}/AI/chat/send-new-session`,
    JSON.stringify({
      question,
      modelType: MODEL_TYPE,
    }),
    jsonHeaders(data.token)
  );

  check(res, {
    'normal chat status is 200': (r) => r.status === 200,
    'normal chat returns success or limiter/circuit signal': (r) => {
      const body = safeJSON(r.body);
      return isAcceptableProtectedResponse(r.status, body);
    },
  });

  sleepThinkTime();
}

export function streamChat(data) {
  const question = `压测流式聊天请求，vu=${__VU}, iter=${__ITER}, ts=${Date.now()}`;
  const res = http.post(
    `${BASE_URL}/AI/chat/send-stream-new-session`,
    JSON.stringify({
      question,
      modelType: MODEL_TYPE,
    }),
    jsonHeaders(data.token)
  );

  check(res, {
    'stream chat http accepted': (r) => r.status === 200 || r.status === 429 || r.status === 503,
    'stream chat returns sse or limiter/circuit signal': (r) => {
      if (r.status === 429 || r.status === 503) {
        return true;
      }
      const body = safeJSON(r.body);
      if (body && isKnownProtectiveCode(body.status_code)) {
        return true;
      }
      return r.body.includes('data:') && (r.body.includes('[DONE]') || r.body.includes('sessionId'));
    },
  });

  sleepThinkTime();
}

function safeJSON(raw) {
  try {
    return JSON.parse(raw);
  } catch (e) {
    return null;
  }
}

function isKnownProtectiveCode(code) {
  return code === 429 || code === 503 || code === 4001;
}

function isAcceptableProtectedResponse(httpStatus, body) {
  if (httpStatus === 429 || httpStatus === 503) {
    return true;
  }
  if (!body) {
    return false;
  }
  return body.status_code === 1000 || isKnownProtectiveCode(body.status_code);
}
