// Package pgxuuid integrates github.com/google/uuid with pgx UUID binary codecs.
package pgxuuid

import (
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type UUID uuid.UUID

func (u *UUID) ScanUUID(v pgtype.UUID) error {
	if !v.Valid {
		return errors.New("cannot scan NULL into *uuid.UUID")
	}
	*u = v.Bytes
	return nil
}

func (u UUID) UUIDValue() (pgtype.UUID, error) {
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func tryWrapUUIDEncodePlan(value any) (pgtype.WrappedEncodePlanNextSetter, any, bool) {
	switch u := value.(type) {
	case uuid.UUID:
		return &uuidEncodePlan{}, UUID(u), true
	case *uuid.UUID:
		return &uuidEncodePlan{}, UUID{}, true
	default:
		return nil, nil, false
	}
}

type uuidEncodePlan struct{ next pgtype.EncodePlan }

func (p *uuidEncodePlan) SetNext(next pgtype.EncodePlan) { p.next = next }

func (p *uuidEncodePlan) Encode(value any, buf []byte) ([]byte, error) {
	if u, ok := value.(*uuid.UUID); ok {
		if u == nil {
			return nil, nil
		}
		return p.next.Encode(UUID(*u), buf)
	}
	return p.next.Encode(UUID(value.(uuid.UUID)), buf)
}

func tryWrapUUIDScanPlan(target any) (pgtype.WrappedScanPlanNextSetter, any, bool) {
	u, ok := target.(*uuid.UUID)
	if !ok {
		return nil, nil, false
	}
	return &uuidScanPlan{}, (*UUID)(u), true
}

type uuidScanPlan struct{ next pgtype.ScanPlan }

func (p *uuidScanPlan) SetNext(next pgtype.ScanPlan) { p.next = next }

func (p *uuidScanPlan) Scan(src []byte, dst any) error {
	return p.next.Scan(src, (*UUID)(dst.(*uuid.UUID)))
}

// Register enables direct binary UUID encoding and scanning for google/uuid.
func Register(tm *pgtype.Map) {
	tm.TryWrapEncodePlanFuncs = append(
		[]pgtype.TryWrapEncodePlanFunc{tryWrapUUIDEncodePlan},
		tm.TryWrapEncodePlanFuncs...)
	tm.TryWrapScanPlanFuncs = append([]pgtype.TryWrapScanPlanFunc{tryWrapUUIDScanPlan}, tm.TryWrapScanPlanFuncs...)
}
