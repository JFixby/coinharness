package memwallet

import (
	"encoding/json"
	"github.com/jfixby/coinharness"
)




// NotificationHandlers defines callback function pointers to invoke with
// notifications.  Since all of the functions are nil by default, all
// notifications are effectively ignored until their handlers are set to a
// concrete callback.
//
// NOTE: Unless otherwise documented, these handlers must NOT directly call any
// blocking calls on the client instance since the input reader goroutine blocks
// until the callback has completed.  Doing so will result in a deadlock
// situation.
type NotificationHandlers struct {
	// OnClientConnected is invoked when the client connects or reconnects
	// to the RPC server.  This callback is run async with the rest of the
	// notification handlers, and is safe for blocking client requests.
	OnClientConnected func()

	// OnBlockConnected is invoked when a block is connected to the longest
	// (best) chain.  It will only be invoked if a preceding call to
	// NotifyBlocks has been made to register for the notification and the
	// function is non-nil.
	OnBlockConnected func(blockHeader []byte, transactions [][]byte)

	// OnBlockDisconnected is invoked when a block is disconnected from the
	// longest (best) chain.  It will only be invoked if a preceding call to
	// NotifyBlocks has been made to register for the notification and the
	// function is non-nil.
	OnBlockDisconnected func(blockHeader []byte)

	// OnRelevantTxAccepted is invoked when an unmined transaction passes
	// the client's transaction filter.
	OnRelevantTxAccepted func(transaction []byte)

	// OnReorganization is invoked when the blockchain begins reorganizing.
	// It will only be invoked if a preceding call to NotifyBlocks has been
	// made to register for the notification and the function is non-nil.
	OnReorganization func(oldHash coinharness.Hash, oldHeight int32,
		newHash coinharness.Hash, newHeight int32)

	// OnWinningTickets is invoked when a block is connected and eligible tickets
	// to be voted on for this chain are given.  It will only be invoked if a
	// preceding call to NotifyWinningTickets has been made to register for the
	// notification and the function is non-nil.
	OnWinningTickets func(blockHash coinharness.Hash,
		blockHeight int64,
		tickets []coinharness.Hash)

	// OnSpentAndMissedTickets is invoked when a block is connected to the
	// longest (best) chain and tickets are spent or missed.  It will only be
	// invoked if a preceding call to NotifySpentAndMissedTickets has been made to
	// register for the notification and the function is non-nil.
	OnSpentAndMissedTickets func(hash coinharness.Hash,
		height int64,
		stakeDiff int64,
		tickets map[coinharness.Hash]bool)

	// OnNewTickets is invoked when a block is connected to the longest (best)
	// chain and tickets have matured to become active.  It will only be invoked
	// if a preceding call to NotifyNewTickets has been made to register for the
	// notification and the function is non-nil.
	OnNewTickets func(hash coinharness.Hash,
		height int64,
		stakeDiff int64,
		tickets []coinharness.Hash)

	// OnStakeDifficulty is invoked when a block is connected to the longest
	// (best) chain and a new stake difficulty is calculated.  It will only
	// be invoked if a preceding call to NotifyStakeDifficulty has been
	// made to register for the notification and the function is non-nil.
	OnStakeDifficulty func(hash coinharness.Hash,
		height int64,
		stakeDiff int64)

	// OnTxAccepted is invoked when a transaction is accepted into the
	// memory pool.  It will only be invoked if a preceding call to
	// NotifyNewTransactions with the verbose flag set to false has been
	// made to register for the notification and the function is non-nil.
	OnTxAccepted func(hash coinharness.Hash, amount coinharness.CoinsAmount)

	// OnTxAccepted is invoked when a transaction is accepted into the
	// memory pool.  It will only be invoked if a preceding call to
	// NotifyNewTransactions with the verbose flag set to true has been
	// made to register for the notification and the function is non-nil.
	//OnTxAcceptedVerbose func(txDetails *dcrjson.TxRawResult)

	// OnDcrdConnected is invoked when a wallet connects or disconnects from
	// dcrd.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnDcrdConnected func(connected bool)

	// OnAccountBalance is invoked with account balance updates.
	//
	// This will only be available when speaking to a wallet server
	// such as dcrwallet.
	OnAccountBalance func(account string, balance coinharness.CoinsAmount, confirmed bool)

	// OnWalletLockState is invoked when a wallet is locked or unlocked.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnWalletLockState func(locked bool)

	// OnTicketsPurchased is invoked when a wallet purchases an SStx.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnTicketsPurchased func(TxHash coinharness.Hash, amount coinharness.CoinsAmount)

	// OnVotesCreated is invoked when a wallet generates an SSGen.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnVotesCreated func(txHash coinharness.Hash,
		blockHash coinharness.Hash,
		height int32,
		sstxIn coinharness.Hash,
		voteBits uint16)

	// OnRevocationsCreated is invoked when a wallet generates an SSRtx.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnRevocationsCreated func(txHash coinharness.Hash,
		sstxIn coinharness.Hash)

	// OnUnknownNotification is invoked when an unrecognized notification
	// is received.  This typically means the notification handling code
	// for this package needs to be updated for a new notification type or
	// the caller is using a custom notification this package does not know
	// about.
	OnUnknownNotification func(method string, params []json.RawMessage)
}
