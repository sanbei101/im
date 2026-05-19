import ws from 'k6/ws';
import { check } from 'k6';
import { Trend, Counter } from 'k6/metrics';
import { v7 as uuidv7 } from 'https://unpkg.com/uuid@14.0.0/dist/index.js';
import http from 'k6/http';
const SingleRoomNum = 500;
const GroupRoom = [100, 100];
const groupTotal = GroupRoom.reduce((sum, val) => sum + val, 0);
const VU_NUM = SingleRoomNum * 2 + groupTotal;
const DURATION = '30s';

/**
 * @typedef {Object} BenchMockUserInfo
 * @property {string} user_id
 * @property {string} username
 * @property {string} token
 */

/**
 * @typedef {Object} SingleRoomResp
 * @property {string} room_id
 * @property {BenchMockUserInfo[]} users
 */

/**
 * @typedef {Object} GroupRoomResp
 * @property {string} room_id
 * @property {number} room_size
 * @property {BenchMockUserInfo[]} users
 */

/**
 * @typedef {Object} BatchMockResp
 * @property {SingleRoomResp[]} single_rooms
 * @property {GroupRoomResp[]} group_rooms
 * @property {number} total_user_num
 */

/**
 * @typedef {Object} VuConfig
 * @property {BenchMockUserInfo} user
 * @property {string} room_id
 * @property {"single"|"group"} type
 */

export const wsMsgLatency = new Trend('ws_msg_latency', true);
export const wsMsgUnmatched = new Counter('ws_msg_unmatched');
export const options = {
  vus: VU_NUM,
  duration: DURATION,
};

const ServerIP = __ENV.TARGET_HOST;
const API_BASE = `http://${ServerIP}:8801`;
const WS_URL = `ws://${ServerIP}:8800/ws`;

const api = {
  /**
   * POST helper
   * @param {string} path
   * @param {any} payload
   * @param {string} token
   * @returns {{status:number, body:string, json:()=>any}}
   */
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

  /**
   * Create bench mock
   * @param {Object} payload
   * @returns {BatchMockResp|null}
   */
  createBenchMock: (payload) => {
    const res = api.post('/api/v1/bench/mock', payload, '');
    if (res.status !== 201) {
      console.error(`Create bench mock failed: ${res.status} ${res.body}`);
      return null;
    }
    return res.json();
  }
};

/**
 * Flatten BatchMockResp into per-VU configs
 * @param {BatchMockResp} batchRes
 * @returns {VuConfig[]}
 */
function flattenBenchData(batchRes) {
  /** @type {VuConfig[]} */
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

/**
 * K6 setup: create bench mock and return vuData
 * @returns {{vuData:VuConfig[]}}
 */
export function setup() {
  const payload = {
    single_room_num: SingleRoomNum,
    group_room: GroupRoom,
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

/**
 * Default VU function
 * @param {{vuData:VuConfig[]}} data
 */
export default function (data) {
  const vuIndex = __VU - 1;
  const myConfig = data.vuData[vuIndex];
  if (!myConfig || !myConfig.user) return;

  /** @type {Map<string, number>} */
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
      }, 500);
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