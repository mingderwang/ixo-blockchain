package bonds

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ixofoundation/ixo-blockchain/x/bonds/internal/keeper"
	"github.com/ixofoundation/ixo-blockchain/x/bonds/internal/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"strconv"
	"strings"
)

func NewHandler(keeper keeper.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case types.MsgCreateBond:
			return handleMsgCreateBond(ctx, keeper, msg)
		case types.MsgEditBond:
			return handleMsgEditBond(ctx, keeper, msg)
		case types.MsgBuy:
			return handleMsgBuy(ctx, keeper, msg)
		case types.MsgSell:
			return handleMsgSell(ctx, keeper, msg)
		case types.MsgSwap:
			return handleMsgSwap(ctx, keeper, msg)
		case types.MsgMakeOutcomePayment:
			return handleMsgMakeOutcomePayment(ctx, keeper, msg)
		case types.MsgWithdrawShare:
			return handleMsgWithdrawShare(ctx, keeper, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized bonds Msg type: %v", msg.Type())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func EndBlocker(ctx sdk.Context, keeper keeper.Keeper) []abci.ValidatorUpdate {

	iterator := keeper.GetBondIterator(ctx)
	for ; iterator.Valid(); iterator.Next() {
		bond := keeper.MustGetBondByKey(ctx, iterator.Key())
		batch := keeper.MustGetBatch(ctx, bond.BondDid)

		// Subtract one block
		batch.BlocksRemaining = batch.BlocksRemaining.SubUint64(1)
		keeper.SetBatch(ctx, bond.BondDid, batch)

		// If blocks remaining > 0 do not perform orders
		if !batch.BlocksRemaining.IsZero() {
			continue
		}

		// Perform orders
		keeper.PerformOrders(ctx, bond.BondDid)

		// Get bond again just in case current supply was updated
		// Get batch again just in case orders were cancelled
		bond = keeper.MustGetBond(ctx, bond.BondDid)
		batch = keeper.MustGetBatch(ctx, bond.BondDid)

		// For augmented, if hatch phase and newSupply >= S0, go to open phase
		if bond.FunctionType == types.AugmentedFunction &&
			bond.State == types.HatchState {
			args := bond.FunctionParameters.AsMap()
			if bond.CurrentSupply.Amount.ToDec().GTE(args["S0"]) {
				keeper.SetBondState(ctx, bond.BondDid, types.OpenState)
				bond = keeper.MustGetBond(ctx, bond.BondDid) // get bond again
				bond.AllowSells = true                       // enable sells
				keeper.SetBond(ctx, bond.BondDid, bond)      // update bond
			}
		}

		// Save current batch as last batch and reset current batch
		keeper.SetLastBatch(ctx, bond.BondDid, batch)
		keeper.SetBatch(ctx, bond.BondDid, types.NewBatch(bond.BondDid, bond.Token, bond.BatchBlocks))
	}
	return []abci.ValidatorUpdate{}
}

func handleMsgCreateBond(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgCreateBond) sdk.Result {
	if keeper.BankKeeper.BlacklistedAddr(msg.FeeAddress) {
		return sdk.ErrUnauthorized(fmt.Sprintf("%s is not allowed to receive transactions", msg.FeeAddress)).Result()
	}

	// Check that bond and bond DID do not already exist
	if keeper.BondExists(ctx, msg.BondDid) {
		return types.ErrBondAlreadyExists(DefaultCodespace, msg.BondDid).Result()
	} else if keeper.BondDidExists(ctx, msg.Token) {
		return types.ErrBondTokenIsTaken(DefaultCodespace, msg.Token).Result()
	}

	// Check that bond token not reserved
	if keeper.ReservedBondToken(ctx, msg.Token) {
		return types.ErrReservedBondToken(DefaultCodespace, msg.Token).Result()
	}

	// Set state to open by default (overridden below if augmented function)
	state := types.OpenState

	// If augmented, add R0, S0, V0 as parameters for quick access
	// Also, override AllowSells and set to False if S0 > 0
	if msg.FunctionType == types.AugmentedFunction {
		paramsMap := msg.FunctionParameters.AsMap()
		d0, _ := paramsMap["d0"]
		p0, _ := paramsMap["p0"]
		theta, _ := paramsMap["theta"]
		kappa, _ := paramsMap["kappa"]

		R0 := d0.Mul(sdk.OneDec().Sub(theta))
		S0 := d0.Quo(p0)
		V0 := types.Invariant(R0, S0, kappa.TruncateInt64())
		// TODO: consider calculating these on-the-fly, especially R0 and S0

		msg.FunctionParameters = append(msg.FunctionParameters,
			types.FunctionParams{
				types.NewFunctionParam("R0", R0),
				types.NewFunctionParam("S0", S0),
				types.NewFunctionParam("V0", V0),
			}...)

		// Set state to Hatch and disable sells. Note that it is never the case
		// that we start with OpenState because S0>0, since S0=d0/p0 and d0>0
		state = types.HatchState
		msg.AllowSells = false
	}

	bond := types.NewBond(msg.Token, msg.Name, msg.Description, msg.CreatorDid,
		msg.FunctionType, msg.FunctionParameters, msg.ReserveTokens,
		msg.TxFeePercentage, msg.ExitFeePercentage, msg.FeeAddress,
		msg.MaxSupply, msg.OrderQuantityLimits, msg.SanityRate,
		msg.SanityMarginPercentage, msg.AllowSells, msg.BatchBlocks,
		msg.OutcomePayment, state, msg.BondDid)

	keeper.SetBond(ctx, bond.BondDid, bond)
	keeper.SetBondDid(ctx, bond.Token, bond.BondDid)
	keeper.SetBatch(ctx, bond.BondDid, types.NewBatch(bond.BondDid, bond.Token, msg.BatchBlocks))

	logger := keeper.Logger(ctx)
	logger.Info(fmt.Sprintf("bond %s [%s] with reserve(s) [%s] created by %s", msg.Token,
		msg.FunctionType, strings.Join(bond.ReserveTokens, ","), msg.CreatorDid))

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateBond,
			sdk.NewAttribute(types.AttributeKeyBondDid, msg.BondDid),
			sdk.NewAttribute(types.AttributeKeyToken, msg.Token),
			sdk.NewAttribute(types.AttributeKeyName, msg.Name),
			sdk.NewAttribute(types.AttributeKeyDescription, msg.Description),
			sdk.NewAttribute(types.AttributeKeyFunctionType, msg.FunctionType),
			sdk.NewAttribute(types.AttributeKeyFunctionParameters, msg.FunctionParameters.String()),
			sdk.NewAttribute(types.AttributeKeyReserveTokens, types.StringsToString(msg.ReserveTokens)),
			sdk.NewAttribute(types.AttributeKeyTxFeePercentage, msg.TxFeePercentage.String()),
			sdk.NewAttribute(types.AttributeKeyExitFeePercentage, msg.ExitFeePercentage.String()),
			sdk.NewAttribute(types.AttributeKeyFeeAddress, msg.FeeAddress.String()),
			sdk.NewAttribute(types.AttributeKeyMaxSupply, msg.MaxSupply.String()),
			sdk.NewAttribute(types.AttributeKeyOrderQuantityLimits, msg.OrderQuantityLimits.String()),
			sdk.NewAttribute(types.AttributeKeySanityRate, msg.SanityRate.String()),
			sdk.NewAttribute(types.AttributeKeySanityMarginPercentage, msg.SanityMarginPercentage.String()),
			sdk.NewAttribute(types.AttributeKeyAllowSells, strconv.FormatBool(msg.AllowSells)),
			sdk.NewAttribute(types.AttributeKeyBatchBlocks, msg.BatchBlocks.String()),
			sdk.NewAttribute(types.AttributeKeyOutcomePayment, msg.OutcomePayment.String()),
			sdk.NewAttribute(types.AttributeKeyState, state),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.CreatorDid),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func handleMsgEditBond(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgEditBond) sdk.Result {

	bond, found := keeper.GetBond(ctx, msg.BondDid)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, msg.BondDid).Result()
	}

	if bond.CreatorDid != msg.EditorDid {
		errMsg := fmt.Sprintf("Editor must be the creator of the bond")
		return sdk.ErrInternal(errMsg).Result()
	}

	if msg.Name != types.DoNotModifyField {
		bond.Name = msg.Name
	}
	if msg.Description != types.DoNotModifyField {
		bond.Description = msg.Description
	}

	if msg.OrderQuantityLimits != types.DoNotModifyField {
		orderQuantityLimits, err := sdk.ParseCoins(msg.OrderQuantityLimits)
		if err != nil {
			return sdk.ErrInvalidCoins(err.Error()).Result()
		}
		bond.OrderQuantityLimits = orderQuantityLimits
	}

	if msg.SanityRate != types.DoNotModifyField {
		var sanityRate, sanityMarginPercentage sdk.Dec
		if msg.SanityRate == "" {
			sanityRate = sdk.ZeroDec()
			sanityMarginPercentage = sdk.ZeroDec()
		} else {
			parsedSanityRate, err := sdk.NewDecFromStr(msg.SanityRate)
			if err != nil {
				return types.ErrArgumentMissingOrNonFloat(types.DefaultCodespace, "sanity rate").Result()
			} else if parsedSanityRate.IsNegative() {
				return types.ErrArgumentCannotBeNegative(types.DefaultCodespace, "sanity rate").Result()
			}
			parsedSanityMarginPercentage, err := sdk.NewDecFromStr(msg.SanityMarginPercentage)
			if err != nil {
				return types.ErrArgumentMissingOrNonFloat(types.DefaultCodespace, "sanity margin percentage").Result()
			} else if parsedSanityMarginPercentage.IsNegative() {
				return types.ErrArgumentCannotBeNegative(types.DefaultCodespace, "sanity margin percentage").Result()
			}
			sanityRate = parsedSanityRate
			sanityMarginPercentage = parsedSanityMarginPercentage
		}
		bond.SanityRate = sanityRate
		bond.SanityMarginPercentage = sanityMarginPercentage
	}

	logger := keeper.Logger(ctx)
	logger.Info(fmt.Sprintf("bond %s edited by %s",
		msg.BondDid, msg.EditorDid))

	keeper.SetBond(ctx, bond.BondDid, bond)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeEditBond,
			sdk.NewAttribute(types.AttributeKeyBondDid, msg.BondDid),
			sdk.NewAttribute(types.AttributeKeyToken, msg.Token),
			sdk.NewAttribute(types.AttributeKeyName, msg.Name),
			sdk.NewAttribute(types.AttributeKeyDescription, msg.Description),
			sdk.NewAttribute(types.AttributeKeyOrderQuantityLimits, msg.OrderQuantityLimits),
			sdk.NewAttribute(types.AttributeKeySanityRate, msg.SanityRate),
			sdk.NewAttribute(types.AttributeKeySanityMarginPercentage, msg.SanityMarginPercentage),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.EditorDid),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func handleMsgBuy(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgBuy) sdk.Result {
	buyerAddr := keeper.DidKeeper.MustGetDidDoc(ctx, msg.BuyerDid).Address()

	bond, found := keeper.GetBond(ctx, msg.BondDid)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, msg.BondDid).Result()
	}

	// Check that bond token used belongs to this bond
	if msg.Amount.Denom != bond.Token {
		return types.ErrBondTokenDoesNotMatchBond(types.DefaultCodespace).Result()
	}

	// Check current state is HATCH/OPEN, max prices, order quantity limits
	if bond.State != types.OpenState && bond.State != types.HatchState {
		return types.ErrInvalidStateForAction(types.DefaultCodespace).Result()
	} else if !bond.ReserveDenomsEqualTo(msg.MaxPrices) {
		return types.ErrReserveDenomsMismatch(types.DefaultCodespace, msg.MaxPrices.String(), bond.ReserveTokens).Result()
	} else if bond.AnyOrderQuantityLimitsExceeded(sdk.Coins{msg.Amount}) {
		return types.ErrOrderQuantityLimitExceeded(types.DefaultCodespace).Result()
	}

	// For the swapper, the first buy is the initialisation of the reserves
	// The max prices are used as the actual prices and one token is minted
	// The amount of token serves to define the price of adding more liquidity
	if bond.CurrentSupply.IsZero() && bond.FunctionType == types.SwapperFunction {
		return performFirstSwapperFunctionBuy(ctx, keeper, msg)
	}

	// Take max that buyer is willing to pay (enforces maxPrice <= balance)
	err := keeper.SupplyKeeper.SendCoinsFromAccountToModule(ctx, buyerAddr,
		types.BatchesIntermediaryAccount, msg.MaxPrices)
	if err != nil {
		return err.Result()
	}

	// Create order
	order := types.NewBuyOrder(msg.BuyerDid, msg.Amount, msg.MaxPrices)

	// Get buy price and check if can add buy order to batch
	buyPrices, sellPrices, err := keeper.GetUpdatedBatchPricesAfterBuy(ctx, bond.BondDid, order)
	if err != nil {
		return err.Result()
	}

	// Add buy order to batch
	keeper.AddBuyOrder(ctx, bond.BondDid, order, buyPrices, sellPrices)

	// Cancel unfulfillable orders
	keeper.CancelUnfulfillableOrders(ctx, bond.BondDid)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeBuy,
			sdk.NewAttribute(types.AttributeKeyBondDid, msg.BondDid),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyMaxPrices, msg.MaxPrices.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.BuyerDid),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func performFirstSwapperFunctionBuy(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgBuy) sdk.Result {
	buyerAddr := keeper.DidKeeper.MustGetDidDoc(ctx, msg.BuyerDid).Address()

	// TODO: investigate effect that a high amount has on future buyers' ability to buy.

	bond, found := keeper.GetBond(ctx, msg.BondDid)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, msg.BondDid).Result()
	}

	// Check that bond token used belongs to this bond
	if msg.Amount.Denom != bond.Token {
		return types.ErrBondTokenDoesNotMatchBond(types.DefaultCodespace).Result()
	}

	// Check if initial liquidity violates sanity rate
	if bond.ReservesViolateSanityRate(msg.MaxPrices) {
		return types.ErrValuesViolateSanityRate(types.DefaultCodespace).Result()
	}

	// Use max prices as the amount to send to the liquidity pool (i.e. price)
	err := keeper.DepositReserve(ctx, bond.BondDid, buyerAddr, msg.MaxPrices)
	if err != nil {
		return err.Result()
	}

	// Mint bond tokens
	err = keeper.SupplyKeeper.MintCoins(ctx, types.BondsMintBurnAccount,
		sdk.Coins{msg.Amount})
	if err != nil {
		return err.Result()
	}

	// Send bond tokens to buyer
	err = keeper.SupplyKeeper.SendCoinsFromModuleToAccount(ctx,
		types.BondsMintBurnAccount, buyerAddr, sdk.Coins{msg.Amount})
	if err != nil {
		return err.Result()
	}

	// Update supply
	keeper.SetCurrentSupply(ctx, bond.BondDid, bond.CurrentSupply.Add(msg.Amount))

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeInitSwapper,
			sdk.NewAttribute(types.AttributeKeyBondDid, msg.BondDid),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyChargedPrices, msg.MaxPrices.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.BuyerDid),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func handleMsgSell(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgSell) sdk.Result {
	sellerAddr := keeper.DidKeeper.MustGetDidDoc(ctx, msg.SellerDid).Address()

	bond, found := keeper.GetBond(ctx, msg.BondDid)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, msg.BondDid).Result()
	}

	// Check sells allowed, current state is OPEN, and order limits not exceeded
	if !bond.AllowSells {
		return types.ErrBondDoesNotAllowSelling(types.DefaultCodespace).Result()
	} else if bond.State != types.OpenState {
		return types.ErrInvalidStateForAction(types.DefaultCodespace).Result()
	} else if bond.AnyOrderQuantityLimitsExceeded(sdk.Coins{msg.Amount}) {
		return types.ErrOrderQuantityLimitExceeded(types.DefaultCodespace).Result()
	}

	// Check that bond token used belongs to this bond
	if msg.Amount.Denom != bond.Token {
		return types.ErrBondTokenDoesNotMatchBond(types.DefaultCodespace).Result()
	}

	// Send coins to be burned from seller (enforces sellAmount <= balance)
	err := keeper.SupplyKeeper.SendCoinsFromAccountToModule(ctx, sellerAddr,
		types.BondsMintBurnAccount, sdk.Coins{msg.Amount})
	if err != nil {
		return err.Result()
	}

	// Burn bond tokens to be sold
	err = keeper.SupplyKeeper.BurnCoins(ctx, types.BondsMintBurnAccount,
		sdk.Coins{msg.Amount})
	if err != nil {
		return err.Result()
	}

	// Create order
	order := types.NewSellOrder(msg.SellerDid, msg.Amount)

	// Get sell price and check if can add sell order to batch
	buyPrices, sellPrices, err := keeper.GetUpdatedBatchPricesAfterSell(ctx, bond.BondDid, order)
	if err != nil {
		return err.Result()
	}

	// Add sell order to batch
	keeper.AddSellOrder(ctx, bond.BondDid, order, buyPrices, sellPrices)

	//// Cancel unfulfillable orders (Note: no need)
	//keeper.CancelUnfulfillableOrders(ctx, bond.BondDid)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSell,
			sdk.NewAttribute(types.AttributeKeyBondDid, msg.BondDid),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.Amount.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.SellerDid),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func handleMsgSwap(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgSwap) sdk.Result {
	swapperAddr := keeper.DidKeeper.MustGetDidDoc(ctx, msg.SwapperDid).Address()

	bond, found := keeper.GetBond(ctx, msg.BondDid)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, msg.BondDid).Result()
	}

	// Confirm that function type is swapper_function and state is OPEN
	if bond.FunctionType != types.SwapperFunction {
		return types.ErrFunctionNotAvailableForFunctionType(types.DefaultCodespace).Result()
	} else if bond.State != types.OpenState {
		return types.ErrInvalidStateForAction(types.DefaultCodespace).Result()
	}

	// Check that from and to use reserve token names
	fromAndTo := sdk.NewCoins(msg.From, sdk.NewCoin(msg.ToToken, sdk.OneInt()))
	fromAndToDenoms := msg.From.Denom + "," + msg.ToToken
	if !bond.ReserveDenomsEqualTo(fromAndTo) {
		return types.ErrReserveDenomsMismatch(types.DefaultCodespace, fromAndToDenoms, bond.ReserveTokens).Result()
	}

	// Check if order quantity limit exceeded
	if bond.AnyOrderQuantityLimitsExceeded(sdk.Coins{msg.From}) {
		return types.ErrOrderQuantityLimitExceeded(types.DefaultCodespace).Result()
	}

	// Take coins to be swapped from swapper (enforces swapAmount <= balance)
	err := keeper.SupplyKeeper.SendCoinsFromAccountToModule(ctx, swapperAddr,
		types.BatchesIntermediaryAccount, sdk.Coins{msg.From})
	if err != nil {
		return err.Result()
	}

	// Create order
	order := types.NewSwapOrder(msg.SwapperDid, msg.From, msg.ToToken)

	// Add swap order to batch
	keeper.AddSwapOrder(ctx, bond.BondDid, order)

	//// Cancel unfulfillable orders (Note: no need)
	//keeper.CancelUnfulfillableOrders(ctx, bond.BondDid)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSwap,
			sdk.NewAttribute(types.AttributeKeyBondDid, bond.BondDid),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.From.Amount.String()),
			sdk.NewAttribute(types.AttributeKeySwapFromToken, msg.From.Denom),
			sdk.NewAttribute(types.AttributeKeySwapToToken, msg.ToToken),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.SwapperDid),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func handleMsgMakeOutcomePayment(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgMakeOutcomePayment) sdk.Result {
	senderAddr := keeper.DidKeeper.MustGetDidDoc(ctx, msg.SenderDid).Address()

	bond, found := keeper.GetBond(ctx, msg.BondDid)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, msg.BondDid).Result()
	}

	// Confirm that state is OPEN and that outcome payment is not nil
	if bond.State != types.OpenState {
		return types.ErrInvalidStateForAction(types.DefaultCodespace).Result()
	} else if bond.OutcomePayment.Empty() {
		return types.ErrCannotMakeZeroOutcomePayment(types.DefaultCodespace).Result()
	}

	// Send outcome payment to reserve
	err := keeper.DepositReserve(ctx, bond.BondDid, senderAddr, bond.OutcomePayment)
	if err != nil {
		return err.Result()
	}

	// Set bond state to SETTLE
	keeper.SetBondState(ctx, bond.BondDid, types.SettleState)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeMakeOutcomePayment,
			sdk.NewAttribute(types.AttributeKeyBondDid, msg.BondDid),
			sdk.NewAttribute(types.AttributeKeyAddress, senderAddr.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.SenderDid),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func handleMsgWithdrawShare(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgWithdrawShare) sdk.Result {
	recipientAddr := keeper.DidKeeper.MustGetDidDoc(ctx, msg.RecipientDid).Address()

	bond, found := keeper.GetBond(ctx, msg.BondDid)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, msg.BondDid).Result()
	}

	// Check that state is SETTLE
	if bond.State != types.SettleState {
		return types.ErrInvalidStateForAction(types.DefaultCodespace).Result()
	}

	// Get number of bond tokens owned by the recipient
	bondTokensOwnedAmount := keeper.BankKeeper.GetCoins(ctx, recipientAddr).AmountOf(bond.Token)
	if bondTokensOwnedAmount.IsZero() {
		return types.ErrNoBondTokensOwned(types.DefaultCodespace).Result()
	}
	bondTokensOwned := sdk.NewCoin(bond.Token, bondTokensOwnedAmount)

	// Send coins to be burned from recipient
	err := keeper.SupplyKeeper.SendCoinsFromAccountToModule(
		ctx, recipientAddr, types.BondsMintBurnAccount, sdk.NewCoins(bondTokensOwned))
	if err != nil {
		return err.Result()
	}

	// Burn bond tokens
	err = keeper.SupplyKeeper.BurnCoins(ctx, types.BondsMintBurnAccount,
		sdk.NewCoins(sdk.NewCoin(bond.Token, bondTokensOwnedAmount)))
	if err != nil {
		return err.Result()
	}

	// Calculate amount owned
	remainingReserve := keeper.GetReserveBalances(ctx, bond.BondDid)
	bondTokensShare := bondTokensOwnedAmount.ToDec().QuoInt(bond.CurrentSupply.Amount)
	reserveOwedDec := sdk.NewDecCoins(remainingReserve).MulDec(bondTokensShare)
	reserveOwed, _ := reserveOwedDec.TruncateDecimal()

	// Send coins owed to recipient
	err = keeper.WithdrawReserve(ctx, bond.BondDid, recipientAddr, reserveOwed)
	if err != nil {
		return err.Result()
	}

	// Update supply
	keeper.SetCurrentSupply(ctx, bond.BondDid, bond.CurrentSupply.Sub(bondTokensOwned))

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeWithdrawShare,
			sdk.NewAttribute(types.AttributeKeyBondDid, msg.BondDid),
			sdk.NewAttribute(types.AttributeKeyAddress, recipientAddr.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, reserveOwed.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.RecipientDid),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}
