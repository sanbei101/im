package render

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"net/http"
	"strings"
	"sync"

	validator "github.com/kamalyes/go-argus"
	"github.com/phuslu/log"
)

var validate = validator.New()

func init() {
	validator.SetLocale("zh")
}

type Response[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type ResponseWithoutData struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type errorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	buf := getRespBuf()
	if err := json.MarshalWrite(buf, v); err != nil {
		putRespBuf(buf)
		log.Error().Err(err).Msg("marshal response")
		if code == http.StatusOK {
			code = http.StatusInternalServerError
		}
		Error(w, code, "internal error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Error().Err(err).Msg("write response")
	}
	putRespBuf(buf)
}

func getRespBuf() *bytes.Buffer {
	if buf, ok := respBufPool.Get().(*bytes.Buffer); ok {
		return buf
	}
	return new(bytes.Buffer)
}

func putRespBuf(buf *bytes.Buffer) {
	if buf.Cap() <= maxRespBufBytes {
		buf.Reset()
		respBufPool.Put(buf)
	}
}

var respBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

const maxRespBufBytes = 64 << 10 // 64 KiB

func Success[T any](w http.ResponseWriter, msg string, data T) {
	writeJSON(w, http.StatusOK, Response[T]{
		Code: http.StatusOK,
		Msg:  msg,
		Data: data,
	})
}

func SuccessNoData(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, ResponseWithoutData{
		Code: code,
		Msg:  msg,
	})
}

func Error(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, errorResponse{
		Code: code,
		Msg:  msg,
	})
}

const maxRequestBodyBytes = 1 << 20 // 1 MiB

func ReadBody[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var body T
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	if err := json.UnmarshalRead(r.Body, &body); err != nil {
		log.Error().Err(err).Msg("Failed to read request body")
		Error(w, http.StatusBadRequest, "JSON 格式非法")
		return body, err
	}

	if err := validate.Struct(body); err != nil {
		errs := validator.TranslateValidationErrors(err, "zh")
		errorMsgs := make([]string, 0, len(errs))
		for i := range errs {
			errorMsgs = append(errorMsgs, errs[i].Field+": "+errs[i].Message)
		}
		fullErrorMsg := strings.Join(errorMsgs, "; ")
		Error(w, http.StatusBadRequest, fullErrorMsg)
		return body, err
	}

	return body, nil
}

// ClientFrame 客户端上行帧
type ClientFrame struct {
	Type        string         `json:"type"`
	RequestID   string         `json:"request_id"`
	ClientMsgID string         `json:"client_msg_id"`
	RoomID      string         `json:"room_id"`
	MsgType     string         `json:"msg_type"`
	Payload     jsontext.Value `json:"payload"`
	ReplyToSeq  uint64         `json:"reply_to_seq"`
	Ext         jsontext.Value `json:"ext"`
}

// AckFrame 下行 ack 帧。
type AckFrame struct {
	Type        string `json:"type"`
	RequestID   string `json:"request_id"`
	ClientMsgID string `json:"client_msg_id"`
	RoomID      string `json:"room_id"`
	RoomSeq     uint64 `json:"room_seq"`
	ServerTime  int64  `json:"server_time"`
	Code        int32  `json:"code"`
	Error       string `json:"error"`
}

// PushFrame 下行消息推送帧
type PushFrame struct {
	Type       string         `json:"type"`
	SenderID   string         `json:"sender_id"`
	RoomID     string         `json:"room_id"`
	RoomSeq    uint64         `json:"room_seq"`
	ServerTime int64          `json:"server_time"`
	MsgType    string         `json:"msg_type"`
	Payload    jsontext.Value `json:"payload"`
	ReplyToSeq uint64         `json:"reply_to_seq,omitempty"`
	Ext        jsontext.Value `json:"ext"`
}

// ErrorFrame 下行错误帧。
type ErrorFrame struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

// TypingFrame 正在输入状态帧
type TypingFrame struct {
	Type   string `json:"type"`
	RoomID string `json:"room_id"`
	UserID string `json:"user_id,omitempty"`
}

// NotificationFrame 下行业务通知帧
type NotificationFrame struct {
	Type  string         `json:"type"`
	Event string         `json:"event"`
	Data  jsontext.Value `json:"data,omitempty"`
}

// FrameWriter 持有可复用的 jsontext.Encoder,把下行帧流式编码进内部 buffer
type FrameWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
	enc *jsontext.Encoder
}

// NewFrameWriter 构造帧编码器。
func NewFrameWriter() *FrameWriter { return &FrameWriter{} }

func (fw *FrameWriter) EncodeFrame[T any](v T) ([]byte, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if fw.enc == nil {
		fw.enc = jsontext.NewEncoder(&fw.buf)
	}
	if err := json.MarshalEncode(fw.enc, v); err != nil {
		fw.buf.Reset()
		fw.enc = jsontext.NewEncoder(&fw.buf)
		return nil, err
	}
	b := fw.buf.Bytes()
	n := len(b)
	if n > 0 && b[n-1] == '\n' {
		n--
	}
	frame := append([]byte(nil), b[:n]...)
	fw.buf.Reset()
	return frame, nil
}

type FrameReader struct {
	dec *jsontext.Decoder
}

func NewFrameReader(r io.Reader) *FrameReader {
	fr := &FrameReader{}
	if r != nil {
		fr.dec = jsontext.NewDecoder(r)
	}
	return fr
}

func (fr *FrameReader) Reset(r io.Reader) {
	if fr.dec == nil {
		fr.dec = jsontext.NewDecoder(r)
	} else {
		fr.dec.Reset(r)
	}
}

// ReadFrame decodes one frame of type T from the reader.
func (fr *FrameReader) ReadFrame[T any]() (T, error) {
	var out T
	err := json.UnmarshalDecode(fr.dec, &out)
	return out, err
}
