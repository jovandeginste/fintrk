package main

import (
	"github.com/spf13/cobra"
)

func (a *App) RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Short: "track your investments",
	}

	cmd.AddCommand(a.UpdateValuationsCmd())
	cmd.AddCommand(a.UpdateSharesCmd())
	cmd.AddCommand(a.ShowCmd())
	cmd.AddCommand(a.CreateTransactionCmd())
	cmd.AddCommand(a.AddISINCmd())

	return cmd
}

func (a *App) AddISINCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-isin",
		Short: "add ISIN code to start tracking",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, i := range args {
				if err := a.DB().AddOrUpdateISIN(i); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func (a *App) ShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "show current state of tracked funds",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.ShowCurrentState()
		},
	}
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
	t := Transaction{}
	d := ""

	cmd := &cobra.Command{
		Use:   "new-transaction",
		Short: "create a new transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := t.SetDate(d); err != nil {
				return err
			}

			if err := a.DB().CreateTransaction(&t); err != nil {
				return err
			}

			a.logger.Info("Created transaction:")
			a.logger.Info(t.String())

			return a.DB().UpdateShares(t.ISIN)
		},
	}

	cmd.Flags().StringVarP(&d, "date", "d", "", "transaction date (YYYY-MM-DD; empty for today)")
	cmd.Flags().StringVarP(&t.ISIN, "isin", "i", "", "ISIN")
	cmd.Flags().Float64VarP(&t.TotalShares, "shares", "s", 0, "total amount of shares")
	cmd.Flags().Float64VarP(&t.TotalValue, "value", "v", 0, "total amount of value")

	cmd.MarkFlagRequired("isin")   //nolint:errcheck
	cmd.MarkFlagRequired("shares") //nolint:errcheck

	return cmd
}
