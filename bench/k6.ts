import "fast-text-encoding";
import ws, { Socket } from "k6/ws";
import { check } from "k6";
import { Trend, Counter } from "k6/metrics";
import { v7 as uuidv7 } from "uuid";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import {
  SendMessageReqSchema,
  SendMessageResp,
  SendMessageRespSchema,
  MessageType,
  TextMessagePayloadSchema,
} from "../gen/ts/proto/im/v1/message_pb";
import http, { RefinedResponse, ResponseType } from "k6/http";

export interface BenchMockUserInfo {
  user_id: string;
  username: string;
  token: string;
}

export interface SingleRoomResp {
  room_id: string;
  users: BenchMockUserInfo[];
}

export interface GroupRoomResp {
  room_id: string;
  room_size: number;
  users: BenchMockUserInfo[];
}

export interface BatchMockResp {
  single_rooms: SingleRoomResp[];
  group_rooms: GroupRoomResp[];
  total_user_num: number;
}

export interface VuConfig {
  user: BenchMockUserInfo;
  room_id: string;
  type: "single" | "group";
}

export interface SetupData {
  vuData: VuConfig[];
}

interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}

// ==================== 2. 常量与指标配置 ====================

const SingleRoomNum = 500;
const GroupRoom = [100, 100];
const groupTotal = GroupRoom.reduce((sum: number, val: number) => sum + val, 0);
const VU_NUM = SingleRoomNum * 2 + groupTotal;
const DURATION = "30s";

export const wsMsgLatency = new Trend("ws_msg_latency", true);
export const wsMsgUnmatched = new Counter("ws_msg_unmatched");

export const options = {
  vus: VU_NUM,
  duration: DURATION,
};

const ServerIP = __ENV.TARGET_HOST;
const API_BASE = `http://${ServerIP}:8801`;
const WS_URL = `ws://${ServerIP}:8800/ws`;

// ==================== 3. API 辅助对象 ====================

const api = {
  /**
   * POST 请求封装
   */
  post: (path: string, payload: unknown, token: string): RefinedResponse<ResponseType> => {
    const url = `${API_BASE}${path}`;
    const params = {
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
    };
    return http.post(url, JSON.stringify(payload), params);
  },

  /**
   * 创建 Mock 数据
   */
  createBenchMock: (payload: {
    single_room_num: number;
    group_room: number[];
  }): BatchMockResp | null => {
    const res = api.post("/api/v1/bench/mock", payload, "");
    if (res.status !== 200) {
      console.error(`Create bench mock failed: ${res.status} ${res.body}`);
      return null;
    }
    const wrapper = res.json() as unknown as ApiResponse<BatchMockResp>;
    return wrapper.data;
  },
};

/**
 * 将 BatchMockResp 展平为单 VU 对应的配置列表
 */
function flattenBenchData(batchRes: BatchMockResp): VuConfig[] {
  const vuData: VuConfig[] = [];

  for (const room of batchRes.single_rooms || []) {
    for (const user of room.users || []) {
      vuData.push({
        user,
        room_id: room.room_id,
        type: "single",
      });
    }
  }

  for (const room of batchRes.group_rooms || []) {
    for (const user of room.users || []) {
      vuData.push({
        user,
        room_id: room.room_id,
        type: "group",
      });
    }
  }

  return vuData;
}

// ==================== 4. K6 生命周期 ====================

/**
 * Setup 阶段：生成测试数据并分发给 VU
 */
export function setup(): SetupData {
  const payload = {
    single_room_num: SingleRoomNum,
    group_room: GroupRoom,
  };

  const batchRes = api.createBenchMock(payload);
  if (!batchRes) {
    throw new Error("Create bench mock failed");
  }

  const vuData = flattenBenchData(batchRes);
  if (vuData.length < batchRes.total_user_num) {
    throw new Error(`Expected ${batchRes.total_user_num} users, got ${vuData.length}`);
  }

  console.log(
    `Setup complete: ${batchRes.single_rooms.length} single rooms, ${batchRes.group_rooms.length} group rooms, ${vuData.length} users.`,
  );
  return { vuData };
}

/**
 * VU 压测主逻辑
 */
export default function (data: SetupData): void {
  const vuIndex = __VU - 1;
  const myConfig = data.vuData[vuIndex];
  if (!myConfig || !myConfig.user) return;

  const pending = new Map<string, number>();

  const res = ws.connect(`${WS_URL}?token=${myConfig.user.token}`, null, (socket: Socket) => {
    socket.on("open", () => {
      socket.setInterval(() => {
        const clientMsgId = uuidv7();
        const now = Date.now();
        const textPayload = create(TextMessagePayloadSchema, {
          text: `[VU${__VU}] hello`,
          atUserIds: [],
        });
        const payloadBytes = toBinary(TextMessagePayloadSchema, textPayload);
        const msg = create(SendMessageReqSchema, {
          clientMsgId: clientMsgId,
          roomId: myConfig.room_id,
          msgType: MessageType.TEXT,
          payload: payloadBytes,
        });

        const binaryData = toBinary(SendMessageReqSchema, msg);

        pending.set(clientMsgId, now);

        // 发送二进制 ArrayBuffer
        socket.sendBinary(binaryData.buffer);
      }, 500);
    });

    socket.on("binaryMessage", (raw: ArrayBuffer) => {
      const msg = fromBinary<SendMessageResp>(SendMessageRespSchema, new Uint8Array(raw));
      const id = msg.clientMsgId;
      if (!id) return;

      const start = pending.get(id);
      if (!start) return;

      const latency = Date.now() - start;
      wsMsgLatency.add(latency);
      pending.delete(id);
    });

    socket.on("close", () => {
      wsMsgUnmatched.add(pending.size);
      pending.clear();
    });
  });

  check(res, {
    "handshake success 101": (r) => r !== null && r.status === 101,
  });
}
