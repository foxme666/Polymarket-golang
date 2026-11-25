package polymarket

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	ordersigner "github.com/polymarket/go-order-utils/pkg/signer"
)

const testPrivateKey = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f093f8bf6b79f57a67"

func TestBuildSignedOrderV2SignsRecoverableOrder(t *testing.T) {
	privateKey, err := crypto.HexToECDSA(testPrivateKey)
	if err != nil {
		t.Fatalf("private key: %v", err)
	}
	signerAddress := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

	domain := orderDomain{chainID: Polygon, exchange: PolygonCTFExchangeV2, version: "2"}
	order, err := buildSignedOrderV2(privateKey, &OrderDataV2{
		Maker:         signerAddress,
		Signer:        signerAddress,
		TokenID:       "123456789",
		MakerAmount:   "1000000",
		TakerAmount:   "2000000",
		Side:          0,
		SignatureType: 0,
		Timestamp:     1713398400000,
		Metadata:      "0x1111111111111111111111111111111111111111111111111111111111111111",
		Builder:       "0x2222222222222222222222222222222222222222222222222222222222222222",
		Expiration:    1713400000,
	}, domain)
	if err != nil {
		t.Fatalf("build signed order: %v", err)
	}

	orderHash, err := buildOrderHashV2(order, domain)
	if err != nil {
		t.Fatalf("build order hash: %v", err)
	}

	ok, err := ordersigner.ValidateSignature(order.Signer, orderHash, order.Signature)
	if err != nil {
		t.Fatalf("validate signature: %v", err)
	}
	if !ok {
		t.Fatal("signature did not recover signer")
	}
	if order.Timestamp.String() != "1713398400000" {
		t.Fatalf("unexpected timestamp: %s", order.Timestamp.String())
	}
	if order.Metadata.Hex() != "0x1111111111111111111111111111111111111111111111111111111111111111" {
		t.Fatalf("unexpected metadata: %s", order.Metadata.Hex())
	}
	if order.Builder.Hex() != "0x2222222222222222222222222222222222222222222222222222222222222222" {
		t.Fatalf("unexpected builder: %s", order.Builder.Hex())
	}
}

func testOrderV2() *SignedOrder {
	return &SignedOrder{
		Salt:          big.NewInt(12345),
		Maker:         common.HexToAddress("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"),
		Signer:        common.HexToAddress("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"),
		Taker:         common.HexToAddress(ZeroAddress),
		TokenId:       big.NewInt(123456789),
		MakerAmount:   big.NewInt(1000000),
		TakerAmount:   big.NewInt(2000000),
		Expiration:    big.NewInt(1713400000),
		Nonce:         big.NewInt(99),
		FeeRateBps:    big.NewInt(50),
		Side:          big.NewInt(0),
		SignatureType: big.NewInt(0),
		Timestamp:     big.NewInt(1713398400000),
		Metadata:      common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
		Builder:       common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
		Signature:     []byte{1, 2, 3},
	}
}

func TestOrderToJSONWithPostOnlyUsesV2WireShape(t *testing.T) {
	order := testOrderV2()
	payload := OrderToJSON(order, "api-key", OrderTypeGTC)
	orderPayload, ok := payload["order"].(map[string]interface{})
	if !ok {
		t.Fatalf("order payload type = %T", payload["order"])
	}

	for _, removed := range []string{"nonce", "feeRateBps"} {
		if _, exists := orderPayload[removed]; exists {
			t.Fatalf("V2 order JSON must not contain %s", removed)
		}
	}
	if orderPayload["salt"] != int64(12345) {
		t.Fatalf("unexpected salt: %v (%T)", orderPayload["salt"], orderPayload["salt"])
	}
	if orderPayload["taker"] != "0x0000000000000000000000000000000000000000" {
		t.Fatalf("unexpected taker: %v", orderPayload["taker"])
	}
	if orderPayload["timestamp"] != "1713398400000" {
		t.Fatalf("unexpected timestamp: %v", orderPayload["timestamp"])
	}
	if orderPayload["builder"] != "0x2222222222222222222222222222222222222222222222222222222222222222" {
		t.Fatalf("unexpected builder: %v", orderPayload["builder"])
	}
	if orderPayload["signature"] != "0x010203" {
		t.Fatalf("unexpected signature: %v", orderPayload["signature"])
	}
	if payload["deferExec"] != false {
		t.Fatalf("unexpected deferExec: %v", payload["deferExec"])
	}
	if payload["postOnly"] != false {
		t.Fatalf("unexpected default postOnly: %v", payload["postOnly"])
	}

	postOnlyPayload := OrderToJSONWithPostOnly(order, "api-key", OrderTypeGTC, true)
	if postOnlyPayload["postOnly"] != true {
		t.Fatalf("unexpected postOnly: %v", postOnlyPayload["postOnly"])
	}
}

