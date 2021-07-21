package pgslap

import "time"

const (
	ThrottleInterrupt = 1 * time.Millisecond
)

func loopWithThrottle(rate int, proc func(i int) (bool, error)) error {
	orgLimit := time.Duration(0)

	if rate > 0 {
		// XXX: Add 1 to get closer to the actual rate...
		orgLimit = time.Second / time.Duration(rate+1)
	}

	thrInt := time.NewTicker(ThrottleInterrupt)
	defer thrInt.Stop()
	blockStart := time.Now()
	currLimit := orgLimit
	var txCnt int64
	thrStart := time.Now()

	for i := 0; ; i++ {
		cont, err := proc(i)

		if !cont || err != nil {
			return err
		}

		txCnt++

		select {
		case <-thrInt.C:
			thrEnd := time.Now()
			procElapsed := thrEnd.Sub(thrStart)
			actualLimit := procElapsed / time.Duration(txCnt)
			currLimit += (orgLimit - actualLimit)

			if currLimit < 0 {
				currLimit = 0
			}

			thrStart = thrEnd
			txCnt = 0
		default:
			// Nothing to do
		}

		blockEnd := time.Now()
		time.Sleep(currLimit - blockEnd.Sub(blockStart))
		blockStart = time.Now()
	}
}
