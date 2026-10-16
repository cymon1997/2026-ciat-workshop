package promo

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

// Promo is one promo code and its redemption quota.
type Promo struct {
	ID    string
	Quota int // maximum number of redemptions allowed
	Used  int // number of redemptions so far
}

// Redemption records one user using one promo on one order.
type Redemption struct {
	PromoID    string
	UserID     string
	OrderID    string
	RedeemedAt time.Time
}

var (
	ErrPromoNotFound       = errors.New("promo not found")
	ErrDuplicateRedemption = errors.New("duplicate redemption for this user and promo")
)

// Store is the persistence layer of the promo service.
//
// In production this is MongoDB. Every method below is ONE round-trip to the
// database, and every single method is atomic on its own, exactly like one
// MongoDB operation.
type Store interface {
	// GetPromo reads one promo document (findOne).
	GetPromo(ctx context.Context, id string) (Promo, error)

	// IncrementUsed adds 1 to promo.Used (updateOne with $inc).
	IncrementUsed(ctx context.Context, id string) error

	// InsertRedemption stores one redemption (insertOne).
	// When the store was created WithUniqueRedemptionPerUser, a second
	// redemption of the same promo by the same user returns
	// ErrDuplicateRedemption, like a unique index on (promoId, userId).
	InsertRedemption(ctx context.Context, r Redemption) error

	// DeleteRedemption removes the redemption of promoID for orderID (deleteOne).
	DeleteRedemption(ctx context.Context, promoID, orderID string) error

	// CountRedemptions counts stored redemptions for a promo (countDocuments).
	CountRedemptions(ctx context.Context, promoID string) (int, error)
}

// ---------------------------------------------------------------------------
// MemoryStore: an in-memory stand-in for MongoDB, used by the tests.
//
// s.mu plays the role of MongoDB's own internal locking: it makes each method
// atomic, like one database operation. It is NOT something the service can
// hold across calls; a real database client gives you no such lock.
// ---------------------------------------------------------------------------

type Option func(*MemoryStore)

// WithUniqueRedemptionPerUser simulates a unique index on (promoId, userId).
func WithUniqueRedemptionPerUser() Option {
	return func(s *MemoryStore) { s.uniquePerUser = true }
}

// WithLatency sets the simulated network round-trip per call.
func WithLatency(min, max time.Duration) Option {
	return func(s *MemoryStore) { s.minLatency, s.maxLatency = min, max }
}

type MemoryStore struct {
	mu            sync.Mutex
	promos        map[string]Promo
	redemptions   []Redemption
	uniquePerUser bool
	minLatency    time.Duration
	maxLatency    time.Duration
}

func NewMemoryStore(opts ...Option) *MemoryStore {
	s := &MemoryStore{
		promos:     map[string]Promo{},
		minLatency: 1 * time.Millisecond,
		maxLatency: 3 * time.Millisecond,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// SeedPromo creates or replaces a promo. Test helper.
func (s *MemoryStore) SeedPromo(p Promo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.promos[p.ID] = p
}

// roundTrip simulates network latency to the database.
func (s *MemoryStore) roundTrip(ctx context.Context) error {
	d := s.minLatency
	if s.maxLatency > s.minLatency {
		d += time.Duration(rand.Int63n(int64(s.maxLatency - s.minLatency)))
	}
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *MemoryStore) GetPromo(ctx context.Context, id string) (Promo, error) {
	if err := s.roundTrip(ctx); err != nil {
		return Promo{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.promos[id]
	if !ok {
		return Promo{}, ErrPromoNotFound
	}
	return p, nil
}

func (s *MemoryStore) IncrementUsed(ctx context.Context, id string) error {
	if err := s.roundTrip(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.promos[id]
	if !ok {
		return ErrPromoNotFound
	}
	p.Used++
	s.promos[id] = p
	return nil
}

func (s *MemoryStore) InsertRedemption(ctx context.Context, r Redemption) error {
	if err := s.roundTrip(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.uniquePerUser {
		for _, existing := range s.redemptions {
			if existing.PromoID == r.PromoID && existing.UserID == r.UserID {
				return ErrDuplicateRedemption
			}
		}
	}
	s.redemptions = append(s.redemptions, r)
	return nil
}

func (s *MemoryStore) DeleteRedemption(ctx context.Context, promoID, orderID string) error {
	if err := s.roundTrip(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.redemptions {
		if r.PromoID == promoID && r.OrderID == orderID {
			s.redemptions = append(s.redemptions[:i], s.redemptions[i+1:]...)
			return nil
		}
	}
	return nil
}

func (s *MemoryStore) CountRedemptions(ctx context.Context, promoID string) (int, error) {
	if err := s.roundTrip(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, r := range s.redemptions {
		if r.PromoID == promoID {
			n++
		}
	}
	return n, nil
}