// Expected digests were computed independently from the EIP-712 type strings and ABI words.
func TestMarketVersionOrderDomains(t *testing.T) {
	client, err := NewClobClient("", Polygon, testPrivateKey, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	const standardHash = "0x9c26ee23f073ffd4801ed92fd2aa6bb046616fb615b6a9872bc60d7637d0a03e"
	const negRiskHash = "0x733ab60523eba17a543470f0786a4c1d762983851256fb84cb5165ac06db1f59"
	const exchangeV3Hash = "0x6c0e1c4a6f70b1b25f67c1f2b8e45e3f04392279efccf90d14ea52ec83d28ab0"
	tick := TickSize001
	client.tickSizes["123"] = tick
	standard, negRisk := false, true
	for _, tc := range []struct {
		name    string
		version MarketVersion
		negRisk *bool
		want    string
	}{
		{"legacy default", "", &standard, standardHash},
		{"v1 standard", MarketVersionV1, &standard, standardHash},
		{"v1 neg risk", MarketVersionV1, &negRisk, negRiskHash},
		{"v2", MarketVersionV2, nil, exchangeV3Hash},
		{"v2 neg risk", MarketVersionV2, &negRisk, exchangeV3Hash},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := &PartialCreateOrderOptions{MarketVersion: tc.version, NegRisk: tc.negRisk, TickSize: &tick, RawOrder: true}
			digest, err := client.GetOrderHash(testOrderV2(), opts)
			if err != nil || digest.Hex() != tc.want {
				t.Fatalf("order hash = %s, err = %v, want %s", digest.Hex(), err, tc.want)
			}
			limit, err := client.CreateOrder(&OrderArgs{TokenID: "123", Price: 0.5, Size: 2, Side: BUY}, opts)
			if err != nil {
				t.Fatal(err)
			}
			market, err := client.CreateMarketOrder(&MarketOrderArgs{TokenID: "123", Price: 0.5, Amount: 1, Side: SELL}, opts)
			if err != nil {
				t.Fatal(err)
			}
			for _, order := range []*SignedOrder{limit, market} {
				digest, err := client.GetOrderHash(order, opts)
				if err != nil {
					t.Fatal(err)
				}
				ok, err := ordersigner.ValidateSignature(order.Signer, digest, order.Signature)
				if !ok || err != nil {
					t.Fatalf("signature verification = %v, err = %v", ok, err)
				}
			}
		})
	}
	if _, err := client.GetOrderHash(testOrderV2(), &PartialCreateOrderOptions{MarketVersion: "v3"}); err == nil {
		t.Fatal("accepted an unsupported market version")
	}
}

func TestMarketOrderHashResolvesNegRiskWithRawOptions(t *testing.T) {
	client, err := NewClobClient("", Polygon, testPrivateKey, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	tick := TickSize001
	client.tickSizes["123"] = tick
	client.negRisk["123"] = true
	opts := &PartialCreateOrderOptions{MarketVersion: MarketVersionV1, TickSize: &tick, RawOrder: true}
	order, err := client.CreateMarketOrder(&MarketOrderArgs{TokenID: "123", Price: 0.5, Amount: 1, Side: SELL}, opts)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := client.GetOrderHash(order, opts)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := ordersigner.ValidateSignature(order.Signer, digest, order.Signature); !ok || err != nil {
		t.Fatalf("signature verification = %v, err = %v", ok, err)
	}
	if _, err := client.CreateOrder(&OrderArgs{TokenID: "123", Price: 0.5, Size: 2, Side: BUY}, opts); err == nil {
		t.Fatal("raw CTF limit order accepted missing NegRisk")
	}
}

func TestOrderHashRejectsOutOfRangeIntegers(t *testing.T) {
	client := &ClobClient{chainID: Polygon}
	for _, mutate := range []func(*SignedOrder){
		func(o *SignedOrder) { o.Side.SetUint64(256) },
		func(o *SignedOrder) { o.SignatureType.SetInt64(-1) },
		func(o *SignedOrder) { o.Salt.Lsh(big.NewInt(1), 256) },
		func(o *SignedOrder) { o.MakerAmount.SetInt64(-1) },
	} {
		order := testOrderV2()
		mutate(order)
		if _, err := client.GetOrderHash(order, &PartialCreateOrderOptions{MarketVersion: MarketVersionV2}); err == nil {
			t.Fatal("accepted an integer outside its EIP-712 field width")
		}
	}
}

func TestParseBytes32RejectsInvalidBuilderCode(t *testing.T) {
	if _, err := parseBytes32("builder", "0x1234"); err == nil {
		t.Fatal("expected length error")
	}
	if _, err := parseBytes32("builder", "0xzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"); err == nil {
		t.Fatal("expected hex error")
	}
}
