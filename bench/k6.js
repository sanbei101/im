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

const api = {
  post: (path, payload) => {
    const url = `${API_BASE}${path}`;
    const params = { headers: { 'Content-Type': 'application/json' } };
    return http.post(url, JSON.stringify(payload), params);
  },

  createUsers: (count) => {
    const res = api.post('/api/v1/users/batch', { count });
    const users = res.json('users');
    if (!users || users.length < count) {
      throw new Error(`Failed to get ${count} users. Status: ${res.status}`);
    }
    return users;
  },

  createSingleRoom: (userId1, userId2) => {
    const res = api.post('/api/v1/rooms', { user_id_1: userId1, user_id_2: userId2 });
    const roomId = res.json('room_id');
    if (!roomId) console.error(`Single room creation failed: ${res.status} ${res.body}`);
    return roomId;
  },

  createGroupRoom: (name, memberIds) => {
    const res = api.post('/api/v1/rooms/group', { name, member_ids: memberIds });
    const roomId = res.json('room_id');
    if (!roomId) console.error(`Group room creation failed: ${res.status} ${res.body}`);
    return roomId;
  }
};

export function setup() {
  const userCount = 500;
  const users = api.createUsers(userCount);
  console.log(`Created ${users.length} users`);

  const vuData = new Array(userCount);

  for (let i = 0; i < 300; i += 2) {
    const roomId = api.createSingleRoom(users[i].user_id, users[i + 1].user_id);
    vuData[i] = { user: users[i], room_id: roomId, type: 'single' };
    vuData[i + 1] = { user: users[i + 1], room_id: roomId, type: 'single' };
  }

  const group1Users = users.slice(300, 400);
  const roomIdG1 = api.createGroupRoom('Group 1', group1Users.map(u => u.user_id));
  for (let i = 300; i < 400; i++) {
    vuData[i] = { user: users[i], room_id: roomIdG1, type: 'group' };
  }

  const group2Users = users.slice(400, 500);
  const roomIdG2 = api.createGroupRoom('Group 2', group2Users.map(u => u.user_id));
  for (let i = 400; i < 500; i++) {
    vuData[i] = { user: users[i], room_id: roomIdG2, type: 'group' };
  }

  console.log(`Setup complete: 150 single rooms, 2 group rooms.`);
  return { vuData };
}

export default function (data) {
  const vuIndex = __VU - 1;
  const myConfig = data.vuData[vuIndex];

  if (!myConfig || !myConfig.user) {
    return;
  }

  const REQ_HEADERS = {
    'Authorization': myConfig.user.token,
  };

  const res = ws.connect(WS_URL, { headers: REQ_HEADERS }, (socket) => {
    socket.on('open', () => {
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