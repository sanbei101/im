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
const SINGLE_CHAT_RATIO = 0.6; // 60% 单聊, 40% 群聊
const GROUP_SIZE = 5; // 群聊每组人数

let rooms = [];

export function setup() {
  // 批量创建用户
  const userCount = 50;
  const payload = JSON.stringify({ count: userCount });
  const res = http.post(`${API_BASE}/api/v1/users/batch`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  const users = res.json('users');
  if (!users || users.length < userCount) {
    throw new Error(`Failed to get ${userCount} users, got ${users ? users.length : 0}`);
  }

  console.log(`Created ${users.length} users`);

  // 创建单聊房间
  const singleChatRooms = [];
  for (let i = 0; i < users.length - 1; i += 2) {
    const res = http.post(
      `${API_BASE}/api/v1/rooms`,
      JSON.stringify({ user_id_1: users[i].user_id, user_id_2: users[i + 1].user_id }),
      { headers: { 'Content-Type': 'application/json' } }
    );
    const roomId = res.json('room_id');
    singleChatRooms.push({ room_id: roomId, users: [users[i], users[i + 1]] });
    console.log(`Created single chat room ${roomId} for users ${i} and ${i + 1}`);
  }

  // 创建群聊房间
  const groupChatRooms = [];
  const groupStartIndex = Math.floor(users.length * 0.6); // 从60%位置开始用于群聊
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
    console.log(`Created group chat room ${roomId} with ${GROUP_SIZE} members starting at index ${i}`);
  }

  rooms = [...singleChatRooms, ...groupChatRooms];
  console.log(`Total rooms: ${rooms.length} (${singleChatRooms.length} single, ${groupChatRooms.length} group)`);

  return { users, rooms };
}

export default function (data) {
  const { users, rooms } = data;
  const vuIndex = __VU - 1;

  // 分配用户和房间
  // VU 0-29 分配给 rooms[0-29]
  const roomIndex = vuIndex % rooms.length;
  const room = rooms[roomIndex];
  const isGroupChat = room.users.length > 2;

  // 当前 VU 对应的用户
  const myUser = room.users[vuIndex % room.users.length];
  if (!myUser) {
    console.error(`[VU ${__VU}] No user assigned`);
    return;
  }

  const REQ_HEADERS = {
    'Authorization': myUser.token,
  };

  let sendCount = 0;

  const res = ws.connect(WS_URL, { headers: REQ_HEADERS }, (socket) => {
    socket.on('open', () => {
      console.log(`[VU ${__VU}] User ${myUser.user_id.slice(0, 8)} connected to ${isGroupChat ? 'group' : 'single'} chat room ${room.room_id.slice(0, 8)}`);

      socket.setInterval(() => {
        sendCount++;

        const msgType = isGroupChat ? 'group' : 'single';
        const targetDesc = isGroupChat ? `group(${room.room_id.slice(0, 8)})` : `user(${room.users.find(u => u.user_id !== myUser.user_id)?.user_id.slice(0, 8) || 'unknown'})`;

        const message = {
          client_msg_id: uuidv7(),
          room_id: room.room_id,
          msg_type: 'text',
          payload: {
            text: `[VU${__VU}] ${msgType} msg #${sendCount}`,
          },
          ext: {},
        };

        socket.send(JSON.stringify(message));

        if (sendCount % 50 === 0) {
          console.log(`[VU ${__VU}] Sent ${sendCount} messages to ${targetDesc}`);
        }
      }, 100);

      socket.setTimeout(() => {
        console.log(`[VU ${__VU}] Final send count: ${sendCount}`);
        socket.close();
      }, 30000);
    });

    socket.on('message', (msg) => {
      if (sendCount % 50 === 0) {
        console.log(`[VU ${__VU}] Received: ${msg.slice(0, 100)}`);
      }
    });

    socket.on('error', (e) => console.error(`[VU ${__VU}] Error:`, e.error()));
    socket.on('close', () => console.log(`[VU ${__VU}] Disconnected`));
  });

  check(res, { 'handshake success 101': (r) => r && r.status === 101 });
}
