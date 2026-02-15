package utils

import (
	"bufio"
	"errors"
	"io"
	"os"
	"time"
)

func TailPollUntilIdle(path string, out io.Writer, idle, pollEvery time.Duration) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	reader := bufio.NewReader(f)
	lastActivity := time.Now()

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			_, err = out.Write(line)
			if err != nil {
				return err
			}
		}

		if err == io.EOF {
			if time.Since(lastActivity) > idle {
				return nil
			}

			time.Sleep(pollEvery)
			continue
		}

		if err != nil {
			return err
		}

		lastActivity = time.Now()
	}
}
