package types

import sdkErrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	DefaultCodespace = "ixo"
	ErrInternal      = sdkErrors.Register(DefaultCodespace, 114, "not allowed format")
)