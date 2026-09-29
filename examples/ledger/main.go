package main

import (
	"errors"
	"fmt"

	"github.com/wallyqs/zizek.go"
)

const limit int64 = 1_000_000
const portableLedgerFailure = "AXicVYyxCQAgEAMvYmXtILr/cipq+C/ChRykA4K2w4eohyMuWU0ryUWvXBSLdK4FQmgBCA=="

type ledger struct {
	balances map[string]int64
}

func newLedger() ledger {
	return ledger{balances: make(map[string]int64)}
}

func (l *ledger) credit(account string, amount int64) {
	l.balances[account] += amount
}

func (l *ledger) debit(account string, amount int64) {
	l.balances[account] -= amount
}

func (l *ledger) transfer(from, to string, amount int64) {
	fromBalance := l.balances[from]
	if from != to && amount-fromBalance <= 1 {
		l.debit(from, amount)
		l.credit(to, amount)
	}
}

type ledgerTest struct {
	ledger   ledger
	accounts *zizek.Pool[string]
}

func (state *ledgerTest) createAccount(tc *zizek.TestCase) error {
	account, err := zizek.Draw(tc, zizek.Text().MinSize(1))
	if err != nil {
		return err
	}
	tc.Note(fmt.Sprintf("create account %q", account))
	return state.accounts.Add(tc, account)
}

func (state *ledgerTest) credit(tc *zizek.TestCase) error {
	account, err := zizek.Draw(tc, state.accounts.ValuesReusable())
	if err != nil {
		return err
	}
	amount, err := zizek.Draw(
		tc,
		zizek.Integers().MinValue(0).MaxValue(limit),
	)
	if err != nil {
		return err
	}
	tc.Note(fmt.Sprintf("credit %q with %d", *account, amount))
	state.ledger.credit(*account, amount)
	return nil
}

func (state *ledgerTest) transfer(tc *zizek.TestCase) error {
	from, err := zizek.Draw(tc, state.accounts.ValuesReusable())
	if err != nil {
		return err
	}
	to, err := zizek.Draw(tc, state.accounts.ValuesReusable())
	if err != nil {
		return err
	}
	amount, err := zizek.Draw(
		tc,
		zizek.Integers().MinValue(0).MaxValue(limit),
	)
	if err != nil {
		return err
	}
	tc.Note(fmt.Sprintf("transfer %d from %q to %q", amount, *from, *to))
	state.ledger.transfer(*from, *to, amount)
	return nil
}

func (state *ledgerTest) nonnegativeBalances(_ *zizek.TestCase) error {
	for account, balance := range state.ledger.balances {
		if balance < 0 {
			return fmt.Errorf(
				"account %q has negative balance %d",
				account,
				balance,
			)
		}
	}
	return nil
}

func property(tc *zizek.TestCase) error {
	state := ledgerTest{
		ledger:   newLedger(),
		accounts: zizek.NewPool[string](tc),
	}
	machine := zizek.Machine[ledgerTest]{
		Rules: []zizek.Rule[ledgerTest]{
			{Name: "create_account", Apply: (*ledgerTest).createAccount},
			{Name: "credit", Apply: (*ledgerTest).credit},
			{Name: "transfer", Apply: (*ledgerTest).transfer},
		},
		Invariants: []zizek.Rule[ledgerTest]{
			{
				Name:  "nonnegative_balances",
				Apply: (*ledgerTest).nonnegativeBalances,
			},
		},
	}
	return zizek.Run(&state, machine, tc)
}

func findAndReplayBug() (*zizek.Failure, error) {
	err := zizek.Check(property, zizek.Config{
		Examples: 2_000,
		Seed:     9,
	})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("ledger bug was not found: %w", err)
	}
	if err := zizek.Check(
		property,
		zizek.Config{Reproduce: failure.Blob},
	); !errors.As(err, &failure) {
		return nil, fmt.Errorf("ledger failure did not replay: %w", err)
	}
	return failure, nil
}

func main() {
	failure, err := findAndReplayBug()
	if err != nil {
		panic(err)
	}
	fmt.Println(failure)
}
