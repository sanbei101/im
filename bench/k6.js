import ws from 'k6/ws';
import { check } from 'k6';
import { Trend, Counter } from 'k6/metrics';
import { v7 as uuidv7 } from 'https://unpkg.com/uuid@14.0.0/dist/index.js';
import http from 'k6/http';

export const wsMsgLatency = new Trend('ws_msg_latency', true);
export const wsMsgUnmatched = new Counter('ws_msg_unmatched');
export const options = {
  vus: 500,
  duration: '30s',
};

const ServerIP = __ENV.TARGET_HOST;
const API_BASE = `http://${ServerIP}:8801`;
const WS_URL = `ws://${ServerIP}:8800/ws`;

const api = {
  post: (path, payload, token) => {
    const url = `${API_BASE}${path}`;
    const params = {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
    };
    return http.post(url, JSON.stringify(payload), params);
  },

  createBenchMock: (payload) => {
    const res = api.post('/api/v1/bench/mock', payload, '');
    if (res.status !== 201) {
      console.error(`Create bench mock failed: ${res.status} ${res.body}`);
      return null;
    }
    return res.json();
  }
};

function flattenBenchData(batchRes) {
  const vuData = [];

  for (const room of batchRes.single_rooms || []) {
    for (const user of room.users || []) {
      vuData.push({
        user,
        room_id: room.room_id,
        type: 'single',
      });
    }
  }

  for (const room of batchRes.group_rooms || []) {
    for (const user of room.users || []) {
      vuData.push({
        user,
        room_id: room.room_id,
        type: 'group',
      });
    }
  }

  return vuData;
}

export function setup() {
  const payload = {
    single_room_num: 500,
    group_room: [100, 100],
  };

  const batchRes = api.createBenchMock(payload);
  if (!batchRes) {
    throw new Error('Create bench mock failed');
  }

  const vuData = flattenBenchData(batchRes);
  if (vuData.length < batchRes.total_user_num) {
    throw new Error(`Expected ${batchRes.total_user_num} users, got ${vuData.length}`);
  }

  console.log(`Setup complete: ${batchRes.single_rooms.length} single rooms, ${batchRes.group_rooms.length} group rooms, ${vuData.length} users.`);
  return { vuData };
}

export default function (data) {
  const vuIndex = __VU - 1;
  const myConfig = data.vuData[vuIndex];
  if (!myConfig || !myConfig.user) return;

  const pending = new Map();

  const res = ws.connect(`${WS_URL}?token=${myConfig.user.token}`, null, (socket) => {
    socket.on('open', () => {
      socket.setInterval(() => {
        const clientMsgId = uuidv7();
        const now = Date.now();

        const message = {
          client_msg_id: clientMsgId,
          room_id: myConfig.room_id,
          msg_type: 'text',
          payload: {
            content: `[VU${__VU}] hello`,
          },
        };

        pending.set(clientMsgId, now);
        socket.send(JSON.stringify(message));
      }, 300);
    });

    socket.on('message', (raw) => {
      let arr;
      try {
        arr = JSON.parse(raw);
      } catch (e) {
        console.error(`[VU ${__VU}] Error parsing message:`, e);
        return;
      }
      for (const msg of arr) {
        const id = msg.client_msg_id;
        if (!id) continue;

        const start = pending.get(id);
        if (!start) continue;

        if (msg.sender_id && msg.sender_id !== myConfig.user.user_id) {
          continue;
        }

        const latency = Date.now() - start;
        wsMsgLatency.add(latency);
        pending.delete(id);
      }
    });

    socket.on('close', () => {
      wsMsgUnmatched.add(pending.size);
      pending.clear();
    });
  });

  check(res, {
    'handshake success 101': (r) => r && r.status === 101,
  });
}