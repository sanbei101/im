package store

import (
	"encoding/json/jsontext"
	"testing"
	"uuid"
)

func BenchmarkArchiveParquetCodec(b *testing.B) {
	rows := make([]ParquetMessage, 2000)
	for i := range rows {
		rows[i] = newParquetMessage(&Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(), RoomID: uuid.NewV7(),
			RoomSeq: uint64(i + 1), ServerTime: int64(i), MsgType: MsgTypeText,
			Payload: jsontext.Value(`{"text":"parquet archive benchmark payload"}`),
		})
	}

	sample, err := encodeParquet(rows)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("encode", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(sample)))
		for b.Loop() {
			if _, err := encodeParquet(rows); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("decode", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(sample)))
		for b.Loop() {
			decoded, err := decodeParquet(sample)
			if err != nil || len(decoded) != len(rows) {
				b.Fatalf("decode parquet: rows=%d err=%v", len(decoded), err)
			}
		}
	})
}
