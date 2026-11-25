package polymarket

import (
	"fmt"
	"strconv"

	obuilder "github.com/foxme666/Polymarket-golang/polymarket/order_builder"
)

// resolveTickSize uses the supplied tick for raw orders and checks the market minimum otherwise.
func (c *ClobClient) resolveTickSize(tokenID string, options *PartialCreateOrderOptions) (TickSize, error) {
	var tickSize *TickSize
	if options != nil {
		tickSize = options.TickSize
		if options.RawOrder {
			if tickSize == nil {
				return "", fmt.Errorf("RawOrder mode requires TickSize to be provided in options")
			}
			return *tickSize, nil
		}
	}
	minTickSize, err := c.GetTickSize(tokenID)
	if err != nil {
		return "", err
	}

	if tickSize != nil {
		if IsTickSizeSmaller(*tickSize, minTickSize) {
			return "", fmt.Errorf("invalid tick size (%s), minimum for the market is %s", *tickSize, minTickSize)
		}
		return *tickSize, nil
	}

	return minTickSize, nil
}

// ResolveFeeRate 解析手续费率
func (c *ClobClient) resolveFeeRate(tokenID string, userFeeRate int) (int, error) {
	marketFeeRateBps, err := c.GetFeeRateBps(tokenID)
	if err != nil {
		return 0, err
	}

	// 如果市场手续费率和用户提供的手续费率都不为零，验证它们是否匹配
	if marketFeeRateBps > 0 && userFeeRate > 0 && userFeeRate != marketFeeRateBps {
		return 0, fmt.Errorf("invalid user provided fee rate: (%d), fee rate for the market must be %d", userFeeRate, marketFeeRateBps)
	}

	return marketFeeRateBps, nil
}

// CreateOrder 创建并签名订单（限价订单）
// 需要L1认证
// options.RawOrder = true 时跳过从服务器获取 tick_size，但必须通过 options.TickSize 提供
func (c *ClobClient) CreateOrder(orderArgs *OrderArgs, options *PartialCreateOrderOptions) (*SignedOrder, error) {
	if err := c.assertLevel1Auth(); err != nil {
		return nil, err
	}

	allowLookup := options == nil || !options.RawOrder
	domain, err := c.resolveOrderDomain(orderArgs.TokenID, options, allowLookup)
	if err != nil {
		return nil, err
	}
	tickSize, err := c.resolveTickSize(orderArgs.TokenID, options)
	if err != nil {
		return nil, err
	}
	if options == nil || !options.RawOrder {
		orderArgs.FeeRateBps = 0
	}

	// 验证价格
	if !PriceValid(orderArgs.Price, tickSize) {
		tickSizeFloat, _ := strconv.ParseFloat(string(tickSize), 64)
		return nil, fmt.Errorf("price (%.6f), min: %s - max: %.6f", orderArgs.Price, tickSize, 1.0-tickSizeFloat)
	}

	// 获取舍入配置
	roundConfig, ok := obuilder.RoundingConfig[string(tickSize)]
	if !ok {
		return nil, fmt.Errorf("unsupported tick size: %s", tickSize)
	}

	// 获取订单金额（带舍入）
	side, makerAmount, takerAmount, err := c.builder.GetOrderAmounts(
		orderArgs.Side,
		orderArgs.Size,
		orderArgs.Price,
		roundConfig,
	)
	if err != nil {
		return nil, err
	}

	orderData := &OrderDataV2{
		Maker:         c.builder.GetFunder(),
		TokenID:       orderArgs.TokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          side,
		Signer:        c.signer.Address(),
		SignatureType: c.builder.GetSigType(),
		Timestamp:     orderArgs.Timestamp,
		Metadata:      orderArgs.Metadata,
		Builder:       orderArgs.BuilderCode,
		Expiration:    orderArgs.Expiration,
	}

	return buildSignedOrderV2(c.signer.privateKey, orderData, domain)
}

