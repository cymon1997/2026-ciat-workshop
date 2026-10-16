package promo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

var (
	ErrInvalidRequest  = errors.New("invalid request")
	ErrQuotaExhausted  = errors.New("promo quota exhausted")
	ErrAlreadyRedeemed = errors.New("user already redeemed this promo") // not enforced yet
)

type RedeemRequest struct {
	PromoCode string
	UserID    string
	OrderID   string
}

// RedeemService applies a promo code to an order.
// One RedeemService runs in each pod of promo-svc.
type RedeemService struct {
	store  Store
	logger *log.Logger
	now    func() time.Time
}

func NewRedeemService(store Store, logger *log.Logger) *RedeemService {
	return &RedeemService{store: store, logger: logger, now: time.Now}
}

// Redeem uses one unit of the promo's quota for the given order.
func (s *RedeemService) Redeem(ctx context.Context, req RedeemRequest) (Redemption, error) {
	if req.PromoCode == "" || req.UserID == "" || req.OrderID == "" {
		return Redemption{}, ErrInvalidRequest
	}

	promo, err := s.store.GetPromo(ctx, req.PromoCode)
	if err != nil {
		return Redemption{}, fmt.Errorf("get promo %s: %w", req.PromoCode, err)
	}

	// Quota check.
	if promo.Used >= promo.Quota {
		s.logger.Printf("rejected promo=%s user=%s reason=quota_exhausted used=%d quota=%d",
			promo.ID, req.UserID, promo.Used, promo.Quota)
		return Redemption{}, ErrQuotaExhausted
	}

	r := Redemption{
		PromoID:    promo.ID,
		UserID:     req.UserID,
		OrderID:    req.OrderID,
		RedeemedAt: s.now(),
	}
	if err := s.store.InsertRedemption(ctx, r); err != nil {
		return Redemption{}, fmt.Errorf("insert redemption: %w", err)
	}

	if err := s.store.IncrementUsed(ctx, promo.ID); err != nil {
		return Redemption{}, fmt.Errorf("increment used: %w", err)
	}

	s.logger.Printf("redeemed promo=%s user=%s order=%s used=%d quota=%d",
		promo.ID, req.UserID, req.OrderID, promo.Used+1, promo.Quota)
	return r, nil
}
