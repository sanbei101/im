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

  batchCreateUsers: (count) => {
    const res = api.post('/api/v1/users/batch', { count }, '');
    const users = res.json('users');
    if (!users || users.length < count) {
      throw new Error(`Failed to get ${count} users. Status: ${res.status}`);
    }
    return users;
  },

  batchCreateRooms: (singleRooms, groupRooms, token) => {
    const payload = {
      single_rooms: singleRooms || [],
      group_rooms: groupRooms || [],
    };
    const res = api.post('/api/v1/rooms/batch', payload, token);
    if (res.status !== 201) {
      console.error(`Batch create rooms failed: ${res.status} ${res.body}`);
      return null;
    }
    return res.json();
  }
};

export function setup() {
  const userCount = 500;
  const users = api.batchCreateUsers(userCount);
  console.log(`Created ${users.length} users`);

  const tokens = users.map(u => u.token);

  const singleRooms = [];
  for (let i = 0; i < 300; i += 2) {
    singleRooms.push({ user_id_1: users[i].user_id, user_id_2: users[i + 1].user_id });
  }

  const groupRooms = [
    { name: 'Group 1', member_ids: users.slice(300, 400).map(u => u.user_id) },
    { name: 'Group 2', member_ids: users.slice(400, 500).map(u => u.user_id) },
  ];

  const batchRes = api.batchCreateRooms(singleRooms, groupRooms, tokens[0]);
  if (!batchRes) {
    throw new Error('Batch create rooms failed');
  }

  const vuData = new Array(userCount);

  for (let i = 0; i < 300; i += 2) {
    const roomId = batchRes.single_rooms[i / 2].room_id;
    vuData[i] = { user: users[i], room_id: roomId, type: 'single' };
    vuData[i + 1] = { user: users[i + 1], room_id: roomId, type: 'single' };
  }

  const roomIdG1 = batchRes.group_rooms[0].room_id;
  for (let i = 300; i < 400; i++) {
    vuData[i] = { user: users[i], room_id: roomIdG1, type: 'group' };
  }

  const roomIdG2 = batchRes.group_rooms[1].room_id;
  for (let i = 400; i < 500; i++) {
    vuData[i] = { user: users[i], room_id: roomIdG2, type: 'group' };
  }

  console.log(`Setup complete: ${batchRes.single_rooms.length} single rooms, ${batchRes.group_rooms.length} group rooms.`);
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