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

func Success[T any](w http.ResponseWriter, msg string, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.MarshalWrite(w, Response[T]{
		Code: http.StatusOK,
		Msg:  msg,
		Data: data,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to write success response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func SuccessNoData(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.MarshalWrite(w, ResponseWithoutData{
		Code: code,
		Msg:  msg,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to write success response without data")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func Error(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.MarshalWrite(w, errorResponse{
		Code: code,
		Msg:  msg,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to write error response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func ReadBody[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var body T

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
	Type         string         `json:"type"`
	RequestID    string         `json:"request_id"`
	ClientMsgID  string         `json:"client_msg_id"`
	RoomID       string         `json:"room_id"`
	MsgType      string         `json:"msg_type"`
	Payload      jsontext.Value `json:"payload"`
	ReplyToMsgID string         `json:"reply_to_msg_id"`
	Ext          jsontext.Value `json:"ext"`
}

// AckFrame 下行 ack 帧。
type AckFrame struct {
	Type        string `json:"type"`
	RequestID   string `json:"request_id"`
	ClientMsgID string `json:"client_msg_id"`
	MsgID       string `json:"msg_id"`
	RoomID      string `json:"room_id"`
	RoomSeq     uint64 `json:"room_seq"`
	ServerTime  int64  `json:"server_time"`
	Code        int32  `json:"code"`
	Error       string `json:"error"`
}

// PushFrame 下行消息推送帧
type PushFrame struct {
	Type        string         `json:"type"`
	MsgID       string         `json:"msg_id"`
	ClientMsgID string         `json:"client_msg_id"`
	SenderID    string         `json:"sender_id"`
	RoomID      string         `json:"room_id"`
	RoomSeq     uint64         `json:"room_seq"`
	ServerTime  int64          `json:"server_time"`
	MsgType      string         `json:"msg_type"`
	Payload      jsontext.Value `json:"payload"`
	ReplyToMsgID string         `json:"reply_to_msg_id,omitempty"`
	Ext          jsontext.Value `json:"ext"`
}

// ErrorFrame 下行错误帧。
type ErrorFrame struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

// PongFrame 心跳应答帧。
type PongFrame struct {
	Type string `json:"type"`
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
	return &FrameReader{dec: jsontext.NewDecoder(r)}
}

func (fr *FrameReader) ReadFrame(out any) error {
	return json.UnmarshalDecode(fr.dec, out)
}
