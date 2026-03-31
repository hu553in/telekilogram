package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Token        string            `env:"TOKEN,required,notEmpty"`
	AllowedUsers []int64           `env:"ALLOWED_USERS"`
	DBPath       string            `env:"DB_PATH"                 envDefault:"db.sqlite"`
	OpenAIAPIKey string            `env:"OPENAI_API_KEY"`
	OpenAI       OpenAIConfig      `                                                     envPrefix:"OPENAI_"`
	Scheduler    SchedulerConfig   `                                                     envPrefix:"SCHEDULER_"`
	RateLimiter  RateLimiterConfig `                                                     envPrefix:"RATE_LIMITER_"`
	Feed         FeedConfig        `                                                     envPrefix:"FEED_"`
	Telegram     TelegramConfig    `                                                     envPrefix:"TELEGRAM_"`
	Bot          BotConfig         `                                                     envPrefix:"BOT_"`
}

type OpenAIConfig struct {
	BaseMaxOutputTokens         int64  `env:"BASE_MAX_OUTPUT_TOKENS"          envDefault:"512"`
	LimitMaxOutputTokens        int64  `env:"LIMIT_MAX_OUTPUT_TOKENS"         envDefault:"2048"`
	MaxOutputTokensGrowthFactor int64  `env:"MAX_OUTPUT_TOKENS_GROWTH_FACTOR" envDefault:"2"`
	SystemPrompt                string `env:"SYSTEM_PROMPT"                   envDefault:"Summarize the Telegram post in one ultra-short sentence.\n\nRules:\n- ≤25 words (hard limit 40).\n- Include only core idea and critical context (dates, numbers, names, calls to action).\n- No lists, no examples — compress into one general statement.\n- Neutral tone.\n- Remove fillers, emojis, hashtags, links unless essential.\n- Output exactly one line in the same language as the input."`
	AIModel                     string `env:"AI_MODEL"                        envDefault:"gpt-5.4-nano"`
	ServiceTier                 string `env:"SERVICE_TIER"                    envDefault:"flex"`
	ReasoningEffort             string `env:"REASONING_EFFORT"                envDefault:"low"`
}

type SchedulerConfig struct {
	CheckHourFeedsTimeout time.Duration `env:"CHECK_HOUR_FEEDS_TIMEOUT" envDefault:"15m"`
}

type RateLimiterConfig struct {
	PrivateChatRate time.Duration `env:"PRIVATE_CHAT_RATE" envDefault:"1s"`
	GroupChatRate   time.Duration `env:"GROUP_CHAT_RATE"   envDefault:"3s"`
	QueueSize       int           `env:"QUEUE_SIZE"        envDefault:"1000"`
}

type FeedConfig struct {
	TelegramSummaryCacheMaxEntries       int           `env:"TELEGRAM_SUMMARY_CACHE_MAX_ENTRIES"        envDefault:"1024"`
	TelegramSummariesMaxParallelism      int           `env:"TELEGRAM_SUMMARIES_MAX_PARALLELISM"        envDefault:"4"`
	ParseFeedGracePeriod                 time.Duration `env:"PARSE_FEED_GRACE_PERIOD"                   envDefault:"10m"`
	FallbackTelegramSummaryMaxChars      int           `env:"FALLBACK_TELEGRAM_SUMMARY_MAX_CHARS"       envDefault:"200"`
	FetchFeedsMaxConcurrencyGrowthFactor int           `env:"FETCH_FEEDS_MAX_CONCURRENCY_GROWTH_FACTOR" envDefault:"10"`
}

type TelegramConfig struct {
	UserAgent     string        `env:"USER_AGENT"     envDefault:"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36"`
	ClientTimeout time.Duration `env:"CLIENT_TIMEOUT" envDefault:"20s"`
}

type BotConfig struct {
	UpdateProcessingTimeout time.Duration `env:"UPDATE_PROCESSING_TIMEOUT" envDefault:"60s"`
	IssueURL                string        `env:"ISSUE_URL"                 envDefault:"https://github.com/hu553in/telekilogram/issues/new"`
}

func LoadConfig() Config {
	return env.Must(env.ParseAs[Config]())
}
