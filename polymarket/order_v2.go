package polymarket

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/polymarket/go-order-utils/pkg/eip712"
	ordersigner "github.com/polymarket/go-order-utils/pkg/signer"
)

var (
	orderV2ProtocolName    = crypto.Keccak256Hash([]byte("Polymarket CTF Exchange"))
	orderV2ProtocolVersion = crypto.Keccak256Hash([]byte("2"))
	orderV2StructureHash   = crypto.Keccak256Hash([]byte("Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)"))
	orderV2Structure       = []abi.Type{
		eip712.Bytes32, // typehash
		eip712.Uint256, // salt
		eip712.Address, // maker
		eip712.Address, // signer
		eip712.Uint256, // tokenId
		eip712.Uint256, // makerAmount
		eip712.Uint256, // takerAmount
		eip712.Uint8,   // side
		eip712.Uint8,   // signatureType
		eip712.Uint256, // timestamp
		eip712.Bytes32, // metadata
		eip712.Bytes32, // builder
	}
)

func buildSignedOrderV2(privateKey *ecdsa.PrivateKey, data *OrderDataV2, exchangeAddr string, chainID int) (*SignedOrder, error) {
	order, err := buildOrderV2(data)
	if err != nil {
		return nil, err
	}

	orderHash, err := buildOrderHashV2(order, exchangeAddr, chainID)
	if err != nil {
		return nil, err
	}

	signature, err := crypto.Sign(orderHash.Bytes(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign V2 order: %w", err)
	}
	signature[64] += 27

	ok, err := ordersigner.ValidateSignature(order.Signer, orderHash, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to validate V2 order signature: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("V2 order signature validation failed")
	}

	order.Signature = signature
	return order, nil
}

func buildOrderV2(data *OrderDataV2) (*SignedOrder, error) {
	if data == nil {
		return nil, fmt.Errorf("order data is required")
	}

	tokenID, err := parseBigInt("TokenID", data.TokenID)
	if err != nil {
		return nil, err
	}
	makerAmount, err := parseBigInt("MakerAmount", data.MakerAmount)
	if err != nil {
		return nil, err
	}
	takerAmount, err := parseBigInt("TakerAmount", data.TakerAmount)
	if err != nil {
		return nil, err
	}
	metadata, err := parseBytes32("metadata", data.Metadata)
	if err != nil {
		return nil, err
	}
	builder, err := parseBytes32("builder", data.Builder)
	if err != nil {
		return nil, err
	}
	salt, err := generateOrderSalt()
	if err != nil {
		return nil, err
	}

	timestamp := data.Timestamp
	if timestamp == 0 {
		timestamp = time.Now().UnixMilli()
	}

	signer := common.HexToAddress(data.Signer)
	if data.Signer == "" {
		signer = common.HexToAddress(data.Maker)
	}

	return &SignedOrder{
		Salt:          salt,
		Maker:         common.HexToAddress(data.Maker),
		Signer:        signer,
		Taker:         common.HexToAddress(ZeroAddress),
		TokenId:       tokenID,
		MakerAmount:   makerAmount,
		TakerAmount:   takerAmount,
		Expiration:    big.NewInt(int64(data.Expiration)),
		Nonce:         big.NewInt(0),
		FeeRateBps:    big.NewInt(0),
		Side:          big.NewInt(int64(data.Side)),
		SignatureType: big.NewInt(int64(data.SignatureType)),
		Timestamp:     big.NewInt(timestamp),
		Metadata:      metadata,
		Builder:       builder,
	}, nil
}

func buildOrderHashV2(order *SignedOrder, exchangeAddr string, chainID int) (common.Hash, error) {
	if order == nil {
		return common.Hash{}, fmt.Errorf("order is required")
	}
	domainSeparator, err := eip712.BuildEIP712DomainSeparator(
		orderV2ProtocolName,
		orderV2ProtocolVersion,
		big.NewInt(int64(chainID)),
		common.HexToAddress(exchangeAddr),
	)
	if err != nil {
		return common.Hash{}, err
	}

	values := []interface{}{
		orderV2StructureHash,
		order.Salt,
		order.Maker,
		order.Signer,
		order.TokenId,
		order.MakerAmount,
		order.TakerAmount,
		uint8(order.Side.Uint64()),
		uint8(order.SignatureType.Uint64()),
		order.Timestamp,
		order.Metadata,
		order.Builder,
	}

	return eip712.HashTypedDataV4(domainSeparator, orderV2Structure, values)
}

func parseBigInt(name, value string) (*big.Int, error) {
	out, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("can't parse %s: %s as valid *big.Int", name, value)
	}
	return out, nil
}

func parseBytes32(name, value string) (common.Hash, error) {
	if value == "" {
		return common.Hash{}, nil
	}
	value = strings.TrimPrefix(value, "0x")
	if len(value) != 64 {
		return common.Hash{}, fmt.Errorf("%s must be a 32-byte hex string", name)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return common.Hash{}, fmt.Errorf("%s must be a hex string: %w", name, err)
	}
	return common.HexToHash("0x" + value), nil
}

func generateOrderSalt() (*big.Int, error) {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(math.Pow(2, 32))))
	if err != nil {
		return nil, err
	}
	return nBig, nil
}
