// Copyright (c) 2014 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// NOTE: This file is intended to house the RPC websocket notifications that are
// supported by a wallet server.

package soterjson

const (
	// AccountBalanceNtfnMethod is the method used for account balance
	// notifications.
	AccountBalanceNtfnMethod = "accountbalance"

	// SoterdConnectedNtfnMethod is the method used for notifications when
	// a wallet server is connected to a chain server.
	SoterdConnectedNtfnMethod = "soterdconnected"

	// WalletLockStateNtfnMethod is the method used to notify the lock state
	// of a wallet has changed.
	WalletLockStateNtfnMethod = "walletlockstate"

	// NewTxNtfnMethod is the method used to notify that a wallet server has
	// added a new transaction to the transaction store.
	NewTxNtfnMethod = "newtx"
)

// AccountBalanceNtfn defines the accountbalance JSON-RPC notification.
type AccountBalanceNtfn struct {
	Account   string
	Balance   float64 // In SOTER
	Confirmed bool    // Whether Balance is confirmed or unconfirmed.
}

// NewAccountBalanceNtfn returns a new instance which can be used to issue an
// accountbalance JSON-RPC notification.
func NewAccountBalanceNtfn(account string, balance float64, confirmed bool) *AccountBalanceNtfn {
	return &AccountBalanceNtfn{
		Account:   account,
		Balance:   balance,
		Confirmed: confirmed,
	}
}

// SoterdConnectedNtfn defines the soterdconnected JSON-RPC notification.
type SoterdConnectedNtfn struct {
	Connected bool
}

// NewSoterdConnectedNtfn returns a new instance which can be used to issue a
// soterdconnected JSON-RPC notification.
func NewSoterdConnectedNtfn(connected bool) *SoterdConnectedNtfn {
	return &SoterdConnectedNtfn{
		Connected: connected,
	}
}

// WalletLockStateNtfn defines the walletlockstate JSON-RPC notification.
type WalletLockStateNtfn struct {
	Locked bool
}

// NewWalletLockStateNtfn returns a new instance which can be used to issue a
// walletlockstate JSON-RPC notification.
func NewWalletLockStateNtfn(locked bool) *WalletLockStateNtfn {
	return &WalletLockStateNtfn{
		Locked: locked,
	}
}

// NewTxNtfn defines the newtx JSON-RPC notification.
type NewTxNtfn struct {
	Account string
	Details ListTransactionsResult
}

// NewNewTxNtfn returns a new instance which can be used to issue a newtx
// JSON-RPC notification.
func NewNewTxNtfn(account string, details ListTransactionsResult) *NewTxNtfn {
	return &NewTxNtfn{
		Account: account,
		Details: details,
	}
}

func init() {
	// The commands in this file are only usable with a wallet server via
	// websockets and are notifications.
	flags := UFWalletOnly | UFWebsocketOnly | UFNotification

	MustRegisterCmd(AccountBalanceNtfnMethod, (*AccountBalanceNtfn)(nil), flags)
	MustRegisterCmd(SoterdConnectedNtfnMethod, (*SoterdConnectedNtfn)(nil), flags)
	MustRegisterCmd(WalletLockStateNtfnMethod, (*WalletLockStateNtfn)(nil), flags)
	MustRegisterCmd(NewTxNtfnMethod, (*NewTxNtfn)(nil), flags)
}
