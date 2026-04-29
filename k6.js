import ws from 'k6/ws';
import { check } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import http from 'k6/http';
export const options = {
  vus: 50,
  duration: '30s',
};
const API_Url = 'http://localhost:8801/api/v1/users/batch'; 
const WS_URL = 'ws://localhost:8800/ws';


export function setup() {
  const payload = JSON.stringify({
    count: 50
  });
  const res = http.post(API_Url, payload);

  const users = res.json('users');

  if (!users || users.length < 50) {
    throw new Error(`获取用户数据失败,仅获取到 ${users ? users.length : 0} 条数据。`);
  }
  return users; 
}

export default function (users) {
  const vuIndex = __VU - 1;
  
  const me = users[vuIndex];

  const isEven = (vuIndex % 2 === 0);
  const partnerIndex = isEven ? vuIndex + 1 : vuIndex - 1;
  const partner = users[partnerIndex];

  
  const REQ_HEADERS = {
    'Authorization': me.token, 
  };

  let sendCount = 0;

  const res = ws.connect(WS_URL, { headers: REQ_HEADERS }, (socket) => {
    socket.on('open', () => {
      console.log(`[VU ${__VU}] (${me.user_id}) 连接成功,正准备向 (${partner.user_id}) 发送消息...`);
      
      socket.setInterval(() => {
        sendCount++;
        
        const message = {
          client_msg_id: uuidv4(), 
          room_id: partner.user_id,
          chat_type: "single",
          msg_type: "text",
          payload: {
            text: `你好,我是 VU${__VU},这是发给你的第 ${sendCount} 条消息!`,
          },
          ext: {},
        };

        socket.send(JSON.stringify(message));
        
        if (sendCount % 100 === 0) {
          console.log(`[VU ${__VU}] 已发送 ${sendCount} 条`);
        }
      }, 100);
      
      socket.setTimeout(() => {
        console.log(`[VU ${__VU}] 最终发送:${sendCount} 条`);
        socket.close();
      }, 30000);
    });

    socket.on('message', (msg) => {
       if (sendCount % 100 === 0) {
        console.log(`[VU ${__VU}] 收到消息: ${msg}`);
       }
    });
    
    socket.on('error', (e) => console.error(`[VU ${__VU}] 错误`, e.error()));
    socket.on('close', () => console.log(`[VU ${__VU}] 断开连接`));
  });

  check(res, { '握手成功 101': (r) => r && r.status === 101 });
}