package store

import (
	"bytes"
	"encoding/json/jsontext"
	"fmt"

	"github.com/parquet-go/parquet-go"
)

// ParquetMessage is the columnar archive representation of Message. UUIDs are
// dictionary coded, monotonic seq/time delta coded, and the JSON columns are
// zstd compressed, so cold files shrink to a fraction of the Pebble footprint.
type ParquetMessage struct {
	MsgID        [16]byte `parquet:"msg_id,dict"`
	ClientMsgID  [16]byte `parquet:"client_msg_id,dict"`
	SenderID     [16]byte `parquet:"sender_id,dict"`
	RoomID       [16]byte `parquet:"room_id,dict"`
	RoomSeq      uint64   `parquet:"room_seq,delta"`
	ServerTime   int64    `parquet:"server_time,delta"`
	ReplyToMsgID [16]byte `parquet:"reply_to_msg_id,dict"`
	MsgType      int32    `parquet:"msg_type"`
	Payload      []byte   `parquet:"payload,zstd"`
	Ext          []byte   `parquet:"ext,zstd"`
}

func newParquetMessage(m *Message) ParquetMessage {
	return ParquetMessage{
		MsgID:        m.MsgID,
		ClientMsgID:  m.ClientMsgID,
		SenderID:     m.SenderID,
		RoomID:       m.RoomID,
		RoomSeq:      m.RoomSeq,
		ServerTime:   m.ServerTime,
		ReplyToMsgID: m.ReplyToMsgID,
		MsgType:      int32(m.MsgType),
		Payload:      m.Payload,
		Ext:          m.Ext,
	}
}

func (p ParquetMessage) message() Message {
	return Message{
		MsgID:        p.MsgID,
		ClientMsgID:  p.ClientMsgID,
		SenderID:     p.SenderID,
		RoomID:       p.RoomID,
		RoomSeq:      p.RoomSeq,
		ServerTime:   p.ServerTime,
		ReplyToMsgID: p.ReplyToMsgID,
		MsgType:      MsgType(p.MsgType),
		Payload:      jsontext.Value(p.Payload),
		Ext:          jsontext.Value(p.Ext),
	}
}

// encodeParquet serializes rows into a self-contained parquet file buffer.
func encodeParquet(rows []ParquetMessage) ([]byte, error) {
	buffer := new(bytes.Buffer)
	writer := parquet.NewGenericWriter[ParquetMessage](buffer)
	if _, err := writer.Write(rows); err != nil {
		writer.Close()
		return nil, fmt.Errorf("write parquet rows: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close parquet writer: %w", err)
	}
	return buffer.Bytes(), nil
}

// decodeParquet parses a parquet file buffer back into rows.
func decodeParquet(data []byte) ([]ParquetMessage, error) {
	rows, err := parquet.Read[ParquetMessage](bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("read parquet: %w", err)
	}
	return rows, nil
}
