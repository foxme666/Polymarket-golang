package polymarket

import "testing"

func TestGetContractConfigUsesPolygonV2Addresses(t *testing.T) {
	standard := getContractConfig(Polygon, false)
	if standard.Exchange != PolygonCTFExchangeV2 {
		t.Fatalf("standard exchange = %s", standard.Exchange)
	}
	if standard.Collateral != PolygonPUSD {
		t.Fatalf("standard collateral = %s", standard.Collateral)
	}

	negRisk := getContractConfig(Polygon, true)
	if negRisk.Exchange != PolygonNegRiskExchangeV2 {
		t.Fatalf("neg risk exchange = %s", negRisk.Exchange)
	}
	if negRisk.Collateral != PolygonPUSD {
		t.Fatalf("neg risk collateral = %s", negRisk.Collateral)
	}
}

func TestGetClobMarketInfoCachesV2MarketParams(t *testing.T) {
	client, err := NewClobClient("https://clob.polymarket.com", Polygon, "", nil, nil, "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	info, err := parseClobMarketInfo("0xcondition", map[string]interface{}{
		"gst":  nil,
		"r":    map[string]interface{}{"min_size": float64(10)},
		"t":    []interface{}{map[string]interface{}{"t": "111", "o": "Yes"}, map[string]interface{}{"t": "222", "o": "No"}},
		"mos":  float64(5),
		"mts":  float64(0.01),
		"mbf":  float64(0),
		"tbf":  float64(20),
		"nr":   true,
		"rfqe": false,
		"fd":   map[string]interface{}{"r": float64(0.02), "e": float64(2), "to": true},
	})
	if err != nil {
		t.Fatalf("market info: %v", err)
	}
	client.cacheClobMarketInfo("0xcondition", info)

	if len(info.Tokens) != 2 {
		t.Fatalf("token count = %d", len(info.Tokens))
	}
	if info.MinimumTickSize != 0.01 {
		t.Fatalf("tick size = %v", info.MinimumTickSize)
	}
	if info.FeeDetails == nil || info.FeeDetails.Rate != 0.02 || info.FeeDetails.Exponent != 2 || !info.FeeDetails.TakerOnly {
		t.Fatalf("fee details = %#v", info.FeeDetails)
	}

	tick, err := client.GetTickSize("111")
	if err != nil {
		t.Fatalf("cached tick: %v", err)
	}
	if tick != TickSize001 {
		t.Fatalf("cached tick = %s", tick)
	}

	negRisk, err := client.GetNegRisk("222")
	if err != nil {
		t.Fatalf("cached neg risk: %v", err)
	}
	if !negRisk {
		t.Fatal("expected cached neg risk")
	}
}
