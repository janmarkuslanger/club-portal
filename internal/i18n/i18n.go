package i18n

const defaultLocale = "de"

const keyAppName = "app.name"

var translations = map[string]map[string]string{
	"de": {
		keyAppName: "Mein Club",
	},
}

func AppName() string {
	return Text(keyAppName)
}

func Text(key string) string {
	return TextForLocale(defaultLocale, key)
}

func TextForLocale(locale, key string) string {
	if values, ok := translations[locale]; ok {
		if value, ok := values[key]; ok {
			return value
		}
	}
	if values, ok := translations[defaultLocale]; ok {
		if value, ok := values[key]; ok {
			return value
		}
	}
	return key
}
