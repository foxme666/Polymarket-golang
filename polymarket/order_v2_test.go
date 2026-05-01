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
	}, PolygonCTFExchangeV2, 137)
	if err != nil {
		t.Fatalf("build signed order: %v", err)
	}

	orderHash, err := buildOrderHashV2(order, PolygonCTFExchangeV2, 137)
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

func TestOrderToJSONWithPostOnlyUsesV2WireShape(t *testing.T) {
	order := &SignedOrder{
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

	payload := OrderToJSON(order, "api-key", OrderTypeGTC)
	orderPayload, ok := payload["order"].(map[string]interface{})
	if !ok {
		t.Fatalf("order payload type = %T", payload["order"])
	}

	for _, removed := range []string{"taker", "nonce", "feeRateBps"} {
		if _, exists := orderPayload[removed]; exists {
			t.Fatalf("V2 order JSON must not contain %s", removed)
		}
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
	if _, exists := payload["postOnly"]; exists {
		t.Fatalf("V2 default order payload must not contain postOnly=false")
	}

	postOnlyPayload := OrderToJSONWithPostOnly(order, "api-key", OrderTypeGTC, true)
	if postOnlyPayload["postOnly"] != true {
		t.Fatalf("unexpected postOnly: %v", postOnlyPayload["postOnly"])
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
