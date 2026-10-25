package engine

import (
	"bytes"
	"encoding/json/jsontext"
	"fmt"

	"github.com/parquet-go/parquet-go"
)

// ParquetMessage is the columnar archive representation of Message. Messages
// are addressed by (room, seq), so no per-message UUID columns exist; the
// low-cardinality UUIDs are dictionary coded, monotonic seq/time delta coded,
// and the JSON columns are zstd compressed.
type ParquetMessage struct {
	SenderID   [16]byte `parquet:"sender_id,dict"`
	RoomID     [16]byte `parquet:"room_id,dict"`
	RoomSeq    uint64   `parquet:"room_seq,delta"`
	ServerTime int64    `parquet:"server_time,delta"`
	ReplyToSeq uint64   `parquet:"reply_to_seq,delta"`
	MsgType    int32    `parquet:"msg_type,dict"`
	Payload    []byte   `parquet:"payload,zstd"`
	Ext        []byte   `parquet:"ext,zstd"`
}

func newParquetMessage(m *Message) ParquetMessage {
	return ParquetMessage{
		SenderID:   m.SenderID,
		RoomID:     m.RoomID,
		RoomSeq:    m.RoomSeq,
		ServerTime: m.ServerTime,
		ReplyToSeq: m.ReplyToSeq,
		MsgType:    int32(m.MsgType),
		Payload:    m.Payload,
		Ext:        m.Ext,
	}
}

func (p ParquetMessage) message() Message {
	return Message{
		SenderID:   p.SenderID,
		RoomID:     p.RoomID,
		RoomSeq:    p.RoomSeq,
		ServerTime: p.ServerTime,
		ReplyToSeq: p.ReplyToSeq,
		MsgType:    MsgType(p.MsgType),
		Payload:    jsontext.Value(p.Payload),
		Ext:        jsontext.Value(p.Ext),
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
