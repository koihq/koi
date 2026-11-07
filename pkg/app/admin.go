package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"

	"github.com/koihq/koi/pkg/api"
	"github.com/koihq/koi/pkg/database"
	"github.com/koihq/koi/pkg/domain"
)

const adminUsage = `usage:
  koi admin reset-password <email>        reads the new password from stdin
  koi admin export-account <email>        writes the account's data as JSON to stdout
  koi admin delete-account <email> --yes  deletes the account and its organization`

// RunAdmin runs an administrator's command against the instance's database. It reads only
// KOI_DATABASE_URL, so it also works when the rest of the server's configuration does not.
func RunAdmin(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	_ = godotenv.Load()

	if len(args) < 2 {
		return errors.New(adminUsage)
	}

	databaseURL, err := requireEnv("KOI_DATABASE_URL")
	if err != nil {
		return err
	}

	db, err := database.Open(databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("sql db: %w", err)
	}
	defer sqlDB.Close()

	store := database.NewStore(db)

	sess := scs.New()
	sess.Store = postgresstore.NewWithCleanupInterval(sqlDB, 0)

	command, email := args[0], args[1]

	switch command {
	case "reset-password":
		return adminResetPassword(ctx, store, sess, email, stdin, stdout)
	case "export-account":
		return adminExportAccount(ctx, store, email, stdout)
	case "delete-account":
		if len(args) != 3 || args[2] != "--yes" {
			return errors.New("delete-account is permanent: repeat it with --yes")
		}

		return adminDeleteAccount(ctx, store, sess, email, stdout)
	default:
		return errors.New(adminUsage)
	}
}

func adminResetPassword(ctx context.Context, store domain.Store, sess *scs.SessionManager, email string, stdin io.Reader, stdout io.Writer) error {
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read password: %w", err)
	}

	commander := domain.NewAuthCommander(store, domain.AuthOptions{})

	userID, err := commander.SetPassword(ctx, email, strings.TrimRight(line, "\r\n"))
	if err != nil {
		return err
	}

	// The password has already changed, so say so even if signing out the sessions fails.
	if err := api.EndUserSessions(ctx, sess, userID); err != nil {
		return fmt.Errorf("password changed, but sessions were not ended: %w", err)
	}

	_, err = fmt.Fprintln(stdout, "Password changed. Sessions and connected agents were signed out.")

	return err
}

func adminExportAccount(ctx context.Context, store domain.Store, email string, stdout io.Writer) error {
	export, err := domain.ExportAccount(ctx, store, email)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")

	return encoder.Encode(export)
}

func adminDeleteAccount(ctx context.Context, store domain.Store, sess *scs.SessionManager, email string, stdout io.Writer) error {
	userID, customerID, err := domain.DeleteAccount(ctx, store, email)
	if err != nil {
		return err
	}

	if err := api.EndUserSessions(ctx, sess, userID); err != nil {
		return fmt.Errorf("account deleted, but sessions were not ended: %w", err)
	}

	if _, err := fmt.Fprintln(stdout, "Account deleted."); err != nil {
		return err
	}

	if customerID == "" {
		return nil
	}

	_, err = fmt.Fprintf(stdout, "Delete Stripe customer %s in the Stripe dashboard to finish.\n", customerID)

	return err
}
