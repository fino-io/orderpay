// seed-demo imports clearly marked local UI fixtures; it never calls payment channels.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	record "github.com/fino-io/orderpay/backend/modules/orderpay-server/infra/repo/model/sqlite"
	pb "github.com/fino-io/orderpay/go/orderpay"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	database := flag.String("db", "orderpay.db", "existing local SQLite database")
	merchant := flag.String("merchant", "019a4a70-0000-7000-8000-000000000001", "merchant UUID")
	flag.Parse()
	if _, err := uuid.Parse(*merchant); err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(*database); err != nil {
		log.Fatal("start the local service to initialize the database first: ", err)
	}

	db, err := gorm.Open(sqlite.Open(*database+"?_busy_timeout=5000&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	pool.SetMaxOpenConns(1)

	created := 0
	err = db.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < 30; i++ {
			number := fmt.Sprintf("MOCK-DEMO-%03d", i+1)
			var count int64
			if err := tx.Model(&record.Order{}).Where("merchant_id = ? AND merchant_order_no = ?", *merchant, number).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			if err := insertDemo(tx, *merchant, number, i); err != nil {
				return err
			}
			created++
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Imported %d mock orders; existing demo orders left unchanged. No channel calls or scheduled payments.\n", created)
}

func insertDemo(tx *gorm.DB, merchant, number string, index int) error {
	descriptions := []string{"协作平台月费", "短信通知套餐", "云存储扩容", "技术支持服务", "企业培训课程", "报表订阅", "设备维护", "团队席位增购", "数据备份服务", "项目实施服务"}
	amounts := []int64{19900, 5000, 9900, 29900, 128000, 4900, 68000, 39900, 15900, 298000}
	kind := index % 10
	created := time.Now().UTC().Add(-48*time.Hour + time.Duration(index)*75*time.Minute)
	updated := created.Add(5 * time.Minute)
	order := record.Order{
		ID:              model.OrderID(uuid.Must(uuid.NewV7()).String()),
		MerchantId:      merchant,
		MerchantOrderNo: number,
		AmountMinor:     amounts[kind],
		Description:     "[模拟数据] " + descriptions[kind],
		Status:          pb.OrderStatus_ORDER_STATUS_PENDING_PAYMENT,
		CreateTime:      core.FromTime(created),
		UpdateTime:      core.FromTime(created),
	}
	if kind == 3 || kind == 4 {
		order.Status = pb.OrderStatus_ORDER_STATUS_PAID
	}
	if kind == 5 || kind == 6 || kind == 9 {
		order.Status = pb.OrderStatus_ORDER_STATUS_CANCELLED
	}
	if kind != 0 {
		order.UpdateTime = core.FromTime(updated)
	}
	if err := tx.Create(&order).Error; err != nil {
		return err
	}
	if err := appendAudit(tx, merchant, string(order.ID), order.Status.String(), created); err != nil {
		return err
	}
	if kind == 0 || kind == 5 {
		return nil
	}

	payment := record.Payment{
		ID:               model.PaymentID(uuid.Must(uuid.NewV7()).String()),
		MerchantId:       merchant,
		OrderId:          string(order.ID),
		AmountMinor:      order.AmountMinor,
		MandateReference: "MOCK-NONEXECUTABLE-" + number,
		ChannelAccountId: "MOCK-CHANNEL-DISABLED",
		Status:           pb.PaymentStatus_PAYMENT_STATUS_CREATED,
		CreateTime:       core.FromTime(created.Add(time.Minute)),
		UpdateTime:       core.FromTime(updated),
	}
	switch kind {
	case 2, 8:
		payment.Status = pb.PaymentStatus_PAYMENT_STATUS_PENDING
	case 3, 4, 9:
		payment.Status = pb.PaymentStatus_PAYMENT_STATUS_SUCCEEDED
	case 6, 7:
		payment.Status = pb.PaymentStatus_PAYMENT_STATUS_DECLINED
	}
	if kind == 6 {
		payment.ReasonCode = "ORDER_CANCELLED"
	}
	if kind == 8 {
		payment.ReasonCode = "RECONCILIATION_REQUIRED"
	}
	if kind == 9 {
		payment.ReasonCode = "COMPENSATION_REQUIRED"
	}
	if kind != 1 && kind != 6 {
		payment.SubmittedAt = core.FromTime(created.Add(2 * time.Minute))
		payment.SubmitAttempts = 1
		payment.ProviderReference = "MOCK-REFERENCE-" + number
	}
	// Deliberately leave next_check_at and retry_deadline_at empty: fixtures are never scheduled.
	if err := tx.Create(&payment).Error; err != nil {
		return err
	}
	return appendAudit(tx, merchant, string(payment.ID), payment.Status.String(), updated)
}

func appendAudit(tx *gorm.DB, merchant, resource, status string, at time.Time) error {
	return tx.Create(&record.AuditEvent{
		ID:                model.AuditEventID(uuid.Must(uuid.NewV7()).String()),
		MerchantId:        merchant,
		ActorId:           "local-demo-seed",
		Action:            "MOCK_DATA_IMPORTED",
		ResourceId:        resource,
		RequestId:         "mock-" + resource,
		AfterStatus:       status,
		ReasonCode:        "MOCK_DATA",
		EvidenceReference: "mock://local-demo/" + resource,
		CreateTime:        core.FromTime(at),
	}).Error
}
