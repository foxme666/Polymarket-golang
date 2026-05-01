//go:build smoke

package polymarket

import (
	"fmt"
	"os"
	"testing"
)

func TestReadOnlyClobSmoke(t *testing.T) {
	host := os.Getenv("POLY_CLOB_HOST")
	if host == "" {
		host = "https://clob.polymarket.com"
	}

	client, err := NewClobClient(host, Polygon, "", nil, nil, "")
	if err != nil {
		t.Fatalf("NewClobClient: %v", err)
	}

	conditionID, tokenID := os.Getenv("POLY_SMOKE_CONDITION_ID"), os.Getenv("POLY_SMOKE_TOKEN_ID")
	if conditionID == "" || tokenID == "" {
		var err error
		conditionID, tokenID, err = findSmokeMarketToken(client)
		if err != nil {
			t.Fatalf("find smoke market token: %v", err)
		}
	}
	t.Logf("using condition_id=%s token_id=%s", conditionID, tokenID)

	info, err := client.GetClobMarketInfo(conditionID)
	if err != nil {
		t.Fatalf("GetClobMarketInfo(%s): %v", conditionID, err)
	}
	if len(info.Tokens) == 0 {
		t.Fatalf("GetClobMarketInfo(%s): no tokens returned", conditionID)
	}

	tickSize, err := client.GetTickSize(tokenID)
	if err != nil {
		t.Fatalf("GetTickSize(%s): %v", tokenID, err)
	}
	if tickSize == "" {
		t.Fatalf("GetTickSize(%s): empty tick size", tokenID)
	}

	_, err = client.GetNegRisk(tokenID)
	if err != nil {
		t.Fatalf("GetNegRisk(%s): %v", tokenID, err)
	}

	book, err := client.GetOrderBook(tokenID)
	if err != nil {
		t.Fatalf("GetOrderBook(%s): %v", tokenID, err)
	}
	if book.AssetID == "" {
		t.Fatalf("GetOrderBook(%s): empty asset id", tokenID)
	}

	books, err := client.GetOrderBooks([]BookParams{{TokenID: tokenID}})
	if err != nil {
		t.Fatalf("GetOrderBooks(%s): %v", tokenID, err)
	}
	if len(books) != 1 || books[0] == nil || books[0].AssetID == "" {
		t.Fatalf("GetOrderBooks(%s): invalid response %#v", tokenID, books)
	}
}

func findSmokeMarketToken(client *ClobClient) (string, string, error) {
	resp, err := client.GetSamplingMarkets("MA==")
	if err == nil {
		if respMap, ok := resp.(map[string]interface{}); ok {
			if conditionID, tokenID, ok := firstOrderableMarketToken(respMap); ok {
				return conditionID, tokenID, nil
			}
		}
	}

	nextCursor := "MA=="
	for page := 0; page < 10 && nextCursor != ""; page++ {
		resp, err := client.GetMarkets(nextCursor)
		if err != nil {
			return "", "", err
		}

		respMap, ok := resp.(map[string]interface{})
		if !ok {
			return "", "", fmt.Errorf("invalid markets response")
		}

		if conditionID, tokenID, ok := firstOrderableMarketToken(respMap); ok {
			return conditionID, tokenID, nil
		}

		nextCursor = getStringFromMap(respMap, "next_cursor")
		if nextCursor == "LTE=" {
			break
		}
	}

	return "", "", fmt.Errorf("no active orderable market found in first pages")
}

func firstOrderableMarketToken(respMap map[string]interface{}) (string, string, bool) {
	data, ok := respMap["data"].([]interface{})
	if !ok {
		return "", "", false
	}

	for _, item := range data {
		market, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if !getBoolFromMap(market, "active") || getBoolFromMap(market, "closed") || !getBoolFromMap(market, "accepting_orders") {
			continue
		}

		conditionID := getStringFromMap(market, "condition_id")
		tokens, ok := market["tokens"].([]interface{})
		if conditionID == "" || !ok {
			continue
		}
		for _, tokenRaw := range tokens {
			token, ok := tokenRaw.(map[string]interface{})
			if !ok {
				continue
			}
			tokenID := getStringFromMap(token, "token_id")
			if tokenID != "" {
				return conditionID, tokenID, true
			}
		}
	}

	return "", "", false
}
