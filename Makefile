unittest:
	@go test -race -count=5 ./promo/...

extended:
	@cp admin/extended/extended_test.go promo/extended_test.go
	@echo "extended tests copied to promo/extended_test.go"

.PHONY: unittest extended
