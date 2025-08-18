package types_test

import (
	"os"
	"testing"

	"github.com/istchain/istchain/app"
)

func TestMain(m *testing.M) {
	app.SetSDKConfig()
	os.Exit(m.Run())
}
