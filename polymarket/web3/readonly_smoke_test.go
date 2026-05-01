//go:build smoke

package web3

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestReadOnlyPolygonContractsSmoke(t *testing.T) {
	cfg, ok := chainConfigs[137]
	if !ok {
		t.Fatal("missing polygon chain config")
	}

	addresses := map[string]string{
		"exchange_v2":            cfg.Exchange.Hex(),
		"neg_risk_exchange_v2":   cfg.NegRiskExchange.Hex(),
		"p_usd":                  cfg.Collateral.Hex(),
		"usdc_e":                 cfg.USDCE.Hex(),
		"conditional_tokens":     cfg.ConditionalTokens.Hex(),
		"ctf_collateral_adapter": CtfCollateralAdapterAddress.Hex(),
		"neg_risk_ctf_adapter":   NegRiskCtfCollateralAdapterAddress.Hex(),
	}

	rpcURLs := polygonSmokeRPCURLs()
	var failures []string
	for _, rpcURL := range rpcURLs {
		if err := assertContractCodes(rpcURL, addresses, t); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", rpcURL, err))
			continue
		}
		return
	}

	t.Fatalf("all polygon RPC smoke attempts failed:\n%s", strings.Join(failures, "\n"))
}

func polygonSmokeRPCURLs() []string {
	if rpcURL := os.Getenv("RPC_URL"); rpcURL != "" {
		return []string{rpcURL}
	}
	return []string{
		"https://polygon-bor-rpc.publicnode.com",
		"https://1rpc.io/matic",
		DefaultPolygonRPC,
	}
}

func assertContractCodes(rpcURL string, addresses map[string]string, t *testing.T) error {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("ethclient.Dial: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	for name, addr := range addresses {
		code, err := client.CodeAt(ctx, common.HexToAddress(addr), nil)
		if err != nil {
			return fmt.Errorf("CodeAt(%s=%s): %w", name, addr, err)
		}
		if len(code) == 0 {
			return fmt.Errorf("CodeAt(%s=%s): empty code", name, addr)
		}
		t.Logf("%s via %s %s code_size=%d", name, rpcURL, addr, len(code))
	}

	return nil
}
