package main

import (
	"time"

	"github.com/spf13/cobra"
)

func (a *App) RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Short: "track your investments",
	}

	cmd.AddCommand(a.UpdateValuationsCmd())
	cmd.AddCommand(a.UpdateSharesCmd())
	cmd.AddCommand(a.ShowCmd())
	cmd.AddCommand(a.ShowAtCmd())
	cmd.AddCommand(a.ShowSinceCmd())
	cmd.AddCommand(a.CreateTransactionCmd())
	cmd.AddCommand(a.AddISINCmd())

	return cmd
}

func (a *App) AddISINCmd() *cobra.Command {
	source := ""

	cmd := &cobra.Command{
		Use:   "add-isin",
		Short: "add ISIN code to start tracking",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, i := range args {
				if err := a.DB().AddOrUpdateISIN(i, source); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&source, "source", "s", "FT", "source to fetch data from")

	return cmd
}

func (a *App) ShowCmd() *cobra.Command {
	var tableFormat string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "show current state of tracked funds",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.ShowCurrentState(tableFormat)
		},
	}

	cmd.Flags().StringVarP(&tableFormat, "format", "f", "ascii", "rendering format (ascii, markdown)")

	return cmd
}

func (a *App) ShowAtCmd() *cobra.Command {
	var tableFormat string

	cmd := &cobra.Command{
		Use:   "show-at",
		Short: "show state of tracked funds at date",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := time.Parse("2006-01-02", args[0])
			if err != nil {
				return err
			}

			return a.ShowStateAt(tableFormat, d)
		},
	}

	cmd.Flags().StringVarP(&tableFormat, "format", "f", "ascii", "rendering format (ascii, markdown)")

	return cmd
}

func (a *App) ShowSinceCmd() *cobra.Command {
	var tableFormat string

	cmd := &cobra.Command{
		Use:   "show-since",
		Short: "show change of tracked funds since date",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := time.Parse("2006-01-02", args[0])
			if err != nil {
				return err
			}

			return a.ShowStateSince(tableFormat, d)
		},
	}

	cmd.Flags().StringVarP(&tableFormat, "format", "f", "ascii", "rendering format (ascii, markdown)")

	return cmd
}

func (a *App) UpdateValuationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "update all tracked funds",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.DB().UpdateValuationsAll()
		},
	}
}

func (a *App) UpdateSharesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update-shares",
		Short: "update share count of a fund",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.DB().UpdateShares(args[0])
		},
	}
}

func (a *App) CreateTransactionCmd() *cobra.Command {
	transaction := Transaction{}
	txDate := ""

	cmd := &cobra.Command{
		Use:   "new-transaction",
		Short: "create a new transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := transaction.SetDate(txDate); err != nil {
				return err
			}

			if err := a.DB().CreateTransaction(&transaction); err != nil {
				return err
			}

			a.logger.Info("Created transaction:")
			a.logger.Info(transaction.String())

			return a.DB().UpdateShares(transaction.ISIN)
		},
	}

	cmd.Flags().StringVarP(&txDate, "date", "d", "", "transaction date (YYYY-MM-DD; empty for today)")
	cmd.Flags().StringVarP(&transaction.ISIN, "isin", "i", "", "ISIN")
	cmd.Flags().Float64VarP(&transaction.TotalShares, "shares", "s", 0, "total amount of shares")
	cmd.Flags().Float64VarP(&transaction.TotalValue, "value", "v", 0, "total amount of value")

	cmd.MarkFlagRequired("isin")   //nolint:errcheck
	cmd.MarkFlagRequired("shares") //nolint:errcheck

	return cmd
}
