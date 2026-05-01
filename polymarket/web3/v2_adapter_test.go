package web3

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestPolygonChainConfigUsesV2CollateralAddresses(t *testing.T) {
	cfg := chainConfigs[137]
	if cfg.Exchange != common.HexToAddress("0xE111180000d2663C0091e4f400237545B87B996B") {
		t.Fatalf("exchange = %s", cfg.Exchange.Hex())
	}
	if cfg.NegRiskExchange != common.HexToAddress("0xe2222d279d744050d28e00520010520000310F59") {
		t.Fatalf("neg risk exchange = %s", cfg.NegRiskExchange.Hex())
	}
	if cfg.Collateral != common.HexToAddress("0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB") {
		t.Fatalf("collateral = %s", cfg.Collateral.Hex())
	}
	if cfg.USDCE != common.HexToAddress("0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174") {
		t.Fatalf("usdce = %s", cfg.USDCE.Hex())
	}
}

func TestCtfAdapterAddressSelection(t *testing.T) {
	client := &BaseWeb3Client{
		CtfCollateralAdapter: CtfCollateralAdapterAddress,
		NegRiskCtfAdapter:    NegRiskCtfCollateralAdapterAddress,
	}
	if got := client.ctfAdapterAddress(false); got != CtfCollateralAdapterAddress {
		t.Fatalf("standard adapter = %s", got.Hex())
	}
	if got := client.ctfAdapterAddress(true); got != NegRiskCtfCollateralAdapterAddress {
		t.Fatalf("neg risk adapter = %s", got.Hex())
	}
}

func TestEncodeMergeUsesPUSDCollateralForAdapterSignature(t *testing.T) {
	client := &BaseWeb3Client{
		CollateralAddress: common.HexToAddress("0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB"),
	}
	conditionID := common.HexToHash("0xabc")

	data, err := client.encodeMerge(conditionID, big.NewInt(1_000_000))
	if err != nil {
		t.Fatalf("encode merge: %v", err)
	}

	args, err := ConditionalTokensABI.Methods["mergePositions"].Inputs.Unpack(data[4:])
	if err != nil {
		t.Fatalf("unpack merge: %v", err)
	}
	if args[0].(common.Address) != client.CollateralAddress {
		t.Fatalf("collateral arg = %s", args[0].(common.Address).Hex())
	}
	if args[2].([32]byte) != conditionID {
		t.Fatalf("condition id mismatch")
	}
	if args[4].(*big.Int).Cmp(big.NewInt(1_000_000)) != 0 {
		t.Fatalf("amount = %s", args[4].(*big.Int).String())
	}
}
