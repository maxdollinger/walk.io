package main

import (
	"context"
	"fmt"
	"os"

	"github.com/maxdollinger/walk.io/internal/db"
)

func main() {
	walkDB, err := db.NewDB("/var/lib/walkio/walk.db")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err = db.InitSchema(ctx, walkDB); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
