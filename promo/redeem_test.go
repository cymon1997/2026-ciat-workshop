package promo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"testing"
)

func quietLogger() *log.Logger { return log.New(io.Discard, "", 0) }

func TestRedeem_RejectsInvalidRequest(t *testing.T) {
	store := NewMemoryStore()
	svc := NewRedeemService(store, quietLogger())

	_, err := svc.Redeem(context.Background(), RedeemRequest{PromoCode: "FLASH1010"})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
}

func TestRedeem_Sequential(t *testing.T) {
	store := NewMemoryStore()
	store.SeedPromo(Promo{ID: "FLASH1010", Quota: 5})
	svc := NewRedeemService(store, quietLogger())
	ctx := context.Background()

	success, exhausted := 0, 0
	for i := 0; i < 7; i++ {
		_, err := svc.Redeem(ctx, RedeemRequest{
			PromoCode: "FLASH1010",
			UserID:    fmt.Sprintf("u-%d", i),
			OrderID:   fmt.Sprintf("o-%d", i),
		})
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrQuotaExhausted):
			exhausted++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if success != 5 || exhausted != 2 {
		t.Fatalf("expected 5 success and 2 exhausted, got %d success and %d exhausted", success, exhausted)
	}
}

// Flash sale: many users press "Pay" at the same moment.
func TestRedeem_ConcurrentFlashSale(t *testing.T) {
	const quota, users = 100, 300

	store := NewMemoryStore()
	store.SeedPromo(Promo{ID: "FLASH1010", Quota: quota})
	svc := NewRedeemService(store, quietLogger())
	ctx := context.Background()

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		success int
	)
	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := svc.Redeem(ctx, RedeemRequest{
				PromoCode: "FLASH1010",
				UserID:    fmt.Sprintf("u-%d", i),
				OrderID:   fmt.Sprintf("o-%d", i),
			})
			if err != nil && !errors.Is(err, ErrQuotaExhausted) {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	promo, _ := store.GetPromo(ctx, "FLASH1010")
	stored, _ := store.CountRedemptions(ctx, "FLASH1010")

	if success != quota {
		t.Errorf("successful redemptions: got %d, want %d", success, quota)
	}
	if stored != quota {
		t.Errorf("stored redemptions: got %d, want %d", stored, quota)
	}
	if promo.Used != quota {
		t.Errorf("promo.Used: got %d, want %d", promo.Used, quota)
	}
}