// CreateMarketOrder 创建并签名市价订单
// 需要L1认证
func (c *ClobClient) CreateMarketOrder(orderArgs *MarketOrderArgs, options *PartialCreateOrderOptions) (*SignedOrder, error) {
	if err := c.assertLevel1Auth(); err != nil {
		return nil, err
	}

	// RawOrder is a limit-order option. Market orders still check the market minimum tick.
	var marketOptions PartialCreateOrderOptions
	if options != nil {
		marketOptions = *options
		marketOptions.RawOrder = false
	}
	domain, err := c.resolveOrderDomain(orderArgs.TokenID, options, true)
	if err != nil {
		return nil, err
	}
	tickSize, err := c.resolveTickSize(orderArgs.TokenID, &marketOptions)
	if err != nil {
		return nil, err
	}

	// 如果价格未设置或为0，计算市价
	if orderArgs.Price <= 0 {
		price, err := c.CalculateMarketPrice(orderArgs.TokenID, orderArgs.Side, orderArgs.Amount, orderArgs.OrderType)
		if err != nil {
			return nil, err
		}
		orderArgs.Price = price
	}

	// 验证价格
	if !PriceValid(orderArgs.Price, tickSize) {
		tickSizeFloat, _ := strconv.ParseFloat(string(tickSize), 64)
		return nil, fmt.Errorf("price (%.6f), min: %s - max: %.6f", orderArgs.Price, tickSize, 1.0-tickSizeFloat)
	}

	// V2订单签名不再携带 feeRateBps；市场手续费由撮合时决定。
	orderArgs.FeeRateBps = 0

	// 获取舍入配置
	roundConfig, ok := obuilder.RoundingConfig[string(tickSize)]
	if !ok {
		return nil, fmt.Errorf("unsupported tick size: %s", tickSize)
	}

	// 获取订单金额
	side, makerAmount, takerAmount, err := c.builder.GetMarketOrderAmounts(
		orderArgs.Side,
		orderArgs.Amount,
		orderArgs.Price,
		roundConfig,
	)
	if err != nil {
		return nil, err
	}

	orderData := &OrderDataV2{
		Maker:         c.builder.GetFunder(),
		TokenID:       orderArgs.TokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          side,
		Signer:        c.signer.Address(),
		SignatureType: c.builder.GetSigType(),
		Timestamp:     orderArgs.Timestamp,
		Metadata:      orderArgs.Metadata,
		Builder:       orderArgs.BuilderCode,
		Expiration:    0,
	}

	return buildSignedOrderV2(c.signer.privateKey, orderData, domain)
}

// CreateAndPostOrder 创建并提交订单（便捷方法）
// 支持通过 options.OrderType 指定订单类型：GTC, FOK, GTD, FAK（默认 GTC）
// 返回 PostOrderResult，包含原始 Payload 和 API 响应
func (c *ClobClient) CreateAndPostOrder(orderArgs *OrderArgs, options *PartialCreateOrderOptions) (*PostOrderResult, error) {
	order, err := c.CreateOrder(orderArgs, options)
	if err != nil {
		return nil, err
	}

	// 确定订单类型，默认为 GTC
	orderType := OrderTypeGTC
	if options != nil && options.OrderType != nil {
		orderType = *options.OrderType
	}

	return c.PostOrder(order, orderType)
}

// CalculateMarketPrice 计算市价
func (c *ClobClient) CalculateMarketPrice(tokenID, side string, amount float64, orderType OrderType) (float64, error) {
	book, err := c.GetOrderBook(tokenID)
	if err != nil {
		return 0, fmt.Errorf("no orderbook: %w", err)
	}

	if side == BUY {
		if len(book.Asks) == 0 {
			return 0, fmt.Errorf("no match")
		}
		return c.builder.CalculateBuyMarketPrice(ConvertOrderSummaries(book.Asks), amount, string(orderType))
	} else {
		if len(book.Bids) == 0 {
			return 0, fmt.Errorf("no match")
		}
		return c.builder.CalculateSellMarketPrice(ConvertOrderSummaries(book.Bids), amount, string(orderType))
	}
}

// ConvertOrderSummaries 转换OrderSummary为order_builder.OrderSummary接口（导出函数）
func ConvertOrderSummaries(summaries []OrderSummary) []interface{} {
	result := make([]interface{}, len(summaries))
	for i, s := range summaries {
		result[i] = &OrderSummaryWrapper{OrderSummary: s}
	}
	return result
}
