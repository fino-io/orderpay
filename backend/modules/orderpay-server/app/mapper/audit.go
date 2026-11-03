package mapper

import (
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	pb "github.com/fino-io/orderpay/go/orderpay/v1"
)

func ToAuditEventViews(entities []*model.AuditEvent) []*pb.AuditEventView {
	result := make([]*pb.AuditEventView, 0, len(entities))
	for _, entity := range entities {
		snapshot := entity.Snapshot()
		result = append(result, &pb.AuditEventView{
			Id:                string(snapshot.ID),
			MerchantId:        snapshot.MerchantId,
			ActorId:           snapshot.ActorId,
			Action:            snapshot.Action,
			ResourceId:        snapshot.ResourceId,
			RequestId:         snapshot.RequestId,
			BeforeStatus:      snapshot.BeforeStatus,
			AfterStatus:       snapshot.AfterStatus,
			ReasonCode:        snapshot.ReasonCode,
			EvidenceReference: snapshot.EvidenceReference,
			CreateTime:        snapshot.CreateTime,
		})
	}
	return result
}
