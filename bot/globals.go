package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

const updateTimeout = 60

const welcomeText = `🤖 *Welcome to Telekilogram\!*

I'm your feed assistant\. I can help you:

– Follow RSS/Atom feeds and public Telegram channels by sending me URLs
– Get feed list with /list
– Unfollow feeds directly from list
– Receive 24h auto\-digest daily automatically \(default \- 00:00 UTC\)
– Receive 24h digest with /digest
– Configure user settings with /settings`

const settingsText = `*⚙️ Settings*

Current UTC time is %s\.

Current auto\-digest hour \(UTC\) setting is %s\.

You can choose different setting below:`

const filterText = `Telekilogram does not support filtering\.\.\.

But you can use awesome [siftrss](https://siftrss.com/) instead\! ✨
It's totally great\. Bot author is also using it\.`

var (
	menuKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📄 Feed list", "menu_list"),
			tgbotapi.NewInlineKeyboardButtonData("👈 24h digest", "menu_digest"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Settings", "menu_settings"),
		},
	}
	returnKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.NewInlineKeyboardButtonData("⬅️ Return to menu", "menu")},
	}
	settingsAutoDigestHourUTCKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData(
				"00",
				"settings_auto_digest_hour_utc_0",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"01",
				"settings_auto_digest_hour_utc_1",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"02",
				"settings_auto_digest_hour_utc_2",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"03",
				"settings_auto_digest_hour_utc_3",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"04",
				"settings_auto_digest_hour_utc_4",
			),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(
				"05",
				"settings_auto_digest_hour_utc_5",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"06",
				"settings_auto_digest_hour_utc_6",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"07",
				"settings_auto_digest_hour_utc_7",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"08",
				"settings_auto_digest_hour_utc_8",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"09",
				"settings_auto_digest_hour_utc_9",
			),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(
				"10",
				"settings_auto_digest_hour_utc_10",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"11",
				"settings_auto_digest_hour_utc_11",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"12",
				"settings_auto_digest_hour_utc_12",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"13",
				"settings_auto_digest_hour_utc_13",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"14",
				"settings_auto_digest_hour_utc_14",
			),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(
				"15",
				"settings_auto_digest_hour_utc_15",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"16",
				"settings_auto_digest_hour_utc_16",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"17",
				"settings_auto_digest_hour_utc_17",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"18",
				"settings_auto_digest_hour_utc_18",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"19",
				"settings_auto_digest_hour_utc_19",
			),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(
				"20",
				"settings_auto_digest_hour_utc_20",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"21",
				"settings_auto_digest_hour_utc_21",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"22",
				"settings_auto_digest_hour_utc_22",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"23",
				"settings_auto_digest_hour_utc_23",
			),
		},
		returnKeyboard[0],
	}
)
