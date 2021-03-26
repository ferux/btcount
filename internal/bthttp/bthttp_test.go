package bthttp

import (
	"log"
	"os"
	"testing"

	"github.com/ferux/btcount/internal/bttest"
)

func TestMain(m *testing.M) {
	err := bttest.Prepare()
	if err != nil {
		log.Fatal(err)
	}

	code := m.Run()

	err = bttest.Finish()
	if err != nil {
		log.Print(err)
	}

	os.Exit(code)
}
