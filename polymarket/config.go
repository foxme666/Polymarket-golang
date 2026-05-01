package polymarket

const (
	PolygonCTFExchangeV2        = "0xE111180000d2663C0091e4f400237545B87B996B"
	PolygonNegRiskExchangeV2    = "0xe2222d279d744050d28e00520010520000310F59"
	PolygonConditionalTokens    = "0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"
	PolygonPUSD                 = "0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB"
	PolygonUSDCE                = "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"
	PolygonCollateralOnramp     = "0x93070a847efEf7F70739046A929D47a521F5B8ee"
	PolygonCollateralOfframp    = "0x2957922Eb93258b93368531d39fAcCA3B4dC5854"
	PolygonCtfCollateralAdapter = "0xADa100874d00e3331D00F2007a9c336a65009718"
	PolygonNegRiskCtfAdapter    = "0xAdA200001000ef00D07553cEE7006808F895c6F1"
	PolygonNegRiskAdapterV2     = "0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296"
)

// getContractConfig 获取链的合约配置
func getContractConfig(chainID int, negRisk bool) *ContractConfig {
	// 标准配置
	config := map[int]*ContractConfig{
		137: { // Polygon
			Exchange:          PolygonCTFExchangeV2,
			Collateral:        PolygonPUSD,
			ConditionalTokens: PolygonConditionalTokens,
		},
		80002: { // Amoy
			Exchange:          "0xdFE02Eb6733538f8Ea35D585af8DE5958AD99E40",
			Collateral:        "0x9c4e1703476e875070ee25b56a58b008cfb8fa78",
			ConditionalTokens: "0x69308FB512518e39F9b16112fA8d994F4e2Bf8bB",
		},
	}

	// 负风险配置
	negRiskConfig := map[int]*ContractConfig{
		137: { // Polygon
			Exchange:          PolygonNegRiskExchangeV2,
			Collateral:        PolygonPUSD,
			ConditionalTokens: PolygonConditionalTokens,
		},
		80002: { // Amoy
			Exchange:          "0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296",
			Collateral:        "0x9c4e1703476e875070ee25b56a58b008cfb8fa78",
			ConditionalTokens: "0x69308FB512518e39F9b16112fA8d994F4e2Bf8bB",
		},
	}

	if negRisk {
		cfg := negRiskConfig[chainID]
		if cfg == nil {
			panic("Invalid chainID for neg risk: " + string(rune(chainID)))
		}
		return cfg
	}

	cfg := config[chainID]
	if cfg == nil {
		panic("Invalid chainID: " + string(rune(chainID)))
	}
	return cfg
}
