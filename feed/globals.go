package feed

import (
	"runtime"
	"time"

	"github.com/mmcdole/gofeed"
)

const parseFeedGracePeriod = 10 * time.Minute
const telegramMessageMaxLength = 4096

var libParser = gofeed.NewParser()
var fetchFeedsMaxConcurrency = runtime.NumCPU() * 10
