import ws from 'k6/ws';
import { check } from 'k6';
import { v7 as uuidv7 } from 'https://unpkg.com/uuid@14.0.0/dist/index.js';
import http from 'k6/http';

export const options = {
  vus: 500,
  duration: '30s',
};

const API_BASE = 'http://localhost:8801';
const WS_URL = 'ws://localhost:8800/ws';

export function setup() {
  const userCount = 500;
  const payload = JSON.stringify({ count: userCount });
  const res = http.post(`${API_BASE}/api/v1/users/batch`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  const users = res.json('users');
  if (!users || users.length < userCount) {
    throw new Error(`Failed to get ${userCount} users`);
  }

  console.log(`Created ${users.length} users`);

  const vuData = new Array(userCount);

  for (let i = 0; i < 300; i += 2) {
    const res = http.post(
      `${API_BASE}/api/v1/rooms`,
      JSON.stringify({ user_id_1: users[i].user_id, user_id_2: users[i + 1].user_id }),
      { headers: { 'Content-Type': 'application/json' } }
    );
    const roomId = res.json('room_id');
    if (!roomId) {
      console.error(`Single creation failed: ${res.status} ${res.body}`);
    }
    vuData[i] = { user: users[i], room_id: roomId, type: 'single' };
    vuData[i + 1] = { user: users[i + 1], room_id: roomId, type: 'single' };
  }

  const group1Users = users.slice(300, 400);
  const resG1 = http.post(
    `${API_BASE}/api/v1/rooms/group`,
    JSON.stringify({ name: `Group 1`, member_ids: group1Users.map(u => u.user_id) }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  const roomIdG1 = resG1.json('room_id');
  if (!roomIdG1) console.error(`Group 1 creation failed: ${resG1.status} ${resG1.body}`);
  
  for (let i = 300; i < 400; i++) {
    vuData[i] = { user: users[i], room_id: roomIdG1, type: 'group' };
  }

  const group2Users = users.slice(400, 500);
  const resG2 = http.post(
    `${API_BASE}/api/v1/rooms/group`,
    JSON.stringify({ name: `Group 2`, member_ids: group2Users.map(u => u.user_id) }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  const roomIdG2 = resG2.json('room_id');
  if (!roomIdG2) console.error(`Group 2 creation failed: ${resG2.status} ${resG2.body}`);

  for (let i = 400; i < 500; i++) {
    vuData[i] = { user: users[i], room_id: roomIdG2, type: 'group' };
  }

  console.log(`Setup complete: 150 single rooms, 2 group rooms.`);
  
  return { vuData };
}

export default function (data) {
  // k6 中的 __VU 是从 1 开始的，而数组是从 0 开始的
  const vuIndex = __VU - 1;
  const myConfig = data.vuData[vuIndex];

  // 安全检查：如果数据未就绪或溢出则直接退出
  if (!myConfig || !myConfig.user) {
    return;
  }

  const REQ_HEADERS = {
    'Authorization': myConfig.user.token,
  };

  const res = ws.connect(WS_URL, { headers: REQ_HEADERS }, (socket) => {
    socket.on('open', () => {
      // 频率修改：每隔 0.5s (500ms) 发送一条消息
      socket.setInterval(() => {
        const message = {
          client_msg_id: uuidv7(),
          room_id: myConfig.room_id,
          msg_type: 'text',
          payload: {
            text: `[VU${__VU}] ${myConfig.type} msg`,
          },
          ext: {},
        };

        socket.send(JSON.stringify(message));
      }, 500); 

      socket.setTimeout(() => {
        socket.close();
      }, 30000);
    });

    socket.on('error', (e) => console.error(`[VU ${__VU}] Error:`, e.error()));
  });

  check(res, { 'handshake success 101': (r) => r && r.status === 101 });
}