package feed

import (
	"net/http"
	"regexp"
	"runtime"
	"time"

	"github.com/mmcdole/gofeed"
)

const (
	parseFeedGracePeriod            = 10 * time.Minute
	telegramMessageMaxLength        = 4096
	TelegramHost                    = "t.me"
	fallbackTelegramSummaryMaxChars = 200
)

var (
	libParser                = gofeed.NewParser()
	fetchFeedsMaxConcurrency = runtime.NumCPU() * 10
	telegramClient           = &http.Client{Timeout: 20 * time.Second}
	telegramSlugRe           = regexp.MustCompile(`^\w{5,32}$`)
	telegramAtSignSlugRe     = regexp.MustCompile(`(\s|^)@(\w{5,32})(\s|$)`)
	userAgent                = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36"
)
