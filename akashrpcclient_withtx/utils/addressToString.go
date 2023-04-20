// Logic copied from https://github.com/cosmos/cosmos-sdk/blob/main/types/address.go
// Code implemented locally to not rely on GetConfig().GetBech32AccountAddrPrefix() which by default expects a cosmos prefixed address

package utils

import (
	"unsafe"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/hashicorp/golang-lru/simplelru"
)

var accAddrCache *simplelru.LRU

func String(aa types.AccAddress) string {
	if aa.Empty() {
		return ""
	}

	key := UnsafeBytesToStr(aa)

	return cacheBech32Addr("akash", aa, accAddrCache, key)
}

// cacheBech32Addr is not concurrency safe. Concurrent access to cache causes race condition.
func cacheBech32Addr(prefix string, addr types.AccAddress, cache *simplelru.LRU, cacheKey string) string {
	bech32Addr, err := bech32.ConvertAndEncode(prefix, addr)
	if err != nil {
		panic(err)
	}

	return bech32Addr
}

func UnsafeBytesToStr(b types.AccAddress) string {
	return *(*string)(unsafe.Pointer(&b))
}
