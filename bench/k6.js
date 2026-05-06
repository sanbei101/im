import ws from 'k6/ws';
import { check } from 'k6';
import { v7 as uuidv7 } from 'https://unpkg.com/uuid@14.0.0/dist/index.js';
import http from 'k6/http';

export const options = {
  vus: 30,
  duration: '30s',
};

const API_BASE = 'http://localhost:8801';
const WS_URL = 'ws://localhost:8800/ws';
const SINGLE_CHAT_RATIO = 0.6;
const GROUP_SIZE = 5;

let rooms = [];

export function setup() {
  const userCount = 50;
  const payload = JSON.stringify({ count: userCount });
  const res = http.post(`${API_BASE}/api/v1/users/batch`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  const users = res.json('users');
  if (!users || users.length < userCount) {
    throw new Error(`Failed to get ${userCount} users`);
  }

  console.log(`Created ${users.length} users`);

  const singleChatRooms = [];
  for (let i = 0; i < users.length - 1; i += 2) {
    const res = http.post(
      `${API_BASE}/api/v1/rooms`,
      JSON.stringify({ user_id_1: users[i].user_id, user_id_2: users[i + 1].user_id }),
      { headers: { 'Content-Type': 'application/json' } }
    );
    const roomId = res.json('room_id');
    singleChatRooms.push({ room_id: roomId, users: [users[i], users[i + 1]] });
  }

  const groupChatRooms = [];
  const groupStartIndex = Math.floor(users.length * 0.6);
  for (let i = groupStartIndex; i + GROUP_SIZE <= users.length; i += GROUP_SIZE) {
    const memberIds = users.slice(i, i + GROUP_SIZE).map(u => u.user_id);
    const res = http.post(
      `${API_BASE}/api/v1/rooms/group`,
      JSON.stringify({ name: `Group ${i / GROUP_SIZE}`, member_ids: memberIds }),
      { headers: { 'Content-Type': 'application/json' } }
    );
    const roomId = res.json('room_id');
    const groupUsers = users.slice(i, i + GROUP_SIZE);
    groupChatRooms.push({ room_id: roomId, users: groupUsers });
  }

  rooms = [...singleChatRooms, ...groupChatRooms];
  console.log(`Total rooms: ${rooms.length}`);

  return { users, rooms };
}

export default function (data) {
  const { users, rooms } = data;
  const vuIndex = __VU - 1;

  const roomIndex = vuIndex % rooms.length;
  const room = rooms[roomIndex];
  const isGroupChat = room.users.length > 2;

  const myUser = room.users[vuIndex % room.users.length];
  if (!myUser) {
    return;
  }

  const REQ_HEADERS = {
    'Authorization': myUser.token,
  };

  const res = ws.connect(WS_URL, { headers: REQ_HEADERS }, (socket) => {
    socket.on('open', () => {
      socket.setInterval(() => {
        const msgType = isGroupChat ? 'group' : 'single';
        
        const message = {
          client_msg_id: uuidv7(),
          room_id: room.room_id,
          msg_type: 'text',
          payload: {
            text: `[VU${__VU}] ${msgType} msg`,
          },
          ext: {},
        };

        socket.send(JSON.stringify(message));
      }, 100);

      socket.setTimeout(() => {
        socket.close();
      }, 30000);
    });

    socket.on('error', (e) => console.error(`[VU ${__VU}] Error:`, e.error()));
  });

  check(res, { 'handshake success 101': (r) => r && r.status === 101 });
}