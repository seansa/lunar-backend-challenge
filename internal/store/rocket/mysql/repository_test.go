package mysql

import (
	"strings"
	"testing"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
)

func TestUpsertBindsEveryRocketColumn(t *testing.T) {
	rocket := domain.Rocket{Channel: "rocket-1"}
	args := rocketArgs(rocket)

	if got, want := len(args), 13; got != want {
		t.Fatalf("rocketArgs() returned %d arguments, want %d", got, want)
	}

	if got, want := strings.Count(upsertRocket, "?"), 25; got != want {
		t.Fatalf("upsertRocket has %d placeholders, want %d", got, want)
	}

	updateArgs := append([]any{}, args[1:]...)
	boundArgs := append(args, updateArgs...)
	if got, want := len(boundArgs), strings.Count(upsertRocket, "?"); got != want {
		t.Fatalf("upsert binds %d arguments for %d placeholders", got, want)
	}
}
