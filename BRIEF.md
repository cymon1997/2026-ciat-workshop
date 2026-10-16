# Incident: FLASH1010 promo was oversold

**Severity:** P2. **Reported:** 2026-10-11 09:12 by Promo Ops. **You are:** the on-call backend team.

## What happened

On 2026-10-10 at 20:00 we ran a flash sale. Promo code **FLASH1010** gives Rp 50,000 off, for the
**first 100 orders only** (`quota = 100`).

The nightly reconcile job reported this:

```
promo=FLASH1010 quota=100 used_counter=103 redemption_docs=103 status=OVER_QUOTA
```

103 customers got the discount. Finance pays for 3 discounts that are not in the budget. Promo Ops
wants to run a bigger flash sale (quota 1,000) next week, and they ask: *"Can this happen again?"*

## Context

- `promo-svc` is a Go service. It runs on Kubernetes and autoscales between **2 and 6 pods**.
- The database is MongoDB. In this kit `MemoryStore` (`promo/store.go`) stands in for it.
- The redeem flow is in `promo/redeem.go`.
- Logs from the flash sale are in `logs/promo-svc-2026-10-10.log`.

## Your tasks

1. **Investigate.** Find the root cause. Point to the evidence in the logs and in the code.
2. **Explain.** Write the root cause in 2 or 3 sentences that Promo Ops can understand.
3. **Fix.** Change `promo/redeem.go` so the promo can never be oversold.
4. **Prove it.** All tests pass: `go test ./...`
5. **Decide.** What do we do about the 3 extra customers who already got the discount?

## Rules

- Do not change the existing tests. You may add new tests.
- You may change `promo/store.go`, but treat `MemoryStore` as **the real MongoDB**: any method you
  add must be something MongoDB can do in **one operation**. Be ready to name that operation.
- Ask the facilitator if you need information. The facilitator is your on-call senior engineer.

## Run

```bash
go test ./...            # one test fails now
go test -run Sequential -v ./...
```

Requires Go 1.21 or newer.

## Present (3 minutes)

1. Root cause and the evidence you used.
2. The options you considered, and why you chose your fix.
3. Your answer to task 5.
